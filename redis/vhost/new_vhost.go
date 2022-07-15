package vhost

import (
	"fmt"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/manifest"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/redis/runtime"
	"io/fs"
	"net/http"
	"time"
)

type constHttpClientFactory struct {
	timeout int64
}

func (c *constHttpClientFactory) GetHttpClient(url string) (hpl.HttpClient, error) {
	return &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}, nil
}

func initmodule(x string, config pl.EvalConfig, fs fs.FS) (*pl.Module, error) {
	p, err := pl.CompileModule(x, fs)
	if err != nil {
		return nil, err
	}

	session := &constHttpClientFactory{}
	hpl := runtime.NewRuntimeWithModule(p)

	if err := hpl.OnGlobal(session); err != nil {
		return nil, err
	}

	if err := hpl.OnConfig(config, session); err != nil {
		return nil, err
	}

	return p, nil
}

func wrapErr(
	context string,
	phase string,
	path string,
	err error,
) error {

	return fmt.Errorf(
		"Under context %s, module file %s, has error in phase %s:\n\n%s",
		context,
		path,
		phase,
		err.Error(),
	)
}

func initVHost(
	path string,
	fsp fs.FS,
) (*VHost, error) {

	vhostSource, err := fs.ReadFile(fsp, path)
	if err != nil {
		return nil, err
	}

	vhostConfig := &VHostConfig{}
	vhostConfigBuilder := &VHostConfigBuilder{
		config: vhostConfig,
	}

	p, err := initmodule(string(vhostSource), vhostConfigBuilder, fsp)
	if err != nil {
		return nil, wrapErr(
			"redis_vhost",
			"initialization",
			path,
			err,
		)
	}

	return vhostConfig.Compose(p)
}

func CreateVHost(
	manifest *manifest.Manifest,
) (*VHost, error) {
	vhost, err := initVHost(manifest.Main, manifest.FS)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}
