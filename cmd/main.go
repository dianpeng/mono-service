package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/dianpeng/mono-service/application"
	"github.com/dianpeng/mono-service/server"
)

func printHelp() {
	flag.PrintDefaults()
}

func main() {
	path := flag.String("path", "", "the path to your project")
	flag.Parse()

	if *path == "" {
		printHelp()
		return
	}

	hsvc, err := server.NewHService([]string{*path})
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	}

	fmt.Println("Service start to run")
	hsvc.Run()
}
