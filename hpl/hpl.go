package hpl

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	// router
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/service"
	hrouter "github.com/julienschmidt/httprouter"

	// library usage
	"encoding/base64"
	"github.com/google/uuid"
	"strings"
	"time"
)

func must(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("must: %s", msg))
	}
}

func unreachable(msg string) {
	panic(fmt.Sprintf("unreachable: %s", msg))
}

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

func (h *HplHttpBody) Index(_ interface{}, name Val) (Val, error) {
	if name.Type != ValStr {
		return NewValNull(), fmt.Errorf("invalid index, http body field name must be string")
	}

	switch name.String {
	case "string":
		return NewValStr(h.DupString()), nil
	case "length":
		if h.hasDup {
			return NewValInt(len(h.dupString)), nil
		} else {
			return NewValNull(), nil
		}
	default:
		return NewValNull(), fmt.Errorf("invalid index, unknown name %s", name.String)
	}
}

func (h *HplHttpBody) Dot(x interface{}, name string) (Val, error) {
	return h.Index(x, NewValStr(name))
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

func NewHplHttpBodyValFromStream(stream io.Reader) Val {
	x := HplHttpBodyFromStream(stream)
	return NewValUsr(
		x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.body"
		},
	)
}

func NewHplHttpBodyValFromString(data string) Val {
	x := HplHttpBodyFromString(data)
	return NewValUsr(
		x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.body"
		},
	)
}

// * --------------------------------------------------------------------------
//   Header
// * --------------------------------------------------------------------------
type HplHttpHeader struct {
	header http.Header
}

func (h *HplHttpHeader) Index(_ interface{}, key Val) (Val, error) {
	if key.Type != ValStr {
		return NewValNull(), fmt.Errorf("invalid index, header name must be string")
	}
	return NewValStr(h.header.Get(key.String)), nil
}

func (h *HplHttpHeader) Dot(x interface{}, key string) (Val, error) {
	return h.Index(x, NewValStr(key))
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

func NewHplHttpHeaderVal(header http.Header) Val {
	x := &HplHttpHeader{
		header: header,
	}

	return NewValUsr(
		x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.header"
		})
}

// * --------------------------------------------------------------------------
//   URL
// * --------------------------------------------------------------------------

type HplHttpUrl struct {
	url *url.URL
}

func (h *HplHttpUrl) Index(_ interface{}, key Val) (Val, error) {
	if key.Type != ValStr {
		return NewValNull(), fmt.Errorf("invalid index, URL name must be string")
	}

	switch key.String {
	case "scheme":
		return NewValStr(h.url.Scheme), nil
	case "host":
		return NewValStr(h.url.Host), nil
	case "path":
		return NewValStr(h.url.Path), nil
	case "query":
		return NewValStr(h.url.RawQuery), nil
	case "url":
		return NewValStr(h.url.String()), nil
	case "userInfo":
		return NewValStr(h.url.User.String()), nil
	default:
		return NewValNull(), fmt.Errorf("unknown component %s in URL", key.String)
	}
}

func (h *HplHttpUrl) Dot(x interface{}, name string) (Val, error) {
	return h.Index(x, NewValStr(name))
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

func NewHplHttpUrlVal(url *url.URL) Val {
	x := &HplHttpUrl{
		url: url,
	}
	return NewValUsr(
		&x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.url"
		})
}

