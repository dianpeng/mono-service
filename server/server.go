package server

import (
	"fmt"
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

func NewDefaultListenerConfig() ListenerConfig {
	return ListenerConfig{
		Name:              "",
		Endpoint:          "",
		ReadTimeout:       20,
		WriteTimeout:      20,
		IdleTimeout:       90,
		ReadHeaderTimeout: 10,
		MaxHeaderSize:     1024 * 64,
	}
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
