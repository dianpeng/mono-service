package http

import (
	"encoding/json"
	"github.com/dianpeng/mono-service/http/vhost"
	"github.com/dianpeng/mono-service/server"
	"net/http"
	"strconv"
	"strings"
	"time"
  "fmt"
)

type listenerConfig struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Endpoint          string `json:"endpoint"`
	ReadTimeout       int64  `json:"read_timeout"`
	WriteTimeout      int64  `json:"write_timeout"`
	IdleTimeout       int64  `json:"idle_timeout"`
	ReadHeaderTimeout int64  `json:"read_header_timeout"`
	MaxHeaderSize     int64  `json:"max_header_size"`
}

type listener struct {
	name   string
	server *http.Server // the server
	vlist  vhostlist
}

func (lc *listenerConfig) TypeName() string {
	return lc.Type
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
	sopt server.ListenerConfig,
) (server.Listener, error) {
	opt := sopt.(*listenerConfig)

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

func (f *fac) ParseConfigJson(input string) (server.ListenerConfig, error) {
	o := &listenerConfig{
		Name:              "",
		Type:              "",
		Endpoint:          "",
		ReadTimeout:       20,
		WriteTimeout:      20,
		IdleTimeout:       90,
		ReadHeaderTimeout: 10,
		MaxHeaderSize:     1024 * 64,
	}
	if err := json.Unmarshal([]byte(input), o); err != nil {
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

func (f *fac) ParseConfigCompact(input string) (server.ListenerConfig, error) {
	conf := &listenerConfig{}
	x := strings.Split(input, ",")
	if len(x) < 3 {
		return conf, fmt.Errorf("invalid listener config: %s, at least 3 elements are needed", input)
	}

	conf.Type = x[0]
	conf.Name = x[1]
	conf.Endpoint = x[2]

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

	if err := parseInt("ReadTimeout", 3, &conf.ReadTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("WriteTimeout", 4, &conf.WriteTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("IdleTimeout", 5, &conf.IdleTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("ReadHeaderTimeout", 6, &conf.ReadHeaderTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("MaxHeaderSize", 7, &conf.MaxHeaderSize); err != nil {
		return conf, err
	}

	return conf, nil
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
	v server.VHost,
) error {
	return l.vlist.add(v.(*vhost.VHost))
}

// try to update a VHost in the current listener
func (l *listener) UpdateVHost(
	v server.VHost,
) {
	l.vlist.update(v.(*vhost.VHost))
}

func (l *listener) RemoveVHost(
	serverName string,
) {
	l.vlist.remove(serverName)
}

func (l *listener) GetVHost(
	serverName string,
) server.VHost {
	return l.vlist.get(serverName)
}

func init() {
	server.AddListenerFactory(
		"http",
		&fac{},
	)
}
