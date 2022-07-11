package hpl

import (
	"fmt"
	"io/fs"
	"net/http"
	"time"

	// router
	"github.com/dianpeng/mono-service/alog"
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

type HttpClientFactory interface {
	GetHttpClient(url string) (HttpClient, error)
}

type ConstSessionWrapper interface {
	HttpClientFactory
}

type HplContext interface {
	// HPL/PL scriptting callback
	OnLoadVar(*pl.Evaluator, string) (pl.Val, error)
	OnStoreVar(*pl.Evaluator, string, pl.Val) error
	OnAction(*pl.Evaluator, string, pl.Val) error
}

type HplRuntime interface {
	// special function used for exposing other utilities
	HttpClientFactory
}

// Interface used by VHost to bridge SessionWrapper/Service object into the HPL
// world. HPL knows nothing about the service/session and also hservice. HPL is
// just a wrapper around PL but with all the http/networking utilities
type SessionWrapper interface {
	HplContext
	HplRuntime
}

// ----------------------------------------------------------------------------
type Hpl struct {
	Eval   *pl.Evaluator
	Module *pl.Module

	// Session related internal state, each HTTP transaction should set it to
	// the corresponding HTTP context when invoke OnInit call
	request    pl.Val
	params     pl.Val
	respWriter pl.Val
	log        pl.Val

	hplCtx HplContext
	hplRt  HplRuntime
}

func NewHpl() *Hpl {
	p := &Hpl{}
	p.Eval = pl.NewEvaluatorSimple()
	return p
}

func NewHplWithModule(module *pl.Module) *Hpl {
	p := &Hpl{}
	p.Eval = pl.NewEvaluatorSimple()
	p.SetModule(module)
	return p
}

func (h *Hpl) CompileModule(input string, fs fs.FS) error {
	p, err := pl.CompileModule(input, fs)
	if err != nil {
		return err
	}
	h.Module = p
	return nil
}

func (h *Hpl) SetModule(p *pl.Module) {
	h.Module = p
}

// Derive a HPL state from another existed HPL, suitable for using in background
func (h *Hpl) Derive(that *Hpl) {
	h.Module = that.Module

	// notes, we currently do not have a way to duplicate session state from that
	// HPL to our HPL and due to the thread issue, we cannot safely just do shallow
	// copy of the session object. Therefore, we do not support session variable
	// for now and user will get an error in background rule/function if they
	// manage to use session variable
}

func (p *Hpl) loadVarBasic(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := p.loadFnVar(x, n); ok {
		return v, nil
	}
	switch n {
	case "request":
		return p.request, nil
	case "params":
		return p.params, nil
	case "response":
		return p.respWriter, nil
	case "log":
		return p.log, nil
	default:
		return p.hplCtx.OnLoadVar(x, n)
	}
}

func (h *Hpl) getHttpClientFactory() HttpClientFactory {
	return h.hplRt
}

func (h *Hpl) fnHttp(args []pl.Val,
	entry func(HttpClientFactory, []pl.Val) (pl.Val, error)) (pl.Val, error) {

	fac := h.getHttpClientFactory()
	if fac == nil {
		return pl.NewValNull(), fmt.Errorf("http client factory is not setup")
	}
	return entry(fac, args)
}

func (p *Hpl) loadFnVar(_ *pl.Evaluator, n string) (pl.Val, bool) {
	switch n {
	case "http::do":
		return pl.NewValNativeFunction(
			"http::do",
			func(args []pl.Val) (pl.Val, error) {
				return p.fnHttp(args, fnHttpDo)
			},
		), true

	case "http::get":
		return pl.NewValNativeFunction(
			"http::do",
			func(args []pl.Val) (pl.Val, error) {
				return p.fnHttp(args, fnHttpGet)
			},
		), true

	case "http::post":
		return pl.NewValNativeFunction(
			"http::do",
			func(args []pl.Val) (pl.Val, error) {
				return p.fnHttp(args, fnHttpPost)
			},
		), true

	default:
		break
	}

	return pl.NewValNull(), false
}

// -----------------------------------------------------------------------------
// customize phase
func (h *Hpl) customizeLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return h.hplCtx.OnLoadVar(x, n)
}

func (p *Hpl) customizeStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.hplCtx.OnStoreVar(x, n, v)
}

func (h *Hpl) customizeAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return h.hplCtx.OnAction(x, actionName, arg)
}

