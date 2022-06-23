package pl

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"strings"
)

const (
	ConfigRule       = "@config"
	SessionRule      = "@session"
	GlobalRule       = "@global"
	defaultStackSize = 2048
)

// Config Population
type EvalConfig interface {
	PushConfig(*Evaluator, string, Val) error
	PopConfig(*Evaluator) error
	ConfigProperty(*Evaluator, string, Val, Val) error
	ConfigCommand(*Evaluator, string, []Val, Val) error
}

type EvalContext interface {
	LoadVar(*Evaluator, string) (Val, error)
	StoreVar(*Evaluator, string, Val) error
	Action(*Evaluator, string, Val) error
}

type cbEvalContext struct {
	loadVarFn  func(*Evaluator, string) (Val, error)
	storeVarFn func(*Evaluator, string, Val) error
	actionFn   func(*Evaluator, string, Val) error
}

const (
	// continue draining event until we are exhausted
	EventContextContinue = 0

	// stop draining event and leave the queue as is
	EventContextPuase = 1

	// stop draining event and clear all the pending queue
	EventContextStopAndClear = 2
)

type EventContext interface {
	// called when an async event is invoked, and it has an error. If this
	// function returns false, then event queue execution will be halt
	OnEventError(
		string,
		error,
	) int
}

func NewCbEvalContext(
	f0 func(*Evaluator, string) (Val, error),
	f1 func(*Evaluator, string, Val) error,
	f2 func(*Evaluator, string, Val) error,
) EvalContext {
	return &cbEvalContext{
		loadVarFn:  f0,
		storeVarFn: f1,
		actionFn:   f2,
	}
}

func NewNullEvalContext() EvalContext {
	return NewCbEvalContext(
		nil,
		nil,
		nil,
	)
}

func (x *cbEvalContext) LoadVar(a *Evaluator, b string) (Val, error) {
	if x.loadVarFn != nil {
		return x.loadVarFn(a, b)
	} else {
		return NewValNull(), fmt.Errorf("load_var: %s is unknown", b)
	}
}

func (x *cbEvalContext) StoreVar(a *Evaluator, b string, c Val) error {
	if x.storeVarFn != nil {
		return x.storeVarFn(a, b, c)
	} else {
		return fmt.Errorf("store_var: %s is unknown", b)
	}
}

func (x *cbEvalContext) Action(a *Evaluator, b string, c Val) error {
	if x.actionFn != nil {
		return x.actionFn(a, b, c)
	} else {
		return fmt.Errorf("action: %s is unknown", b)
	}
}

// Notes on function call frame layout
// The parser will generate bytecode to allow following stack value layout
//
// [funcframe] (a user value)
// [arg:N]
// [arg:N-1]
//   ...
// [arg:1]          <------- framep+1 (where local stack start)
// [index:function] <------- framep

type Evaluator struct {
	Stack   []Val
	Session []Val
	Context EvalContext
	Config  EvalConfig
	Event   EventContext

	// internal states -----------------------------------------------------------
	// current frame, ie the one that is been executing
	curframe     funcframe
	curexcep     Val
	eventQ       EventQueue
	inEventQueue bool
}

type exception struct {
	// where should this exception goes to
	handlerPc int
	stackSize int
}

// internal structure which we used to record the current frame and information
// will be setup inside of the function's caller reserve slot
type funcframe struct {
	pc     int
	prog   *program
	framep int
	farg   int
	excep  []exception
	sfunc  *scriptFunc
	nfunc  *nativeFunc
	event  Val
}

func dupFuncFrameForErr(fr *funcframe) *funcframe {
	return &funcframe{
		pc:     fr.pc,
		prog:   fr.prog,
		framep: fr.framep,
		farg:   fr.farg,
	}
}

func (ff *funcframe) isTop() bool {
	return ff.prog == nil
}

func (ff *funcframe) isNative() bool {
	return ff.nfunc != nil
}

func (ff *funcframe) frameInfo() string {
	if ff.prog != nil {
		return fmt.Sprintf("[pc=%d]"+
			"[framep=%d]"+
			"[argcount=%d]"+
			"[instr=%s]"+
			"[type=%s]"+
			"[name=%s]"+
			"[localcount=%d]"+
			"[source=]:\n%s",
			ff.pc,
			ff.framep,
			ff.farg,
			ff.prog.bcList[ff.pc].dumpWithProgram(ff.prog),
			ff.prog.typeName(),
			ff.prog.name,
			ff.prog.localSize,
			ff.prog.dbgList[ff.pc].where(),
		)
	} else {
		must(ff.nfunc != nil, "must have native function")
		return fmt.Sprintf("[pc=%d]"+
			"[framep=%d]"+
			"[argcount=%d]"+
			"[native_function]"+
			"[id=%s]",
			ff.pc,
			ff.framep,
			ff.farg,
			ff.nfunc.Id(),
		)
	}
}

func (e *Evaluator) curevent() Val {
	must(e.curframe.prog.progtype == progRule, "must be rule")
	return e.Stack[e.curframe.framep+1]
}

func (e *Evaluator) curfuncframeVal() Val {
	return newValFrame(&e.curframe)
}

