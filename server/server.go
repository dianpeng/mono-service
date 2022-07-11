package server

import (
	"fmt"
	"sync"

	"github.com/dianpeng/mono-service/http/vhost"

	// for side effect
	_ "github.com/dianpeng/mono-service/http/ext/application"
	_ "github.com/dianpeng/mono-service/http/ext/request"
	_ "github.com/dianpeng/mono-service/http/ext/response"
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
	return listener.AddVHost(vhost)
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
	vhost *vhost.VHost,
) error {
	lis := s.getListener(
		vhost.Config.Listener,
	)
	if lis == nil {
		return fmt.Errorf("listener %s is not existed", vhost.Config.Listener)
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
	vhost *vhost.VHost,
) {
	l := s.getListener(vhost.Config.Listener)
	if l != nil {
		l.UpdateVHost(vhost)
	}
}
