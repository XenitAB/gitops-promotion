package manifest

import (
	"log"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type KustomizeFs struct {
	fs afero.Fs
}

func NewKustomizeFs(fs afero.Fs) KustomizeFs {
	return KustomizeFs{fs}
}

func (kfs KustomizeFs) Create(path string) (filesys.File, error) {
	return kfs.fs.Create(path)
}

func (kfs KustomizeFs) Mkdir(path string) error {
	return kfs.fs.Mkdir(path, 0755)
}

func (kfs KustomizeFs) MkdirAll(path string) error {
	return kfs.fs.MkdirAll(path, 0755)
}

func (kfs KustomizeFs) RemoveAll(path string) error {
	return kfs.fs.RemoveAll(path)
}

func (kfs KustomizeFs) Open(path string) (filesys.File, error) {
	return kfs.fs.Open(path)
}

func (kfs KustomizeFs) IsDir(path string) bool {
	ok, err := afero.IsDir(kfs.fs, path)
	if err != nil {
		return false
	}
	return ok
}

func (kfs KustomizeFs) ReadDir(path string) ([]string, error) {
	fileInfos, err := afero.ReadDir(kfs.fs, path)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, fileInfo := range fileInfos {
		paths = append(paths, fileInfo.Name())
	}
	return paths, nil
}

func (kfs KustomizeFs) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	if kfs.IsDir(path) {
		return filesys.ConfirmedDir(path), "", nil
	}
	d := filepath.Dir(path)
	if !kfs.IsDir(d) {
		// Programmer/assumption error.
		log.Fatalf("first part of '%s' not a directory", path)
	}
	if d == path {
		// Programmer/assumption error.
		log.Fatalf("d '%s' should be a subset of deLinked", d)
	}
	f := filepath.Base(path)
	if filepath.Join(d, f) != path {
		// Programmer/assumption error.
		log.Fatalf("these should be equal: '%s', '%s'", filepath.Join(d, f), path)
	}
	return filesys.ConfirmedDir(d), f, nil
}

func (kfs KustomizeFs) Exists(path string) bool {
	ok, err := afero.Exists(kfs.fs, path)
	if err != nil {
		return false
	}
	return ok
}

func (kfs KustomizeFs) Glob(pattern string) ([]string, error) {
	return afero.Glob(kfs.fs, pattern)
}

func (kfs KustomizeFs) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(kfs.fs, path)
}

func (kfs KustomizeFs) WriteFile(path string, data []byte) error {
	return afero.WriteFile(kfs.fs, path, data, 0600)
}

func (kfs KustomizeFs) Walk(path string, walkFn filepath.WalkFunc) error {
	return afero.Walk(kfs.fs, path, walkFn)
}
