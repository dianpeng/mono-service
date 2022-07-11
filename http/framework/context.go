package framework

import (
	"github.com/dianpeng/mono-service/http/runtime"
)

type ServiceContext interface {
	Runtime() *runtime.Runtime
	HplSessionWrapper() runtime.SessionWrapper
}
