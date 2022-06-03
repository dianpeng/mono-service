package hpl

import (
	"fmt"
	"io"
	"net/http"

	// router
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"
	hrouter "github.com/julienschmidt/httprouter"

	// library usage
	"strings"
)

func must(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("must: %s", msg))
	}
}

func unreachable(msg string) {
	panic(fmt.Sprintf("unreachable: %s", msg))
}

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
	Eval   *pl.Evaluator
	Policy *pl.Policy

	UserLoadFn pl.EvalLoadVar
	UserCall   pl.EvalCall
	UserAction pl.EvalAction

	// internal status during evaluation context
	request    pl.Val
	respWriter *hplResponseWriter
	params     pl.Val
	session    service.Session

	log       *alog.SessionLog
	isRunning bool
}

func foreachHeaderKV(arg pl.Val, fn func(key string, val string)) bool {
	if arg.Type == pl.ValList {
		for _, v := range arg.List.Data {
			if v.Type == pl.ValPair && v.Pair.First.Type == pl.ValStr && v.Pair.Second.Type == pl.ValStr {
				fn(v.Pair.First.String, v.Pair.Second.String)
			}
		}
	} else if arg.Type == pl.ValPair {
		if arg.Pair.First.Type == pl.ValStr && arg.Pair.Second.Type == pl.ValStr {
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

func foreachStr(arg pl.Val, fn func(key string)) bool {
	if arg.Type == pl.ValList {
		for _, v := range arg.List.Data {
			if v.Type == pl.ValStr {
				fn(v.String)
			}
		}
	} else if arg.Type == pl.ValStr {
		fn(arg.String)
	} else {
		return false
	}
	return true
}

func (p *Hpl) httpEvalAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	// http response generation action, ie builting actions
	switch actionName {
	case "status":
		if arg.Type != pl.ValInt {
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
		if arg.Type == pl.ValStr {
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
func (p *Hpl) httpLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	switch n {
	case "request":
		return p.request, nil
	case "params":
		return p.params, nil
	case "serviceName":
		return pl.NewValStr(p.session.Service().Name()), nil
	case "serviceIdl":
		return pl.NewValStr(p.session.Service().IDL()), nil
	case "servicePolicy":
		return pl.NewValStr(p.session.Service().Policy()), nil
	case "serviceRouter":
		return pl.NewValStr(p.session.Service().Router()), nil
	case "serviceMethodList":
		r := pl.NewValList()
		for _, v := range p.session.Service().MethodList() {
			r.AddList(pl.NewValStr(v))
		}
		return r, nil
	default:
		if p.UserLoadFn != nil {
			return p.UserLoadFn(x, n)
		} else {
			return pl.NewValNull(), fmt.Errorf("unknown variable %s", n)
		}
	}
}

func (h *Hpl) fnHttp(args []pl.Val) (pl.Val, error) {
	if h.session == nil {
		return pl.NewValNull(), fmt.Errorf("function: http cannot be executed, session not bound")
	}
	sres := h.session.SessionResource()
	if sres == nil {
		return pl.NewValNull(), fmt.Errorf("function: http cannot be executed, session resource empty")
	}
	return fnDoHttp(sres, args)
}

func (p *Hpl) httpEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
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
		return pl.NewValNull(), fmt.Errorf("function %s is unknown", n)
	}
}

func NewHpl(f0 pl.EvalLoadVar, f1 pl.EvalCall, f2 pl.EvalAction) *Hpl {
	p := &Hpl{
		UserLoadFn: f0,
		UserCall:   f1,
		UserAction: f2,
	}

	p.Eval = pl.NewEvaluatorSimple()
	return p
}

func NewHplWithPolicy(f0 pl.EvalLoadVar, f1 pl.EvalCall, f2 pl.EvalAction, policy *pl.Policy) *Hpl {
	p := &Hpl{
		UserLoadFn: f0,
		UserCall:   f1,
		UserAction: f2,
	}

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

// -----------------------------------------------------------------------------
// prepare phase
func (h *Hpl) prepareLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	switch n {
	case "serviceName":
		return pl.NewValStr(h.session.Service().Name()), nil
	case "serviceIdl":
		return pl.NewValStr(h.session.Service().IDL()), nil
	case "servicePolicy":
		return pl.NewValStr(h.session.Service().Policy()), nil
	case "serviceRouter":
		return pl.NewValStr(h.session.Service().Router()), nil
	case "serviceMethodList":
		r := pl.NewValList()
		for _, v := range h.session.Service().MethodList() {
			r.AddList(pl.NewValStr(v))
		}
		return r, nil
	default:
		break
	}
	if h.UserLoadFn != nil {
		return h.UserLoadFn(x, n)
	} else {
		return pl.NewValNull(), fmt.Errorf("unknown variable %s", n)
	}
}

func (h *Hpl) prepareEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.httpEvalCall(x, n, args)
}

func (h *Hpl) prepareActionCall(x *pl.Evaluator, actionName string, arg pl.Val) error {
	if h.UserAction != nil {
		return h.UserAction(x, actionName, arg)
	} else {
		return fmt.Errorf("unknown action %s", actionName)
	}
}

func (h *Hpl) OnSession(session service.Session) error {
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

	return h.Eval.EvalSession(h.Policy)
}

// -----------------------------------------------------------------------------
// session log phase
func (h *Hpl) logLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	switch n {
	case "logFormat":
		return pl.NewValStr(h.log.Format.Raw), nil
	default:
		break
	}
	return h.httpLoadVar(x, n)
}

func (h *Hpl) logEvalCall(x *pl.Evaluator, n string, args []pl.Val) (pl.Val, error) {
	return h.httpEvalCall(x, n, args)
}

func (h *Hpl) logActionCall(x *pl.Evaluator, actionName string, arg pl.Val) error {
	switch actionName {
	case "format":
		if arg.Type != pl.ValStr {
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
		case pl.ValStr, pl.ValReal, pl.ValBool, pl.ValNull, pl.ValInt, pl.ValRegexp:
			str, _ := arg.ToString()
			h.log.Appendix = append(h.log.Appendix, str)
			break

		case pl.ValList:
			for _, e := range arg.List.Data {
				if e.Type == pl.ValStr {
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
