package hpl

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
)

// * --------------------------------------------------------------------------
//   Body
// * --------------------------------------------------------------------------
type HplHttpBody struct {
	Stream    io.Reader
	dupString string // maybe populated to locally cache the content of stream
	hasDup    bool   // indicate whether the dupString is valid or not
}

func HplHttpBodyFromString(data string) *HplHttpBody {
	return &HplHttpBody{
		Stream:    strings.NewReader(data),
		dupString: data,
		hasDup:    true,
	}
}

func HplHttpBodyFromStream(stream io.Reader) *HplHttpBody {
	return &HplHttpBody{
		Stream:    stream,
		dupString: "",
		hasDup:    false,
	}
}

func (h *HplHttpBody) DupString() string {
	if h.hasDup {
		return h.dupString
	}
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, h.Stream)
	h.dupString = buf.String()
	h.hasDup = true
	h.Stream = strings.NewReader(h.dupString)
	return h.dupString
}

func (h *HplHttpBody) ToStream() io.Reader {
	return h.Stream
}

func (h *HplHttpBody) SetStream(stream io.Reader) {
	h.dupString = ""
	h.hasDup = false
	h.Stream = stream
}

func (h *HplHttpBody) SetString(data string) {
	h.dupString = data
	h.hasDup = true
	h.Stream = strings.NewReader(h.dupString)
}

func (h *HplHttpBody) Index(_ interface{}, name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, http body field name must be string")
	}

	switch name.String() {
	case "string":
		return pl.NewValStr(h.DupString()), nil
	case "length":
		if h.hasDup {
			return pl.NewValInt(len(h.dupString)), nil
		} else {
			return pl.NewValNull(), nil
		}
	default:
		return pl.NewValNull(), fmt.Errorf("invalid index, unknown name %s", name.String())
	}
}

func (h *HplHttpBody) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *HplHttpBody) ToString(_ interface{}) (string, error) {
	return h.DupString(), nil
}

func (h *HplHttpBody) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(
		map[string]interface{}{
			"content": h.DupString(),
		},
	)
	return string(blob), nil
}

func (h *HplHttpBody) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.body:%s is unknown", name)
}

func (h *HplHttpBody) info(_ interface{}) string {
	return "http.body"
}

func NewHplHttpBodyValFromStream(stream io.Reader) pl.Val {
	x := HplHttpBodyFromStream(stream)
	return pl.NewValUsrImmutable(
		x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.body"
		},
		x.info,
		nil,
	)
}

func NewHplHttpBodyValFromString(data string) pl.Val {
	x := HplHttpBodyFromString(data)
	return pl.NewValUsrImmutable(
		x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.body"
		},
		x.info,
		nil,
	)
}

// * --------------------------------------------------------------------------
//   Header
// * --------------------------------------------------------------------------
type HplHttpHeader struct {
	header http.Header
}

func (h *HplHttpHeader) Index(_ interface{}, key pl.Val) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, header name must be string")
	}
	return pl.NewValStr(h.header.Get(key.String())), nil
}

func (h *HplHttpHeader) Dot(x interface{}, key string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(key))
}

func (h *HplHttpHeader) ToString(_ interface{}) (string, error) {
	var b bytes.Buffer
	for key, val := range h.header {
		b.WriteString(fmt.Sprintf("%s => %s\n", key, val))
	}
	return b.String(), nil
}

func (h *HplHttpHeader) ToJSON(_ interface{}) (string, error) {
	blob, err := json.Marshal(h.header)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

func (h *HplHttpHeader) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.header:%s is unknown", name)
}

func (h *HplHttpHeader) info(_ interface{}) string {
	return "http.header"
}

func NewHplHttpHeaderVal(header http.Header) pl.Val {
	x := &HplHttpHeader{
		header: header,
	}

	return pl.NewValUsrImmutable(
		x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.header"
		},
		x.info,
		nil,
	)
}

// * --------------------------------------------------------------------------
//   URL
// * --------------------------------------------------------------------------

type HplHttpUrl struct {
	url *url.URL
}

