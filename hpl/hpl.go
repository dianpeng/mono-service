package hpl

import (
	"fmt"
	"io"
	"net/http"

	// router
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
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

// Interface used by VHost to bridge SessionWrapper/Service object into the HPL
// world. HPL knows nothing about the service/session and also hservice. HPL is
// just a wrapper around PL but with all the http/networking utilities
type SessionWrapper interface {
	// HPL/PL scriptting callback
	OnLoadVar(*pl.Evaluator, string) (pl.Val, error)
	OnStoreVar(*pl.Evaluator, string, pl.Val) error
	OnCall(*pl.Evaluator, string, []pl.Val) (pl.Val, error)
	OnAction(*pl.Evaluator, string, pl.Val) error

	// other utilities
	GetPhaseName() string
	GetErrorDescription() string

	// special function used for exposing other utilities
	GetHttpClient(url string) (HttpClient, error)
}

// ----------------------------------------------------------------------------
// A net/http adapter to help user to evaluate certain things under the http
// context

type hplResponseWriter struct {
	bodyIsStream bool
	bodyStream   io.ReadCloser
	bodyString   string
	statusCode   int
	header       http.Header

	// do not use it
	_writer http.ResponseWriter
}

func (h *hplResponseWriter) clearHeader() {
	h.header = make(http.Header)
}

func (h *hplResponseWriter) writeBodyString(data string) {
	h.bodyIsStream = false
	h.bodyString = data
}

func (h *hplResponseWriter) writeBodyStream(data io.ReadCloser) {
	h.bodyIsStream = true
	h.bodyStream = data
}

func (h *hplResponseWriter) toBodyStream() io.ReadCloser {
	if !h.bodyIsStream {
		h.bodyIsStream = true
		h.bodyStream = neweofByteReadCloserFromString(h.bodyString)
	}
	return h.bodyStream
}

func (h *hplResponseWriter) setResponse(resp *http.Response) {
	h.statusCode = resp.StatusCode
	h.header = resp.Header
	h.writeBodyStream(resp.Body)
}

func (h *hplResponseWriter) finish() error {
	defer func() {
		if h.bodyIsStream {
			h.bodyStream.Close()
		}
	}()
	for k, v := range h.header {
		h._writer.Header()[k] = v
	}
	h._writer.WriteHeader(h.statusCode)
	_, _ = io.Copy(h._writer, h.toBodyStream())
	return nil
}

func newHplResponseWriter(w http.ResponseWriter) *hplResponseWriter {
	return &hplResponseWriter{
		bodyIsStream: false,
		bodyStream:   nil,
		bodyString:   "",
		statusCode:   200,
		header:       make(http.Header),
		_writer:      w,
	}
}

type Hpl struct {
	Eval   *pl.Evaluator
	Policy *pl.Policy

	// internal status during evaluation context
	request    pl.Val
	respWriter *hplResponseWriter
	params     pl.Val
	session    SessionWrapper

	log       *alog.SessionLog
	isRunning bool
}

func NewHpl() *Hpl {
	p := &Hpl{}
	p.Eval = pl.NewEvaluatorSimple()
	return p
}

func NewHplWithPolicy(policy *pl.Policy) *Hpl {
	p := &Hpl{}
	p.Eval = pl.NewEvaluatorSimple()
	p.SetPolicy(policy)
	return p
}

func (h *Hpl) CompilePolicy(input string) error {
	p, err := pl.CompilePolicy(input)
	if err != nil {
		return err
	}
	h.Policy = p
	return nil
}

func (h *Hpl) SetPolicy(p *pl.Policy) {
	h.Policy = p
}

func (p *Hpl) loadVarBasic(x *pl.Evaluator, n string) (pl.Val, error) {
	switch n {
	case "request":
		return p.request, nil
	case "params":
		return p.params, nil
	default:
		return p.session.OnLoadVar(x, n)
	}
}

func (h *Hpl) fnHttp(args []pl.Val) (pl.Val, error) {
	if h.session == nil {
    return pl.NewValNull(), fmt.Errorf("http::do cannot be executed, session not bound")
	}
	return fnDoHttp(h.session, args)
}

func (p *Hpl) evalCallBasic(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	switch n {
  case "http::do":
		return p.fnHttp(args)
  case "http::concate_body":
		return fnConcateHttpBody(args)
  case "http::new_url":
    return fnNewUrl(args)
	default:
		return p.session.OnCall(x, n, args)
	}
}

// -----------------------------------------------------------------------------
// init phase
func (h *Hpl) initLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return h.session.OnLoadVar(x, n)
}

func (p *Hpl) initStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.session.OnStoreVar(x, n, v)
}

func (h *Hpl) initEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.session.OnCall(x, n, args)
}

func (h *Hpl) initAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return h.session.OnAction(x, actionName, arg)
}

