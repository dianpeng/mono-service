package hpl

import (
	"fmt"
	"net/http"

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

// Interface used by VHost to bridge SessionWrapper/Service object into the HPL
// world. HPL knows nothing about the service/session and also hservice. HPL is
// just a wrapper around PL but with all the http/networking utilities
type SessionWrapper interface {
	// HPL/PL scriptting callback
	OnLoadVar(string) (pl.Val, error)
	OnStoreVar(string, pl.Val) error
	OnAction(string, pl.Val) error

	// special function used for exposing other utilities
	HttpClientFactory
}

// ----------------------------------------------------------------------------
type Hpl struct {
	Eval   *pl.Evaluator
	Policy *pl.Policy

	isRunning bool

	// Session related internal state, each HTTP transaction should set it to
	// the corresponding HTTP context when invoke OnInit call
	request    pl.Val
	params     pl.Val
	respWriter pl.Val

	session      SessionWrapper
	constSession ConstSessionWrapper
	log          *alog.SessionLog
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
	default:
		return p.session.OnLoadVar(n)
	}
}

func (h *Hpl) getHttpClientFactory() HttpClientFactory {
	if h.session != nil {
		return h.session
	}
	if h.constSession != nil {
		return h.constSession
	}
	return nil
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
	return h.session.OnLoadVar(n)
}

func (p *Hpl) customizeStoreVar(_ *pl.Evaluator, n string, v pl.Val) error {
	return p.session.OnStoreVar(n, v)
}

func (h *Hpl) customizeAction(_ *pl.Evaluator, actionName string, arg pl.Val) error {
	return h.session.OnAction(actionName, arg)
}

func (h *Hpl) OnCustomize(selector string, session SessionWrapper) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}

	h.isRunning = true
	h.session = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.customizeLoadVar,
		h.customizeStoreVar,
		h.customizeAction,
	)

	defer func() {
		h.isRunning = false
		h.respWriter = pl.NewValNull()
		h.session = nil
	}()

	return h.Eval.Eval(selector, h.Policy)
}

// -----------------------------------------------------------------------------
// const phase
func (h *Hpl) constLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	if v, ok := h.loadFnVar(x, n); ok {
		return v, nil
	}
	return pl.NewValNull(), fmt.Errorf("const initialization: unknown variable %s", n)
}

func (p *Hpl) constStoreVar(x *pl.Evaluator, n string, v pl.Val) error {
	return fmt.Errorf("const initialization: unknown variable set %s", n)
}

func (h *Hpl) constAction(x *pl.Evaluator, actionName string, arg pl.Val) error {
	return fmt.Errorf("const initialization: unknown action %s", actionName)
}

func (h *Hpl) OnConst(session ConstSessionWrapper) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}

	h.isRunning = true
	h.constSession = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.constLoadVar,
		h.constStoreVar,
		h.constAction,
	)

	defer func() {
		h.isRunning = false
		h.constSession = nil
	}()

	return h.Eval.EvalConst(h.Policy)
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
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}

	h.isRunning = true
	h.constSession = session

	h.Eval.Context = pl.NewCbEvalContext(
		h.constLoadVar,
		h.constStoreVar,
		h.constAction,
	)
	h.Eval.Config = context

	defer func() {
		h.isRunning = false
		h.constSession = nil
	}()

	return h.Eval.EvalConfig(h.Policy)
}

// -----------------------------------------------------------------------------
// init phase
func (h *Hpl) initLoadVar(x *pl.Evaluator, n string) (pl.Val, error) {
	return h.loadVarBasic(x, n)
}

func (p *Hpl) initStoreVar(_ *pl.Evaluator, n string, v pl.Val) error {
	return p.session.OnStoreVar(n, v)
}

func (h *Hpl) initAction(_ *pl.Evaluator, actionName string, arg pl.Val) error {
	return h.session.OnAction(actionName, arg)
}

// invoked every HTTP transaction/session
func (h *Hpl) OnInit(
	request pl.Val,
	hrouter pl.Val,
	respWriter pl.Val,
	session SessionWrapper,
	log *alog.SessionLog,
) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}

	h.isRunning = true
	h.request = request
	h.params = hrouter
	h.respWriter = respWriter
	h.session = session
	h.log = log

	h.Eval.Context = pl.NewCbEvalContext(
		h.initLoadVar,
		h.initStoreVar,
		h.initAction,
	)

	defer func() {
		h.isRunning = false
	}()

	return h.Eval.EvalSession(h.Policy)
}

// -----------------------------------------------------------------------------
func (h *Hpl) Run(name string) error {
	if h.Policy == nil {
		return fmt.Errorf("the Hpl engine does not have any policy binded")
	}
	if h.isRunning {
		return fmt.Errorf("the Hpl engine is running, it does not support re-enter")
	}

	h.isRunning = true
	defer func() {
		h.isRunning = false
	}()

	return h.Eval.Eval(name, h.Policy)
}