func (h *Hpl) OnCustomize(selector string, session SessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}

	oldCtx := h.hplCtx
	oldRt := h.hplRt

	h.hplCtx = session
	h.hplRt = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.customizeLoadVar,
		h.customizeStoreVar,
		h.customizeAction,
	)

	defer func() {
		h.respWriter = pl.NewValNull()
		h.hplCtx = oldCtx
		h.hplRt = oldRt
	}()

	return h.Eval.Eval(selector, h.Module)
}

// -----------------------------------------------------------------------------
// global phase
func (h *Hpl) globalLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return pl.NewValNull(), fmt.Errorf("global initialization: unknown variable %s", n)
}

func (p *Hpl) globalStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return fmt.Errorf("global initialization: unknown variable set %s", n)
}

func (h *Hpl) globalAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return fmt.Errorf("global initialization: unknown action %s", actionName)
}

func (h *Hpl) OnGlobal(session ConstSessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}

	h.hplRt = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.globalLoadVar,
		h.globalStoreVar,
		h.globalAction,
	)

	defer func() {
		h.hplRt = nil
		h.hplCtx = nil
	}()

	return h.Eval.EvalGlobal(h.Module)
}

// -----------------------------------------------------------------------------
// config phase
func (h *Hpl) configLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return pl.NewValNull(), fmt.Errorf("config initialization: unknown variable %s", n)
}

func (p *Hpl) configStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return fmt.Errorf("config initialization: unknown variable set %s", n)
}

func (h *Hpl) configAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return fmt.Errorf("config initialization: unknown action %s", actionName)
}

func (h *Hpl) OnConfig(context pl.EvalConfig, session ConstSessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}
	h.hplRt = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.configLoadVar,
		h.configStoreVar,
		h.configAction,
	)
	h.Eval.Config = context

	defer func() {
		h.hplRt = nil
		h.hplCtx = nil
	}()

	return h.Eval.EvalConfig(h.Module)
}

// -----------------------------------------------------------------------------
// init phase
func (h *Hpl) initLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return h.loadVarBasic(x, n)
}

func (p *Hpl) initStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.hplCtx.OnStoreVar(x, n, v)
}

func (h *Hpl) initAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return h.hplCtx.OnAction(x, actionName, arg)
}

// invoked every HTTP transaction/session
func (h *Hpl) OnInit(
	request pl.Val,
	hrouter pl.Val,
	respWriter pl.Val,
	session SessionWrapper,
	log *alog.Log,
) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}
	h.request = request
	h.params = hrouter
	h.respWriter = respWriter

	h.hplCtx = session
	h.hplRt = session
	h.log = NewAccessLogVal(log)

	h.Eval.Context = pl.NewCbEvalContext(
		h.initLoadVar,
		h.initStoreVar,
		h.initAction,
	)

	defer func() {
	}()

	return h.Eval.EvalSession(h.Module)
}

// -----------------------------------------------------------------------------
func (h *Hpl) RunWithContext(name string, context pl.Val) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}

	return h.Eval.EvalWithContext(name, context, h.Module)
}

// =============================================================================
// -----------------------------------------------------------------------------
// This is used for testing purpose and not used in production
type testHttpClientFactory struct {
	timeout int64
}

func (c *testHttpClientFactory) GetHttpClient(url string) (HttpClient, error) {
	return &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}, nil
}

func (h *Hpl) testLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	} else {
		return h.hplCtx.OnLoadVar(x, n)
	}
}

func (h *Hpl) testStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return h.hplCtx.OnStoreVar(x, n, v)
}

func (h *Hpl) testAction(x *pl.Evaluator, n string, v pl.Val) error {
	return h.hplCtx.OnAction(x, n, v)
}

func (h *Hpl) OnTestSession(session SessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}
	h.request = pl.NewValNull()
	h.params = pl.NewValNull()
	h.respWriter = pl.NewValNull()

	h.hplCtx = session
	h.hplRt = session
	h.Eval.Context = pl.NewCbEvalContext(
		h.testLoadVar,
		h.testStoreVar,
		h.testAction,
	)

	return h.Eval.EvalSession(h.Module)
}

func (h *Hpl) OnTest(selector string, context pl.Val) error {
	if h.Module == nil {
		return fmt.Errorf("the Hpl engine does not have any module binded")
	}
	h.request = pl.NewValNull()
	h.params = pl.NewValNull()
	h.respWriter = pl.NewValNull()

	h.Eval.Context = pl.NewCbEvalContext(
		h.testLoadVar,
		h.testStoreVar,
		h.testAction,
	)

	return h.Eval.EvalWithContext(selector, context, h.Module)
}