// * --------------------------------------------------------------------------
//   Request
// * --------------------------------------------------------------------------
type HplHttpRequest struct {
	request *http.Request
	header  Val
	url     Val
	body    Val
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

func (h *HplHttpRequest) Index(_ interface{}, key Val) (Val, error) {
	if key.Type != ValStr {
		return NewValNull(), fmt.Errorf("invalid index, request's component must be string")
	}

	switch key.String {
	case "header":
		return h.header, nil
	case "method":
		return NewValStr(h.request.Method), nil
	case "proto":
		return NewValStr(h.request.Proto), nil
	case "protoMajor":
		return NewValInt(h.request.ProtoMajor), nil
	case "protoMinor":
		return NewValInt(h.request.ProtoMinor), nil

	case "body":
		return h.body, nil

	// URI related
	case "requestURI":
		return NewValStr(h.request.RequestURI), nil
	case "url":
		return h.url, nil

	// request related special information
	case "host":
		return NewValStr(h.request.Host), nil
	case "remoteAddr":
		return NewValStr(h.request.RemoteAddr), nil

	// tls information
	case "isTLS":
		return NewValBool(h.isTLS()), nil
	case "tlsServerName":
		if h.isTLS() {
			return NewValStr(h.request.TLS.ServerName), nil
		} else {
			return NewValNull(), nil
		}
	case "tlsVersion":
		if h.isTLS() {
			return NewValStr(h.tlsVersion()), nil
		} else {
			return NewValNull(), nil
		}
	case "tlsProtocol":
		if h.isTLS() {
			return NewValStr(h.request.TLS.NegotiatedProtocol), nil
		} else {
			return NewValNull(), nil
		}

	// ContentLength or TransferEncoding
	case "contentLength":
		if h.request.ContentLength >= 0 {
			return NewValInt(int(h.request.ContentLength)), nil
		} else {
			return NewValNull(), nil
		}

	case "transferEncoding":
		ck := false
		for _, xx := range h.request.TransferEncoding {
			if xx == "chunked" {
				ck = true
			}
		}
		return NewValBool(ck), nil
	default:
		break
	}

	return NewValNull(), fmt.Errorf("unknown field name %s for request", key.String)
}

func (h *HplHttpRequest) Dot(x interface{}, name string) (Val, error) {
	return h.Index(x, NewValStr(name))
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

func NewHplHttpRequestVal(req *http.Request) Val {
	x := &HplHttpRequest{
		request: req,
		header:  NewHplHttpHeaderVal(req.Header),
		url:     NewHplHttpUrlVal(req.URL),
		body:    NewHplHttpBodyValFromStream(req.Body),
	}

	return NewValUsr(
		x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.request"
		})
}

// * --------------------------------------------------------------------------
//   Response
// * --------------------------------------------------------------------------
type HplHttpResponse struct {
	response *http.Response
	header   Val
	request  Val
	body     Val
}

func (h *HplHttpResponse) Index(_ interface{}, name Val) (Val, error) {
	if name.Type != ValStr {
		return NewValNull(), fmt.Errorf("invalid index, name must be string")
	}

	switch name.String {
	case "statusText":
		return NewValStr(h.response.Status), nil
	case "status":
		return NewValInt(h.response.StatusCode), nil
	case "proto":
		return NewValStr(h.response.Proto), nil

	case "header":
		return h.header, nil

	case "body":
		return h.body, nil

	case "contentLength":
		if h.response.ContentLength >= 0 {
			return NewValInt(int(h.response.ContentLength)), nil
		} else {
			return NewValNull(), nil
		}

	case "transferEncoding":
		ck := false
		for _, xx := range h.response.TransferEncoding {
			if xx == "chunked" {
				ck = true
			}
		}
		return NewValBool(ck), nil

	case "uncompressed":
		return NewValBool(h.response.Uncompressed), nil

	case "request":
		return h.request, nil

	default:
		break
	}

	return NewValNull(), fmt.Errorf("invalid index, unknown field: %s", name.String)
}

func (h *HplHttpResponse) Dot(x interface{}, name string) (Val, error) {
	return h.Index(x, NewValStr(name))
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

func NewHplHttpResponseVal(response *http.Response) Val {
	var request Val
	if response.Request != nil {
		request = NewHplHttpRequestVal(response.Request)
	} else {
		request = NewValNull()
	}

	x := &HplHttpResponse{
		response: response,
		header:   NewHplHttpHeaderVal(response.Header),
		request:  request,
		body:     NewHplHttpBodyValFromStream(response.Body),
	}

	return NewValUsr(
		x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.request"
		})
}

// * --------------------------------------------------------------------------
//   Router.Params
// * --------------------------------------------------------------------------
type HplHttpRouterParams struct {
	params hrouter.Params
}

func (h *HplHttpRouterParams) Index(_ interface{}, name Val) (Val, error) {
	if name.Type != ValStr {
		return NewValNull(), fmt.Errorf("invalid index, http.router's field must be string")
	}

	return NewValStr(h.params.ByName(name.String)), nil
}

func (h *HplHttpRouterParams) Dot(x interface{}, name string) (Val, error) {
	return h.Index(x, NewValStr(name))
}

func (h *HplHttpRouterParams) ToString(_ interface{}) (string, error) {
	var b bytes.Buffer
	for idx, kv := range h.params {
		b.WriteString(fmt.Sprintf("%d. %s => %s\n", idx, kv.Key, kv.Value))
	}
	return b.String(), nil
}

func (h *HplHttpRouterParams) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(h.params)
	return string(blob), nil
}

