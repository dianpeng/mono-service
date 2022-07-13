package vhost

import (
	"fmt"

	"github.com/gorilla/mux"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/manifest"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/server"
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
	LogFormat   *alog.Format
	Config      *VHostConfig
	Module      *pl.Module
	clientPool  *util.HClientPool
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

		logf, err := alog.CompileFormat(logFormat)
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

	VHost.clientPool = util.NewHClientPool(
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
		return fmt.Errorf("http_vhost config: nested config scope is not allowed")
	}
	if name != "http_vhost" {
		return fmt.Errorf("http_vhost config: unknown config type, expect http_vhost")
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
		return fmt.Errorf("config property must be set inside of http_vhost scope")
	}

	switch key {
	case "name":
		return propSetString(
			value,
			&s.config.Name,
			"http_vhost.name",
		)

	case "comment":
		return propSetString(
			value,
			&s.config.Comment,
			"http_vhost.comment",
		)

	case "server_name":
		return propSetString(
			value,
			&s.config.ServerName,
			"http_vhost.server_name",
		)

	case "listener":
		return propSetString(
			value,
			&s.config.Listener,
			"http_vhost.listener",
		)

	case "log_format":
		return propSetString(
			value,
			&s.config.LogFormat,
			"http_vhost.log_format",
		)

	case "http_client_pool_max_size":
		return propSetInt64(
			value,
			&s.config.HttpClientPoolMaxSize,
			"http_vhost.http_client_pool_max_size",
		)

	case "http_client_pool_timeout":
		return propSetInt64(
			value,
			&s.config.HttpClientPoolTimeout,
			"http_vhost.http_client_pool_timeout",
		)

	case "http_client_pool_max_drain_size":
		return propSetInt64(
			value,
			&s.config.HttpClientPoolMaxDrainSize,
			"http_vhost.http_client_pool_max_drain_size",
		)

	default:
		break
	}

	return fmt.Errorf("http_vhost: unknown property: %s", key)
}

func (x *VHostConfigBuilder) ConfigCommand(
	_ *pl.Evaluator,
	key string,
	_ []pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("http_vhost: unknown command %s", key)
}

func (v *VHost) uploadLog(_ *alog.Log, _ alog.Provider) {
	// TODO(dpeng): Add log sinking services
}

// ----------------------------------------------------------------------------
// server.vhost
func (v *VHost) Name() string {
	return v.Config.Name
}

func (v *VHost) ListenerName() string {
	return v.Config.Listener
}

func (v *VHost) ListenerType() string {
	return "http"
}

type vhostfac struct {
}

func (v *vhostfac) New(x *manifest.Manifest) (server.VHost, error) {
	return CreateVHost(x)
}

func init() {
	server.AddVHostFactory(
		"http",
		&vhostfac{},
	)
}
