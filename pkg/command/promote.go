package command

import (
	"context"
	"fmt"
	"log"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
	"github.com/xenitab/gitops-promotion/pkg/manifest"
)

// PromoteCommand is run after a PR is merged. It creates a new PR for the next environment
// if there is one present.
func PromoteCommand(ctx context.Context, cfg config.Config, repo *git.Repository) (string, error) {
	pr, err := repo.GetPRThatCausedCurrentCommit(ctx)
	if err != nil {
		//nolint:errcheck //best effort for logging
		sha, _ := repo.GetCurrentCommit()
		log.Printf("Failed retrieving pull request for commit %s: %v", sha, err)
		//lint:ignore nilerr should not return error
		return "skipping PR creation as commit does not originate from promotion PR", nil
	}
	return promote(ctx, cfg, repo, &pr.State)
}

func promote(ctx context.Context, cfg config.Config, repo *git.Repository, state *git.PRState) (string, error) {
	if state.GetPRType() == git.PRTypePromote {
		return "not promoting feature branch", nil
	}

	// Check if there is a next env or get next env
	if state.Env != cfg.Environments[0].Name {
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
	err = manifest.UpdateImageTag(manifestPath, state.App, state.Group, state.Tag)
	if err != nil {
		return "", fmt.Errorf("failed updating manifests: %w", err)
	}

	// Push and create PR
	branchName := state.BranchName(cfg.PRFlow == "per-env")
	err = repo.CreateBranch(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not create branch: %w", err)
	}
	sha, err := repo.CreateCommit(branchName, state.Title())
	if err != nil {
		return "", fmt.Errorf("could not commit changes: %w", err)
	}
	err = repo.Push(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not push changes: %w", err)
	}
	auto, err := cfg.IsEnvironmentAutomated(state.Env)
	if err != nil {
		return "", fmt.Errorf("could not get environment automation state: %w", err)
	}
	prid, err := repo.CreatePR(ctx, branchName, auto, state)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %w", err)
	}
	return fmt.Sprintf("created branch %s with pull request %d on commit %s", branchName, prid, sha), nil
}