func NewHplHttpRouterParamsVal(r hrouter.Params) Val {
	x := &HplHttpRouterParams{
		params: r,
	}
	return NewValUsr(
		x,
		x.Index,
		x.Dot,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return "http.router.params"
		})
}

// ----------------------------------------------------------------------------
// builtin functions/libraries for our internal usage

// {{# Codec Library Functions
func fnBase64Encode(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: b64_encode expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: b64_encode's frist argument must be string")
	}

	out := base64.StdEncoding.EncodeToString([]byte(argument[0].String))
	return NewValStr(out), nil
}

func fnBase64Decode(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: b64_decode expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: b64_decode's frist argument must be string")
	}

	out, err := base64.StdEncoding.DecodeString(argument[0].String)
	if err != nil {
		return NewValNull(), err
	} else {
		return NewValStr(string(out)), nil
	}
}

func fnURLEncode(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: url_encode expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: url_encode's frist argument must be string")
	}

	out := url.QueryEscape(argument[0].String)
	return NewValStr(out), nil
}

func fnURLDecode(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: url_decode expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: url_decode's frist argument must be string")
	}

	out, err := url.QueryUnescape(argument[0].String)
	if err != nil {
		return NewValNull(), err
	} else {
		return NewValStr(out), nil
	}
}

func fnPathEscape(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: path_encode expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: path_encode's frist argument must be string")
	}

	out := url.PathEscape(argument[0].String)
	return NewValStr(out), nil
}

func fnPathUnescape(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: path_decode expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: path_decode's frist argument must be string")
	}

	out, err := url.PathUnescape(argument[0].String)
	if err != nil {
		return NewValNull(), err
	} else {
		return NewValStr(out), nil
	}
}

// #}}

// {{# General Functions

func fnToString(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: to_string expect 1 argument")
	}
	x, err := argument[0].ToString()
	if err != nil {
		return NewValNull(), err
	} else {
		return NewValStr(x), nil
	}
}

func fnHttpDate(argument []Val) (Val, error) {
	if len(argument) != 0 {
		return NewValNull(), fmt.Errorf("function: http_date expect 1 argument")
	}

	now := time.Now()
	return NewValStr(now.Format(time.RFC3339)), nil
}

func fnHttpDateNano(argument []Val) (Val, error) {
	if len(argument) != 0 {
		return NewValNull(), fmt.Errorf("function: http_datenano expect 1 argument")
	}

	now := time.Now()
	return NewValStr(now.Format(time.RFC3339Nano)), nil
}

func fnUUID(argument []Val) (Val, error) {
	if len(argument) != 0 {
		return NewValNull(), fmt.Errorf("function: uuid expect 1 argument")
	}
	return NewValStr(uuid.NewString()), nil
}

func fnEcho(argument []Val) (Val, error) {
	// printing the input out, regardless what it is and the echo should print it
	// out always
	var b []interface{}

	for _, x := range argument {
		b = append(b, x.ToNative())
	}

	x, _ := json.MarshalIndent(b, "", "  ")
	log.Println(string(x))
	return NewValNull(), nil
}

