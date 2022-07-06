package server

import (
	"encoding/json"
	"fmt"
	"github.com/dianpeng/mono-service/vhost"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type listener struct {
	name   string
	server *http.Server // the server
	vlist  vhostlist
}

type ListenerConfig struct {
	Name              string `json:"name"`
	Endpoint          string `json:"endpoint"`
	ReadTimeout       int64  `json:"read_timeout"`
	WriteTimeout      int64  `json:"write_timeout"`
	IdleTimeout       int64  `json:"idle_timeout"`
	ReadHeaderTimeout int64  `json:"read_header_timeout"`
	MaxHeaderSize     int64  `json:"max_header_size"`
}

func ParseListenerConfigFromJSON(input string) (ListenerConfig, error) {
	o := ListenerConfig{
		Name:              "",
		Endpoint:          "",
		ReadTimeout:       20,
		WriteTimeout:      20,
		IdleTimeout:       90,
		ReadHeaderTimeout: 10,
		MaxHeaderSize:     1024 * 64,
	}
	if err := json.Unmarshal([]byte(input), &o); err != nil {
		return o, err
	}

	if o.Name == "" {
		return o, fmt.Errorf("must specify Name for listener config")
	}

	if o.Endpoint == "" {
		return o, fmt.Errorf("must specify Endpoint for listener config")
	}

	return o, nil
}

func ParseListenerConfigFromCompact(input string) (ListenerConfig, error) {
	conf := ListenerConfig{}
	x := strings.Split(input, ",")
	if len(x) < 2 {
		return conf, fmt.Errorf("invalid listener config: %s, at least 2 elements are needed", input)
	}

	conf.Name = x[0]
	conf.Endpoint = x[1]

	parseInt := func(field string, index int, out *int64) error {
		if len(x) > index {
			ival, err := strconv.ParseInt(x[index], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid listener config field %s, must be valid "+
					"integer, but has error: %s", field, err.Error())
			}
			*out = ival
		}
		return nil
	}

	if err := parseInt("ReadTimeout", 2, &conf.ReadTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("WriteTimeout", 3, &conf.WriteTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("IdleTimeout", 4, &conf.IdleTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("ReadHeaderTimeout", 5, &conf.ReadHeaderTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("MaxHeaderSize", 6, &conf.MaxHeaderSize); err != nil {
		return conf, err
	}

	return conf, nil
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

	return l
}

func (l *listener) resolveVHost(serverName string) *vhost.VHost {
	return l.vlist.resolve(serverName)
}

func (l *listener) run() error {
	return l.server.ListenAndServe()
}

// the follwing function are thread safe, so can be used to add, update, remove
// virtual host when the listener is executing/running
func (l *listener) addVHost(
	v *vhost.VHost,
) error {
	return l.vlist.add(v)
}

// try to update a VHost in the current listener
func (l *listener) updateVHost(
	v *vhost.VHost,
) {
	l.vlist.update(v)
}

func (l *listener) removeVHost(
	serverName string,
) {
	l.vlist.remove(serverName)
}

func (l *listener) getVHost(
	serverName string,
) *vhost.VHost {
	return l.vlist.get(serverName)
}
