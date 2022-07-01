package vhost

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Each application to be served by mono-service will needs to have a unfied
// entry, which is the manifest object. It contains all the services entries
// along with the FS object which can be used to load teh whole file
type Manifest struct {
	FS          fs.FS
	Main        string
	ServiceFile []string
}

// Create a manifest from local director
func NewManifestFromLocalDir(
	mainPath string,
) (*Manifest, error) {
	manifest := &Manifest{}
	dir := filepath.Dir(mainPath)

	relativeOffset := len(dir) + 1

	manifest.Main = mainPath[relativeOffset:]
	manifest.FS = os.DirFS(dir)

	err := filepath.Walk(
		dir,
		func(path string, info os.FileInfo, e error) error {
			if e != nil {
				return nil
			}
			if path == mainPath {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".pl" {
				return nil
			}

			manifest.ServiceFile = append(manifest.ServiceFile, path[relativeOffset:])
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return manifest, nil
}