func (e *Evaluator) hasExcepHandler() bool {
	return len(e.curframe.excep) == 0
}

func (e *Evaluator) curExcep() *exception {
	sz := len(e.curframe.excep)
	if sz != 0 {
		return &e.curframe.excep[sz-1]
	} else {
		return nil
	}
}

func (e *Evaluator) SetEventQueue(eq EventQueue) {
	e.eventQ = eq
}

func (e *Evaluator) EventQueue() EventQueue {
	return e.eventQ
}

func (e *Evaluator) emitEvent(
	name string,
	context Val,
) {
	must(e.eventQ != nil, "event queue must be setup")
	e.eventQ.OnEvent(name, context)
}

func (e *Evaluator) drainEventQueue(p *Policy) {
	if e.inEventQueue {
		return
	}
	e.inEventQueue = true
	defer func() {
		e.inEventQueue = false
	}()

	var status int
	statusp := &status

	callback := func(
		name string,
		err error,
	) bool {
		if e.Event != nil {
			*statusp = e.Event.OnEventError(
				name,
				err,
			)
		}

		if *statusp != EventContextContinue {
			return false
		} else {
			return true
		}
	}

	e.eventQ.Drain(
		e,
		p,
		callback,
	)

	if *statusp == EventContextStopAndClear {
		e.eventQ.Clear()
	}
}

func (e *Evaluator) pushExcep(pc int, stackSize int) {
	e.curframe.excep = append(e.curframe.excep, exception{
		handlerPc: pc,
		stackSize: stackSize,
	})
}

func (e *Evaluator) popExcep() {
	sz := len(e.curframe.excep)
	e.curframe.excep = e.curframe.excep[:sz-1]
}

func (e *Evaluator) prevframepos() int {
	// offset by 1 to skip the function's index on the local stack
	return e.curframe.framep + e.curframe.farg + 1
}

func (e *Evaluator) localpos() int {
	return e.curframe.framep + 1
}

func (e *Evaluator) prevfuncframe() *funcframe {
	v := e.Stack[e.prevframepos()]
	ff, ok := v.frame().(*funcframe)
	must(ok, "unknown stack, corrupted? %s", v.Id())
	return ff
}

func (e *Evaluator) popfuncframe(prev *funcframe) (int, *program) {
	e.popTo(e.curframe.framep)
	pc := prev.pc - 1
	prog := prev.prog
	e.curframe = *prev
	return pc, prog
}

func newfuncframe(
	pc int,
	prog *program,
	framep int,
	farg int,
	excep []exception,
	sfunc *scriptFunc,
	nfunc *nativeFunc,
) (*funcframe, Val) {
	ff := &funcframe{
		pc:     pc,
		prog:   prog,
		framep: framep,
		farg:   farg,
		excep:  excep,
		sfunc:  sfunc,
		nfunc:  nfunc,
	}
	return ff, newValFrame(ff)
}

func NewEvaluatorSimple() *Evaluator {
	return NewEvaluatorWithContext(
		NewNullEvalContext(),
	)
}

func NewEvaluatorWithContext(context EvalContext) *Evaluator {
	return &Evaluator{
		Stack:   make([]Val, 0, defaultStackSize),
		Session: nil,
		Context: context,
		eventQ:  &defEventQueue{},
	}
}

func NewEvaluatorWithContextCallback(
	f0 func(*Evaluator, string) (Val, error),
	f1 func(*Evaluator, string, Val) error,
	f2 func(*Evaluator, string, Val) error,
) *Evaluator {
	return NewEvaluatorWithContext(NewCbEvalContext(f0, f1, f2))
}

func NewEvaluator(context EvalContext, config EvalConfig) *Evaluator {
	return &Evaluator{
		Stack:   make([]Val, 0, defaultStackSize),
		Session: nil,
		Context: context,
		Config:  config,
		eventQ:  &defEventQueue{},
	}
}

// stack manipulation
func (e *Evaluator) pop() {
	e.popN(1)
}

func (e *Evaluator) popN(cnt int) {
	sz := len(e.Stack)
	must(sz >= cnt, "invalid pop size")
	e.Stack = e.Stack[:sz-cnt]
}

func (e *Evaluator) popTo(size int) {
	must(len(e.Stack) >= size, "invalid popTo size")
	e.Stack = e.Stack[:size]
}

func (e *Evaluator) clearStack() {
	if len(e.Stack) != 0 {
		e.Stack = e.Stack[:0]
	}
}

func (e *Evaluator) stackSize() int {
	return len(e.Stack)
}

func (e *Evaluator) push(v Val) {
	e.Stack = append(e.Stack, v)
}

func (e *Evaluator) topN(where int) Val {
	sz := len(e.Stack)
	must(sz >= where+1, "invalid topN index")
	idx := (sz - where - 1)
	return e.Stack[idx]
}

func (e *Evaluator) top0() Val {
	return e.topN(0)
}

func (e *Evaluator) top1() Val {
	return e.topN(1)
}

func (e *Evaluator) top2() Val {
	return e.topN(2)
}

