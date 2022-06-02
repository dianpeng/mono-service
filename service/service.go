package service

import (
	"fmt"
	"github.com/dianpeng/mono-service/config"
)

type Service interface {
	// return the name of the service
	Name() string

	// return the tag of this service
	Tag() string

	// return the IDL description of the service, which is mostly builtin. Notes
	// not in used for now. Will be used in the future
	IDL() string

	// return the policy string that is been used by this service
	Policy() string

	// returns the dumpped policy bytecode
	PolicyDump() string

	// return a list of router strings
	Router() string

	// return a list of method lists that can be accepted for this service
	MethodList() []string

	// http handler
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
