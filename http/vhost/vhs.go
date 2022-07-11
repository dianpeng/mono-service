package vhost

import (
	"fmt"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"
)

type vHS struct {
	factory     *framework.ServiceFactory
	module      *pl.Module
	config      *vHSConfig
	vhost       *VHost
	servicePool servicePool
}

func (s *vHS) getServiceHandler() (*serviceHandler, error) {
	// try to get a session handler from the pool
	if h := s.servicePool.get(); h != nil {
		return h, nil
	}

	// create a new one
	if service, err := s.factory.NewService(); err != nil {
		return newServiceHandler(nil, s), err
	} else {
		return newServiceHandler(service, s), nil
	}
}

func newvHS(vhost *VHost,
	factory *framework.ServiceFactory,
	config *vHSConfig,
	p *pl.Module,
) (*vHS, error) {
	return &vHS{
		factory:     factory,
		module:      p,
		config:      config,
		vhost:       vhost,
		servicePool: newServicePool(config.MaxSessionCacheSize),
	}, nil
}

// We use a data oriented method, during the bytecode execution, firstly we do
// compose a config object and after the config object is finalized, we use the
// config object to initialize all various component

type vHSMiddlewareConfigEntry struct {
	Name   string
	Config []pl.Val
}

type vHSMiddlewareConfig struct {
	List []vHSMiddlewareConfigEntry
}

type vHSConfig struct {
	// service config property
	Name                string
	Tag                 string
	Comment             string
	Router              string
	MaxSessionCacheSize int

	// middleware part, ie request, response, application etc ...
	Request   vHSMiddlewareConfig
	Response  vHSMiddlewareConfig
	AppName   string
	AppConfig []pl.Val
}

func (cfg *vHSConfig) Compose() (*framework.ServiceFactory, error) {
	o := &framework.ServiceFactory{}
	{
		facList := framework.NewMiddlewareFactoryList(
			"request",
			"",
		)

		for _, x := range cfg.Request.List {
			if err := facList.AddRequest(
				x.Name,
				x.Config,
			); err != nil {
				return nil, err
			}
		}
		o.Request = facList
	}

	{
		facList := framework.NewMiddlewareFactoryList(
			"response",
			"",
		)

		for _, x := range cfg.Response.List {
			if err := facList.AddResponse(
				x.Name,
				x.Config,
			); err != nil {
				return nil, err
			}
		}
		o.Response = facList
	}

	if cfg.AppName == "" {
		return nil, fmt.Errorf("service's application is not set")
	}

	if app := framework.GetApplicationFactory(cfg.AppName); app == nil {
		return nil, fmt.Errorf("application %s is not found", cfg.AppName)
	} else {
		o.App = app
		o.AppConfig = cfg.AppConfig
	}
	return o, nil
}

// config composer, invoked by PL runtime during config scope bytecode execution
const (
	svcConfigInit = iota
	svcConfigService
	svcConfigRequest
	svcConfigResponse
	svcConfigApplication
)

type svcConfigBuilder struct {
	config *vHSConfig
	cur    []int
}

func (s *svcConfigBuilder) curType() int {
	if len(s.cur) == 0 {
		return svcConfigInit
	}
	return s.cur[len(s.cur)-1]
}

func (s *svcConfigBuilder) setCur(x int) {
	s.cur = append(s.cur, x)
}

func (s *svcConfigBuilder) popCur() {
	s.cur = s.cur[:len(s.cur)-1]
}

func (s *svcConfigBuilder) PushConfig(
	_ *pl.Evaluator,
	name string,
	_ pl.Val,
) error {

	if s.curType() == svcConfigInit && name == "service" {
		s.setCur(svcConfigService)
		return nil
	}

	if s.curType() == svcConfigService {
		switch name {
		case "request":
			s.setCur(svcConfigRequest)
			break
		case "response":
			s.setCur(svcConfigResponse)
			break
		case "application":
			s.setCur(svcConfigApplication)
			break
		default:
			return fmt.Errorf("invalid service config status: expect " +
				"request/response/application type")
		}
		return nil
	}

	return fmt.Errorf("cannot have more nested config scope")
}

func (s *svcConfigBuilder) PopConfig(
	_ *pl.Evaluator,
) error {
	s.popCur()
	return nil
}

func (s *svcConfigBuilder) propService(
	key string,
	value pl.Val,
	_ pl.Val,
) error {
	switch key {
	case "name":
		if err := propSetString(
			value,
			&s.config.Name,
			"service.name",
		); err != nil {
			return err
		}
		break

	case "tag":
		if err := propSetString(
			value,
			&s.config.Tag,
			"service.tag",
		); err != nil {
			return err
		}
		break

	case "comment":
		if err := propSetString(
			value,
			&s.config.Comment,
			"service.comment",
		); err != nil {
			return err
		}
		break

	case "router":
		if err := propSetString(
			value,
			&s.config.Router,
			"service.router",
		); err != nil {
			return err
		}
		break

	case "max_session_cache_size":
		if err := propSetInt(
			value,
			&s.config.MaxSessionCacheSize,
			"service.max_session_cache_size",
		); err != nil {
			return err
		}
		break

	default:
		return fmt.Errorf("unknown property %s of service config", key)
	}

	return nil
}

func (s *svcConfigBuilder) propMiddleware(
	key string,
	_ pl.Val,
	_ pl.Val,
	_ *vHSMiddlewareConfig,
) error {
	return fmt.Errorf("unknown property %s of service middleware", key)
}

func (s *svcConfigBuilder) propApplication(
	key string,
	_ pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("unknown property %s of service application", key)
}

func (s *svcConfigBuilder) ConfigProperty(
	_ *pl.Evaluator,
	key string,
	value pl.Val,
	attr pl.Val,
) error {
	switch s.curType() {
	case svcConfigService:
		return s.propService(
			key,
			value,
			attr,
		)

	case svcConfigRequest, svcConfigResponse:
		var x *vHSMiddlewareConfig

		if s.curType() == svcConfigRequest {
			x = &s.config.Request
		} else {
			x = &s.config.Response
		}

		return s.propMiddleware(
			key,
			value,
			attr,
			x,
		)

	case svcConfigApplication:
		return s.propApplication(
			key,
			value,
			attr,
		)

	default:
		break
	}

	return fmt.Errorf("unknown config instruction: %s", key)
}

func (s *svcConfigBuilder) cmdMiddleware(
	key string,
	value []pl.Val,
	_ pl.Val,
	x *vHSMiddlewareConfig,
) error {
	x.List = append(x.List, vHSMiddlewareConfigEntry{
		Name:   key,
		Config: value,
	})
	return nil
}

func (s *svcConfigBuilder) cmdApplication(
	key string,
	value []pl.Val,
	_ pl.Val,
) error {
	s.config.AppName = key
	s.config.AppConfig = value
	return nil
}

func (s *svcConfigBuilder) ConfigCommand(
	_ *pl.Evaluator,
	key string,
	value []pl.Val,
	attr pl.Val,
) error {

	switch s.curType() {
	case svcConfigRequest, svcConfigResponse:
		var x *vHSMiddlewareConfig
		if s.curType() == svcConfigRequest {
			x = &s.config.Request
		} else {
			x = &s.config.Response
		}

		return s.cmdMiddleware(
			key,
			value,
			attr,
			x,
		)

	case svcConfigApplication:
		return s.cmdApplication(
			key,
			value,
			attr,
		)

	default:
		return fmt.Errorf("unknown command %s inside of config service", key)
	}
}
