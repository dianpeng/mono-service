package runtime

import (
	"fmt"
	"io/fs"
	"net/http"
	"time"

	// router
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/pl"
)

type Context interface {
	// HPL/PL scriptting callback
	OnLoadVar(*pl.Evaluator, string) (pl.Val, error)
	OnStoreVar(*pl.Evaluator, string, pl.Val) error
}

type Action interface {
	OnAction(*pl.Evaluator, string, pl.Val) error
}

type Resource interface {
	// special function used for exposing other utilities
	hpl.HttpClientFactory
}

// Interface used by VHost to bridge SessionWrapper/Service object into the HPL
// world. HPL knows nothing about the service/session and also hservice. HPL is
// just a wrapper around PL but with all the http/networking utilities
type SessionWrapper interface {
	Context
	Action
	Resource
}

type ConstSessionWrapper interface {
	Resource
}

// ----------------------------------------------------------------------------
type Runtime struct {
	Eval   *pl.Evaluator
	Module *pl.Module

	// Session related internal state, each HTTP transaction should set it to
	// the corresponding HTTP context when invoke OnInit call
	request    pl.Val
	params     pl.Val
	respWriter pl.Val
	log        pl.Val

	hplCtx    Context
	hplRt     Resource
	hplAction Action
}

func NewRuntime() *Runtime {
	p := &Runtime{}
	p.Eval = pl.NewEvaluatorSimple()
	return p
}

func NewRuntimeWithModule(module *pl.Module) *Runtime {
	p := &Runtime{}
	p.Eval = pl.NewEvaluatorSimple()
	p.SetModule(module)
	return p
}

func (h *Runtime) CompileModule(input string, fs fs.FS) error {
	p, err := pl.CompileModule(input, fs)
	if err != nil {
		return err
	}
	h.Module = p
	return nil
}

func (h *Runtime) SetModule(p *pl.Module) {
	h.Module = p
}

// Derive a HPL state from another existed HPL, suitable for using in background
func (h *Runtime) Derive(that *Runtime) {
	h.Module = that.Module

	// notes, we currently do not have a way to duplicate session state from that
	// HPL to our HPL and due to the thread issue, we cannot safely just do shallow
	// copy of the session object. Therefore, we do not support session variable
	// for now and user will get an error in background rule/function if they
	// manage to use session variable
}

func (p *Runtime) loadVarBasic(x *pl.Evaluator, n string) (pl.Val, error) {
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

func (h *Runtime) getHttpClientFactory() hpl.HttpClientFactory {
	return h.hplRt
}

func (h *Runtime) fnHttp(args []pl.Val,
	entry func(hpl.HttpClientFactory, []pl.Val) (pl.Val, error)) (pl.Val, error) {

	fac := h.getHttpClientFactory()
	if fac == nil {
		return pl.NewValNull(), fmt.Errorf("http client factory is not setup")
	}
	return entry(fac, args)
}

func (p *Runtime) loadFnVar(_ *pl.Evaluator, n string) (pl.Val, bool) {
	switch n {
	case "http::do":
		return pl.NewValNativeFunction(
			"http::do",
			func(args []pl.Val) (pl.Val, error) {
				return p.fnHttp(args, hpl.FnHttpDo)
			},
		), true

	case "http::get":
		return pl.NewValNativeFunction(
			"http::do",
			func(args []pl.Val) (pl.Val, error) {
				return p.fnHttp(args, hpl.FnHttpGet)
			},
		), true

	case "http::post":
		return pl.NewValNativeFunction(
			"http::do",
			func(args []pl.Val) (pl.Val, error) {
				return p.fnHttp(args, hpl.FnHttpPost)
			},
		), true

	default:
		break
	}

	return pl.NewValNull(), false
}

// -----------------------------------------------------------------------------
// customize phase
func (h *Runtime) customizeLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return h.hplCtx.OnLoadVar(x, n)
}

func (p *Runtime) customizeStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.hplCtx.OnStoreVar(x, n, v)
}

