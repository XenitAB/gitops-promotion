package command

import (
	"context"
	"fmt"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func PromoteCommand(ctx context.Context, path, token string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", fmt.Errorf("could not get configuration: %w", err)
	}
	repo, err := getRepository(ctx, path, token)
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
	err = repo.Push(state.BranchName())
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