// concate all input to a single stream, which can be represented as http body
func fnConcateHttpBody(argument []Val) (Val, error) {
	var input []io.Reader
	for idx, a := range argument {
		if a.Id() == "http.body" {
			body, _ := a.Usr.Context.(*HplHttpBody)
			input = append(input, body.ToStream())
		} else {
			str, err := a.ToString()
			if err != nil {
				return NewValNull(),
					fmt.Errorf("concate_http_body: the %d argument cannot be converted to string: %s",
						idx+1, err.Error())
			}
			input = append(input, strings.NewReader(str))
		}
	}

	return NewHplHttpBodyValFromStream(io.MultiReader(input...)), nil
}

// #}}

// {{# String Manipulation Functions

func fnToUpper(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: to_upper expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: to_upper's frist argument must be string")
	}
	return NewValStr(strings.ToUpper(argument[0].String)), nil
}

func fnToLower(argument []Val) (Val, error) {
	if len(argument) != 1 {
		return NewValNull(), fmt.Errorf("function: to_lower expect 1 argument")
	}
	if argument[0].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: to_lower's frist argument must be string")
	}
	return NewValStr(strings.ToLower(argument[0].String)), nil
}

func fnSplit(argument []Val) (Val, error) {
	if len(argument) != 2 {
		return NewValNull(), fmt.Errorf("function: split expect 2 arguments")
	}
	if argument[0].Type != ValStr && argument[1].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: split expects 2 string arguments")
	}
	x := strings.Split(argument[0].String, argument[1].String)
	ret := NewValList()
	for _, xx := range x {
		ret.AddList(NewValStr(xx))
	}
	return ret, nil
}

func fnTrim(argument []Val) (Val, error) {
	if len(argument) != 2 {
		return NewValNull(), fmt.Errorf("function: trim expect 2 arguments")
	}
	if argument[0].Type != ValStr && argument[1].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: trim expects 2 string arguments")
	}
	return NewValStr(strings.Trim(argument[0].String, argument[1].String)), nil
}

func fnLTrim(argument []Val) (Val, error) {
	if len(argument) != 2 {
		return NewValNull(), fmt.Errorf("function: ltrim expect 2 arguments")
	}
	if argument[0].Type != ValStr && argument[1].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: ltrim expects 2 string arguments")
	}
	return NewValStr(strings.TrimLeft(argument[0].String, argument[1].String)), nil
}

func fnRTrim(argument []Val) (Val, error) {
	if len(argument) != 2 {
		return NewValNull(), fmt.Errorf("function: rtrim expect 2 arguments")
	}
	if argument[0].Type != ValStr && argument[1].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: rtrim expects 2 string arguments")
	}
	return NewValStr(strings.TrimRight(argument[0].String, argument[1].String)), nil
}

// #}}

// {{# HTTP/Network Functions

// general HTTP request interfaces, allowing user to specify following method
// http(
//   [0]: url(string)
//   [1]: method(string)
//   [2]: [header](list[string]),
//   [3]: [body](string)
// )
func (h *Hpl) fnHttp(argument []Val) (Val, error) {
	if len(argument) < 2 {
		return NewValNull(), fmt.Errorf("function: http expect at least 2 arguments")
	}
	if h.session == nil {
		return NewValNull(), fmt.Errorf("function: http cannot be executed, session not bound")
	}

	sres := h.session.SessionResource()
	if sres == nil {
		return NewValNull(), fmt.Errorf("function: http cannot be executed, session resource empty")
	}

	url := argument[0]
	method := argument[1]
	if url.Type != ValStr || method.Type != ValStr {
		return NewValNull(), fmt.Errorf("function: http's first 2 arguments must be string")
	}

	var body io.Reader
	if len(argument) == 4 && argument[3].Type == ValStr {
		body = strings.NewReader(argument[3].String)
	} else {
		body = http.NoBody
	}

	req, err := http.NewRequest(method.String, url.String, body)
	if err != nil {
		return NewValNull(), err
	}

	// check header field
	if len(argument) >= 3 && argument[2].Type == ValList {
		for _, hdr := range argument[2].List.Data {
			if hdr.Type == ValPair && hdr.Pair.First.Type == ValStr && hdr.Pair.Second.Type == ValStr {
				req.Header.Add(hdr.Pair.First.String, hdr.Pair.Second.String)
			}
		}
	}

	client, err := sres.GetHttpClient(url.String)
	if err != nil {
		return NewValNull(), err
	}

	resp, err := client.Do(req)
	if err != nil {
		return NewValNull(), err
	}

	// serialize the response back to the normal object
	return NewHplHttpResponseVal(resp), nil
}

