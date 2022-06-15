package hpl

import (
	"fmt"
	"io"
	"net/http"

	// router
	"github.com/dianpeng/mono-service/pl"
)

// Notes, the http::do/get/post is specialized function that is not exposed via
// builtins but exposed by HPL since HPL holds the client pool

// general HTTP request interfaces, allowing user to specify following method
// http(
//   [0]: url(string)
//   [1]: method(string)
//   [2]: [header](list[string]),
//   [3]: [body](string)
// )

var (
	fnProtoHttpGet  = pl.MustNewModFuncProto("http", "get", "{%s}{%s%a}")
	fnProtoHttpPost = pl.MustNewModFuncProto("http", "post", "{%s%a}{%s%a%a}")

	fnProtoHttpDo = pl.MustNewModFuncProto("http", "do",
		"{%U['http.request']}"+ /* just request */
			"{%s%s}"+ /* url, method */
			"{%s%s%a}"+ /* url, method, header */
			"{%s%s%a%a}", /* url, method, header, body(string) */
	)
)

func fnHttpGet(factory HttpClientFactory, argument []pl.Val) (pl.Val, error) {
	asize, err := fnProtoHttpGet.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	hreq, err := http.NewRequest(
		"GET",
		argument[0].String(),
		http.NoBody,
	)

	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::get cannot create request: %s", err.Error())
	}

	if asize == 2 {
		hval, err := NewHeaderValFromVal(argument[1])
		if err != nil {
			return pl.NewValNull(), fmt.Errorf("http::get cannot create header: %s", err.Error())
		}
		hdr, _ := hval.Usr().(*Header)
		hreq.Header = hdr.HttpHeader()
	}

	client, err := factory.GetHttpClient(argument[0].String())
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::get cannot create client: %s", err.Error())
	}

	resp, err := client.Do(hreq)
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::get cannot issue request: %s", err.Error())
	}

	return NewResponseVal(resp), nil
}

func fnHttpPost(factory HttpClientFactory, argument []pl.Val) (pl.Val, error) {
	asize, err := fnProtoHttpPost.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	bodyval := pl.NewValNull()
	headerval := pl.NewValNull()

	if asize == 2 {
		if b, err := NewBodyValFromVal(argument[1]); err != nil {
			return pl.NewValNull(), fmt.Errorf("http::post cannot create body: %s", err.Error())
		} else {
			bodyval = b
		}
	} else {
		if b, err := NewHeaderValFromVal(argument[1]); err != nil {
			return pl.NewValNull(), fmt.Errorf("http::post cannot create header: %s", err.Error())
		} else {
			headerval = b
		}

		if b, err := NewBodyValFromVal(argument[2]); err != nil {
			return pl.NewValNull(), fmt.Errorf("http::post cannot create body: %s", err.Error())
		} else {
			bodyval = b
		}
	}

	body, _ := bodyval.Usr().(*Body)

	hreq, err := http.NewRequest(
		"POST",
		argument[0].String(),
		body.Stream().Stream,
	)

	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::post cannot create request: %s", err.Error())
	}

	if headerval.Type != pl.ValNull {
		hdr, _ := headerval.Usr().(*Header)
		hreq.Header = hdr.HttpHeader()
	}

	client, err := factory.GetHttpClient(argument[0].String())
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::post cannot create client: %s", err.Error())
	}

	resp, err := client.Do(hreq)
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::post cannot issue request: %s", err.Error())
	}

	return NewResponseVal(resp), nil
}

func fnHttpDo(factory HttpClientFactory, argument []pl.Val) (pl.Val, error) {
	asize, err := fnProtoHttpDo.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	var req *http.Request

	if asize > 1 {
		url := argument[0].String()
		method := argument[1].String()

		var body io.Reader
		body = http.NoBody

		if asize == 4 && argument[3].Type != pl.ValNull {
			bodyval, err := NewBodyValFromVal(argument[3])
			if err != nil {
				return pl.NewValNull(), fmt.Errorf("http::do cannot create body: %s", err.Error())
			}
			b, _ := bodyval.Usr().(*Body)
			body = b.Stream().Stream
		}

		if hreq, err := http.NewRequest(method, url, body); err != nil {
			return pl.NewValNull(), fmt.Errorf("http::do cannot create request: %s", err.Error())
		} else {
			req = hreq
		}

		if asize >= 3 && argument[3].Type != pl.ValNull {
			hdrval, err := NewHeaderValFromVal(argument[2])
			if err != nil {
				return pl.NewValNull(), fmt.Errorf("http::do cannot create header: %s", err.Error())
			}
			b, _ := hdrval.Usr().(*Header)
			req.Header = b.HttpHeader()
		}
	} else {
		hreq, _ := argument[0].Usr().(*Request)
		req = hreq.HttpRequest()
	}

	client, err := factory.GetHttpClient(req.URL.String())
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::do cannot create client: %s", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::do cannot issue request: %s", err.Error())
	}

	// serialize the response back to the normal object
	return NewResponseVal(resp), nil
}
