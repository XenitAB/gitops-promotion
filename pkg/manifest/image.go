package manifest

import (
	"fmt"
	"log"

	"github.com/fluxcd/image-automation-controller/pkg/update"
	imagev1_reflect "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateImageTag changes the image tag in the kustomization file.
// TODO: Should change to using fs objects.
func UpdateImageTag(path, app, group, tag string) error {
	policies := []imagev1_reflect.ImagePolicy{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app,
				Namespace: group,
			},
			Status: imagev1_reflect.ImagePolicyStatus{
				LatestImage: fmt.Sprintf("%s:%s", app, tag),
			},
		},
	}
	log.Printf("Updating images with %s:%s:%s in %s\n", group, app, tag, path)
	_, err := update.UpdateWithSetters(logr.Discard(), path, path, policies)
	if err != nil {
		return fmt.Errorf("failed updating manifests: %w", err)
	}
	return nil
}
