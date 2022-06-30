package server

import (
	"fmt"
	"github.com/dianpeng/mono-service/vhost"
	"net/http"
	"time"
)

// A listener is a endpoint which wait for http traffic and run bunch of
// servername oriented services on top of it. A listener does not know
// which server name should be used.
type vhostentry struct {
	h *vhost.VHost
}

// TOOD(dpeng): support atomic pointer to allow concurrent updating
func (v *vhostentry) vhost() *vhost.VHost {
	return v.h
}

type listener struct {
	name   string
	list   []vhostentry
	sn     map[string]int // TODO(dpeng): allowing better servername matching
	server *http.Server   // the server
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

func newListener(
	opt ListenerConfig,
) *listener {
	l := &listener{
		name: opt.Name,
		sn:   make(map[string]int),
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

	return l
}

func (l *listener) addVHost(
	v *vhost.VHost,
) error {
	serverName := v.Config.ServerName

	_, ok := l.sn[serverName]
	if ok {
		return fmt.Errorf("server name %s already existed", serverName)
	}

	idx := len(l.list)
	l.list = append(l.list, vhostentry{h: v})
	l.sn[serverName] = idx
	return nil
}

func (l *listener) resolveVHost(serverName string) *vhost.VHost {
	idx, ok := l.sn[serverName]
	if ok {
		return l.list[idx].vhost()
	} else {
		return nil
	}
}

func (l *listener) run() error {
	return l.server.ListenAndServe()
}
