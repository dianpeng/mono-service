package runtime

import (
	"fmt"
	"io/fs"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/pl"
)

type Resource interface {
	// special function used for exposing other utilities
	hpl.HttpClientFactory
}

type Runtime struct {
	Eval   *pl.Evaluator
	Module *pl.Module

	conn     pl.Val
	log      pl.Val
	resource Resource
}

func NewRuntime() *Runtime {
	p := &Runtime{}
	p.Eval = pl.NewEvaluatorSimple()
	return p
}

func NewRuntimeWithModule(m *pl.Module) *Runtime {
	p := &Runtime{}
	p.Eval = pl.NewEvaluatorSimple()
	p.SetModule(m)
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

func (r *Runtime) SetModule(p *pl.Module) {
	r.Module = p
}

func (h *Runtime) getHttpClientFactory() hpl.HttpClientFactory {
	return h.resource
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

func (p *Runtime) loadVar(
	x *pl.Evaluator,
	n string,
) (pl.Val, error) {

	if v, ok := p.loadFnVar(x, n); ok {
		return v, nil
	}

	switch n {
	case "conn":
		return p.conn, nil
	case "log":
		return p.log, nil
	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("Runtime: unknown variable %s", n)
}

func (p *Runtime) storeVar(
	_ *pl.Evaluator,
	n string,
	_ pl.Val,
) error {
	return fmt.Errorf("Runtime: unknown variable set %s", n)
}

func (p *Runtime) action(
	_ *pl.Evaluator,
	n string,
	_ pl.Val,
) error {
	return fmt.Errorf("Runtime: action %s is unknown", n)
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

func (h *Runtime) OnGlobal(resource Resource) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}
	h.resource = resource

	h.Eval.Context = pl.NewCbEvalContext(
		h.globalLoadVar,
		h.globalStoreVar,
		h.globalAction,
	)

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

func (h *Runtime) OnConfig(context pl.EvalConfig, resource Resource) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}
	h.resource = resource

	h.Eval.Context = pl.NewCbEvalContext(
		h.configLoadVar,
		h.configStoreVar,
		h.configAction,
	)
	h.Eval.Config = context
	return h.Eval.EvalConfig(h.Module)
}

func (h *Runtime) OnInit(
	conn pl.Val,
	resource Resource,
	log *alog.Log,
) error {
	if h.Module == nil {
		return fmt.Errorf("Runtime engine does not have any module binded")
	}

	h.conn = conn
	h.log = hpl.NewAccessLogVal(log)

	h.resource = resource
	h.Eval.Context = pl.NewCbEvalContext(
		h.loadVar,
		h.storeVar,
		h.action,
	)
	return h.Eval.EvalSession(h.Module)
}

func (h *Runtime) Emit(
	name string,
	context pl.Val,
) (pl.Val, error) {

	if h.Module == nil {
		return pl.NewValNull(), fmt.Errorf("Runtime engine does not have any module binded")
	}

	return h.Eval.EvalWithContext(
		name,
		context,
		h.Module,
	)
}
