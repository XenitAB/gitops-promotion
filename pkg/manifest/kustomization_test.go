package manifest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

// TODO: Extend testing of edge cases with existing resources

func TestDuplicateApplication(t *testing.T) {
	osFs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/duplicate-application")
	memFs := afero.NewMemMapFs()
	fs := afero.NewCopyOnWriteFs(osFs, memFs)

	state := git.PRState{
		Group:   "apps",
		App:     "nginx",
		Env:     "dev",
		Tag:     "hash",
		Feature: "feature",
	}
	err := DuplicateApplication(fs, state, map[string]string{"app": "nginx"})
	require.NoError(t, err)

	isDir, err := afero.IsDir(memFs, filepath.Join(state.Group, state.Env, fmt.Sprintf("%s-%s", state.App, state.Feature)))
	require.NoError(t, err)
	require.True(t, isDir)

	rootKustomization, err := afero.ReadFile(memFs, filepath.Join(state.Group, state.Env, "kustomization.yaml"))
	require.NoError(t, err)
	expectedRootKustomization := `resources:
- ../base
- nginx-feature
images:
- name: nginx
  newTag: app # {"$imagepolicy": "apps:nginx:tag"}
`
	require.Equal(t, expectedRootKustomization, string(rootKustomization))
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
	fmt.Println(string(b))
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
