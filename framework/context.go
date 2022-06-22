package framework

import (
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/pl"
)

type ServiceContext interface {
	Hpl() *hpl.Hpl
	HplSessionWrapper() hpl.SessionWrapper
}

type HplContext interface {
	OnLoadVar(int, string) (pl.Val, error)
	OnStoreVar(int, string, pl.Val) error
	OnAction(int, string, pl.Val) error
}
