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

const (
	featureLabel = "gitops-promotion.xenit.io/feature"
)

// DuplicateApplication duplicates the application manifests based on the label selector.
// It assumes that the fs is a base fs in the repository directory.
// nolint:gocognit,gocritic // ignore
func DuplicateApplication(fs afero.Fs, state git.PRState, labelSelector map[string]string) error {
	// Write feature app manifests
	resources, err := manfifestsMatchingSelector(fs, state.EnvPath(), labelSelector)
	if err != nil {
		return fmt.Errorf("could not get manifets from selector: %w", err)
	}
	dirExists, err := createOrReplaceDirectory(fs, state.AppPath())
	if err != nil {
		return err
	}
	commonLabels := map[string]string{featureLabel: state.Feature}
	for k, v := range labelSelector {
		commonLabels[k] = fmt.Sprintf("%s-%s", v, state.Feature)
	}
	kustomization := &kustypes.Kustomization{}
	kustomization.NameSuffix = fmt.Sprintf("-%s", state.Feature)
	kustomization.CommonLabels = commonLabels
	for _, res := range resources {
		b, err := patchResource(res, state.Tag, state.Feature)
		if err != nil {
			return err
		}
		id := strings.Join([]string{res.GetGvk().String(), res.GetNamespace(), res.GetName()}, "-")
		id = fmt.Sprintf("%s.yaml", id)
		if err := afero.WriteFile(fs, filepath.Join(state.AppPath(), id), b, 0600); err != nil {
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
	if err := afero.WriteFile(fs, state.AppKustomizationPath(), b, 0600); err != nil {
		return err
	}

	// Append feature kustomization to root resources if it does not exist
	// TODO: Replace with a separate check instead
	if !dirExists {
		b, err = afero.ReadFile(fs, state.EnvKustomizationPath())
		if err != nil {
			return err
		}
		node, err := kyaml.Parse(string(b))
		if err != nil {
			return err
		}
		// TODO: Replace with function from state
		resourcePath := fmt.Sprintf("%s-%s", state.App, state.Feature)
		rNode := node.Field("resources")
		yNode := rNode.Value.YNode()
		yNode.Content = append(yNode.Content, kyaml.NewStringRNode(resourcePath).YNode())
		rNode.Value.SetYNode(yNode)
		data, err := node.String()
		if err != nil {
			return err
		}
		if err := afero.WriteFile(fs, state.EnvKustomizationPath(), []byte(data), 0600); err != nil {
			return err
		}
	}

	return nil
}

func RemoveApplication(fs afero.Fs, state git.PRState) error {
	// Remove application manifest directory
	err := fs.RemoveAll(state.AppPath())
	if err != nil {
		return err
	}

	// Remove reference to path in environment Kustomization
	b, err := afero.ReadFile(fs, state.EnvKustomizationPath())
	if err != nil {
		return err
	}
	node, err := kyaml.Parse(string(b))
	if err != nil {
		return err
	}
	rNode := node.Field("resources")
	yNode := rNode.Value.YNode()
	newContent := []*kyaml.Node{}
	for _, content := range yNode.Content {
		if content.Value == fmt.Sprintf("%s-%s", state.App, state.Feature) {
			continue
		}
		newContent = append(newContent, content)
	}
	yNode.Content = newContent
	rNode.Value.SetYNode(yNode)
	data, err := node.String()
	if err != nil {
		return err
	}
	if err := afero.WriteFile(fs, state.EnvKustomizationPath(), []byte(data), 0600); err != nil {
		return err
	}
	return nil
}

func manfifestsMatchingSelector(fs afero.Fs, path string, labelSelector map[string]string) ([]*resource.Resource, error) {
	selector, err := labels.ValidatedSelectorFromSet(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("could not create label selector: %w", err)
	}
	selectorString := selector.String()
	if selectorString == "" {
		return nil, fmt.Errorf("selector string should not be empty")
	}
	selectorString = fmt.Sprintf("%s,!%s", selectorString, featureLabel)
	kSelector := &kustypes.Selector{LabelSelector: selectorString}
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := k.Run(NewKustomizeFs(fs), path)
	if err != nil {
		return nil, fmt.Errorf("could not build kustomization: %w", err)
	}
	resources, err := resMap.Select(*kSelector)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("returned resources is an empty list")
	}
	return resources, nil
}

func patchResource(res *resource.Resource, tag, feature string) ([]byte, error) {
	b, err := res.AsYAML()
	if err != nil {
		return nil, err
	}
	switch res.GetKind() {
	case "Ingress":
		b, err = patchIngress(b, feature)
		if err != nil {
			return nil, err
		}
	case "Deployment":
		b, err = patchDeployment(b, tag)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	return b, nil
}

func patchIngress(b []byte, feature string) ([]byte, error) {
	ingress := &networkingv1.Ingress{}
	err := yaml.Unmarshal(b, ingress)
	if err != nil {
		return nil, err
	}
	for i, rule := range ingress.Spec.Rules {
		ingress.Spec.Rules[i].Host = fmt.Sprintf("%s.%s", feature, rule.Host)
	}
	// nolint:gocritic // ignore
	for i, tls := range ingress.Spec.TLS {
		for j, host := range tls.Hosts {
			ingress.Spec.TLS[i].Hosts[j] = fmt.Sprintf("%s.%s", feature, host)
		}
	}
	return yaml.Marshal(ingress)
}

func patchDeployment(b []byte, tag string) ([]byte, error) {
	deployment := &appsv1.Deployment{}
	err := yaml.Unmarshal(b, deployment)
	if err != nil {
		return nil, err
	}
	// TODO: Do not override every single image tag
	// nolint:gocritic // ignore
	for i, container := range deployment.Spec.Template.Spec.Containers {
		name, _ := image.Split(container.Image)
		deployment.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", name, tag)
	}
	return yaml.Marshal(deployment)
}
