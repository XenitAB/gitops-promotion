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
		return "skipping promotion as commit does not originate from PR", nil
	}
	if pr.State == nil {
		return "skipping promotion as PR is not created by gitops-promotion", nil
	}
	if pr.State.GetPRType() == git.PRTypeFeature {
		return "skipping promotion of feature", nil
	}
	if !cfg.HasNextEnvironment(pr.State.Env) {
		return "no next environment to promote to", nil
	}
	headID, err := repo.GetCurrentCommit()
	if err != nil {
		return "", fmt.Errorf("could not get latest commit: %w", err)
	}
	nextEnv, err := cfg.NextEnvironment(pr.State.Env)
	if err != nil {
		return "", fmt.Errorf("could not get next environment: %w", err)
	}
	state := &git.PRState{
		Group: pr.State.Group,
		App:   pr.State.App,
		Tag:   pr.State.Tag,
		Env:   nextEnv.Name,
		Sha:   headID.String(),
		Type:  pr.State.Type,
	}
	return promote(ctx, cfg, repo, state)
}

func promote(ctx context.Context, cfg config.Config, repo *git.Repository, state *git.PRState) (string, error) {
	// Update image tag
	manifestPath := fmt.Sprintf("%s/%s/%s", repo.GetRootDir(), state.Group, state.Env)
	err := manifest.UpdateImageTag(manifestPath, state.App, state.Group, state.Tag)
	if err != nil {
		return "", fmt.Errorf("failed updating manifests: %w", err)
	}

	// Push and create PR
	branchName := state.BranchName(cfg.PRFlow == "per-env")
	title := state.Title()
	description, err := state.Description()
	if err != nil {
		return "", err
	}
	err = repo.CreateBranch(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not create branch: %w", err)
	}
	sha, err := repo.CreateCommit(branchName, title)
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
	prid, err := repo.CreatePR(ctx, branchName, auto, title, description)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %w", err)
	}
	return fmt.Sprintf("created branch %s with pull request %d on commit %s", branchName, prid, sha), nil
}
