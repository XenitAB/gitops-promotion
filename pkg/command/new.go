package command

import (
	"context"
	"fmt"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

// NewCommand creates the initial PR which is going to be merged to the first environment. The main
// difference to PromoteCommand is that it does not use a previous PR to create the first PR.
func NewCommand(ctx context.Context, cfg config.Config, repo *git.Repository, group, app, tag string) (string, error) {
	headID, err := repo.GetCurrentCommit()
	if err != nil {
		return "", fmt.Errorf("could not get latest commit: %w", err)
	}
	state := git.PRState{
		Group: group,
		App:   app,
		Tag:   tag,
		Env:   cfg.Environments[0].Name,
		Sha:   headID.String(),
		Type:  git.PRTypePromote,
	}
	return promote(ctx, cfg, repo, &state)
}
