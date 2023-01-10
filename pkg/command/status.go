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
//
//nolint:gocognit // not convinced that extracting bits would make it more readable
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
	if pr.State.GetPRType() == git.PRTypeFeature {
		return "Automatically allowing feature branch PR", nil
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
	deadline := time.Now().Add(cfg.StatusTimeout)
	for {
		if time.Now().After(deadline) {
			break
		}
		status, err := repo.GetStatus(ctx, pr.State.Sha, pr.State.Group, prevEnv.Name)
		if err == nil {
			if !status.Succeeded {
				return "", fmt.Errorf("failed reconciliation for %s-%s found on %q", pr.State.Group, prevEnv.Name, pr.State.Sha)
			}
			return fmt.Sprintf("successful reconciliation for %s-%s found on %q", pr.State.Group, prevEnv.Name, pr.State.Sha), nil
		}
		head, err := repo.FetchBranch(git.DefaultBranch)
		if err != nil {
			return "", fmt.Errorf("failed to fetch new commits: %w", err)
		}
		status, err = repo.GetStatus(ctx, head.String(), pr.State.Group, prevEnv.Name)
		if err == nil {
			if !status.Succeeded {
				return "", fmt.Errorf("failed reconciliation for %s-%s found on %s at %s", pr.State.Group, prevEnv.Name, git.DefaultBranch, head)
			}
			return fmt.Sprintf("successful reconciliation for %s-%s found on %s at %s", pr.State.Group, prevEnv.Name, git.DefaultBranch, head), nil
		}
		fmt.Printf("retrying status check for %s-%s: %v\n", pr.State.Group, prevEnv.Name, err)
		time.Sleep(5 * time.Second)
	}
	return "", fmt.Errorf("commit status check for %s-%s has timed out %q", pr.State.Group, prevEnv.Name, pr.State.Sha)
}