func (h *HplHttpUrl) Index(_ interface{}, key pl.Val) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, URL name must be string")
	}

	switch key.String() {
	case "scheme":
		return pl.NewValStr(h.url.Scheme), nil
	case "host":
		return pl.NewValStr(h.url.Host), nil
	case "path":
		return pl.NewValStr(h.url.Path), nil
	case "query":
		return pl.NewValStr(h.url.RawQuery), nil
	case "url":
		return pl.NewValStr(h.url.String()), nil
	case "userInfo":
		return pl.NewValStr(h.url.User.String()), nil
	default:
		return pl.NewValNull(), fmt.Errorf("unknown component %s in URL", key.String())
	}
}

func (h *HplHttpUrl) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *HplHttpUrl) ToString(_ interface{}) (string, error) {
	return h.url.String(), nil
}

func (h *HplHttpUrl) ToJSON(_ interface{}) (string, error) {
	blob, err := json.Marshal(h.url)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

func (h *HplHttpUrl) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.url:%s is unknown", name)
}

func (h *HplHttpUrl) info(_ interface{}) string {
	return "http.url"
}

func NewHplHttpUrlVal(url *url.URL) pl.Val {
	x := &HplHttpUrl{
		url: url,
	}
	return pl.NewValUsrImmutable(
		&x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.url"
		},
		x.info,
		nil,
	)
}

// * --------------------------------------------------------------------------
//   Request
// * --------------------------------------------------------------------------
type HplHttpRequest struct {
	request *http.Request
	header  pl.Val
	url     pl.Val
	body    pl.Val
}

func (h *HplHttpRequest) isTLS() bool {
	return h.request.TLS != nil
}

func (h *HplHttpRequest) tlsVersion() string {
	if !h.isTLS() {
		return ""
	} else {
		switch h.request.TLS.Version {
		case tls.VersionTLS10:
			return "tls10"
		case tls.VersionTLS11:
			return "tls11"
		case tls.VersionTLS12:
			return "tls12"
		case tls.VersionTLS13:
			return "tls13"
		case tls.VersionSSL30:
			return "ssl3"
		default:
			return "unknown"
		}
	}
}

func (h *HplHttpRequest) Index(_ interface{}, key pl.Val) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, request's component must be string")
	}

	switch key.String() {
	case "header":
		return h.header, nil
	case "method":
		return pl.NewValStr(h.request.Method), nil
	case "proto":
		return pl.NewValStr(h.request.Proto), nil
	case "protoMajor":
		return pl.NewValInt(h.request.ProtoMajor), nil
	case "protoMinor":
		return pl.NewValInt(h.request.ProtoMinor), nil

	case "body":
		return h.body, nil

	// URI related
	case "requestURI":
		return pl.NewValStr(h.request.RequestURI), nil
	case "url":
		return h.url, nil

	// request related special information
	case "host":
		return pl.NewValStr(h.request.Host), nil
	case "remoteAddr":
		return pl.NewValStr(h.request.RemoteAddr), nil

	// tls information
	case "isTLS":
		return pl.NewValBool(h.isTLS()), nil
	case "tlsServerName":
		if h.isTLS() {
			return pl.NewValStr(h.request.TLS.ServerName), nil
		} else {
			return pl.NewValNull(), nil
		}
	case "tlsVersion":
		if h.isTLS() {
			return pl.NewValStr(h.tlsVersion()), nil
		} else {
			return pl.NewValNull(), nil
		}
	case "tlsProtocol":
		if h.isTLS() {
			return pl.NewValStr(h.request.TLS.NegotiatedProtocol), nil
		} else {
			return pl.NewValNull(), nil
		}

	// ContentLength or TransferEncoding
	case "contentLength":
		if h.request.ContentLength >= 0 {
			return pl.NewValInt(int(h.request.ContentLength)), nil
		} else {
			return pl.NewValNull(), nil
		}

	case "transferEncoding":
		ck := false
		for _, xx := range h.request.TransferEncoding {
			if xx == "chunked" {
				ck = true
			}
		}
		return pl.NewValBool(ck), nil
	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("unknown field name %s for request", key.String())
}

func (h *HplHttpRequest) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *HplHttpRequest) ToString(_ interface{}) (string, error) {
	return "http request]", nil
}