// header related functions
func fnHeaderHas(argument []Val) (Val, error) {
	if len(argument) != 2 {
		return NewValNull(), fmt.Errorf("function: header_has requires 2 arguments")
	}

	if argument[0].Id() != "http.header" || argument[1].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: header_has's " +
			"first argument must be header and second " +
			"argument must be string")
	}

	hdr, ok := argument[0].Usr.Context.(*HplHttpHeader)
	must(ok, "must be http.header")

	if vv := hdr.header.Get(argument[1].String); vv == "" {
		return NewValBool(false), nil
	} else {
		return NewValBool(true), nil
	}
}

func doHeaderDelete(hdr http.Header, key string) int {
	prefixWildcard := strings.HasPrefix(key, "*")
	suffixWildcard := strings.HasSuffix(key, "*")
	cnt := 0

	if prefixWildcard && suffixWildcard {
		midfix := key[1 : len(key)-1]
		for key, _ := range hdr {
			if strings.Contains(key, midfix) {
				hdr.Del(key)
				cnt++
			}
		}
	} else if prefixWildcard {
		target := key[1:]
		for key, _ := range hdr {
			if strings.HasSuffix(key, target) {
				hdr.Del(key)
				cnt++
			}
		}
	} else if suffixWildcard {
		target := key[:len(key)-1]
		for key, _ := range hdr {
			if strings.HasPrefix(key, target) {
				hdr.Del(key)
				cnt++
			}
		}
	} else {
		hdr.Del(key)
		cnt++
	}

	return cnt
}

func fnHeaderDelete(argument []Val) (Val, error) {
	if len(argument) != 2 {
		return NewValNull(), fmt.Errorf("function: header_delete requires 2 arguments")
	}

	if argument[0].Id() != "http.header" || argument[1].Type != ValStr {
		return NewValNull(), fmt.Errorf("function: header_delete's " +
			"first argument must be header and second " +
			"argument must be string")
	}

	hdr, ok := argument[0].Usr.Context.(*HplHttpHeader)
	must(ok, "must be http.header")

	cnt := doHeaderDelete(hdr.header, argument[1].String)
	return NewValInt(cnt), nil
}

// #}}

// ----------------------------------------------------------------------------
// A net/http adapter to help user to evaluate certain things under the http
// context

type hplResponseWriter struct {
	bodyIsStream bool
	bodyStream   io.Reader
	bodyString   string
	statusCode   int

	writer http.ResponseWriter
}

func (h *hplResponseWriter) writeBodyString(data string) {
	h.bodyIsStream = false
	h.bodyString = data
}

func (h *hplResponseWriter) writeBodyStream(data io.Reader) {
	h.bodyIsStream = true
	h.bodyStream = data
}

func (h *hplResponseWriter) toBodyStream() io.Reader {
	if !h.bodyIsStream {
		h.bodyIsStream = true
		h.bodyStream = strings.NewReader(h.bodyString)
	}
	return h.bodyStream
}

func (h *hplResponseWriter) finish() error {
	h.writer.WriteHeader(h.statusCode)
	_, _ = io.Copy(h.writer, h.toBodyStream())
	return nil
}

func newHplResponseWriter(w http.ResponseWriter) *hplResponseWriter {
	return &hplResponseWriter{
		bodyIsStream: false,
		bodyStream:   nil,
		bodyString:   "",
		statusCode:   200,
		writer:       w,
	}
}

