package vhost

import (
	"fmt"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/http/runtime"
	"github.com/dianpeng/mono-service/manifest"
	"github.com/dianpeng/mono-service/pl"
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
			"http_vhost",
			"initialization",
			path,
			err,
		)
	}

	return vhostConfig.Compose(p)
}

func initVHostSVC(
	path string,
	vhost *VHost,
	fsp fs.FS,
) (*vHS, error) {

	src, err := fs.ReadFile(fsp, path)
	if err != nil {
		return nil, err
	}

	cfg := &vHSConfig{}
	builder := &svcConfigBuilder{
		config: cfg,
	}

	p, err := initmodule(
		string(src),
		builder,
		fsp,
	)
	if err != nil {
		return nil, wrapErr(
			"service",
			"initialization",
			path,
			err,
		)
	}

	fac, err := cfg.Compose()
	if err != nil {
		return nil, wrapErr(
			"service",
			"middleware composition",
			path,
			err,
		)
	}

	return newvHS(
		vhost,
		fac,
		cfg,
		p,
	)
}

func doRoute(
	vhs *vHS,
	writer http.ResponseWriter,
	req *http.Request,
) {

	handler, err := vhs.getServiceHandler()

	handler.main(
		req,
		hrouter.NewParams(req),
		writer,
		err,
	)
}

// Create the full VHOST object from the manifest object
func CreateVHost(
	manifest *manifest.Manifest,
) (*VHost, error) {

	vhost, err := initVHost(manifest.Main, manifest.FS)
	if err != nil {
		return nil, err
	}

	for _, cfg := range manifest.ServiceFile {
		if svc, err := initVHostSVC(
			cfg,
			vhost,
			manifest.FS,
		); err != nil {
			return nil, err
		} else {
			_, err = newRouter(
				svc.config.Router,
				vhost.Router,
				func(a http.ResponseWriter, b *http.Request) {
					doRoute(svc, a, b)
				},
			)
			if err != nil {
				return nil, err
			}
			vhost.ServiceList = append(vhost.ServiceList, svc)
		}
	}

	return vhost, nil
}
