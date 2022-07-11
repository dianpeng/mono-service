package framework

import (
	"github.com/dianpeng/mono-service/hpl"
)

type ServiceContext interface {
	Hpl() *hpl.Hpl
	HplSessionWrapper() hpl.SessionWrapper
}
