package framework

import (
	"github.com/dianpeng/mono-service/pl"
)

type ServiceFactory struct {
	Request  MiddlewareFactory // a request middleware
	Response MiddlewareFactory // a response middleware

	App       ApplicationFactory // a application
	AppConfig []pl.Val           // application config
}

type Service struct {
	Request  Middleware
	Response Middleware
	App      Application
}

func (s *ServiceFactory) NewService() (*Service, error) {
	req, err := s.Request.Create(nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Response.Create(nil)
	if err != nil {
		return nil, err
	}

	app, err := s.App.Create(s.AppConfig)
	if err != nil {
		return nil, err
	}

	return &Service{
		Request:  req,
		Response: resp,
		App:      app,
	}, nil
}
