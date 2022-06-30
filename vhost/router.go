package vhost

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

var httpmethodlist = []string{
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"PURGE",
	"OPTION",
	"HEAD",
}

// parse the compact router representation to gorilla's mux
func newRouter(
	router string,
	r *mux.Router,
	callback func(http.ResponseWriter, *http.Request),
) (*mux.Route, error) {

	methodList := []string{}
	path := ""

	{
		startSqr := strings.Index(router, "[")
		if startSqr == -1 {
			return nil, fmt.Errorf("invalid router: %s, cannot find [", router)
		}

		endSqr := strings.Index(router, "]")
		if endSqr == -1 {
			return nil, fmt.Errorf("invalid router: %s, cannot find ]", router)
		}

		mlist := strings.TrimSpace(router[startSqr+1 : endSqr])
		if mlist == "*" {
			methodList = httpmethodlist
		} else {
			x := strings.Split(mlist, ",")
			for _, xx := range x {
				methodList = append(methodList,
					strings.ToUpper(
						strings.TrimSpace(xx),
					),
				)
			}
		}

		// now get the rest as path
		path = router[endSqr+1:]
		path = checkrouterpath(path)
	}

	rr := r.HandleFunc(path, callback)

	if len(methodList) != 0 {
		rr.Methods(methodList...)
	}

	return rr, nil
}

func checkrouterpath(p string) string {
	idx := strings.LastIndex(p, "*")
	if idx == -1 {
		return p
	} else {
		suffix := p[:idx]
		if len(suffix) != 0 && suffix[len(suffix)-1] == '/' {
			return fmt.Sprintf("%s{_Rest:.*}", suffix)
		} else {
			return fmt.Sprintf("%s/{_Rest:.*}", suffix)
		}
	}
}
