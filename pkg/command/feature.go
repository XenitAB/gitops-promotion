package command

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
	"github.com/xenitab/gitops-promotion/pkg/manifest"
)

// FeatureCommand is similar to NewCommand but creates a PR with a temporary deployment of the application.
// A totally new application will be created instead of overriding the existing application deployment.
func FeatureNewCommand(ctx context.Context, cfg config.Config, repo *git.Repository, group, app, tag, feature string) (string, error) {
	// The feature name has to be alpha numeric or "-" as both Kubernetes
	// resources and domain names have this requirement. For this reason the
	// inputed feature name is sanitized to remove any offending characters
	// and lowercase all characters.
	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	feature = reg.ReplaceAllString(feature, "")
	feature = strings.ToLower(feature)

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

func FeatureDeleteStaleCommand(ctx context.Context, cfg config.Config, repo *git.Repository, maxAge time.Duration) (string, error) {
	environmentName := cfg.Environments[0].Name
	fs := afero.NewBasePathFs(afero.NewOsFs(), repo.GetRootDir())

	// Find directory names that are feature deployments
	states := []git.PRState{}
	for groupKey, group := range cfg.Groups {
		for appKey := range group.Applications {
			globKey := fmt.Sprintf("%s-*", appKey)
			matches, err := afero.Glob(fs, filepath.Join(groupKey, environmentName, globKey))
			if err != nil {
				return "", err
			}
			for _, match := range matches {
				// Skip non directories
				if fi, err := fs.Stat(match); err != nil && !fi.IsDir() {
					continue
				}

				// TODO: Replace with strings.Cut in Go 1.18
				comps := strings.Split(match, "-")
				feature := strings.Join(comps[1:], "-")

				state := git.PRState{
					Group:   groupKey,
					App:     appKey,
					Env:     environmentName,
					Feature: feature,
					Type:    git.PRTypeFeature,
				}
				states = append(states, state)
			}
		}
	}

	// Remove feature directories that have not been committed to for longer than max age
	removedApplication := false
	for _, state := range states {
		commit, err := repo.GetLastCommitForPath(state.AppPath())
		if err != nil {
			return "", err
		}
		if time.Now().Sub(commit.Author().When) < maxAge {
			continue
		}
		err = manifest.RemoveApplication(fs, state)
		if err != nil {
			return "", fmt.Errorf("could not remove application: %w", err)
		}
		removedApplication = true
	}
	if !removedApplication {
		return "No stale application to remove, exiting early.", nil
	}

	// Commit, push branch, create PR
	branchName := "remove/stale-feature"
	err := repo.CreateBranch(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not create branch: %w", err)
	}
	title := "Remove stale review features"
	description := ""
	sha, err := repo.CreateCommit(branchName, title)
	if err != nil {
		return "", fmt.Errorf("could not commit changes: %w", err)
	}
	err = repo.Push(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not push changes: %w", err)
	}
	auto, err := cfg.IsEnvironmentAutomated(environmentName)
	if err != nil {
		return "", fmt.Errorf("could not get environment automation state: %w", err)
	}
	prid, err := repo.CreatePR(ctx, branchName, auto, title, description)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %w", err)
	}
	return fmt.Sprintf("created branch %s with pull request %d on commit %s", branchName, prid, sha), nil
}
