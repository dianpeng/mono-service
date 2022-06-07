package service

import (
	"fmt"
	"github.com/dianpeng/mono-service/config"
)

type Service interface {
	Name() string

	IDL() string

	NewSession() (Session, error)
}

type ServiceFactory interface {
	Name() string

	Comment() string

	IDL() string

	Create(*config.Service) (Service, error)
}

var serviceMap map[string]ServiceFactory = make(map[string]ServiceFactory)

func ForeachServiceFactory(fn func(ServiceFactory)) {
	for _, v := range serviceMap {
		fn(v)
	}
}

func RegisterServiceFactory(sf ServiceFactory) error {
	_, ok := serviceMap[sf.Name()]
	if ok {
		return fmt.Errorf("service %s already existed", sf.Name())
	}
	serviceMap[sf.Name()] = sf
	return nil
}

func GetServiceFactory(name string) ServiceFactory {
	x, ok := serviceMap[name]
	if !ok {
		return nil
	} else {
		return x
	}
}