func (h *Hpl) OnInit(session SessionWrapper) error {
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
	h.session = session

	h.Eval.LoadVarFn = h.initLoadVar
	h.Eval.StoreVarFn = h.initStoreVar
	h.Eval.CallFn = h.initEvalCall
	h.Eval.ActionFn = h.initAction

	defer func() {
		h.isRunning = false
		h.respWriter = nil
		h.session = nil
	}()

	return h.Eval.EvalSession(h.Policy)
}

// -----------------------------------------------------------------------------
// access phase
func (h *Hpl) accessLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return h.loadVarBasic(x, n)
}

func (h *Hpl) accessStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return h.session.OnStoreVar(x, n, v)
}

func (h *Hpl) accessEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.session.OnCall(x, n, args)
}

func (h *Hpl) accessAction(x *pl.Evaluator, n string, arg pl.Val) error {
	return h.session.OnAction(x, n, arg)
}

func (h *Hpl) OnAccess(selector string, req *http.Request, param hrouter.Params, session SessionWrapper) error {
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
	h.request = NewRequestVal(req)
	h.params = NewRouterParamsVal(param)
	h.session = session

	h.Eval.LoadVarFn = h.accessLoadVar
	h.Eval.StoreVarFn = h.accessStoreVar
	h.Eval.CallFn = h.accessEvalCall
	h.Eval.ActionFn = h.accessAction

	defer func() {
		h.isRunning = false
		h.session = nil
	}()

	return h.Eval.Eval(selector, h.Policy)
}

// -----------------------------------------------------------------------------
// request phase
func (h *Hpl) httpRequestLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return h.loadVarBasic(x, n)
}

func (h *Hpl) httpRequestStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return h.session.OnStoreVar(x, n, v)
}

func (h *Hpl) httpRequestEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.session.OnCall(x, n, args)
}

func (h *Hpl) httpRequestAction(x *pl.Evaluator, n string, arg pl.Val) error {
	return h.session.OnAction(x, n, arg)
}

func (h *Hpl) OnRequest(selector string, req *http.Request, param hrouter.Params, session SessionWrapper) error {
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
	h.request = NewRequestVal(req)
	h.params = NewRouterParamsVal(param)
	h.session = session

	h.Eval.LoadVarFn = h.httpRequestLoadVar
	h.Eval.StoreVarFn = h.httpRequestStoreVar
	h.Eval.CallFn = h.httpRequestEvalCall
	h.Eval.ActionFn = h.httpRequestAction

	defer func() {
		h.isRunning = false
		h.session = nil
	}()

	return h.Eval.Eval(selector, h.Policy)
}

// -----------------------------------------------------------------------------
// response phase

func foreachHeaderKV(arg pl.Val, fn func(key string, val string)) bool {
	if arg.Type == pl.ValList {
		for _, v := range arg.List().Data {
			if v.Type == pl.ValPair && v.Pair.First.Type == pl.ValStr && v.Pair.Second.Type == pl.ValStr {
				fn(v.Pair.First.String(), v.Pair.Second.String())
			}
		}
	} else if arg.Type == pl.ValPair {
		if arg.Pair.First.Type == pl.ValStr && arg.Pair.Second.Type == pl.ValStr {
			fn(arg.Pair.First.String(), arg.Pair.Second.String())
		}
	} else if ValIsHttpHeader(arg) {
		// known special user type to us, then just foreach the header
		hdr := arg.Usr().Context.(*Header)
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

func foreachStr(arg pl.Val, fn func(key string)) bool {
	if arg.Type == pl.ValList {
		for _, v := range arg.List().Data {
			if v.Type == pl.ValStr {
				fn(v.String())
			}
		}
	} else if arg.Type == pl.ValStr {
		fn(arg.String())
	} else {
		return false
	}
	return true
}

func (p *Hpl) httpResponseAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	switch actionName {

	case "response":
		if ValIsHttpResponse(arg) {
			r, _ := arg.Usr().Context.(*Response)
			p.respWriter.setResponse(r.response)
			return nil
		} else {
			return fmt.Errorf("invalid argument type for action response")
		}

	case "status":
		if arg.Type != pl.ValInt {
			return fmt.Errorf("status action must have a int argument")
		} else {
			p.respWriter.statusCode = int(arg.Int())
		}
		return nil

	case "header":
		if ValIsHttpHeader(arg) {
			h, _ := arg.Usr().Context.(*Header)
			p.respWriter.header = h.header
		} else {
			p.respWriter.clearHeader()
			if !foreachHeaderKV(
				arg,
				func(key, val string) {
					p.respWriter.header.Add(key, val)
				},
			) {
				return fmt.Errorf("invalid header value type")
			}
		}
		return nil

	case "header_add":
		if !foreachHeaderKV(
			arg,
			func(key string, val string) {
				p.respWriter.header.Add(key, val)
			}) {
			return fmt.Errorf("invalid header_add value type")
		}
		return nil

	case "header_set":
		if !foreachHeaderKV(
			arg,
			func(key string, val string) {
				p.respWriter.header.Set(key, val)
			}) {
			return fmt.Errorf("invalid header_set value type")
		}
		return nil

	case "header_try_set":
		if !foreachHeaderKV(
			arg,
			func(key string, val string) {
				hdr := p.respWriter.header
				if hdr.Get(key) == "" {
					hdr.Set(key, val)
				}
			}) {
			return fmt.Errorf("invalid header_try_set value type")
		}
		return nil

	case "body":
		if arg.Type == pl.ValStr {
			p.respWriter.writeBodyString(arg.String())
		} else if ValIsReadableStream(arg) {
			stream, ok := arg.Usr().Context.(*ReadableStream)
			must(ok, "invalid body type(.readablestream)")
			p.respWriter.writeBodyStream(stream.Stream)
		} else if ValIsHttpBody(arg) {
			body, ok := arg.Usr().Context.(*Body)
			must(ok, "invalid body type(http.body)")
			p.respWriter.writeBodyStream(body.Stream().Stream)
		} else {
			return fmt.Errorf("invalid body value type")
		}
		return nil

	default:
		break
	}

	return p.session.OnAction(x, actionName, arg)
}

func (p *Hpl) httpResponseLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return p.loadVarBasic(x, n)
}

