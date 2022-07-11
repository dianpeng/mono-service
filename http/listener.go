package http

import (
	"github.com/dianpeng/mono-service/http/vhost"
	"github.com/dianpeng/mono-service/server"
	"net/http"
	"time"
)

type listener struct {
	name   string
	server *http.Server // the server
	vlist  vhostlist
}

func (l *listener) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	x := l.resolveVHost(r.Host)
	if x != nil {
		x.Router.ServeHTTP(w, r)
	} else {
		w.WriteHeader(403)
	}
}

type fac struct{}

func (f *fac) New(
	opt server.ListenerConfig,
) (server.Listener, error) {
	l := &listener{
		name:  opt.Name,
		vlist: newvhostlist(),
	}

	l.server = &http.Server{
		Addr:              opt.Endpoint,
		Handler:           l,
		ReadHeaderTimeout: time.Second * time.Duration(opt.ReadHeaderTimeout),
		ReadTimeout:       time.Second * time.Duration(opt.ReadTimeout),
		WriteTimeout:      time.Second * time.Duration(opt.WriteTimeout),
		IdleTimeout:       time.Second * time.Duration(opt.IdleTimeout),
		MaxHeaderBytes:    int(opt.MaxHeaderSize),
	}

	return l, nil
}

func (l *listener) resolveVHost(serverName string) *vhost.VHost {
	return l.vlist.resolve(serverName)
}

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Type() string {
	return "http"
}

func (l *listener) Run() error {
	return l.server.ListenAndServe()
}

// the follwing function are thread safe, so can be used to add, update, remove
// virtual host when the listener is executing/running
func (l *listener) AddVHost(
	v *vhost.VHost,
) error {
	return l.vlist.add(v)
}

// try to update a VHost in the current listener
func (l *listener) UpdateVHost(
	v *vhost.VHost,
) {
	l.vlist.update(v)
}

func (l *listener) RemoveVHost(
	serverName string,
) {
	l.vlist.remove(serverName)
}

func (l *listener) GetVHost(
	serverName string,
) *vhost.VHost {
	return l.vlist.get(serverName)
}

func init() {
	server.AddListenerFactory(
		"http",
		&fac{},
	)
}
