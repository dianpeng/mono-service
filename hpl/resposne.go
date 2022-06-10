package hpl

import (
	"encoding/json"
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
)

type Response struct {
	response *http.Response
	header   pl.Val
	request  pl.Val
	body     pl.Val
}

func (h *Response) isChunked() bool {
	for _, xx := range h.response.TransferEncoding {
		if xx == "chunked" {
			return true
		}
	}
	return false
}

func (r *Response) setHeader(v pl.Val) error {
	if ValIsHttpHeader(v) {
		r.header = v
		return nil
	}
	return fmt.Errorf("http.response.header set, invalid type")
}

func (r *Response) setBody(v pl.Val) error {
	if ValIsHttpBody(v) {
		r.body = v
		return nil
	}

	if v.Type == pl.ValStr {
		r.body = NewBodyValFromString(v.String())
		return nil
	}

	return fmt.Errorf("http.response.body set, invalid type")
}

func (h *Response) Index(_ interface{}, name pl.Val) (pl.Val, error) {
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
		return pl.NewValBool(h.isChunked()), nil

	case "uncompressed":
		return pl.NewValBool(h.response.Uncompressed), nil

	case "request":
		return h.request, nil

	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("invalid index, unknown field: %s", name.String())
}

func (h *Response) IndexSet(x interface{}, key pl.Val, value pl.Val) error {
	if key.Type != pl.ValStr {
		return fmt.Errorf("http.response index set, invalid key type")
	}
	return h.DotSet(x, key.String(), value)
}

func (h *Response) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *Response) DotSet(_ interface{}, key string, value pl.Val) error {
	switch key {
	case "header":
		return h.setHeader(value)

	case "url":
		return h.setBody(value)

	default:
		return fmt.Errorf("http.response set, unknown field: %s", key)
	}
}

func (h *Response) ToString(_ interface{}) (string, error) {
	return "[http response]", nil
}

func (h *Response) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(
		map[string]interface{}{
			"status": h.response.Status,
			"proto":  h.response.Proto,
			"header": h.response.Header,
		},
	)

	return string(blob), nil
}

func (h *Response) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("http.response method %s is unknown", name)
}

func (h *Response) info(_ interface{}) string {
	return "http.response"
}

func NewResponseVal(response *http.Response) pl.Val {
	var request pl.Val
	if response.Request != nil {
		request = NewRequestVal(response.Request)
	} else {
		request = pl.NewValNull()
	}

	x := &Response{
		response: response,
		header:   NewHeaderVal(response.Header),
		request:  request,
		body:     NewBodyValFromStream(response.Body),
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