// local variable is stored at the top of the stack from bot to top and the
// expression variable is been manipulated at the top of the stack in reverse
// direction
func (e *Evaluator) localslot(arg int) int {
	return e.curframe.framep + 1 + arg
}

func mustReal(x Val) float64 {
	if x.Type == ValInt {
		return float64(x.Int())
	} else {
		return x.Real()
	}
}

func powI(n, m int64) int64 {
	if m == 0 {
		return 1
	}
	result := n
	var i int64
	for i = 2; i <= m; i++ {
		result *= n
	}
	return result
}

// binary operation interpreter, we just do simple operations.
func (e *Evaluator) doBin(lhs, rhs Val, op int) (Val, error) {
	switch op {
	case bcSub:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(lhs.Int() - rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real() - rhs.Real()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) - mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for -")

	case bcMul:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(lhs.Int() * rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real() * rhs.Real()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) * mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcPow:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(powI(lhs.Int(), rhs.Int())), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(math.Pow(lhs.Real(), rhs.Real())), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(math.Pow(mustReal(lhs), mustReal(rhs))), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcMod:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				if rhs.Int() == 0 {
					return NewValNull(), fmt.Errorf("divide zero")
				}
				return NewValInt64(lhs.Int() % rhs.Int()), nil
			}
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcDiv:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				if rhs.Int() == 0 {
					return NewValNull(), fmt.Errorf("divide zero")
				}
				return NewValInt64(lhs.Int() / rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real() / rhs.Real()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) / mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcAdd:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(lhs.Int() + rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real() + rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValStr(lhs.String() + rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) + mustReal(rhs)), nil
		} else if lhs.Type == ValStr || rhs.Type == ValStr {
			if lhsStr, e1 := lhs.ToString(); e1 == nil {
				if rhsStr, e2 := rhs.ToString(); e2 == nil {
					return NewValStr(lhsStr + rhsStr), nil
				}
			}
		}
		return NewValNull(), fmt.Errorf("invalid operator for +")

	case bcEq:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int() == rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real() == rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String() == rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) == mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for ==")

	case bcNe:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int() != rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real() != rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String() != rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) != mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for !=")

	case bcLt:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int() < rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real() < rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String() < rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) < mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for <")

	case bcLe:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int() <= rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real() <= rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String() <= rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) <= mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for <=")

	case bcGt:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int() > rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real() > rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String() > rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) > mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for >")

	case bcGe:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int() >= rhs.Int()), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real() >= rhs.Real()), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String() >= rhs.String()), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) >= mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for >=")

	case bcRegexpMatch:
		if lhs.Type == ValStr && rhs.Type == ValRegexp {
			r := rhs.Regexp().Match([]byte(lhs.String()))
			return NewValBool(r), nil
		} else {
			return NewValNull(), fmt.Errorf("regexp operator ~ must be applied on string and regexp")
		}

	case bcRegexpNMatch:
		if lhs.Type == ValStr && rhs.Type == ValRegexp {
			r := rhs.Regexp().Match([]byte(lhs.String()))
			return NewValBool(!r), nil
		} else {
			return NewValNull(), fmt.Errorf("regexp operator !~ must be applied on string and regexp")
		}

	default:
		unreachable("binary operator")
		break
	}

	return NewValNull(), nil
}

func (e *Evaluator) doNegate(v Val) (Val, error) {
	switch v.Type {
	case ValInt:
		return NewValInt64(-v.Int()), nil
	case ValReal:
		return NewValReal(-v.Real()), nil
	default:
		return NewValNull(), fmt.Errorf("invalid operand for !")
	}
}

// Generate a human readable backtrace for reporting errors. Notes the backtrace
// list should be given by the caller since when we really need to return error
// it means the function frame has already been poped up
type btlist []*funcframe

func (e *Evaluator) backtrace(curframe *program, max int, bt btlist) string {
	sep := "....................."
	var b []string

	for idx, cf := range bt {
		b = append(b, fmt.Sprintf("%d>%s\n%s\n%s\n", idx, sep, cf.frameInfo(), sep))

		if idx == max {
			b = append(b, ".........\n")
			break
		}
	}
	return strings.Join(b, "")
}

// TODO(dpeng): Optimize diagnostic information
func (e *Evaluator) doErr(bt btlist, p *program, pc int, err error) error {
	if p != nil {
		dbg := p.dbgList[pc]
		return fmt.Errorf("symbol(%s), %s has error: %s\n%s",
			p.name, dbg.where(), err.Error(), e.backtrace(p, 10, bt))
	} else {
		return fmt.Errorf("symbol([native function]): %s", err.Error())
	}
}

// Return 3 tuple elements
// [1]: the program stops the execution, notes due to call, we can enter into
//      other script function
// [2]: the pc that stops the execution
// [3]: the error if we have

type runresult struct {
	prog *program
	pc   int
	e    error
}

func rrErr(p *program, pc int, e error) runresult {
	return runresult{
		prog: p,
		pc:   pc,
		e:    e,
	}
}

func rrErrf(p *program, pc int, format string, a ...interface{}) runresult {
	return rrErr(p, pc, fmt.Errorf(format, a...))
}

func rrDone() runresult {
	return runresult{}
}

func (rr *runresult) isDone() bool {
	return rr.e == nil
}

