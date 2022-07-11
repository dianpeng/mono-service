package application

import (
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"
)

type bgApplicationWrapper struct {
	parent      hpl.SessionWrapper
	application framework.Application
}

func (b *bgApplicationWrapper) OnLoadVar(x *pl.Evaluator, name string) (pl.Val, error) {
	return b.parent.OnLoadVar(x, name)
}

func (b *bgApplicationWrapper) OnStoreVar(x *pl.Evaluator, name string, value pl.Val) error {
	return b.parent.OnStoreVar(x, name, value)
}

func (b *bgApplicationWrapper) OnAction(x *pl.Evaluator, name string, val pl.Val) error {
	return b.parent.OnAction(x, name, val)
}

func (b *bgApplicationWrapper) GetHttpClient(url string) (hpl.HttpClient, error) {
	// parent's GetHttpClient is always thread safe
	return b.parent.GetHttpClient(url)
}

func newBgApplicationWrapper(
	parent hpl.SessionWrapper,
	application framework.Application,
) *bgApplicationWrapper {
	return &bgApplicationWrapper{
		parent:      parent,
		application: application,
	}
}
