package command

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
	"github.com/xenitab/gitops-promotion/pkg/manifest"
)

// FeatureCommand is similar to NewCommand but creates a PR with a temporary deployment of the application.
// A totally new application will be created instead of overriding the existing application deployment.
func FeatureCommand(ctx context.Context, cfg config.Config, repo *git.Repository, group, app, tag, feature string) (string, error) {
	state := git.PRState{
		Env:     cfg.Environments[0].Name,
		Group:   group,
		App:     app,
		Tag:     tag,
		Sha:     "",
		Feature: feature,
		Type:    git.PRTypeFeature,
	}
	featureLabelSelector, err := cfg.GetFeatureLabelSelector(state.Group, app)
	if err != nil {
		return "", fmt.Errorf("feature deployment does not work without configuring a feature label selector: %w", err)
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), repo.GetRootDir())
	err = manifest.DuplicateApplication(fs, state, featureLabelSelector)
	if err != nil {
		return "", err
	}

	branchName := state.BranchName(false)
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
	prid, err := repo.CreatePR(ctx, branchName, auto, &state)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %w", err)
	}
	return fmt.Sprintf("created branch %s with pull request %d on commit %s", branchName, prid, sha), nil
}
