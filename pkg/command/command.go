package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fluxcd/image-automation-controller/pkg/update"
	imagev1alpha1_reflect "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func getConfig(path string) (config.Config, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("could not load config: %v", err)
	}
	return cfg, nil
}

func getRepository(ctx context.Context, path, token string) (*git.Repository, error) {
	repo, err := git.LoadRepository(ctx, path, git.ProviderTypeAzdo, token)
	if err != nil {
		return nil, fmt.Errorf("could not load repository: %v", err)
	}
	return repo, nil
}

func NewCommand(ctx context.Context, path, token, group, app, tag string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", err
	}
	repo, err := getRepository(ctx, path, token)
	if err != nil {
		return "", err
	}
	state := git.PRState{
		Env:   "",
		Group: group,
		App:   app,
		Tag:   tag,
		Sha:   "",
	}
	return promote(ctx, cfg, repo, state)
}

func PromoteCommand(ctx context.Context, path, token string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", fmt.Errorf("could not get configuration: %v", err)
	}
	repo, err := getRepository(ctx, path, token)
	if err != nil {
		return "", fmt.Errorf("could not get repository: %v", err)
	}
	pr, err := repo.GetPRThatCausedCurrentCommit(ctx)
	if err != nil {
		return "skipping PR creation as commit does not originate from promotion PR", nil
	}
	return promote(ctx, cfg, repo, pr.State)
}

func promote(ctx context.Context, cfg config.Config, repo *git.Repository, state git.PRState) (string, error) {
	// Check if there is a next env or get next env
	if state.Env == "" {
		state.Env = cfg.Environments[0].Name
	} else {
		if !cfg.HasNextEnvironment(state.Env) {
			return "no next environment to promote to", nil
		}
		nextEnv, err := cfg.NextEnvironment(state.Env)
		if err != nil {
			return "", fmt.Errorf("could not get next environment: %v", err)
		}
		state.Env = nextEnv.Name
	}

	// Set sha to be included in the next PR
	headID, err := repo.GetCurrentCommit()
	if err != nil {
		return "", fmt.Errorf("could not get latest commit: %v", err)
	}
	state.Sha = headID.String()

	// Update image tag
	manifestPath := fmt.Sprintf("%s/%s/%s", repo.GetRootDir(), state.Group, state.Env)
	err = updateImageTag(manifestPath, state.App, state.Group, state.Tag)
	if err != nil {
		return "", fmt.Errorf("failed updating manifests: %v", err)
	}

	// Push and create PR
	err = repo.CreateBranch(state.BranchName(), true)
	if err != nil {
		return "", fmt.Errorf("could not create branch: %v", err)
	}
	_, err = repo.CreateCommit(state.BranchName(), state.Title())
	if err != nil {
		return "", fmt.Errorf("could not commit changes: %v", err)
	}
	err = repo.Push(state.BranchName())
	if err != nil {
		return "", fmt.Errorf("could not push changes: %v", err)
	}
	auto, err := cfg.IsEnvironmentAutomated(state.Env)
	if err != nil {
		return "", fmt.Errorf("could not get environment automation state: %v", err)
	}
	err = repo.CreatePR(ctx, state.BranchName(), auto, state)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %v", err)
	}
	return "created promotions pull request", nil
}

func StatusCommand(ctx context.Context, path, token string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", err
	}
	repo, err := getRepository(ctx, path, token)
	if err != nil {
		return "", err
	}

	// If branch does not contain promote it was manual, return early
	branchName, err := repo.GetBranchName()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(branchName, git.PromoteBranch) {
		return "Promotion was manual, skipping check", nil
	}

	// get current pr
	pr, err := repo.GetPRForCurrentBranch(ctx)
	if err != nil {
		return "", fmt.Errorf("failed getting pr for current branch: %v", err)
	}

	// Skip the status check if this is the first environment
	if cfg.Environments[0].Name == pr.State.Env {
		return fmt.Sprintf("%q is the first environment so status check is skipped", pr.State.Env), nil
	}

	// Check status of commit
	prevEnv, err := cfg.PrevEnvironment(pr.State.Env)
	if err != nil {
		return "", err
	}
	// TODO: Check if current commit is stale
	deadline := time.Now().Add(5 * time.Minute)
	for {
		if time.Now().After(deadline) {
			break
		}

		status, err := repo.GetStatus(ctx, pr.State.Sha, pr.State.Group, prevEnv.Name)
		if err != nil {
			fmt.Printf("retrying status check: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if !status.Succeeded {
			return "", fmt.Errorf("commit status check has failed %q", pr.State.Sha)
		}
		return "status check has succeed", nil
	}
	return "", fmt.Errorf("commit status check has timed out %q", pr.State.Sha)
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

	_, err := update.UpdateWithSetters(path, path, policies)
	if err != nil {
		return fmt.Errorf("failed updating manifests: %v", err)
	}

	return nil
}
