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

	// configuration payload
	var configData string

	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot load from %s: %s", *configPath, err.Error())
			return
		}
		configData = string(data)
	} else {
		printHelp()
		return
	}

	config, err := config.ParseConfig(configData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot parse config: %s", err.Error())
		return
	}

	hsvc, err := hservice.NewHService(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "service creation error: %s", err.Error())
		return
	}

	fmt.Println("Service start to run")
	hsvc.Run()
}
