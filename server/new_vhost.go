package server

import (
	"fmt"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
	"os"
	"path/filepath"
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

func initpolicy(x string, config pl.EvalConfig) (*pl.Policy, error) {
	p, err := pl.CompilePolicy(x)
	if err != nil {
		return nil, err
	}

	session := &constHttpClientFactory{}
	hpl := hpl.NewHpl()

	hpl.SetPolicy(p)

	if err := hpl.OnGlobal(session); err != nil {
		return nil, err
	}

	if err := hpl.OnConfig(config, session); err != nil {
		return nil, err
	}

	return p, nil
}

func wrapE(
	context string,
	phase string,
	path string,
	err error,
) error {

	return fmt.Errorf(
		"Under context %s, policy file %s, has error in phase %s:\n\n%s",
		context,
		path,
		phase,
		err.Error(),
	)
}

func initVHost(
	path string,
) (*vhost, error) {

	vhostSource, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	vhostConfig := &vhostConfig{}
	vhostConfigBuilder := &vhostConfigBuilder{
		config: vhostConfig,
	}

	p, err := initpolicy(string(vhostSource), vhostConfigBuilder)
	if err != nil {
		return nil, wrapE(
			"virtual_host",
			"initialization",
			path,
			err,
		)
	}

	return vhostConfig.Compose(p)
}

func initVHostSVC(
	path string,
	vhost *vhost,
) (*vHS, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &vHSConfig{}
	builder := &svcConfigBuilder{
		config: cfg,
	}

	p, err := initpolicy(
		string(src),
		builder,
	)
	if err != nil {
		return nil, wrapE(
			"service",
			"initialization",
			path,
			err,
		)
	}

	fac, err := cfg.Compose()
	if err != nil {
		return nil, wrapE(
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

func createVHost(
	vhostPath string, // main vhost file path
) (*vhost, error) {
	vhost, err := initVHost(vhostPath)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(vhostPath)

	// now start to walk the directory to find out all the service code definition
	err = filepath.Walk(dir,
		func(path string, info os.FileInfo, e error) error {
			if e != nil {
				return e
			}

			if path == vhostPath {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			// only try file with extension of .pl
			if filepath.Ext(path) != ".pl" {
				return nil
			}

			svc, err := initVHostSVC(
				path,
				vhost,
			)
			if err != nil {
				return err
			}

			// now, populate the router part of the vhs
			_, err = newRouter(
				svc.config.Router,
				vhost.Router,
				func(a http.ResponseWriter, b *http.Request) {
					doRoute(svc, a, b)
				},
			)
			if err != nil {
				return err
			}

			// finally insert the service into the active service list
			vhost.ServiceList = append(vhost.ServiceList, svc)
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return vhost, nil
}