func (p *Hpl) httpResponseStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.session.OnStoreVar(x, n, v)
}

func (p *Hpl) httpResponseEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return p.evalCallBasic(x, n, args)
}

func (h *Hpl) OnResponse(selector string,
	resp http.ResponseWriter,
	req *http.Request,
	param hrouter.Params,
	session SessionWrapper,
) error {
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

	h.request = NewRequestVal(req)
	h.respWriter = newHplResponseWriter(resp)
	h.params = NewRouterParamsVal(param)
	h.session = session

	h.Eval.LoadVarFn = h.httpResponseLoadVar
	h.Eval.StoreVarFn = h.httpResponseStoreVar
	h.Eval.CallFn = h.httpResponseEvalCall
	h.Eval.ActionFn = h.httpResponseAction

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

// -----------------------------------------------------------------------------
// log phase
func (h *Hpl) logLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	switch n {
	case "logFormat":
		return pl.NewValStr(h.log.Format.Raw), nil
	default:
		break
	}
	return h.loadVarBasic(x, n)
}

func (p *Hpl) logStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.session.OnStoreVar(x, n, v)
}

func (h *Hpl) logEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.evalCallBasic(x, n, args)
}

func (h *Hpl) logAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	switch actionName {
	case "format":
		if arg.Type != pl.ValStr {
			return fmt.Errorf("status format must have a string argument")
		} else {
			fmt, err := alog.NewSessionLogFormat(arg.String())
			if err != nil {
				return err
			}
			h.log.Format = fmt
			return nil
		}

	case "appendix":
		switch arg.Type {
		case pl.ValStr, pl.ValReal, pl.ValBool, pl.ValNull, pl.ValInt, pl.ValRegexp:
			str, _ := arg.ToString()
			h.log.Appendix = append(h.log.Appendix, str)
			break

		case pl.ValList:
			for _, e := range arg.List().Data {
				if e.Type == pl.ValStr {
					h.log.Appendix = append(h.log.Appendix, e.String())
				}
			}
			break

		default:
			return fmt.Errorf("status appendix must have a string or string list argument")
		}

		return nil

	default:
		return h.session.OnAction(x, actionName, arg)
	}
}

func (h *Hpl) OnLog(selector string, log *alog.SessionLog, session SessionWrapper) error {
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

	h.request = NewRequestVal(log.HttpRequest)
	h.params = NewRouterParamsVal(log.RouterParams)
	h.session = session
	h.log = log

	h.Eval.LoadVarFn = h.logLoadVar
	h.Eval.StoreVarFn = h.logStoreVar
	h.Eval.CallFn = h.logEvalCall
	h.Eval.ActionFn = h.logAction

	defer func() {
		h.isRunning = false
		h.respWriter = nil
		h.session = nil
	}()

	return h.Eval.Eval(selector, h.Policy)
}

// -----------------------------------------------------------------------------
// error phase
func (h *Hpl) errorLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	switch n {
	case "phase":
		return pl.NewValStr(h.session.GetPhaseName()), nil
	case "error":
		return pl.NewValStr(h.session.GetErrorDescription()), nil
	default:
		break
	}
	return h.loadVarBasic(x, n)
}

func (p *Hpl) errorStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.session.OnStoreVar(x, n, v)
}

func (h *Hpl) errorEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.evalCallBasic(x, n, args)
}

func (h *Hpl) errorAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return h.session.OnAction(x, actionName, arg)
}

func (h *Hpl) OnError(selector string, session SessionWrapper) error {
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

	h.session = session

	h.Eval.LoadVarFn = h.errorLoadVar
	h.Eval.StoreVarFn = h.errorStoreVar
	h.Eval.CallFn = h.errorEvalCall
	h.Eval.ActionFn = h.errorAction

	defer func() {
		h.isRunning = false
		h.session = nil
	}()

	return h.Eval.Eval(selector, h.Policy)
}