func (e *Evaluator) runP(
	prog *program,
	pc int,
	policy *Policy,
) runresult {
	// script function entry label, the bcSCall will setup stack layout and
	// jump(goto) this label for rexecution. prog will be swapped with the
	// function program
FUNC:
	for ; ; pc++ {
		bc := prog.bcList[pc]

		switch bc.opcode {
		case bcAction:
			actName := prog.idxStr(bc.argument)
			val := e.top0()
			if err := e.Context.Action(e, actName, val); err != nil {
				return rrErr(prog, pc, err)
			}
			e.pop()
			break

			// arithmetic
		case bcAdd,
			bcSub,
			bcMul,
			bcDiv,
			bcMod,
			bcPow,
			bcLt,
			bcLe,
			bcGt,
			bcGe,
			bcEq,
			bcNe,
			bcRegexpMatch,
			bcRegexpNMatch:

			lhs := e.top1()
			rhs := e.top0()
			e.popN(2)
			v, err := e.doBin(lhs, rhs, bc.opcode)
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.push(v)
			break

		case bcNot:
			t := e.top0()
			e.pop()
			e.push(NewValBool(!t.ToBoolean()))
			break

		case bcNegate:
			t := e.top0()
			e.pop()
			v, err := e.doNegate(t)
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.push(v)
			break

		// jump
		case bcOr:
			cond := e.top0()
			if cond.ToBoolean() {
				pc = bc.argument - 1
			} else {
				e.pop()
			}
			break

		case bcAnd:
			cond := e.top0()
			if !cond.ToBoolean() {
				pc = bc.argument - 1
			} else {
				e.pop()
			}
			break

		case bcSwap:
			sz := len(e.Stack)

			// top of the stack has been swapped, now t1 becomse t0
			e.Stack[sz-1], e.Stack[sz-2] = e.Stack[sz-2], e.Stack[sz-1]
			break

		case bcPop:
			e.pop()
			break

		case bcDup1:
			tos := e.top0()
			e.push(tos)
			break

		case bcDup2:
			// order matters
			to1 := e.top1()
			to0 := e.top0()

			e.push(to1)
			e.push(to0)

			break

		case bcJfalse:
			cond := e.top0()
			e.pop()
			if !cond.ToBoolean() {
				pc = bc.argument - 1
			}
			break

		case bcJtrue:
			cond := e.top0()
			e.pop()
			if cond.ToBoolean() {
				pc = bc.argument - 1
			}
			break

		case bcTernary:
			cond := e.top0()
			e.pop()

			if cond.ToBoolean() {
				pc = bc.argument - 1
			} else {
				e.pop()
				// fallthrough
			}
			break

		case bcJump:
			pc = bc.argument - 1
			break

		// other constant loading etc ...
		case bcLoadInt:
			e.push(NewValInt64(prog.idxInt(bc.argument)))
			break

		case bcLoadReal:
			e.push(NewValReal(prog.idxReal(bc.argument)))
			break

		case bcLoadStr:
			e.push(NewValStr(prog.idxStr(bc.argument)))
			break

		case bcLoadRegexp:
			e.push(NewValRegexp(prog.idxRegexp(bc.argument)))
			break

		case bcLoadTrue:
			e.push(NewValBool(true))
			break

		case bcLoadFalse:
			e.push(NewValBool(false))
			break

		case bcLoadNull:
			e.push(NewValNull())
			break

		case bcNewList:
			e.push(NewValList())
			break

		case bcAddList:
			cnt := bc.argument
			l := e.topN(cnt)
			must(l.Type == ValList, "must be list")
			for ii := len(e.Stack) - cnt; ii < len(e.Stack); ii++ {
				l.AddList(e.Stack[ii])
			}
			e.popN(cnt)
			break

		case bcNewMap:
			e.push(NewValMap())
			break

		case bcAddMap:
			cnt := bc.argument
			m := e.topN(cnt * 2)
			must(m.Type == ValMap, "must be map")
			for ii := len(e.Stack) - cnt*2; ii < len(e.Stack); {
				name := e.Stack[ii]
				must(name.Type == ValStr, "must be string")
				val := e.Stack[ii+1]
				m.AddMap(name.String(), val)
				ii = ii + 2
			}
			e.popN(cnt * 2)
			break

		case bcNewPair:
			t0 := e.top1()
			t1 := e.top0()
			e.popN(2)
			e.push(NewValPair(t0, t1))
			break

		case bcICall:
			paramSize := bc.argument

			// prepare argument list slice
			argStart := len(e.Stack) - paramSize
			argEnd := len(e.Stack)
			arg := e.Stack[argStart:argEnd]

			funcIndex := e.topN(paramSize)

			must(funcIndex.Type == ValInt,
				fmt.Sprintf("function indext must be indext but %s", funcIndex.Id()))

			must(funcIndex.Int() >= 0,
				fmt.Sprintf("function index must be none negative"))

			fentry := intrinsicFunc[funcIndex.Int()]
			r, err := fentry.entry(e, "$intrinsic$", arg)
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.popN(paramSize + 1)
			e.push(r)
			break

		case bcMCall:
			paramSize := bc.argument
			methodName := e.topN(paramSize)
			must(methodName.Type == ValStr,
				fmt.Sprintf("method name must be string but %s", methodName.Id()))

			// prepare argument list slice
			argStart := len(e.Stack) - paramSize
			argEnd := len(e.Stack)
			arg := e.Stack[argStart:argEnd]

			recv := e.topN(paramSize + 1)
			ret, err := recv.Method(methodName.String(), arg)
			if err != nil {
				return rrErr(prog, pc, err)
			}

			e.popN(paramSize + 2)
			e.push(ret)
			break

			// script function call and return
		case bcSCall, bcVCall:
			paramSize := bc.argument
			funcIndexOrEntry := e.topN(paramSize)

			// create the frame marker for the scriptting call
			_, newFV := newfuncframe(
				pc+1, // next pc
				prog,
				e.curframe.framep,
				e.curframe.farg,
				e.curframe.excep,
				e.curframe.sfunc,
				e.curframe.nfunc,
			)

			// setup new calling frame
			e.push(newFV)

			var sfunc *scriptFunc
			var nfunc *nativeFunc

			// enter into the new call
			if bc.opcode == bcSCall {
				idx := funcIndexOrEntry.Int()
				prog = prog.policy.fn[int(idx)]
				must(prog.freeCall(), "must be freecall")
			} else {
				if funcIndexOrEntry.IsClosure() {
					closure := funcIndexOrEntry.Closure()
					switch closure.Type() {
					case ClosureScript:
						fn, _ := closure.(*scriptFunc)
						prog = fn.prog
						sfunc = fn
						break

					case ClosureNative:
						// native function call, we still need to setup the call
						fn, _ := closure.(*nativeFunc)
						prog = nil
						nfunc = fn
						break

					default:
						return rrErrf(prog, pc, "object must be callable function, "+
							"but got type: %s", funcIndexOrEntry.Id())
					}
				} else {
					return rrErrf(prog, pc, "object must be function, but got type: %s",
						funcIndexOrEntry.Id())
				}
			}

			e.curframe.prog = prog
			e.curframe.pc = 0
			e.curframe.framep = len(e.Stack) - paramSize - 2
			e.curframe.farg = paramSize
			e.curframe.excep = nil

			if prog != nil {
				e.curframe.sfunc = sfunc
				// make sure to reset the PC when entering into the new function
				pc = 0
				goto FUNC
			} else {
				must(nfunc != nil, "nfunc cannot be nil")
				e.curframe.nfunc = nfunc

				// call the native function as if we are in a new frame, this is thee
				// native call trampoline, but written inline here as go code

				stackSize := len(e.Stack)
				argStart := stackSize - paramSize - 1 // NOTES, we just push a frame
				argEnd := stackSize - 1
				args := e.Stack[argStart:argEnd]

				val, err := nfunc.entry(
					args,
				)

				if err != nil {
					return rrErr(prog, pc, err)
				}

				// simulate the inline return and resume from where we stop
				prevcf := e.prevfuncframe()

				// return to where just stopped
				pc, prog = e.popfuncframe(prevcf)

				e.push(val)
				break
			}

			unreachable("bcXCall")

		case bcReturn:
			rv := e.top0()

			prevcf := e.prevfuncframe()

			// return back to the caller
			pc, prog = e.popfuncframe(prevcf)

			// return value push back to the stack
			e.push(rv)
			break

		case bcNewClosure:
			fn := prog.policy.fn[bc.argument]
			sfunc := newScriptFunc(fn)
			for _, uv := range fn.upvalue {
				if uv.onStack {
					sfunc.upvalue = append(sfunc.upvalue, e.Stack[e.localslot(uv.index)])
				} else {
					sfunc.upvalue = append(sfunc.upvalue, e.curframe.sfunc.upvalue[uv.index])
				}
			}

			e.push(newValSFunc(sfunc))
			break

		case bcLoadUpvalue:
			e.push(e.curframe.sfunc.upvalue[bc.argument])
			break

		case bcStoreUpvalue:
			e.curframe.sfunc.upvalue[bc.argument] = e.top0()
			e.pop()
			break

		case bcToStr:
			top := e.top0()
			str, err := top.ToString()
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.pop()
			e.push(NewValStr(str))
			break

		case bcConStr:
			sz := bc.argument
			var b bytes.Buffer
			for ii := len(e.Stack) - sz; ii < len(e.Stack); ii++ {
				v := e.Stack[ii]
				must(v.Type == ValStr, "must be string during concatenation")
				b.WriteString(v.String())
			}
			e.popN(sz)
			e.push(NewValStr(b.String()))
			break

		case bcLoadVar:
			vname := prog.idxStr(bc.argument)
			val, err := e.Context.LoadVar(e, vname)
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.push(val)
			break

		case bcStoreVar:
			top := e.top0()
			e.pop()

			vname := prog.idxStr(bc.argument)
			if err := e.Context.StoreVar(e, vname, top); err != nil {
				return rrErr(prog, pc, err)
			}
			break

		case bcLoadDollar:
			e.push(e.curevent())
			break

		case bcIndex:
			ee := e.top1()
			er := e.top0()
			val, err := ee.Index(er)
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.popN(2)
			e.push(val)
			break

		case bcIndexSet:
			recv := e.top2()
			index := e.top1()
			value := e.top0()
			e.popN(3)

			if err := recv.IndexSet(index, value); err != nil {
				return rrErr(prog, pc, err)
			}
			break

		case bcDot:
			ee := e.top0()
			vname := prog.idxStr(bc.argument)
			val, err := ee.Dot(vname)
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.pop()
			e.push(val)
			break

		case bcDotSet:
			recv := e.top1()
			value := e.top0()
			e.popN(2)

			if err := recv.DotSet(prog.idxStr(bc.argument), value); err != nil {
				return rrErr(prog, pc, err)
			}
			break

		case bcReserveLocal:
			sz := bc.argument
			if sz > 0 {
				for i := 0; i < sz; i++ {
					e.Stack = append(e.Stack, NewValNull())
				}
			}
			break

		case bcStoreLocal:
			e.Stack[e.localslot(bc.argument)] = e.top0()
			e.pop()
			break

		case bcLoadLocal:
			e.push(e.Stack[e.localslot(bc.argument)])
			break

		// special functions, ie intrinsic
		case bcTemplate:
			ctx := e.top0()
			e.pop()
			tmp := prog.idxTemplate(bc.argument)
			data, err := tmp.Execute(ctx)
			if err != nil {
				return rrErr(prog, pc, err)
			}

			e.push(NewValStr(data))
			break

		// session
		case bcSetSession:
			ctx := e.top0()
			e.pop()
			e.Session = append(e.Session, ctx)
			break

		case bcLoadSession:
			if len(e.Session) <= bc.argument {
				return rrErrf(prog, pc, "session variable is not existed")
			} else {
				e.push(e.Session[bc.argument])
			}
			break

		case bcStoreSession:
			if len(e.Session) <= bc.argument {
				return rrErrf(prog, pc, "session variable is not existed")
			} else {
				e.Session[bc.argument] = e.top0()
				e.pop()
			}
			break

		// exception
		case bcPushException:
			e.pushExcep(
				bc.argument,
				e.stackSize(),
			)
			break

		case bcPopException:
			e.popExcep()
			pc = bc.argument - 1
			break

		case bcLoadException:
			e.push(e.curexcep)
			break

		// configuration
		case bcConfigPush, bcConfigPushWithAttr:
			attr := NewValNull()
			if bc.opcode == bcConfigPushWithAttr {
				attr = e.top0()
				e.pop()
			}
			if e.Config != nil {
				if err := e.Config.PushConfig(e, prog.idxStr(bc.argument), attr); err != nil {
					return rrErr(prog, pc, err)
				}
			}
			break

		case bcConfigPop:
			if e.Config != nil {
				if err := e.Config.PopConfig(e); err != nil {
					return rrErr(prog, pc, err)
				}
			}
			break

		case bcConfigPropertySet, bcConfigPropertySetWithAttr:
			attr := NewValNull()
			name := e.top1()
			value := e.top0()
			e.popN(2)

			if bc.opcode == bcConfigPropertySetWithAttr {
				attr = e.top0()
				e.pop()
			}

			str, err := name.ToString()
			if err != nil {
				return rrErr(prog, pc, err)
			}

			if e.Config != nil {
				if err := e.Config.ConfigProperty(
					e,
					str,
					value,
					attr,
				); err != nil {
					return rrErr(prog, pc, err)
				}
			}
			break

		case bcConfigCommand, bcConfigCommandWithAttr:
			attr := NewValNull()
			pcnt := bc.argument
			name := e.topN(pcnt)
			popSize := pcnt + 1

			str, err := name.ToString()
			if err != nil {
				return rrErr(prog, pc, err)
			}

			if bc.opcode == bcConfigPropertySetWithAttr {
				popSize++
				attr = e.topN(pcnt + 1)
			}

			argStart := len(e.Stack) - pcnt
			argEnd := len(e.Stack)

			// notes, during execution of configuration instruction, we do duplication
			// of various input arguments since the configuration part typically will
			// just store the input argument for later use
			arg := Dup(e.Stack[argStart:argEnd])

			if e.Config != nil {
				if err := e.Config.ConfigCommand(
					e,
					str,
					arg,
					attr,
				); err != nil {
					return rrErr(prog, pc, err)
				}
			}

			e.popN(popSize)
			break

		// global
		case bcSetGlobal:
			ctx := e.top0()
			e.pop()
			if !policy.global.add(
				ctx,
			) {
				return rrErrf(prog, pc, "global varaible must store immutable type, "+
					"ie int, real, bool, string, null")
			}
			break

		case bcLoadGlobal:
			val, ok := policy.GetGlobal(bc.argument)
			if !ok {
				return rrErrf(prog, pc, "global variable loading error, "+
					"global variable is not existed")
			} else {
				e.push(val)
			}
			break

		case bcStoreGlobal:
			ctx := e.top0()
			e.pop()
			if !policy.StoreGlobal(bc.argument, ctx) {
				return rrErrf(prog, pc, "global variable storing error, "+
					"value is not immutable or global variable is not existed")
			}
			break

			// iterator
		case bcNewIterator:
			tos := e.top0()
			e.pop()

			itr, err := tos.NewIterator()
			if err != nil {
				return rrErr(prog, pc, err)
			}
			e.push(NewValIter(itr))
			break

		case bcHasIterator:
			tos := e.top0()
			must(tos.IsIter(), "must be iterator(has_iterator)")
			e.push(NewValBool(tos.Iter().Has()))
			break

		case bcDerefIterator:
			tos := e.top0()
			must(tos.IsIter(), "must be iterator(deref_iterator)")
			key, val, err := tos.Iter().Deref()
			if err != nil {
				return rrErr(prog, pc, err)
			}

			// order matters
			e.push(key)
			e.push(val)
			break

		case bcNextIterator:
			tos := e.top0()
			must(tos.IsIter(), "must be iterator(next_iterator)")
			e.push(NewValBool(tos.Iter().Next()))
			break

		case bcHalt:
			return rrDone()

		case bcEmit:
			context := e.top0()
			name := e.top1()
			e.popN(2)
			must(name.IsString(), "event name must be string")

			e.emitEvent(
				name.String(),
				context,
			)
			break

		default:
			log.Fatalf("invalid unknown bytecode %d", bc.opcode)
		}
	}
}

