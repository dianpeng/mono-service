package server

import (
	"github.com/dianpeng/mono-service/manifest"
)

type VHost interface {
	ListenerName() string
	ListenerType() string
	Name() string
}

type VHostFactory interface {
	New(*manifest.Manifest) (VHost, error)
}

var vhostfac = make(map[string]VHostFactory)

func AddVHostFactory(
	n string,
	f VHostFactory,
) {
	vhostfac[n] = f
}

func GetVHostFactory(
	n string,
) VHostFactory {
	v, ok := vhostfac[n]
	if ok {
		return v
	} else {
		return nil
	}
}
