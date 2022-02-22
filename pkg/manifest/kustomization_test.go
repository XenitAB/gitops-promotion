package manifest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/xenitab/gitops-promotion/pkg/git"
)

func testFileContains(t *testing.T, fs afero.Fs, path string, expectedContent string) {
	t.Helper()
	content, err := afero.ReadFile(fs, path)
	require.NoError(t, err)
	require.Equal(t, expectedContent, string(content))
}

func TestDuplicateApplication(t *testing.T) {
	osFs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/duplicate-application")
	memFs := afero.NewMemMapFs()
	fs := afero.NewCopyOnWriteFs(osFs, memFs)

	for _, tag := range []string{"1234", "abcd", "12ab"} {
		state := git.PRState{
			Group:   "apps",
			App:     "nginx",
			Env:     "dev",
			Tag:     tag,
			Feature: "feature",
		}
		err := DuplicateApplication(fs, state, map[string]string{"app": "nginx"})
		require.NoError(t, err)
		appFeatureName := fmt.Sprintf("%s-%s", state.App, state.Feature)
		appFeatureDir := filepath.Join(state.Group, state.Env, appFeatureName)

		expectedRootKustomization := `resources:
- ../base
- existing-feature
- nginx-feature
images:
- name: nginx
  newTag: app # {"$imagepolicy": "apps:nginx:tag"}
`
		testFileContains(t, memFs, filepath.Join(state.Group, state.Env, "kustomization.yaml"), expectedRootKustomization)

		expectedFeatureKustomization := `commonLabels:
  app: nginx-feature
  gitops-promotion.xenit.io/feature: feature
nameSuffix: -feature
resources:
- apps_v1_Deployment--nginx.yaml
- ~G_v1_Service--nginx.yaml
`
		testFileContains(t, memFs, filepath.Join(appFeatureDir, "kustomization.yaml"), expectedFeatureKustomization)

		fis, err := afero.ReadDir(memFs, filepath.Join(state.Group, state.Env, "nginx-feature"))
		require.NoError(t, err)
		files := []string{}
		for _, fi := range fis {
			files = append(files, fi.Name())
		}
		require.Equal(t, []string{"apps_v1_Deployment--nginx.yaml", "kustomization.yaml", "~G_v1_Service--nginx.yaml"}, files)
	}
}

func TestPatchIngress(t *testing.T) {
	yamlTemplate := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  creationTimestamp: null
  name: test
spec:
  rules:
  - host: %s
    http:
      paths:
      - backend:
          service:
            name: service
            port:
              number: 80
        path: /
        pathType: Prefix
status:
  loadBalancer: {}
`

	cases := []struct {
		name     string
		domain   string
		feature  string
		expected string
	}{
		{
			name:     "simple",
			domain:   "foo.bar.example",
			feature:  "baz",
			expected: "baz.foo.bar.example",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := patchIngress([]byte(fmt.Sprintf(yamlTemplate, c.domain)), c.feature)
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf(yamlTemplate, c.expected), string(b))
		})
	}
}

func TestPatchDeployment(t *testing.T) {
	yaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
	b, err := patchDeployment([]byte(yaml), "foobar")
	require.NoError(t, err)
	expectedYaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: nginx
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx:foobar
        name: nginx
        ports:
        - containerPort: 80
        resources: {}
status: {}
`
	require.Equal(t, expectedYaml, string(b))
}
