package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/hclient"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/util"
)

type vhostConfig struct {
	Name       string
	Path       string
	Comment    string
	Endpoint   string
	ServerName string
	LogFormat  string

	MaxHeaderSize int64

	ReadTimeout  int64
	WriteTimeout int64

	HttpClientPoolMaxSize      int64
	HttpClientPoolTimeout      int64
	HttpClientPoolMaxDrainSize int64
}

type vhost struct {
	ServiceList []*vHS
	Router      *mux.Router
	HttpServer  *http.Server
	LogFormat   *alog.SessionLogFormat
	Config      *vhostConfig
	Policy      *pl.Policy
	clientPool  *hclient.HClientPool
}

type vhostConfigBuilder struct {
	configPush bool
	config     *vhostConfig
}

func (config *vhostConfig) Compose(p *pl.Policy) (*vhost, error) {
	vhost := &vhost{}

	{
		logFormat := util.NotZeroStr(
			config.LogFormat,
			g.VHostLogFormat,
		)

		logf, err := alog.NewSessionLogFormat(logFormat)
		if err != nil {
			return nil, err
		}
		vhost.LogFormat = logf
	}

	router := mux.NewRouter()
	if config.ServerName != "" {
		router.Host(config.ServerName)
	}

	// finish the creation of VHost object
	vhost.Config = config
	vhost.Router = router
	vhost.ServiceList = nil
	vhost.Policy = p

	vhost.HttpServer = &http.Server{
		Addr:           config.Endpoint,
		Handler:        router,
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: int(config.MaxHeaderSize),
	}

	vhost.clientPool = hclient.NewHClientPool(
		config.Name,
		util.NotZeroInt64(config.HttpClientPoolMaxSize, g.VHostHttpClientPoolMaxSize),
		util.NotZeroInt64(config.HttpClientPoolTimeout, g.VHostHttpClientPoolTimeout),
		util.NotZeroInt64(config.HttpClientPoolMaxDrainSize, g.VHostHttpClientPoolMaxDrainSize),
	)

	return vhost, nil
}

func (x *vhostConfigBuilder) PushConfig(
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

func (x *vhostConfigBuilder) PopConfig(
	_ *pl.Evaluator,
) error {
	x.configPush = false
	return nil
}

func (s *vhostConfigBuilder) ConfigProperty(
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

	case "endpoint":
		return propSetString(
			value,
			&s.config.Endpoint,
			"virtual_host.endpoint",
		)

	case "server_name":
		return propSetString(
			value,
			&s.config.ServerName,
			"virtual_host.server_name",
		)

	case "log_format":
		return propSetString(
			value,
			&s.config.LogFormat,
			"virtual_host.log_format",
		)

	case "max_header_size":
		return propSetInt64(
			value,
			&s.config.MaxHeaderSize,
			"virtual_host.max_header_size",
		)

	case "read_timeout":
		return propSetInt64(
			value,
			&s.config.ReadTimeout,
			"virtual_host.read_timeout",
		)

	case "write_timeout":
		return propSetInt64(
			value,
			&s.config.WriteTimeout,
			"virtual_host.write_timeout",
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

func (x *vhostConfigBuilder) ConfigCommand(
	_ *pl.Evaluator,
	key string,
	_ []pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("virtual_host: unknown command %s", key)
}

func (v *vhost) uploadLog(_ *alog.SessionLog) {
	// TODO(dpeng): Add log sinking services
}

func (h *vhost) Run() {
	h.HttpServer.ListenAndServe()
}