func (h *HplHttpRequest) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(
		map[string]interface{}{
			"method":     h.request.Method,
			"host":       h.request.Host,
			"requestURI": h.request.RequestURI,
			"url":        h.request.URL.String(),
			"proto":      h.request.Proto,
			"tls":        h.tlsVersion(),
			"header":     h.request.Header,
			"remoteAddr": h.request.RemoteAddr,
		},
	)
	return string(blob), nil
}

func (h *HplHttpRequest) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.request:%s is unknown", name)
}

func (h *HplHttpRequest) info(_ interface{}) string {
	return "http.request"
}

func NewHplHttpRequestVal(req *http.Request) pl.Val {
	x := &HplHttpRequest{
		request: req,
		header:  NewHplHttpHeaderVal(req.Header),
		url:     NewHplHttpUrlVal(req.URL),
		body:    NewHplHttpBodyValFromStream(req.Body),
	}

	return pl.NewValUsrImmutable(
		x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.request"
		},
		x.info,
		nil,
	)
}

// * --------------------------------------------------------------------------
//   Response
// * --------------------------------------------------------------------------
type HplHttpResponse struct {
	response *http.Response
	header   pl.Val
	request  pl.Val
	body     pl.Val
}

func (h *HplHttpResponse) Index(_ interface{}, name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, name must be string")
	}

	switch name.String() {
	case "statusText":
		return pl.NewValStr(h.response.Status), nil
	case "status":
		return pl.NewValInt(h.response.StatusCode), nil
	case "proto":
		return pl.NewValStr(h.response.Proto), nil

	case "header":
		return h.header, nil

	case "body":
		return h.body, nil

	case "contentLength":
		if h.response.ContentLength >= 0 {
			return pl.NewValInt(int(h.response.ContentLength)), nil
		} else {
			return pl.NewValNull(), nil
		}

	case "transferEncoding":
		ck := false
		for _, xx := range h.response.TransferEncoding {
			if xx == "chunked" {
				ck = true
			}
		}
		return pl.NewValBool(ck), nil

	case "uncompressed":
		return pl.NewValBool(h.response.Uncompressed), nil

	case "request":
		return h.request, nil

	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("invalid index, unknown field: %s", name.String())
}

func (h *HplHttpResponse) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *HplHttpResponse) ToString(_ interface{}) (string, error) {
	return "[http response]", nil
}

func (h *HplHttpResponse) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(
		map[string]interface{}{
			"status": h.response.Status,
			"proto":  h.response.Proto,
			"header": h.response.Header,
		},
	)

	return string(blob), nil
}

func (h *HplHttpResponse) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.response:%s is unknown", name)
}

func (h *HplHttpResponse) info(_ interface{}) string {
	return "http.response"
}

func NewHplHttpResponseVal(response *http.Response) pl.Val {
	var request pl.Val
	if response.Request != nil {
		request = NewHplHttpRequestVal(response.Request)
	} else {
		request = pl.NewValNull()
	}

	x := &HplHttpResponse{
		response: response,
		header:   NewHplHttpHeaderVal(response.Header),
		request:  request,
		body:     NewHplHttpBodyValFromStream(response.Body),
	}

	return pl.NewValUsrImmutable(
		x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.request"
		},
		x.info,
		nil,
	)
}

// * --------------------------------------------------------------------------
//   Router.Params
// * --------------------------------------------------------------------------
type HplHttpRouterParams struct {
	params hrouter.Params
}

func (h *HplHttpRouterParams) Index(_ interface{}, name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, http.router's field must be string")
	}

	return pl.NewValStr(h.params.ByName(name.String())), nil
}

func (h *HplHttpRouterParams) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *HplHttpRouterParams) ToString(_ interface{}) (string, error) {
	return h.params.String(), nil
}

func (h *HplHttpRouterParams) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(h.params)
	return string(blob), nil
}

func (h *HplHttpRouterParams) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.router.params:%s is unknown", name)
}

func (h *HplHttpRouterParams) info(_ interface{}) string {
	return "http.router.params"
}

func NewHplHttpRouterParamsVal(r hrouter.Params) pl.Val {
	x := &HplHttpRouterParams{
		params: r,
	}
	return pl.NewValUsrImmutable(
		x,
		x.Index,
		x.Dot,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.router.params"
		},
		x.info,
		nil,
	)
}
