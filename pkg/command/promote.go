package command

import (
	"context"
	"fmt"
	"log"

	"github.com/fluxcd/image-automation-controller/pkg/update"
	imagev1alpha1_reflect "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PromoteCommand(ctx context.Context, providerType string, path, token string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", fmt.Errorf("could not get configuration: %w", err)
	}
	repo, err := getRepository(ctx, providerType, path, token)
	if err != nil {
		return "", fmt.Errorf("could not get repository: %w", err)
	}
	pr, err := repo.GetPRThatCausedCurrentCommit(ctx)
	if err != nil {
		//lint:ignore nilerr should not return error
		return "skipping PR creation as commit does not originate from promotion PR", nil
	}
	return promote(ctx, cfg, repo, &pr.State)
}

func promote(ctx context.Context, cfg config.Config, repo *git.Repository, state *git.PRState) (string, error) {
	// Check if there is a next env or get next env
	if state.Env == "" {
		state.Env = cfg.Environments[0].Name
	} else {
		if !cfg.HasNextEnvironment(state.Env) {
			return "no next environment to promote to", nil
		}
		nextEnv, err := cfg.NextEnvironment(state.Env)
		if err != nil {
			return "", fmt.Errorf("could not get next environment: %w", err)
		}
		state.Env = nextEnv.Name
	}

	// Set sha to be included in the next PR
	headID, err := repo.GetCurrentCommit()
	if err != nil {
		return "", fmt.Errorf("could not get latest commit: %w", err)
	}
	state.Sha = headID.String()

	// Update image tag
	manifestPath := fmt.Sprintf("%s/%s/%s", repo.GetRootDir(), state.Group, state.Env)
	err = updateImageTag(manifestPath, state.App, state.Group, state.Tag)
	if err != nil {
		return "", fmt.Errorf("failed updating manifests: %w", err)
	}

	// Push and create PR
	err = repo.CreateBranch(state.BranchName(), true)
	if err != nil {
		return "", fmt.Errorf("could not create branch: %w", err)
	}
	_, err = repo.CreateCommit(state.BranchName(), state.Title())
	if err != nil {
		return "", fmt.Errorf("could not commit changes: %w", err)
	}
	err = repo.Push(state.BranchName(), true)
	if err != nil {
		return "", fmt.Errorf("could not push changes: %w", err)
	}
	auto, err := cfg.IsEnvironmentAutomated(state.Env)
	if err != nil {
		return "", fmt.Errorf("could not get environment automation state: %w", err)
	}
	err = repo.CreatePR(ctx, state.BranchName(), auto, state)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %w", err)
	}
	return "created promotions pull request", nil
}

func updateImageTag(path, app, group, tag string) error {
	policies := []imagev1alpha1_reflect.ImagePolicy{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app,
				Namespace: group,
			},
			Status: imagev1alpha1_reflect.ImagePolicyStatus{
				LatestImage: fmt.Sprintf("%s:%s", app, tag),
			},
		},
	}

	log.Printf("Updating images with %s:%s:%s in %s\n", group, app, tag, path)
	_, err := update.UpdateWithSetters(path, path, policies)
	if err != nil {
		return fmt.Errorf("failed updating manifests: %w", err)
	}

	return nil
}
