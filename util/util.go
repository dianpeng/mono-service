package util

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func fatal(xx string) {
	log.Fatal(xx)
}

func ErrorRequest(w http.ResponseWriter, e error) {
	w.WriteHeader(500)
	w.Write([]byte(fmt.Sprintf("HPL execution error: %s", e.Error())))
}

func JSONSplitLine(input string) interface{} {
	temp := strings.Split(input, "\n")
	if len(temp) == 1 {
		return input
	} else {
		return temp
	}
}

func LoadFile(path string) (string, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

var hostname string

func GetHostname() string {
	return hostname
}

func init() {
	{
		h, err := os.Hostname()
		if err != nil {
			fatal(err.Error())
		}
		hostname = h
	}
}
