package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ListenerConfig struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Endpoint          string `json:"endpoint"`
	ReadTimeout       int64  `json:"read_timeout"`
	WriteTimeout      int64  `json:"write_timeout"`
	IdleTimeout       int64  `json:"idle_timeout"`
	ReadHeaderTimeout int64  `json:"read_header_timeout"`
	MaxHeaderSize     int64  `json:"max_header_size"`
}

func ParseListenerConfigFromJSON(input string) (ListenerConfig, error) {
	o := ListenerConfig{
		Name:              "",
		Type:              "",
		Endpoint:          "",
		ReadTimeout:       20,
		WriteTimeout:      20,
		IdleTimeout:       90,
		ReadHeaderTimeout: 10,
		MaxHeaderSize:     1024 * 64,
	}
	if err := json.Unmarshal([]byte(input), &o); err != nil {
		return o, err
	}

	if o.Name == "" {
		return o, fmt.Errorf("must specify Name for listener config")
	}

	if o.Endpoint == "" {
		return o, fmt.Errorf("must specify Endpoint for listener config")
	}

	return o, nil
}

func ParseListenerConfigFromCompact(input string) (ListenerConfig, error) {
	conf := ListenerConfig{}
	x := strings.Split(input, ",")
	if len(x) < 3 {
		return conf, fmt.Errorf("invalid listener config: %s, at least 3 elements are needed", input)
	}

	conf.Type = x[0]
	conf.Name = x[1]
	conf.Endpoint = x[2]

	parseInt := func(field string, index int, out *int64) error {
		if len(x) > index {
			ival, err := strconv.ParseInt(x[index], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid listener config field %s, must be valid "+
					"integer, but has error: %s", field, err.Error())
			}
			*out = ival
		}
		return nil
	}

	if err := parseInt("ReadTimeout", 3, &conf.ReadTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("WriteTimeout", 4, &conf.WriteTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("IdleTimeout", 5, &conf.IdleTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("ReadHeaderTimeout", 6, &conf.ReadHeaderTimeout); err != nil {
		return conf, err
	}
	if err := parseInt("MaxHeaderSize", 7, &conf.MaxHeaderSize); err != nil {
		return conf, err
	}

	return conf, nil
}

// listener factory -----------------------------------------------------------
type ListenerFactory interface {
	New(ListenerConfig) (Listener, error)
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
