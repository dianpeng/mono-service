package redis

import (
	"encoding/json"
	"fmt"
	"strconv"

	"crypto/tls"
	"strings"
	"sync/atomic"
	"unsafe"

	RV "github.com/dianpeng/mono-service/redis/vhost"
	"github.com/dianpeng/mono-service/server"
	"github.com/dianpeng/mono-service/util"

	"github.com/tidwall/redcon"
)

type listenerConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Endpoint       string `json:"endpoint"`
	TLSKey         string `json:"tls_key"`
	TLSCertificate string `json:"tls_certificate"`
	IdleTimeout    int64  `json:"idle_timeout"`
}

type redconServer interface {
	SetAcceptError(
		func(error),
	)
	ListenAndServe() error
}

type listener struct {
	name       string
	server     redconServer
	clientPool *util.HClientPool
	vhost      *server.VHost
}

type fac struct{}

type clearRedconServer struct {
	s *redcon.Server
}

type tlsRedconServer struct {
	s *redcon.TLSServer
}

func mkClearServer(
	s *redcon.Server,
) *clearRedconServer {
	return &clearRedconServer{
		s: s,
	}
}

func mkTLSServer(
	s *redcon.TLSServer,
) *tlsRedconServer {
	return &tlsRedconServer{
		s: s,
	}
}

func (x *clearRedconServer) ListenAndServe() error {
	return x.s.ListenAndServe()
}

func (x *tlsRedconServer) ListenAndServe() error {
	return x.s.ListenAndServe()
}

func (x *clearRedconServer) SetAcceptError(
	f func(error),
) {
	x.s.AcceptError = f
}

func (x *tlsRedconServer) SetAcceptError(
	f func(error),
) {
	x.s.AcceptError = f
}

func (f *fac) New(
	sopt server.ListenerConfig,
) (server.Listener, error) {
	opt := sopt.(*listenerConfig)
	return newListener(opt)
}

func (f *fac) ParseConfigJson(
	input string,
) (server.ListenerConfig, error) {
	o := &listenerConfig{
		Name:           "",
		Type:           "redis",
		Endpoint:       "",
		TLSKey:         "",
		TLSCertificate: "",
		IdleTimeout:    90,
	}
	if err := json.Unmarshal([]byte(input), o); err != nil {
		return nil, err
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

	parseString := func(index int, out *string) {
		if len(x) > index {
			*out = x[index]
		}
	}

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

	parseString(3, &conf.TLSKey)
	parseString(4, &conf.TLSCertificate)

	if err := parseInt("IdleTimeout", 5, &conf.IdleTimeout); err != nil {
		return nil, err
	}

	return conf, nil
}

func (l *listener) onEvent(
	conn redcon.Conn,
	cmd redcon.Command,
) {
	vhs := l.vhs()
	if vhs != nil {
		(*vhs).OnEvent(conn, cmd)
	} else {
		conn.WriteError("redis_vhost is not setup")
		conn.Close()
	}
}

func (l *listener) onAccept(
	conn redcon.Conn,
) bool {
	vhs := l.vhs()
	if vhs != nil {
		return (*vhs).OnAccept(conn)
	} else {
		conn.WriteError("redis_vhost is not setup")
		return false
	}
}

func (l *listener) onClose(
	conn redcon.Conn,
	err error,
) {
	vhs := l.vhs()
	if vhs != nil {
		(*vhs).OnClose(conn, err)
	}
}

func newListener(c *listenerConfig) (*listener, error) {
	var s redconServer

	l := &listener{
		name: c.Name,
	}

	if c.TLSKey != "" && c.TLSCertificate != "" {
		cer, err := tls.X509KeyPair(
			[]byte(c.TLSKey),
			[]byte(c.TLSCertificate),
		)
		if err != nil {
			return nil, err
		}

		s = mkTLSServer(
			redcon.NewServerTLS(
				c.Endpoint,
				l.onEvent,
				l.onAccept,
				l.onClose,
				&tls.Config{
					Certificates: []tls.Certificate{cer},
				},
			))
	} else {
		s = mkClearServer(
			redcon.NewServer(
				c.Endpoint,
				l.onEvent,
				l.onAccept,
				l.onClose,
			))
	}

	l.server = s
	return l, nil
}

func (lc *listenerConfig) TypeName() string {
	return lc.Type
}

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Type() string {
	return "redis"
}

func (l *listener) vhs() *RV.VHost {
	vptr := l.vhostPtr()
	return (*vptr).(*RV.VHost)
}

func (l *listener) vhostPtr() *server.VHost {
	x := (*server.VHost)(atomic.LoadPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&l.vhost)),
	))
	return x
}

func (l *listener) AddVHost(x server.VHost) error {
	y := l.vhs()
	if y.Name() == x.Name() {
		return fmt.Errorf("vhost has already been added")
	}
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&l.vhost)),
		unsafe.Pointer(&x),
	)
	return nil
}

func (l *listener) UpdateVHost(x server.VHost) {
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&l.vhost)),
		unsafe.Pointer(&x),
	)
}

func (l *listener) RemoveVHost(n string) {
	for {
		x := l.vhostPtr()
		if (*x).Name() == n {
			if atomic.CompareAndSwapPointer(
				(*unsafe.Pointer)(unsafe.Pointer(&l.vhost)),
				unsafe.Pointer(x),
				nil,
			) {
				break
			}
		} else {
			break
		}
	}
}

func (l *listener) GetVHost(name string) server.VHost {
	x := l.vhs()
	if x.Name() == name {
		return x
	}
	return nil
}

func (l *listener) Run() error {
	return l.server.ListenAndServe()
}

func init() {
	server.AddListenerFactory(
		"redis",
		&fac{},
	)
}
