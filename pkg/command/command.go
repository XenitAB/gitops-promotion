package command

import (
	"context"
	"fmt"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func getConfig(path string) (config.Config, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("could not load config: %w", err)
	}
	return cfg, nil
}

func getRepository(ctx context.Context, path, token string) (*git.Repository, error) {
	repo, err := git.LoadRepository(ctx, path, git.ProviderTypeAzdo, token)
	if err != nil {
		return nil, fmt.Errorf("could not load repository: %w", err)
	}
	return repo, nil
}
