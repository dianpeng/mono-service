package vhost

import (
	"fmt"

	"github.com/gorilla/mux"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/hclient"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/util"
)

type VHostConfig struct {
	Name       string
	Path       string
	Comment    string
	ServerName string
	Listener   string
	LogFormat  string

	HttpClientPoolMaxSize      int64
	HttpClientPoolTimeout      int64
	HttpClientPoolMaxDrainSize int64
}

type VHost struct {
	ServiceList []*vHS
	Router      *mux.Router
	LogFormat   *alog.SessionLogFormat
	Config      *VHostConfig
	Module      *pl.Module
	clientPool  *hclient.HClientPool
}

type VHostConfigBuilder struct {
	configPush bool
	config     *VHostConfig
}

func (config *VHostConfig) Compose(p *pl.Module) (*VHost, error) {
	VHost := &VHost{}

	{
		logFormat := util.NotZeroStr(
			config.LogFormat,
			g.VHostLogFormat,
		)

		logf, err := alog.NewSessionLogFormat(logFormat)
		if err != nil {
			return nil, err
		}
		VHost.LogFormat = logf
	}

	router := mux.NewRouter()

	// finish the creation of VHost object
	VHost.Config = config
	VHost.Router = router
	VHost.ServiceList = nil
	VHost.Module = p

	VHost.clientPool = hclient.NewHClientPool(
		config.Name,
		util.NotZeroInt64(config.HttpClientPoolMaxSize, g.VHostHttpClientPoolMaxSize),
		util.NotZeroInt64(config.HttpClientPoolTimeout, g.VHostHttpClientPoolTimeout),
		util.NotZeroInt64(config.HttpClientPoolMaxDrainSize, g.VHostHttpClientPoolMaxDrainSize),
	)

	return VHost, nil
}

func (x *VHostConfigBuilder) PushConfig(
	_ *pl.Evaluator,
	name string,
	_ pl.Val,
) error {
	if x.configPush {
		return fmt.Errorf("virtual_host config: nested config scope is not allowed")
	}
	if name != "virtual_host" {
		return fmt.Errorf("virtual_host config: unknown config type, expect virtual_host")
	}

	x.configPush = true
	return nil
}

func (x *VHostConfigBuilder) PopConfig(
	_ *pl.Evaluator,
) error {
	x.configPush = false
	return nil
}

func (s *VHostConfigBuilder) ConfigProperty(
	_ *pl.Evaluator,
	key string,
	value pl.Val,
	_ pl.Val,
) error {

	if !s.configPush {
		return fmt.Errorf("config property must be set inside of virtual_host scope")
	}

	switch key {
	case "name":
		return propSetString(
			value,
			&s.config.Name,
			"virtual_host.name",
		)

	case "comment":
		return propSetString(
			value,
			&s.config.Comment,
			"virtual_host.comment",
		)

	case "server_name":
		return propSetString(
			value,
			&s.config.ServerName,
			"virtual_host.server_name",
		)

	case "listener":
		return propSetString(
			value,
			&s.config.Listener,
			"virtual_host.listener",
		)

	case "log_format":
		return propSetString(
			value,
			&s.config.LogFormat,
			"virtual_host.log_format",
		)

	case "http_client_pool_max_size":
		return propSetInt64(
			value,
			&s.config.HttpClientPoolMaxSize,
			"virtual_host.http_client_pool_max_size",
		)

	case "http_client_pool_timeout":
		return propSetInt64(
			value,
			&s.config.HttpClientPoolTimeout,
			"virtual_host.http_client_pool_timeout",
		)

	case "http_client_pool_max_drain_size":
		return propSetInt64(
			value,
			&s.config.HttpClientPoolMaxDrainSize,
			"virtual_host.http_client_pool_max_drain_size",
		)

	default:
		break
	}

	return fmt.Errorf("virtual_host: unknown property: %s", key)
}

func (x *VHostConfigBuilder) ConfigCommand(
	_ *pl.Evaluator,
	key string,
	_ []pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("virtual_host: unknown command %s", key)
}

func (v *VHost) uploadLog(_ *alog.SessionLog) {
	// TODO(dpeng): Add log sinking services
}
