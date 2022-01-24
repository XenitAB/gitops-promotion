package command

import (
	"context"
	"fmt"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func FeatureCommand(ctx context.Context, cfg config.Config, repo *git.Repository) (string, error) {
	fmt.Println("feature")
	return "", nil
}
