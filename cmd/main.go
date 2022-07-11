package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dianpeng/mono-service/server"

	// for side effect
	_ "github.com/dianpeng/mono-service/http"
	"github.com/dianpeng/mono-service/http/vhost"
)

func printHelp() {
	flag.PrintDefaults()
}

type strList []string

func isjson(x string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(x), &js) == nil
}

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
		l := server.ListenerConfig{}
		if isjson(cfg) {
			lc, err := server.ParseListenerConfigFromJSON(cfg)
			if err != nil {
				return nil, err
			}
			l = lc
		} else {
			lc, err := server.ParseListenerConfigFromCompact(cfg)
			if err != nil {
				return nil, err
			}
			l = lc
		}
		o = append(o, l)
	}
	return o, nil
}

func main() {
	var listenerConf strList
	var projPath strList

	flag.Var(&listenerConf, "listener", "list of listener config, in JSON")
	flag.Var(&projPath, "path", "list of path to local fs project's main file")
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

	for _, m := range projPath {
		manifest, err := vhost.NewManifestFromLocalDir(
			m,
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
