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
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

const kustomizationFile = "kustomization.yaml"

// DuplicateApplication duplicates the application manifests based on the label selector.
// It assumes that the fs is a base fs in the repository directory.
// nolint:gocritic // ignore
func DuplicateApplication(fs afero.Fs, state git.PRState, labelSelector map[string]string) error {
	// Get manifests for the application
	selector, err := toKustomizeSelector(labelSelector)
	if err != nil {
		return fmt.Errorf("could not convert to kustomize selector: %w", err)
	}
	envPath := filepath.Join(state.Group, state.Env)
	resources, err := manfifestsMatchingSelector(fs, envPath, selector)
	if err != nil {
		return fmt.Errorf("could not get manifets from selector: %w", err)
	}

	// Create feature app manifest directory
	appPath := filepath.Join(envPath, fmt.Sprintf("%s-%s", state.App, state.Feature))
	dirExists, err := afero.DirExists(fs, appPath)
	if err != nil {
		return err
	}
	if !dirExists {
		if err := fs.RemoveAll(appPath); err != nil {
			return err
		}
	}
	if err := fs.Mkdir(appPath, 0755); err != nil {
		return err
	}

	// Write feature app manifests
	kustomization := &kustypes.Kustomization{}
	kustomization.NameSuffix = state.Feature
	kustomization.CommonLabels = map[string]string{"feature": state.Feature}
	for _, res := range resources {
		b, err := res.AsYAML()
		if err != nil {
			return err
		}
		switch res.GetKind() {
		case "Ingress":
			b, err = patchIngress(b, state.Feature)
			if err != nil {
				return err
			}
		case "Deployment":
			b, err = patchDeployment(b, state.Tag)
			if err != nil {
				return err
			}
		}
		id := fmt.Sprintf("%s-%s-%s.yaml", res.GetGvk().String(), res.GetNamespace(), res.GetName())
		if err := afero.WriteFile(fs, filepath.Join(appPath, id), b, 0600); err != nil {
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
	if err := afero.WriteFile(fs, filepath.Join(appPath, kustomizationFile), b, 0600); err != nil {
		return err
	}

	// Append feature kustomization to root resources if it does not exist
	kustomizationPath := filepath.Join(envPath, kustomizationFile)
	b, err = afero.ReadFile(fs, kustomizationPath)
	if err != nil {
		return err
	}
	node, err := kyaml.Parse(string(b))
	if err != nil {
		return err
	}
	resourcePath := fmt.Sprintf("%s-%s", state.App, state.Feature)
	rNode := node.Field("resources")
	yNode := rNode.Value.YNode()
	fmt.Println(yNode.Content)
	yNode.Content = append(yNode.Content, kyaml.NewStringRNode(resourcePath).YNode())
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
	return &kustypes.Selector{LabelSelector: selectorString}, nil
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