func (h *Runtime) customizeAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	if h.hplAction != nil {
		return h.hplAction.OnAction(x, actionName, arg)
	} else {
		return fmt.Errorf("unknown action: %s", actionName)
	}
}

func (h *Runtime) OnCustomize(selector string, session SessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}

	oldCtx := h.hplCtx
	oldRt := h.hplRt
	oldAct := h.hplAction

	h.hplCtx = session
	h.hplRt = session
	h.hplAction = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.customizeLoadVar,
		h.customizeStoreVar,
		h.customizeAction,
	)

	defer func() {
		h.respWriter = pl.NewValNull()
		h.hplCtx = oldCtx
		h.hplRt = oldRt
		h.hplAction = oldAct
	}()

	return h.Eval.Eval(selector, h.Module)
}

// -----------------------------------------------------------------------------
// global phase
func (h *Runtime) globalLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return pl.NewValNull(), fmt.Errorf("global initialization: unknown variable %s", n)
}

func (p *Runtime) globalStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return fmt.Errorf("global initialization: unknown variable set %s", n)
}

func (h *Runtime) globalAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return fmt.Errorf("global initialization: unknown action %s", actionName)
}

func (h *Runtime) OnGlobal(session ConstSessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
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
func (h *Runtime) configLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return pl.NewValNull(), fmt.Errorf("config initialization: unknown variable %s", n)
}

func (p *Runtime) configStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return fmt.Errorf("config initialization: unknown variable set %s", n)
}

func (h *Runtime) configAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return fmt.Errorf("config initialization: unknown action %s", actionName)
}

func (h *Runtime) OnConfig(context pl.EvalConfig, session ConstSessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
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
func (h *Runtime) initLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return h.loadVarBasic(x, n)
}

func (p *Runtime) initStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return p.hplCtx.OnStoreVar(x, n, v)
}

func (h *Runtime) initAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	if h.hplAction != nil {
		return h.hplAction.OnAction(x, actionName, arg)
	} else {
		return fmt.Errorf("unknown action: %s", actionName)
	}
}

// invoked every HTTP transaction/session
func (h *Runtime) OnInit(
	request pl.Val,
	hrouter pl.Val,
	respWriter pl.Val,
	session SessionWrapper,
	log *alog.Log,
) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}
	h.request = request
	h.params = hrouter
	h.respWriter = respWriter

	h.hplCtx = session
	h.hplRt = session
	h.hplAction = session

	h.log = hpl.NewAccessLogVal(log)

	h.Eval.Context = pl.NewCbEvalContext(
		h.initLoadVar,
		h.initStoreVar,
		h.initAction,
	)

	return h.Eval.EvalSession(h.Module)
}

// -----------------------------------------------------------------------------
func (h *Runtime) Emit(name string, context pl.Val) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}

	return h.Eval.EvalWithContext(name, context, h.Module)
}

// =============================================================================
// -----------------------------------------------------------------------------
// This is used for testing purpose and not used in production
type testHttpClientFactory struct {
	timeout int64
}

func (c *testHttpClientFactory) GetHttpClient(url string) (hpl.HttpClient, error) {
	return &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}, nil
}

func (h *Runtime) testLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	} else {
		return h.hplCtx.OnLoadVar(x, n)
	}
}

func (h *Runtime) testStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return h.hplCtx.OnStoreVar(x, n, v)
}

func (h *Runtime) testAction(x *pl.Evaluator, n string, v pl.Val) error {
	if h.hplAction != nil {
		return h.hplAction.OnAction(x, n, v)
	} else {
		return fmt.Errorf("unknown action: %s", n)
	}
}

func (h *Runtime) OnTestSession(session SessionWrapper) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}
	h.request = pl.NewValNull()
	h.params = pl.NewValNull()
	h.respWriter = pl.NewValNull()

	h.hplCtx = session
	h.hplRt = session
	h.hplAction = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.testLoadVar,
		h.testStoreVar,
		h.testAction,
	)

	return h.Eval.EvalSession(h.Module)
}

func (h *Runtime) OnTest(selector string, context pl.Val) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
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
