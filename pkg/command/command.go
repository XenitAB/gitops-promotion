package command

import (
	"context"
	"fmt"

	"github.com/fluxcd/image-automation-controller/pkg/update"
	imagev1alpha1_reflect "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func updateImageTag(path, app, group, tag string) error {
	policies := []imagev1alpha1_reflect.ImagePolicy{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app,
				Namespace: group,
			},
			Status: imagev1alpha1_reflect.ImagePolicyStatus{
				LatestImage: fmt.Sprintf("%s:%s", app, tag),
			},
		},
	}

	_, err := update.UpdateWithSetters(path, path, policies)
	if err != nil {
		return fmt.Errorf("failed updating manifests: %w", err)
	}

	return nil
}
