package application

import (
	"net/http"

	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"
)

type nullFactory struct{}

type nullApplication struct{}

func (s *nullFactory) Name() string {
	return "null"
}

func (s *nullFactory) Comment() string {
	return "a simple module that does basically nothing"
}

func (s *nullApplication) Done(_ interface{}) {
}

func (s *nullApplication) Prepare(_ *http.Request, _ hrouter.Params) (interface{}, error) {
	return nil, nil
}

func (s *nullApplication) Accept(_ interface{}, _ framework.ServiceContext) (framework.ApplicationResult, error) {
	return framework.ApplicationResult{}, nil
}

func (s *nullFactory) Create(_ []pl.Val) (framework.Application, error) {
	return &nullApplication{}, nil
}

func init() {
	framework.AddApplicationFactory(
		"noop",
		&nullFactory{},
	)
}
