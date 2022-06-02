package impl

import (
	"fmt"
	"net/http"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/service"
	"github.com/dianpeng/mono-service/util"
	hrouter "github.com/julienschmidt/httprouter"
)

type nullService struct {
	config *config.Service
	policy *hpl.Policy
}

func (s *nullService) Name() string {
	return "null"
}

func (s *nullService) IDL() string {
	return ""
}

func (s *nullService) Tag() string {
	return s.config.Tag
}

func (s *nullService) Policy() string {
	return s.config.Policy
}

func (s *nullService) PolicyDump() string {
	return s.policy.Dump()
}

func (s *nullService) Router() string {
	return s.config.Router
}

func (s *nullService) MethodList() []string {
	return s.config.Method
}

type nullSession struct {
	hpl     *hpl.Hpl
	service *nullService
	r       service.SessionResource
}

func (s *nullSession) hplLoadVar(x *hpl.Evaluator, name string) (hpl.Val, error) {
	return hpl.NewValNull(), fmt.Errorf("unknown variable %s", name)
}

func (s *nullSession) Service() service.Service {
	return s.service
}

func (s *nullSession) Start(r service.SessionResource) error {
	s.r = r
	return s.hpl.OnGlobal(s)
}

func (s *nullSession) Done() {
	s.r = nil
}

func (s *nullSession) SessionResource() service.SessionResource {
	return s.r
}

func (s *nullSession) Allow(_ *http.Request, _ hrouter.Params) error {
	return nil
}

func (s *nullSession) Accept(w http.ResponseWriter, req *http.Request, p hrouter.Params) {
	err := s.hpl.OnHttpResponse(
		"response",
		hpl.HttpContext{
			Request:        req,
			ResponseWriter: w,
			QueryParams:    p,
		},
		s,
	)

	if err != nil {
		util.ErrorRequest(w, err)
	}
}

func (s *nullSession) Log(log *alog.SessionLog) error {
	return s.hpl.OnLog(
		"log",
		log,
		s,
	)
}

func (s *nullService) NewSession() (service.Session, error) {
	session := &nullSession{}
	hpl := hpl.NewHplWithPolicy(session.hplLoadVar, nil, nil, s.policy)
	session.hpl = hpl
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

	svc := &nullService{
		config: config,
	}

	policy, err := hpl.CompilePolicy(config.Policy)
	if err != nil {
		return nil, err
	}
	svc.policy = policy

	return svc, nil
}

func init() {
	service.RegisterServiceFactory(&nullServiceFactory{})
}