type Hpl struct {
	Eval   *Evaluator
	Policy *Policy

	UserLoadFn EvalLoadVar
	UserCall   EvalCall
	UserAction EvalAction

	// internal status during evaluation context
	request    Val
	respWriter *hplResponseWriter
	params     Val
	session    service.Session

	log       *alog.SessionLog
	isRunning bool
}

func foreachHeaderKV(arg Val, fn func(key string, val string)) bool {
	if arg.Type == ValList {
		for _, v := range arg.List.Data {
			if v.Type == ValPair && v.Pair.First.Type == ValStr && v.Pair.Second.Type == ValStr {
				fn(v.Pair.First.String, v.Pair.Second.String)
			}
		}
	} else if arg.Type == ValPair {
		if arg.Pair.First.Type == ValStr && arg.Pair.Second.Type == ValStr {
			fn(arg.Pair.First.String, arg.Pair.Second.String)
		}
	} else if arg.Id() == "http.header" {
		// known special user type to us, then just foreach the header
		hdr := arg.Usr.Context.(*HplHttpHeader)
		for k, v := range hdr.header {
			for _, vv := range v {
				fn(k, vv)
			}
		}
	} else {
		return false
	}

	return true
}

func foreachStr(arg Val, fn func(key string)) bool {
	if arg.Type == ValList {
		for _, v := range arg.List.Data {
			if v.Type == ValStr {
				fn(v.String)
			}
		}
	} else if arg.Type == ValStr {
		fn(arg.String)
	} else {
		return false
	}
	return true
}

func (p *Hpl) httpEvalAction(x *Evaluator, actionName string, arg Val) error {
	// http response generation action, ie builting actions
	switch actionName {
	case "status":
		if arg.Type != ValInt {
			return fmt.Errorf("status action must have a int argument")
		} else {
			p.respWriter.statusCode = int(arg.Int)
		}
		return nil

	// header operations
	case "header_add":
		if !foreachHeaderKV(
			arg,
			func(key string, val string) {
				p.respWriter.writer.Header().Add(key, val)
			}) {
			return fmt.Errorf("invalid header_add value type")
		}
		return nil

	case "header_set":
		if !foreachHeaderKV(
			arg,
			func(key string, val string) {
				p.respWriter.writer.Header().Set(key, val)
			}) {
			return fmt.Errorf("invalid header_set value type")
		}
		return nil

	case "header_try_set":
		if !foreachHeaderKV(
			arg,
			func(key string, val string) {
				hdr := p.respWriter.writer.Header()
				if hdr.Get(key) == "" {
					hdr.Set(key, val)
				}
			}) {
			return fmt.Errorf("invalid header_try_set value type")
		}
		return nil

	// Delete header logic, support wildcard matching during header deletion
	// 4 patterns are recognized
	// 1) *-suffix
	// 2) prefix-*
	// 3) *-midfix-*
	// 4) literal key

	case "header_del":
		if !foreachStr(
			arg,
			func(key string) {
				hdr := p.respWriter.writer.Header()
				doHeaderDelete(hdr, key)
			}) {
			return fmt.Errorf("invalid header_add value type")
		}
		return nil

	// body operations
	case "body":
		if arg.Type == ValStr {
			if p.respWriter == nil {
				panic("WFT???")
			}
			p.respWriter.writeBodyString(arg.String)
		} else if arg.Id() == "http.body" {
			body, ok := arg.Usr.Context.(*HplHttpBody)
			must(ok, "invalid body type")
			p.respWriter.writeBodyStream(body.ToStream())
		} else {
			return fmt.Errorf("invalid body value type")
		}
		return nil

	default:
		break
	}

	if p.UserAction != nil {
		return p.UserAction(x, actionName, arg)
	} else {
		return fmt.Errorf("unknown action %s", actionName)
	}
}

