package command

import (
	"context"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

// NewCommand creates the initial PR which is going to be merged to the first environment. The main
// difference to PromoteCommand is that it does not use a previous PR to create the first PR.
func NewCommand(ctx context.Context, cfg config.Config, repo *git.Repository, group, app, tag string) (string, error) {
	state := git.PRState{
		Group: group,
		App:   app,
		Tag:   tag,
		Env:   "",
		Sha:   "",
	}
	return promote(ctx, cfg, repo, &state)
}