// unwind the stack until we hit an exception handler or we are done with all
// the function frames on the stack
func (e *Evaluator) unwindForExcep(toTop bool, err error) (int, *program, btlist, bool) {
	// -------------------------------------------------------------------------
	// ******************* Stack Unwind and Exception Handling *****************
	// -------------------------------------------------------------------------
	// perform error recovery until we hit one and then re-evaluate the frame
	// again. We will have to stackwalk all the function that is on the stack
	// and find out the correct handler to resume the execution accordingly
	// -------------------------------------------------------------------------
	bt := btlist{dupFuncFrameForErr(&e.curframe)}

	for !e.curframe.isTop() {
		// condition for breaking up, if we hit native then we just break now
		if toTop {
			if e.curframe.isTop() {
				break
			}
		} else {
			// as long as it is a native frame or top frame, we just break
			if e.curframe.isNative() || e.curframe.isTop() {
				break
			}
		}

		// start to check the handler
		cf := &e.curframe

		// now check whether the current frame has exception or not
		if xp := e.curExcep(); xp != nil {
			// notes native frame on the stack cannot be used to handle exception,
			// then just jump forward
			if !cf.isNative() {
				// try to handle it, we haven't used a table based exception handler
				// but rely on bytecode so the current exception must be the exception
				// that is matched

				e.popTo(xp.stackSize)
				pc := xp.handlerPc
				prog := cf.prog
				cf.pc = pc

				// currently just convert error to a string
				e.curexcep = NewValStr(err.Error())

				// okay recover the frame
				return pc, prog, bt, true
			}
		}

		// unwind the frame
		pframe := e.prevfuncframe()
		if !pframe.isTop() {
			bt = append(bt, pframe)
		}

		// unwind current frame and go back to the previous frame to check whether
		// we have exception handler or not
		e.popfuncframe(pframe)
	}

	return -1, nil, bt, false
}

