package module

import (
	"fmt"
	"net/http"

	"github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"
)

type nullService struct {
}

func (s *nullService) Name() string {
	return "null"
}

func (s *nullService) IDL() string {
	return ""
}

type nullSession struct {
	service *nullService
	r       service.SessionResource
}

func (s *nullSession) OnLoadVar(_ int, _ *pl.Evaluator, name string) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("module(null): unknown variable %s", name)
}

func (s *nullSession) OnStoreVar(_ int, _ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf("module(null): unknown variable %s for storing", name)
}

func (s *nullSession) OnCall(_ int, _ *pl.Evaluator, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("module(null): unknown function %s", name)
}

func (s *nullSession) OnAction(_ int, _ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf("module(null): unknown action %s", name)
}

func (s *nullSession) Service() service.Service {
	return s.service
}

func (s *nullSession) Start(r service.SessionResource) error {
	s.r = r
	return nil
}

func (s *nullSession) Done(_ interface{}) {
	s.r = nil
}

func (s *nullSession) SessionResource() service.SessionResource {
	return s.r
}

func (s *nullSession) Prepare(_ *http.Request, _ hrouter.Params) (interface{}, error) {
	return nil, nil
}

func (s *nullSession) Accept(_ interface{}, _ *hpl.Hpl, _ hpl.SessionWrapper) (service.SessionResult, error) {
	return service.SessionResult{
		Event: "response",
	}, nil
}

func (s *nullService) NewSession() (service.Session, error) {
	session := &nullSession{}
	session.service = s
	return session, nil
}

type nullServiceFactory struct {
}

func (n *nullServiceFactory) Name() string {
	return "null"
}

func (n *nullServiceFactory) Comment() string {
	return "a basic simple service does nothing except evaluate the policy specified"
}

func (n *nullServiceFactory) IDL() string {
	return ""
}

func (n *nullServiceFactory) Create(config *config.Service) (service.Service, error) {
	return &nullService{}, nil
}

func init() {
	service.RegisterServiceFactory(&nullServiceFactory{})
}
