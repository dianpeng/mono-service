package module

import (
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/phase"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"
)

type bgSessionWrapper struct {
	parent  hpl.SessionWrapper
	session service.Session
}

func (b *bgSessionWrapper) OnLoadVar(x *pl.Evaluator, name string) (pl.Val, error) {
	return b.session.OnLoadVar(phase.PhaseBackground, x, name)
}

func (b *bgSessionWrapper) OnStoreVar(x *pl.Evaluator, name string, value pl.Val) error {
	return b.session.OnStoreVar(phase.PhaseBackground, x, name, value)
}

func (b *bgSessionWrapper) OnCall(x *pl.Evaluator, name string, args []pl.Val) (pl.Val, error) {
	return b.session.OnCall(phase.PhaseBackground, x, name, args)
}

func (b *bgSessionWrapper) OnAction(x *pl.Evaluator, name string, val pl.Val) error {
	return b.session.OnAction(phase.PhaseBackground, x, name, val)
}

func (b *bgSessionWrapper) GetPhaseName() string {
	return ".background"
}

func (b *bgSessionWrapper) GetErrorDescription() string {
	return ""
}

func (b *bgSessionWrapper) GetHttpClient(url string) (hpl.HttpClient, error) {
	// parent's GetHttpClient is always thread safe
	return b.parent.GetHttpClient(url)
}

func newBgSessionWrapper(
	parent hpl.SessionWrapper,
	session service.Session,
) *bgSessionWrapper {
	return &bgSessionWrapper{
		parent:  parent,
		session: session,
	}
}
