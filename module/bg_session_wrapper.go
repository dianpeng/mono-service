package module

import (
	"github.com/dianpeng/mono-service/framework"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/phase"
	"github.com/dianpeng/mono-service/pl"
)

type bgApplicationWrapper struct {
	parent      hpl.SessionWrapper
	application framework.Application
}

func (b *bgApplicationWrapper) OnLoadVar(name string) (pl.Val, error) {
	return b.application.OnLoadVar(phase.PhaseBackground, name)
}

func (b *bgApplicationWrapper) OnStoreVar(name string, value pl.Val) error {
	return b.application.OnStoreVar(phase.PhaseBackground, name, value)
}

func (b *bgApplicationWrapper) OnAction(name string, val pl.Val) error {
	return b.application.OnAction(phase.PhaseBackground, name, val)
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