func (e *Evaluator) runRule(event Val, prog *program, policy *Policy) error {
	must(e.Context != nil, "Evaluator's context is nil!")

	// just clear the stack size if needed before every run, since we need to reuse
	// this evaluator
	e.clearStack()

	// mark exception to be null, ie no exception
	e.curexcep = NewValNull()

	// assign a null scriptFunc
	e.curframe.sfunc = &scriptFunc{}

	// Enter into the VM with a native function call marker. This serves as a
	// frame marker to indicate the end of the script frame which will help us
	// to terminate the frame walk

	// -1 means this is a rule inside of the policy instead of a function call
	e.push(NewValInt(-1))

	// push the event onto the stack to simulate the argument
	e.push(event)

	_, entryV := newfuncframe(
		0,
		nil,
		0,
		0,
		nil,
		&scriptFunc{},
		nil,
	)
	e.push(entryV)

	pc := 0

	// Now let's enter into the bytecode VM
	{
		// point to currently executing program, notes the PC field is not updated
		// until the VM breaks
		e.curframe.prog = prog
		e.curframe.pc = pc
		e.curframe.framep = 0
		e.curframe.farg = 1
		e.curframe.sfunc.prog = prog

	RECOVER:
		rr := e.runP(prog, pc, policy)

		// finish execution
		if rr.isDone() {
			return nil
		}

		var bt btlist

		{
			a, b, c, d := e.unwindForExcep(true, rr.e)
			if d {
				pc = a
				prog = b
				goto RECOVER
			} else {
				bt = c
			}
		}

		return e.doErr(bt, rr.prog, rr.pc, rr.e)
	}

	return nil
}

