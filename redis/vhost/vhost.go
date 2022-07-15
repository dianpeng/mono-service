package vhost

import (
	"fmt"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/manifest"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/server"
	"github.com/dianpeng/mono-service/util"
	"github.com/tidwall/redcon"
)

// For redis, each vhost can only attach to one single listener
type VHostConfig struct {
	Name      string
	Comment   string
	Listener  string
	LogFormat string

	SessionCacheSize           int
	HttpClientPoolMaxSize      int64
	HttpClientPoolTimeout      int64
	HttpClientPoolMaxDrainSize int64
}

type VHost struct {
	Config      *VHostConfig
	Module      *pl.Module
	LogFormat   *alog.Format
	clientPool  *util.HClientPool
	servicePool servicePool
}

type VHostConfigBuilder struct {
	configPush bool
	config     *VHostConfig
}

func (x *VHost) uploadLog(_ *alog.Log, _ alog.Provider) {
}

func (x *VHost) OnAccept(
	conn redcon.Conn,
) bool {
	handler := x.getServiceHandler()
	return handler.onAccept(conn)
}

func (x *VHost) OnEvent(
	conn redcon.Conn,
	cmd redcon.Command,
) {
	handler := x.getServiceHandler()
	handler.onEvent(conn, cmd)
}

func (x *VHost) OnClose(
	conn redcon.Conn,
	err error,
) {
	handler := x.getServiceHandler()
	handler.onClose(conn, err)
}

func (x *VHost) getServiceHandler() *serviceHandler {
	h := x.servicePool.get()
	if h != nil {
		return h
	}
	return newServiceHandler(x)
}

func (v *VHost) ListenerName() string {
	return v.Config.Listener
}

func (v *VHost) ListenerType() string {
	return "redis"
}

func (v *VHost) Name() string {
	return v.Config.Name
}

type vhostfac struct{}

func (v *vhostfac) New(x *manifest.Manifest) (server.VHost, error) {
	return CreateVHost(x)
}

func init() {
	server.AddVHostFactory(
		"redis",
		&vhostfac{},
	)
}

func (config *VHostConfig) Compose(p *pl.Module) (*VHost, error) {
	vhost := &VHost{}
	{
		logFormat := util.NotZeroStr(
			config.LogFormat,
			g.VHostLogFormat,
		)

		logf, err := alog.CompileFormat(logFormat)
		if err != nil {
			return nil, err
		}
		vhost.LogFormat = logf
	}

	vhost.Config = config
	vhost.Module = p
	vhost.clientPool = util.NewHClientPool(
		config.Name,
		util.NotZeroInt64(config.HttpClientPoolMaxSize, g.VHostHttpClientPoolMaxSize),
		util.NotZeroInt64(config.HttpClientPoolTimeout, g.VHostHttpClientPoolTimeout),
		util.NotZeroInt64(config.HttpClientPoolMaxDrainSize, g.VHostHttpClientPoolMaxDrainSize),
	)

	vhost.servicePool = newServicePool(
		int(config.SessionCacheSize),
	)

	return vhost, nil
}

func (x *VHostConfigBuilder) PushConfig(
	_ *pl.Evaluator,
	name string,
	_ pl.Val,
) error {
	if x.configPush {
		return fmt.Errorf("redis_vhost: nested config scope is not allowed")
	}
	if name != "redis_vhost" {
		return fmt.Errorf("redis_vhost config: unknown config type, expect redis_vhost")
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

func (x *VHostConfigBuilder) ConfigProperty(
	_ *pl.Evaluator,
	key string,
	value pl.Val,
	_ pl.Val,
) error {
	if !x.configPush {
		return fmt.Errorf("config property must be inside of redis_vhost scope")
	}

	switch key {
	case "name":
		return propSetString(
			value,
			&x.config.Name,
			"redis_vhost.name",
		)

	case "comment":
		return propSetString(
			value,
			&x.config.Comment,
			"redis_vhost.comment",
		)

	case "listener":
		return propSetString(
			value,
			&x.config.Listener,
			"redis_vhost.listener",
		)

	case "log_format":
		return propSetString(
			value,
			&x.config.LogFormat,
			"redis_vhost.LogFormat",
		)

	case "session_cache_size":
		return propSetInt(
			value,
			&x.config.SessionCacheSize,
			"redis_vhost.SessionCacheSize",
		)

	case "http_client_pool_max_size":
		return propSetInt64(
			value,
			&x.config.HttpClientPoolMaxSize,
			"redis_vhost.HttpClientPoolMaxSize",
		)

	case "http_client_pool_timeout":
		return propSetInt64(
			value,
			&x.config.HttpClientPoolTimeout,
			"redis_vhost.HttpClientPoolTimeout",
		)

	case "http_client_pool_max_drain_size":
		return propSetInt64(
			value,
			&x.config.HttpClientPoolMaxDrainSize,
			"redis_vhost.HttpClientPoolMaxDrainSize",
		)

	default:
		break
	}

	return fmt.Errorf("redis_vhost: unknown property: %s", key)
}

func (x *VHostConfigBuilder) ConfigCommand(
	_ *pl.Evaluator,
	key string,
	_ []pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("redis_vhost: unknown command %s", key)
}
