package server

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ListenerConfig interface {
	TypeName() string
}

type ListenerFactory interface {
	New(ListenerConfig) (Listener, error)
	ParseConfigCompact(string) (ListenerConfig, error)
	ParseConfigJson(string) (ListenerConfig, error)
}

var lisfactory = make(map[string]ListenerFactory)

func AddListenerFactory(
	x string,
	f ListenerFactory,
) {
	lisfactory[x] = f
}

func GetListenerFactory(
	x string,
) ListenerFactory {
	v, ok := lisfactory[x]
	if ok {
		return v
	} else {
		return nil
	}
}

type jsonConfig struct {
	Type string `json:"type"`
}

func tryJson(data string) (string, bool) {
	x := jsonConfig{}
	err := json.Unmarshal(
		[]byte(data),
		&x,
	)
	if err != nil {
		return "", false
	} else {
		return x.Type, true
	}
}

func tryCompact(data string) (string, bool) {
	x := strings.Split(data, ",")
	if len(x) == 0 {
		return "", false
	} else {
		return x[0], true
	}
}

func ParseListenerConfig(content string) (ListenerConfig, error) {
	if t, isJson := tryJson(content); isJson {
		factory := GetListenerFactory(t)
		if factory == nil {
			return nil, fmt.Errorf("unknown listener type: %s", t)
		} else {
			return factory.ParseConfigJson(content)
		}
	} else if t, isCompact := tryCompact(content); isCompact {
		factory := GetListenerFactory(t)
		if factory == nil {
			return nil, fmt.Errorf("unknown listener type: %s", t)
		} else {
			return factory.ParseConfigCompact(content)
		}
	} else {
		return nil, fmt.Errorf("invalid listener config: %s", content)
	}
}