// eval related functions
func (p *Hpl) httpLoadVar(x *Evaluator, n string) (Val, error) {
	switch n {
	case "request":
		return p.request, nil
	case "params":
		return p.params, nil
	case "serviceName":
		return NewValStr(p.session.Service().Name()), nil
	case "serviceIdl":
		return NewValStr(p.session.Service().IDL()), nil
	case "servicePolicy":
		return NewValStr(p.session.Service().Policy()), nil
	case "serviceRouter":
		return NewValStr(p.session.Service().Router()), nil
	case "serviceMethodList":
		r := NewValList()
		for _, v := range p.session.Service().MethodList() {
			r.AddList(NewValStr(v))
		}
		return r, nil
	default:
		if p.UserLoadFn != nil {
			return p.UserLoadFn(x, n)
		} else {
			return NewValNull(), fmt.Errorf("unknown variable %s", n)
		}
	}
}

func (p *Hpl) httpEvalCall(x *Evaluator, n string, args []Val) (Val, error) {
	switch n {
	case "b64_encode":
		return fnBase64Encode(args)
	case "b64_decode":
		return fnBase64Decode(args)
	case "url_encode":
		return fnURLEncode(args)
	case "url_decode":
		return fnURLDecode(args)
	case "path_escape":
		return fnPathEscape(args)
	case "path_unescape":
		return fnPathUnescape(args)
	case "to_string":
		return fnToString(args)
	case "http_date":
		return fnHttpDate(args)
	case "http_datenano":
		return fnHttpDateNano(args)
	case "uuid":
		return fnUUID(args)
	case "concate_http_body":
		return fnConcateHttpBody(args)
	case "echo":
		return fnEcho(args)
	case "to_upper":
		return fnToUpper(args)
	case "to_lower":
		return fnToLower(args)
	case "split":
		return fnSplit(args)
	case "trim":
		return fnTrim(args)
	case "ltrim":
		return fnLTrim(args)
	case "rtrim":
		return fnRTrim(args)
	// networking
	case "http":
		return p.fnHttp(args)
	case "header_has":
		return fnHeaderHas(args)
	case "header_delete":
		return fnHeaderDelete(args)

	default:
		break
	}

	if p.UserCall != nil {
		return p.UserCall(x, n, args)
	} else {
		return NewValNull(), fmt.Errorf("function %s is unknown", n)
	}
}

func NewHpl(f0 EvalLoadVar, f1 EvalCall, f2 EvalAction) *Hpl {
	p := &Hpl{
		UserLoadFn: f0,
		UserCall:   f1,
		UserAction: f2,
	}

	p.Eval = NewEvaluatorSimple()
	return p
}

func NewHplWithPolicy(f0 EvalLoadVar, f1 EvalCall, f2 EvalAction, policy *Policy) *Hpl {
	p := &Hpl{
		UserLoadFn: f0,
		UserCall:   f1,
		UserAction: f2,
	}

	p.Eval = NewEvaluatorSimple()
	p.SetPolicy(policy)
	return p
}

func (h *Hpl) CompilePolicy(input string) error {
	p, err := CompilePolicy(input)
	if err != nil {
		return err
	}
	h.Policy = p
	return nil
}

func (h *Hpl) SetPolicy(p *Policy) {
	h.Policy = p
}

// -----------------------------------------------------------------------------
// prepare phase
func (h *Hpl) prepareLoadVar(x *Evaluator, n string) (Val, error) {
	switch n {
	case "serviceName":
		return NewValStr(h.session.Service().Name()), nil
	case "serviceIdl":
		return NewValStr(h.session.Service().IDL()), nil
	case "servicePolicy":
		return NewValStr(h.session.Service().Policy()), nil
	case "serviceRouter":
		return NewValStr(h.session.Service().Router()), nil
	case "serviceMethodList":
		r := NewValList()
		for _, v := range h.session.Service().MethodList() {
			r.AddList(NewValStr(v))
		}
		return r, nil
	default:
		break
	}
	if h.UserLoadFn != nil {
		return h.UserLoadFn(x, n)
	} else {
		return NewValNull(), fmt.Errorf("unknown variable %s", n)
	}
}

func (h *Hpl) prepareEvalCall(x *Evaluator, n string, args []Val) (Val, error) {
	return h.httpEvalCall(x, n, args)
}

