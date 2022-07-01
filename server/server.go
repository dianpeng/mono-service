package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	_ "github.com/dianpeng/mono-service/application"
	"github.com/dianpeng/mono-service/vhost"
)

type ListenerConfig struct {
	Name              string `json:"name"`
	Endpoint          string `json:"endpoint"`
	ReadTimeout       int64  `json:"read_timeout"`
	WriteTimeout      int64  `json:"write_timeout"`
	IdleTimeout       int64  `json:"idle_timeout"`
	ReadHeaderTimeout int64  `json:"read_header_timeout"`
	MaxHeaderSize     int64  `json:"max_header_size"`
}

type Server struct {
	listener []*listener
	wg       sync.WaitGroup
}

// create a new server with corresponding
func NewServer(cfgList []ListenerConfig) (*Server, error) {
	s := &Server{}
	for _, x := range cfgList {
		l := newListener(x)
		s.listener = append(s.listener, l)
	}
	return s, nil
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

func (s *Server) getListener(x string) *listener {
	for _, l := range s.listener {
		if l.name == x {
			return l
		}
	}
	return nil
}

func (s *Server) AddVirtualHost(
	config *vhost.Manifest,
) error {
	vhost, err := vhost.CreateVHost(config)
	if err != nil {
		return err
	}
	listener := s.getListener(vhost.Config.Listener)
	if listener == nil {
		return fmt.Errorf("listener: %s is not existed", vhost.Config.Listener)
	}
	return listener.addVHost(vhost)
}

// run all the listener
func (s *Server) Run() {
	s.wg.Add(len(s.listener))

	for _, vv := range s.listener {
		go func() {
			defer s.wg.Done()
			err := vv.run()
			if err != nil {
				fmt.Printf("error: %s", err.Error())
			}
		}()
	}

	fmt.Printf("Server has been started")
	s.wg.Wait()
}
