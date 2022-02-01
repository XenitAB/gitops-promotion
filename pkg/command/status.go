package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

// StatusCommand is run inside a PR to check if the PR can be merged.
func StatusCommand(ctx context.Context, cfg config.Config, repo *git.Repository) (string, error) {
	// If branch does not contain promote it was manual, return early
	branchName, err := repo.GetBranchName()
	if err != nil {
		return "", fmt.Errorf("failed to find current branch: %w", err)
	}
	if !strings.HasPrefix(branchName, string(git.PRTypePromote)) {
		return "Promotion was manual, skipping check", nil
	}

	// get current pr
	pr, err := repo.GetPRForCurrentBranch(ctx)
	if err != nil {
		return "", fmt.Errorf("failed getting pr for current branch: %w", err)
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
			fmt.Printf("retrying status check for %s-%s: %v\n", pr.State.Group, prevEnv.Name, err)
			time.Sleep(5 * time.Second)
			continue
		}
		if !status.Succeeded {
			return "", fmt.Errorf("commit status check for %s-%s has failed %q", pr.State.Group, prevEnv.Name, pr.State.Sha)
		}
		return "status check has succeed", nil
	}
	return "", fmt.Errorf("commit status check for %s-%s has timed out %q", pr.State.Group, prevEnv.Name, pr.State.Sha)
}