func (e *Evaluator) funcProlog(fn Val, args []Val) int {
	fp := len(e.Stack)
	// performing arguments shuffling here, ie move user provided function
	// arguments into our own stack and create a valid frame for script function
	e.push(fn)

	// push all the arguments onto the stack
	for _, a := range args {
		e.push(a)
	}

	// push argument count onto the stack
	e.push(NewValInt(len(args)))

	// push current frame onto stack and once we are done we will return from it
	_, newFV := newfuncframe(
		e.curframe.pc,
		e.curframe.prog,
		e.curframe.framep,
		e.curframe.farg,
		e.curframe.excep,
		e.curframe.sfunc,
		e.curframe.nfunc,
	)
	e.push(newFV)

	return fp
}

// used by callback function, ie re-enter into the VM while a native call calls
// back into the VM.
// Due to the interleaved frame limitation, we cannot propagate the exception
// from inner frame to outer frame without major refactory of our APIs, ie
// providing go side awareness of inner exception throwned. Therefore, we uses
// a special error to represent the inner exception and allowing the native frame
// to do proper job etc, or just return the error as they wish. In most cases,
// this is not something that most people needs to be aware of
func (e *Evaluator) runSFunc(
	sfunc *scriptFunc,
	args []Val,
) (Val, error) {

	// performing arguments shuffling here, ie move user provided function
	// arguments into our own stack and create a valid frame for script function
	fp := e.funcProlog(newValSFunc(sfunc), args)

	// now set the current frame point to the script code been executing
	pc := 0
	prog := sfunc.prog
	e.curframe.sfunc = sfunc
	e.curframe.prog = prog
	e.curframe.pc = 0
	e.curframe.framep = fp
	e.curframe.farg = len(args)

	{
	RECOVER:
		rr := e.runP(prog, pc, prog.policy)

		// finish execution
		if rr.isDone() {

			// The value should be just placed on the stack and we should simulate a
			// return instruction here
			ret := e.top0()

			// pop the frame out, ie our frame
			prevcf := e.prevfuncframe()
			e.popfuncframe(prevcf)

			return ret, nil
		}

		var bt btlist

		{
			// unwind until we hit a native frame and then just report the error
			a, b, c, d := e.unwindForExcep(false, rr.e)
			if d {
				prog = b
				pc = a
				goto RECOVER
			} else {
				bt = c
			}
		}

		return NewValNull(), e.doErr(bt, rr.prog, rr.pc, rr.e)
	}
}

