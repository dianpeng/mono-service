package server

type Listener interface {
	Name() string
	Type() string

	AddVHost(VHost) error
	UpdateVHost(VHost)
	RemoveVHost(string)
	GetVHost(string) VHost
	Run() error
}
