package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dianpeng/mono-service/manifest"
	"github.com/dianpeng/mono-service/server"

	// for side effect
	_ "github.com/dianpeng/mono-service/http"
)

func printHelp() {
	flag.PrintDefaults()
}

type strList []string

func (c *strList) String() string {
	return "a list of string"
}

func (c *strList) Set(v string) error {
	*c = append(*c, v)
	return nil
}

func parseListenerConfig(x strList) ([]server.ListenerConfig, error) {
	o := []server.ListenerConfig{}
	for _, cfg := range x {
		c, err := server.ParseListenerConfig(cfg)
		if err != nil {
			return nil, err
		}
		o = append(o, c)
	}
	return o, nil
}

func main() {
	var listenerConf strList
	var httpdir strList

	flag.Var(&listenerConf, "listener", "list of listener config, in Json")
	flag.Var(&httpdir, "http_dir", "list of path to local fs http virtual host")

	flag.Parse()

	lconf, err := parseListenerConfig(listenerConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	}

	srv, err := server.NewServer(lconf)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	}

	for _, m := range httpdir {
		manifest, err := manifest.NewManifestFromLocalDir(
			m,
			"http",
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return
		}

		if err := srv.AddVirtualHost(manifest); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return
		}
	}

	srv.Run()
}
