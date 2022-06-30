package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dianpeng/mono-service/server"
	"github.com/dianpeng/mono-service/vhost"
	"os"
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
		l := server.NewDefaultListenerConfig()
		if err := json.Unmarshal([]byte(cfg), &l); err != nil {
			return nil, err
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

	httpListener := flag.Bool("http", false, "default listener, ie :80")
	testListener := flag.Bool("test", false, "default testing listener, ie :18080")

	flag.Parse()

	if *httpListener {
		listenerConf = append(listenerConf, `
    {
      "name" : "http",
      "endpoint" : ":80"
    }
    `)
	}
	if *testListener {
		listenerConf = append(listenerConf, `
    {
      "name" : "test",
      "endpoint" : ":18080"
    }
    `)
	}

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