func (e *Evaluator) runNFunc(
	nfunc *nativeFunc,
	args []Val,
) (Val, error) {

	fp := e.funcProlog(
		newValNFunc(nfunc),
		args,
	)

	// now set the current frame point to the script code been executing
	e.curframe.nfunc = nfunc
	e.curframe.prog = nil
	e.curframe.pc = 0
	e.curframe.framep = fp
	e.curframe.farg = len(args)

	// now let's just run the native function
	v, err := nfunc.entry(args)

	// since native function does not support recover from exception for now,
	// just pops the frame and return from where we are

	prevcf := e.prevfuncframe()
	e.popfuncframe(prevcf)

	// TODO(dpeng): Decorate native function frame error ?
	return v, err
}

func (e *Evaluator) EvalConfig(p *Policy) error {
	if !p.HasConfig() {
		return nil
	}
	if e.Config == nil {
		return fmt.Errorf("evaluator's Config is not set")
	}
	return e.runRule(NewValNull(), p.config, p)
}

func (e *Evaluator) EvalGlobal(p *Policy) error {
	defer func() {
		e.drainEventQueue(p)
	}()

	if !p.HasGlobal() {
		return nil
	}
	p.global.globalVar = nil
	return e.runRule(NewValNull(), p.global.globalProgram, p)
}

func (e *Evaluator) EvalSession(p *Policy) error {
	defer func() {
		e.drainEventQueue(p)
	}()

	if !p.HasSession() {
		return nil
	}
	e.Session = nil
	return e.runRule(NewValStr(SessionRule), p.session, p)
}

func (e *Evaluator) Eval(event string, p *Policy) error {
	defer func() {
		e.drainEventQueue(p)
	}()

	if prog := p.findEvent(event); prog != nil {
		return e.runRule(NewValNull(), prog, p)
	} else {
		return nil
	}
}

func (e *Evaluator) EvalWithContext(event string, context Val, p *Policy) error {
	defer func() {
		e.drainEventQueue(p)
	}()

	if prog := p.findEvent(event); prog != nil {
		return e.runRule(context, prog, p)
	} else {
		return nil
	}
}

// Notes, this must be used for evaluation of event queue event since inside of
// this function, it will NOT issue event queue call again which prevent from
// been called recursively
func (e *Evaluator) EvalDeferred(
	name string,
	context Val,
	p *Policy,
) error {
	if prog := p.findEvent(name); prog != nil {
		return e.runRule(context, prog, p)
	} else {
		return nil
	}
}

// Used by the config module for !eval directive. Notes the policy should just
// contain one program inside of its .p field
func (e *Evaluator) EvalExpr(p *Policy) (Val, error) {
	if len(p.p) == 0 {
		return NewValNull(), nil
	}
	if len(p.p) != 1 {
		return NewValNull(), fmt.Errorf("not an expression policy")
	}
	if err := e.runRule(NewValNull(), p.p[0], p); err != nil {
		return NewValNull(), err
	}
	return e.top0(), nil
}
