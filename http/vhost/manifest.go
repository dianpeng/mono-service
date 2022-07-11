package vhost

import (
	"io/fs"
)

// Each application to be served by mono-service will needs to have a unfied
// entry, which is the manifest object. It contains all the services entries
// along with the FS object which can be used to load teh whole file
type Manifest struct {
	FS          fs.FS
	Main        string
	ServiceFile []string
}
