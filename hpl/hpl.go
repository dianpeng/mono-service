package hpl

import (
	"fmt"
	"net/http"
)

func must(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("must: %s", msg))
	}
}

func unreachable(msg string) {
	panic(fmt.Sprintf("unreachable: %s", msg))
}

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type HttpClientFactory interface {
	GetHttpClient(url string) (HttpClient, error)
}