func (h *Hpl) prepareActionCall(x *Evaluator, actionName string, arg Val) error {
	if h.UserAction != nil {
		return h.UserAction(x, actionName, arg)
	} else {
		return fmt.Errorf("unknown action %s", actionName)
	}
}

func (h *Hpl) OnGlobal(session service.Session) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}
	if h.respWriter != nil {
		panic("CONCURRENT PROBLEM")
	}

	h.isRunning = true
	h.session = session

	h.Eval.LoadVarFn = h.prepareLoadVar
	h.Eval.CallFn = h.prepareEvalCall
	h.Eval.ActionFn = h.prepareActionCall

	defer func() {
		h.isRunning = false
		h.respWriter = nil
		h.session = nil
	}()

	return h.Eval.EvalGlobal(h.Policy)
}

// -----------------------------------------------------------------------------
// session log phase
func (h *Hpl) logLoadVar(x *Evaluator, n string) (Val, error) {
	switch n {
	case "logFormat":
		return NewValStr(h.log.Format.Raw), nil
	default:
		break
	}
	return h.httpLoadVar(x, n)
}

func (h *Hpl) logEvalCall(x *Evaluator, n string, args []Val) (Val, error) {
	return h.httpEvalCall(x, n, args)
}

func (h *Hpl) logActionCall(x *Evaluator, actionName string, arg Val) error {
	switch actionName {
	case "format":
		if arg.Type != ValStr {
			return fmt.Errorf("status format must have a string argument")
		} else {
			fmt, err := alog.NewSessionLogFormat(arg.String)
			if err != nil {
				return err
			}
			h.log.Format = fmt
			return nil
		}

	case "appendix":
		switch arg.Type {
		case ValStr, ValReal, ValBool, ValNull, ValInt, ValRegexp:
			str, _ := arg.ToString()
			h.log.Appendix = append(h.log.Appendix, str)
			break

		case ValList:
			for _, e := range arg.List.Data {
				if e.Type == ValStr {
					h.log.Appendix = append(h.log.Appendix, e.String)
				}
			}
			break

		default:
			return fmt.Errorf("status appendix must have a string or string list argument")
		}

		return nil

	default:
		if h.UserAction != nil {
			return h.UserAction(x, actionName, arg)
		} else {
			return fmt.Errorf("unknown action %s", actionName)
		}
	}
}

func (h *Hpl) OnLog(selector string, log *alog.SessionLog, session service.Session) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}
	if h.respWriter != nil {
		panic("concurrent access")
	}

	h.isRunning = true

	h.request = NewHplHttpRequestVal(log.HttpRequest)
	h.params = NewHplHttpRouterParamsVal(log.RouterParams)
	h.session = session
	h.log = log

	h.Eval.LoadVarFn = h.logLoadVar
	h.Eval.CallFn = h.logEvalCall
	h.Eval.ActionFn = h.logActionCall

	defer func() {
		h.isRunning = false
		h.respWriter = nil
		h.session = nil
	}()

	return h.Eval.Eval(selector, h.Policy)
}

// the following tights into the pipeline of the web server

type HttpContext struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	QueryParams    hrouter.Params
}

// HTTP response generation phase ----------------------------------------------
func (h *Hpl) OnHttpResponse(selector string, context HttpContext, session service.Session) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}
	if h.respWriter != nil {
		panic("concurrent access")
	}

	h.isRunning = true

	h.request = NewHplHttpRequestVal(context.Request)
	h.respWriter = newHplResponseWriter(context.ResponseWriter)
	h.params = NewHplHttpRouterParamsVal(context.QueryParams)
	h.session = session

	h.Eval.LoadVarFn = h.httpLoadVar
	h.Eval.CallFn = h.httpEvalCall
	h.Eval.ActionFn = h.httpEvalAction

	defer func() {
		h.isRunning = false
		h.respWriter = nil
		h.session = nil
	}()

	err := h.Eval.Eval(selector, h.Policy)
	if err != nil {
		return err
	}
	return h.respWriter.finish()
}
