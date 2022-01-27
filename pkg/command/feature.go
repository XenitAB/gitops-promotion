package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/kustomize/api/image"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resource"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"

	"github.com/xenitab/gitops-promotion/pkg/config"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

// FeatureCommand is similar to NewCommand but creates a PR with a temporary deployment of the application.
// A totally new application will be created instead of overriding the existing application deployment.
func FeatureCommand(ctx context.Context, cfg config.Config, repo *git.Repository, group, app, tag string) (string, error) {
	// Create new state
	state := git.PRState{
		Env:   cfg.Environments[0].Name,
		Group: group,
		App:   fmt.Sprintf("%s-%s", app, tag),
		Tag:   tag,
		Sha:   "",
		Type:  git.PRTypeFeature,
	}
	envPath := filepath.Join(repo.GetRootDir(), state.Group, state.Env)
	featurePath := filepath.Join(envPath, state.App)
	fmt.Println(featurePath)

	// Create label selector
	featureApp, err := cfg.Features.GetFeatureApp(state.Group, app)
	if err != nil {
		return "", err
	}
	selector, err := labels.ValidatedSelectorFromSet(featureApp.LabelSelector)
	if err != nil {
		return "", fmt.Errorf("could not create label selector: %w", err)
	}
	selectorString := selector.String()
	if selectorString == "" {
		return "", fmt.Errorf("selector string should not be empty")
	}
	kSelector := kustypes.Selector{LabelSelector: selector.String()}

	// Prepare Kustomization
	fs := filesys.MakeFsOnDisk()
	opts := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}
	k := krusty.MakeKustomizer(opts)

	// Get manifests for the application
	resMap, err := k.Run(fs, envPath)
	if err != nil {
		return "", fmt.Errorf("could not build kustomization: %w", err)
	}
	resources, err := resMap.Select(kSelector)
	if err != nil {
		return "", err
	}
	if len(resources) == 0 {
		return "", fmt.Errorf("returned resources is an empty list")
	}

	// Write manifets to feature directory
	kustomization := &kustypes.Kustomization{}
	kustomization.NameSuffix = state.Tag
	kustomization.CommonLabels = map[string]string{"feature": state.Tag}
	err = os.Mkdir(filepath.Join(envPath, state.App), 0755)
	if err != nil {
		return "", err
	}
	for _, res := range resources {
		b, err := patchResource(res, state.Tag)
		if err != nil {
			return "", err
		}
		id := fmt.Sprintf("%s-%s-%s.yaml", res.GetGvk().String(), res.GetNamespace(), res.GetName())
		err = os.WriteFile(filepath.Join(featurePath, id), b, 0644)
		if err != nil {
			return "", err
		}
		kustomization.Resources = append(kustomization.Resources, filepath.Join(id))
	}
	errStrings := kustomization.EnforceFields()
	if len(errStrings) != 0 {
		return "", fmt.Errorf("%s", strings.Join(errStrings, ", "))
	}
	b, err := yaml.Marshal(kustomization)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(filepath.Join(featurePath, "kustomization.yaml"), b, 0644)
	if err != nil {
		return "", err
	}

	// Append to root kustomization resources
	kustomizationPath := filepath.Join(envPath, "kustomization.yaml")
	b, err = os.ReadFile(kustomizationPath)
	if err != nil {
		return "", err
	}
	kustomization = &kustypes.Kustomization{}
	err = yaml.Unmarshal(b, kustomization)
	if err != nil {
		return "", err
	}
	kustomization.Resources = append(kustomization.Resources, state.App)
	b, err = yaml.Marshal(kustomization)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(kustomizationPath, b, 0644)
	if err != nil {
		return "", err
	}
	_, err = k.Run(fs, envPath)
	if err != nil {
		return "", err
	}

	// Push and create PR
	branchName := state.BranchName(false)
	err = repo.CreateBranch(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not create branch: %w", err)
	}
	sha, err := repo.CreateCommit(branchName, state.Title())
	if err != nil {
		return "", fmt.Errorf("could not commit changes: %w", err)
	}
	err = repo.Push(branchName, true)
	if err != nil {
		return "", fmt.Errorf("could not push changes: %w", err)
	}
	prid, err := repo.CreatePR(ctx, branchName, true, &state)
	if err != nil {
		return "", fmt.Errorf("could not create a PR: %w", err)
	}
	return fmt.Sprintf("created branch %s with pull request %d on commit %s", branchName, prid, sha), nil
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
	for i, container := range deployment.Spec.Template.Spec.Containers {
		name, _ := image.Split(container.Image)
		deployment.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", name, feature)
	}
	return yaml.Marshal(deployment)
}
