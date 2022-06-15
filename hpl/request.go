package hpl

import (
	"crypto/tls"
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Request struct {
	request *http.Request
	header  pl.Val
	url     pl.Val
	body    pl.Val
}

func ValIsHttpRequest(v pl.Val) bool {
	return v.Id() == HttpRequestTypeId
}

func (r *Request) HttpRequest() *http.Request {
	return r.request
}

func (r *Request) setUrl(v pl.Val) error {
	switch v.Type {
	case pl.ValStr:
		u, err := url.Parse(v.String())
		if err != nil {
			return err
		}
		r.url = NewUrlVal(u)
		return nil

	default:
		if ValIsUrl(v) {
			r.url = v
			return nil
		}
		break
	}

	return fmt.Errorf("http.request.url set, invalid type")
}

func (r *Request) setHeader(v pl.Val) error {
	if ValIsHttpHeader(v) {
		r.header = v
		return nil
	}
	return fmt.Errorf("http.request.header set, invalid type")
}

func (r *Request) setBody(v pl.Val) error {
	if ValIsHttpBody(v) {
		r.body = v
		return nil
	}

	if v.Type == pl.ValStr {
		r.body = NewBodyValFromString(v.String())
		return nil
	}

	return fmt.Errorf("http.request.body set, invalid type")
}

func (h *Request) isTLS() bool {
	return h.request.TLS != nil
}

func (h *Request) tlsVersion() string {
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

func (h *Request) isChunked() bool {
	for _, xx := range h.request.TransferEncoding {
		if xx == "chunked" {
			return true
		}
	}
	return false
}

func (h *Request) Index(key pl.Val) (pl.Val, error) {
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
		return pl.NewValBool(h.isChunked()), nil
	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("unknown field name %s for request", key.String())
}

func (h *Request) Dot(name string) (pl.Val, error) {
	return h.Index(pl.NewValStr(name))
}

func (h *Request) IndexSet(key pl.Val, val pl.Val) error {
	if key.Type == pl.ValStr {
		return h.DotSet(key.String(), val)
	} else {
		return fmt.Errorf("http.request index set, invalid key type")
	}
}

func (h *Request) DotSet(key string, val pl.Val) error {
	switch key {
	case "header":
		return h.setHeader(val)

	case "url":
		return h.setUrl(val)

	case "body":
		return h.setBody(val)

	default:
		return fmt.Errorf("http.request set, unknown field: %s", key)
	}

	return nil
}

func (h *Request) ToString() (string, error) {
	return h.Info(), nil
}

func (h *Request) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"type":       HttpRequestTypeId,
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
}

func (h *Request) Method(name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.request:%s is unknown", name)
}

func (h *Request) Info() string {
	return HttpRequestTypeId
}

func (h *Request) ToNative() interface{} {
	return h.request
}

func (h *Request) Id() string {
	return HttpRequestTypeId
}

func (h *Request) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("http.request does not support iterator")
}

func NewRequestVal(req *http.Request) pl.Val {
	x := &Request{
		request: req,
		header:  NewHeaderVal(req.Header),
		url:     NewUrlVal(req.URL),
		body:    NewBodyValFromStream(req.Body),
	}

	return pl.NewValUsr(x)
}

func NewRequestValFromVal(
	method string,
	url string,
	body pl.Val,
) (pl.Val, error) {

	switch body.Type {
	case pl.ValStr:
		return NewRequestValFromString(method, url, body.String())
	default:
		if ValIsHttpBody(body) {
			b, _ := body.Usr().(*Body)
			return NewRequestValFromStream(method, url, b.Stream().Stream)
		}
		if ValIsReadableStream(body) {
			s, _ := body.Usr().(*ReadableStream)
			return NewRequestValFromStream(method, url, s.Stream)
		}

		return pl.NewValNull(), fmt.Errorf("unknown body type: %s", body.Id())
	}
}

func NewRequestValFromStream(
	method string,
	url string,
	body io.Reader) (pl.Val, error) {

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return pl.NewValNull(), err
	}

	return NewRequestVal(req), nil
}

func NewRequestValFromString(
	method string,
	url string,
	body string) (pl.Val, error) {

	return NewRequestValFromStream(method, url, strings.NewReader(body))
}

func NewRequestValFromBuffer(
	method string,
	url string,
	body []byte) (pl.Val, error) {

	return NewRequestValFromStream(method, url, strings.NewReader(string(body)))
}
