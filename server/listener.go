package server

import (
	"github.com/dianpeng/mono-service/http/vhost"
)

type Listener interface {
	Name() string
	Type() string

	AddVHost(*vhost.VHost) error
	UpdateVHost(*vhost.VHost)
	RemoveVHost(string)
	GetVHost(string) *vhost.VHost

	Run() error
}
