package manifest

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/kustomize/api/image"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resource"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

// DuplicateApplication duplicates the application manifests based on the label selector.
// It assumes that the fs is a base fs in the repository directory.
// nolint:gocritic // ignore
func DuplicateApplication(fs afero.Fs, labelSelector map[string]string, state git.PRState) error {
	envPath := filepath.Join(state.Group, state.Env)
	featurePath := filepath.Join(envPath, fmt.Sprintf("%s-%s", state.App, state.Tag))
	kustomizationPath := filepath.Join(envPath, "kustomization.yaml")

	// Get manifests for the application
	selector, err := toKustomizeSelector(labelSelector)
	if err != nil {
		return fmt.Errorf("could not convert to kustomize selector: %w", err)
	}
	resources, err := manfifestsMatchingSelector(fs, envPath, selector)
	if err != nil {
		return fmt.Errorf("could not get manifets from selector: %w", err)
	}

	// Create feature manifest directory
	kustomization := &kustypes.Kustomization{}
	kustomization.NameSuffix = state.Tag
	kustomization.CommonLabels = map[string]string{"feature": state.Tag}
	if err := fs.Mkdir(featurePath, 0755); err != nil {
		return err
	}
	for _, res := range resources {
		b, err := patchResource(res, state.Tag)
		if err != nil {
			return err
		}
		id := fmt.Sprintf("%s-%s-%s.yaml", res.GetGvk().String(), res.GetNamespace(), res.GetName())
		if err := afero.WriteFile(fs, filepath.Join(featurePath, id), b, 0600); err != nil {
			return err
		}
		kustomization.Resources = append(kustomization.Resources, id)
	}
	errStrings := kustomization.EnforceFields()
	if len(errStrings) != 0 {
		return fmt.Errorf("%s", strings.Join(errStrings, ", "))
	}
	b, err := yaml.Marshal(kustomization)
	if err != nil {
		return err
	}
	if err := afero.WriteFile(fs, filepath.Join(featurePath, "kustomization.yaml"), b, 0600); err != nil {
		return err
	}

	// Append feature kustomization to root resources
	b, err = afero.ReadFile(fs, kustomizationPath)
	if err != nil {
		return err
	}
	node, err := yaml.Parse(string(b))
	if err != nil {
		return err
	}
	resourcePath := fmt.Sprintf("%s-%s", state.App, state.Tag)
	rNode := node.Field("resources")
	yNode := rNode.Value.YNode()
	yNode.Content = append(yNode.Content, yaml.NewStringRNode(resourcePath).YNode())
	rNode.Value.SetYNode(yNode)

	data, err := node.String()
	if err != nil {
		return err
	}
	if err := afero.WriteFile(fs, kustomizationPath, []byte(data), 0600); err != nil {
		return err
	}

	return nil
}

func toKustomizeSelector(labelSelector map[string]string) (*kustypes.Selector, error) {
	selector, err := labels.ValidatedSelectorFromSet(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("could not create label selector: %w", err)
	}
	selectorString := selector.String()
	if selectorString == "" {
		return nil, fmt.Errorf("selector string should not be empty")
	}
	return &kustypes.Selector{LabelSelector: selector.String()}, nil
}

func manfifestsMatchingSelector(fs afero.Fs, path string, selector *kustypes.Selector) ([]*resource.Resource, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := k.Run(NewKustomizeFs(fs), path)
	if err != nil {
		return nil, fmt.Errorf("could not build kustomization: %w", err)
	}
	resources, err := resMap.Select(*selector)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("returned resources is an empty list")
	}
	return resources, nil
}

func patchResource(res *resource.Resource, feature string) ([]byte, error) {
	b, err := res.AsYAML()
	if err != nil {
		return nil, err
	}
	switch res.GetKind() {
	case "Ingress":
		return patchIngress(b, feature)
	case "Deployment":
		return patchDeployment(b, feature)
	default:
		return b, nil
	}
}

func patchIngress(b []byte, feature string) ([]byte, error) {
	ingress := &networkingv1.Ingress{}
	err := yaml.Unmarshal(b, ingress)
	if err != nil {
		return nil, err
	}
	for i, rule := range ingress.Spec.Rules {
		ingress.Spec.Rules[i].Host = fmt.Sprintf("%s-%s", feature, rule.Host)
	}
	// nolint:gocritic // ignore
	for i, tls := range ingress.Spec.TLS {
		for j, host := range tls.Hosts {
			ingress.Spec.TLS[i].Hosts[j] = fmt.Sprintf("%s-%s", feature, host)
		}
	}
	return yaml.Marshal(ingress)
}

func patchDeployment(b []byte, feature string) ([]byte, error) {
	deployment := &appsv1.Deployment{}
	err := yaml.Unmarshal(b, deployment)
	if err != nil {
		return nil, err
	}
	// TODO: Do not override every single image tag
	// nolint:gocritic // ignore
	for i, container := range deployment.Spec.Template.Spec.Containers {
		name, _ := image.Split(container.Image)
		deployment.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", name, feature)
	}
	return yaml.Marshal(deployment)
}
