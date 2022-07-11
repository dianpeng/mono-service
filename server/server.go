package server

import (
	"fmt"
	"sync"

	"github.com/dianpeng/mono-service/manifest"

	// for side effect
	_ "github.com/dianpeng/mono-service/http/prelude"
)

type Server struct {
	listener []Listener
	wg       sync.WaitGroup
}

// create a new server with corresponding
func NewServer(cfgList []ListenerConfig) (*Server, error) {
	s := &Server{}
	for _, x := range cfgList {
		f := GetListenerFactory(x.Type)
		if f == nil {
			return nil, fmt.Errorf("unknown listener type: %s", x.Type)
		}
		l, err := f.New(x)
		if err != nil {
			return nil, fmt.Errorf("cannot create listener: %s", err.Error())
		}
		s.listener = append(s.listener, l)
	}
	return s, nil
}

func (s *Server) getListener(x string) Listener {
	for _, l := range s.listener {
		if l.Name() == x {
			return l
		}
	}
	return nil
}

func (s *Server) AddVirtualHost(
	config *manifest.Manifest,
) error {
	fac := GetVHostFactory(config.Type)
	if fac == nil {
		return fmt.Errorf("listener: unknown manifest type %s", config.Type)
	}
	vhost, err := fac.New(config)
	if err != nil {
		return err
	}
	if vhost.ListenerType() != config.Type {
		return fmt.Errorf("listener: mismatched listener type %s and vhost type %s",
			vhost.ListenerType(),
			config.Type,
		)
	}

	if listener := s.getListener(vhost.ListenerName()); listener == nil {
		return fmt.Errorf("listener: %s is not existed", vhost.ListenerName())
	} else {
		return listener.AddVHost(vhost)
	}
}

// run all the listener
func (s *Server) Run() {
	s.wg.Add(len(s.listener))

	for _, vv := range s.listener {
		go func() {
			defer s.wg.Done()
			err := vv.Run()
			if err != nil {
				fmt.Printf("error: %s", err.Error())
			}
		}()
	}

	fmt.Printf("Server has been started")
	s.wg.Wait()
}

func (s *Server) AddVHost(
	vhost VHost,
) error {
	lis := s.getListener(
		vhost.ListenerName(),
	)
	if lis == nil {
		return fmt.Errorf("listener %s is not existed", vhost.ListenerName())
	}

	return lis.AddVHost(vhost)
}

func (s *Server) RemoveVHostFromListener(
	listenerName string,
	vhostName string,
) {
	lis := s.getListener(
		listenerName,
	)
	lis.RemoveVHost(vhostName)
}

func (s *Server) RemoveVHost(
	vhostName string,
) {
	for _, lis := range s.listener {
		lis.RemoveVHost(vhostName)
	}
}

func (s *Server) UpdateVHost(
	vhost VHost,
) {
	l := s.getListener(vhost.ListenerName())
	if l != nil {
		l.UpdateVHost(vhost)
	}
}
