package module

import (
	"fmt"
	"net/http"

	"github.com/dianpeng/mono-service/framework"
	"github.com/dianpeng/mono-service/hrouter"
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

func (s *nullApplication) OnLoadVar(_ int, name string) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("module(null): unknown variable %s", name)
}

func (s *nullApplication) OnStoreVar(_ int, name string, _ pl.Val) error {
	return fmt.Errorf("module(null): unknown variable %s for storing", name)
}

func (s *nullApplication) OnAction(_ int, name string, _ pl.Val) error {
	return fmt.Errorf("module(null): unknown action %s", name)
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
