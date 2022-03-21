package manifest

import "github.com/spf13/afero"

// createOrReplaceDirectory will for the given path remove any existing directory 
// and then create a new one.
func createOrReplaceDirectory(fs afero.Fs, path string) (bool, error) {
  dirExists, err := afero.DirExists(fs, path)
	if err != nil {
		return false, err
	}
	if dirExists {
		if err := fs.RemoveAll(path); err != nil {
			return false, err
		}
	}
	if err := fs.Mkdir(path, 0755); err != nil {
		return false, err
	}
	return dirExists, err
}
