package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/hservice"
)

func printHelp() {
	flag.PrintDefaults()
}

func main() {
	configPath := flag.String("config", "", "the path to the configuration file")
	flag.Parse()
	if *configPath == "" {
		printHelp()
		return
	} else {
		config, err := config.ParseConfigFromFile(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config parsing failed: %s", err.Error())
			return
		}

		// create the hservice
		hsvc, err := hservice.NewHService(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "service loading error: %s", err.Error())
			return
		}

		// run the hsvc
		hsvc.Run()
	}
}
