package command

import (
	"context"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func NewCommand(ctx context.Context, cfg config.Config, repo *git.Repository, group, app, tag string) (string, error) {
	state := git.PRState{
		Env:   "",
		Group: group,
		App:   app,
		Tag:   tag,
		Sha:   "",
	}
	return promote(ctx, cfg, repo, &state)
}
