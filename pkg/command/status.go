package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

func StatusCommand(ctx context.Context, providerType string, path, token string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", err
	}
	repo, err := getRepository(ctx, providerType, path, token)
	if err != nil {
		return "", err
	}

	// If branch does not contain promote it was manual, return early
	branchName, err := repo.GetBranchName()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(branchName, git.PromoteBranchPrefix) {
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
