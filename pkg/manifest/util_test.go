package manifest

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestNewDirectory(t *testing.T) {
  fs := afero.NewMemMapFs()
  path := "./foo/bar"
  exists, err := createOrReplaceDirectory(fs, path)
  require.NoError(t, err)
  require.False(t, exists)
}

func TestExistingDirectory(t *testing.T) {
  fs := afero.NewMemMapFs()
  path := "./foo/bar"
  err := fs.Mkdir(path, 0655)
  require.NoError(t, err)
  exists, err := createOrReplaceDirectory(fs, path)
  require.NoError(t, err)
  require.True(t, exists)
}
