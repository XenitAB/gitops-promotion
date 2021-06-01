package command

import (
	"context"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

func NewCommand(ctx context.Context, path, token, group, app, tag string) (string, error) {
	cfg, err := getConfig(path)
	if err != nil {
		return "", err
	}
	repo, err := getRepository(ctx, path, token)
	if err != nil {
		return "", err
	}
	state := git.PRState{
		Env:   "",
		Group: group,
		App:   app,
		Tag:   tag,
		Sha:   "",
	}
	return promote(ctx, cfg, repo, &state)
}
