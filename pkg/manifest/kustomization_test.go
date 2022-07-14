package manifest

import (
	"fmt"
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

func TestDuplicateAndRemoveApplication(t *testing.T) {
	osFs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/duplicate-application")
	memFs := afero.NewMemMapFs()
	fs := afero.NewCopyOnWriteFs(osFs, memFs)

	state := git.PRState{
		Group:   "apps",
		App:     "nginx",
		Env:     "dev",
		Feature: "feature",
	}
	for _, tag := range []string{"1234", "abcd", "12ab"} {
		state.Tag = tag

		err := DuplicateApplication(fs, state, map[string]string{"app": "nginx"})
		require.NoError(t, err)

		expectedRootKustomization := `resources:
- ../base
- existing-feature
- nginx-feature
images:
- name: nginx
  newTag: app # {"$imagepolicy": "apps:nginx:tag"}
`
		testFileContains(t, memFs, state.EnvKustomizationPath(), expectedRootKustomization)

		expectedFeatureKustomization := `commonLabels:
  app: nginx-feature
  gitops-promotion.xenit.io/feature: feature
nameSuffix: -feature
resources:
- apps_v1_Deployment--nginx.yaml
- ~G_v1_Service--nginx.yaml
`
		testFileContains(t, memFs, state.AppKustomizationPath(), expectedFeatureKustomization)

		fis, err := afero.ReadDir(memFs, state.AppPath())
		require.NoError(t, err)
		files := []string{}
		for _, fi := range fis {
			files = append(files, fi.Name())
		}
		require.Equal(t, []string{"apps_v1_Deployment--nginx.yaml", "kustomization.yaml", "~G_v1_Service--nginx.yaml"}, files)
	}

	err := RemoveApplication(fs, state)
	require.NoError(t, err)
	expectedKustomization := `resources:
- ../base
- existing-feature
images:
- name: nginx
  newTag: app # {"$imagepolicy": "apps:nginx:tag"}
`
	testFileContains(t, memFs, state.EnvKustomizationPath(), expectedKustomization)
	_, err = fs.Stat(state.AppPath())
	require.EqualError(t, err, "stat testdata/duplicate-application/apps/dev/nginx-feature: no such file or directory")
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
