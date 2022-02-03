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
