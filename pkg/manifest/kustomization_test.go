package manifest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/gitops-promotion/pkg/git"
)

func TestDuplicateApplication(t *testing.T) {
	osFs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/duplicate-application")
	memFs := afero.NewMemMapFs()
	fs := afero.NewCopyOnWriteFs(osFs, memFs)

	state := git.PRState{
		Env:   "dev",
		Group: "apps",
		App:   "nginx",
		Tag:   "feature",
	}
	err := DuplicateApplication(fs, map[string]string{"app": "nginx"}, state)
	require.NoError(t, err)

	isDir, err := afero.IsDir(memFs, filepath.Join(state.Group, state.Env, fmt.Sprintf("%s-%s", state.App, state.Tag)))
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
	yaml := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test
spec:
  rules:
  - host: "foo.bar.com"
    http:
      paths:
      - pathType: Prefix
        path: "/"
        backend:
          service:
            name: service
            port:
              number: 80
`
	b, err := patchIngress([]byte(yaml), "foobar")
	require.NoError(t, err)
	expectedYaml := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  creationTimestamp: null
  name: test
spec:
  rules:
  - host: foobar-foo.bar.com
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
	require.Equal(t, expectedYaml, string(b))
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
