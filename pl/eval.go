package pl

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"strings"
)

// Evaluation part of the VM
type EvalLoadVar func(*Evaluator, string) (Val, error)
type EvalStoreVar func(*Evaluator, string, Val) error
type EvalCall func(*Evaluator, string, []Val) (Val, error)
type EvalAction func(*Evaluator, string, Val) error

// notes on function call frame layout
//
// the parser will generate bytecode to allow following stack value layout
//
// [funcframe] (a user value)
// [arg:N]
// [arg:N-1]
//   ...
// [arg:1]          <-------------- framep+1 (where local stack start)
// [index:function] <-------------- framep

type Evaluator struct {
	Stack      []Val
	Session    []Val
	LoadVarFn  EvalLoadVar
	StoreVarFn EvalStoreVar
	CallFn     EvalCall
	ActionFn   EvalAction

	// internal states

	// current frame, ie the one that is been executing
	curframe funcframe
}

// internal structure which we used to record the current frame and information
// will be setup inside of the function's caller reserve slot
type funcframe struct {
	pc     int
	prog   *program
	framep int
	farg   int
}

func (ff *funcframe) isTop() bool {
	return ff.prog == nil
}

func (ff *funcframe) frameInfo() string {
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
}

func (e *Evaluator) curfuncframeVal() Val {
	return NewValUsrData(&e.curframe)
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
	must(v.Type == ValUsr, "must be user")
	ff, ok := v.Usr().Context.(*funcframe)
	must(ok, "must be frame")
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
) (*funcframe, Val) {
	ff := &funcframe{
		pc:     pc,
		prog:   prog,
		framep: framep,
		farg:   farg,
	}
	return ff, NewValUsrData(ff)
}

func NewEvaluatorSimple() *Evaluator {
	return &Evaluator{}
}

func NewEvaluator(
	f0 EvalLoadVar,
	f1 EvalStoreVar,
	f2 EvalCall,
	f3 EvalAction) *Evaluator {

	return &Evaluator{
		Stack:      nil,
		Session:    nil,
		LoadVarFn:  f0,
		StoreVarFn: f1,
		CallFn:     f2,
		ActionFn:   f3,
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

// Generate a human readable backtrace for reporting errors
func (e *Evaluator) backtrace(curframe *program, max int) string {
	sep := "....................."
	var b []string
	idx := 0

	for {
		cf := &e.curframe
		if cf.isTop() {
			break
		}

		b = append(b, fmt.Sprintf("%d>%s\n%s\n%s\n", idx, sep, cf.frameInfo(), sep))
		idx++

		if idx == max {
			b = append(b, ".........\n")
			break
		}

		// pop it up
		e.popfuncframe(e.prevfuncframe())
	}

	return strings.Join(b, "")
}

func (e *Evaluator) doErrf(p *program, pc int, format string, a ...interface{}) error {
	must(e.curframe.prog == p, "must be compatible")
	e.curframe.pc = pc

	dbg := p.dbgList[pc]
	msg := fmt.Sprintf(format, a...)
	return fmt.Errorf("policy(%s), %s has error: %s\n%s",
		p.name, dbg.where(), msg, e.backtrace(p, 10))
}

func (e *Evaluator) doErr(p *program, pc int, err error) error {
	dbg := p.dbgList[pc]
	return fmt.Errorf("policy(%s), %s has error: %s\n%s",
		p.name, dbg.where(), err.Error(), e.backtrace(p, 10))
}

func (e *Evaluator) doEval(event string, pp []*program, policy *Policy) error {
	// just clear the stack size if needed before every run, since we need to reuse
	// this evaluator
	if cap(e.Stack) > 0 {
		e.Stack = e.Stack[:0]
	}

	// Enter into the VM with a native function call marker. This serves as a
	// frame marker to indicate the end of the script frame which will help us
	// to terminate the frame walk

	// -1 means this is a rule inside of the policy instead of a function call
	e.push(NewValInt(-1))
	_, entryV := newfuncframe(
		0,
		nil,
		0,
		0,
	)
	e.push(entryV)

	// VM execution
VM:
	for _, prog := range pp {
		// point to currently executing program, notes the PC field is not updated
		// until the VM breaks, ie when enter into doErr/doErrf
		e.curframe.prog = prog
		e.curframe.pc = 0
		e.curframe.framep = 0
		e.curframe.farg = 0

		// script function entry label, the bcSCall will setup stack layout and
		// jump(goto) this label for rexecution. prog will be swapped with the
		// function program
	FUNC:
		// actual loop to interpret the internal policy, the policy selection code
		// is been fused inside of the bytecode body and terminated by bcMatch bc
		// make sure the function has enough slot for local variable

	MATCH:
		for pc := 0; ; pc++ {
			bc := prog.bcList[pc]

			switch bc.opcode {
			case bcAction:
				actName := prog.idxStr(bc.argument)
				val := e.top0()
				if e.ActionFn == nil {
					return e.doErrf(prog, pc, "action %s is not allowed", actName)
				}
				if err := e.ActionFn(e, actName, val); err != nil {
					return err
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
					return e.doErr(prog, pc, err)
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
					return e.doErr(prog, pc, err)
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

			case bcPop:
				e.pop()
				break

			case bcTernary, bcJfalse:
				cond := e.top0()
				e.pop()
				if !cond.ToBoolean() {
					pc = bc.argument - 1
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

			case bcCall:
				paramSize := bc.argument

				// prepare argument list slice
				argStart := len(e.Stack) - paramSize
				argEnd := len(e.Stack)
				arg := e.Stack[argStart:argEnd]

				funcName := e.topN(paramSize)
				must(funcName.Type == ValStr,
					fmt.Sprintf("function name must be string but %s", funcName.Id()))

				var r Val
				var err error

				if e.CallFn != nil {
					r, err = e.CallFn(e, funcName.String(), arg)
				} else {
					err = e.doErrf(prog, pc, "function %s is not found", funcName.String())
				}
				if err != nil {
					return e.doErr(prog, pc, err)
				}
				e.popN(paramSize + 1)
				e.push(r)
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
					return e.doErr(prog, pc, err)
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
					return e.doErr(prog, pc, err)
				}

				e.popN(paramSize + 2)
				e.push(ret)
				break

			// script function call and return
			case bcSCall:
				paramSize := bc.argument
				funcIndex := e.topN(paramSize)

				// create the frame marker for the scriptting call
				_, newFV := newfuncframe(
					pc+1, // next pc
					prog,
					e.curframe.framep,
					e.curframe.farg,
				)

				// setup new calling frame
				e.push(newFV)

				// enter into the new call
				newCall := policy.fn[int(funcIndex.Int())]
				prog = newCall

				e.curframe.prog = prog
				e.curframe.pc = 0
				e.curframe.framep = len(e.Stack) - paramSize - 2
				e.curframe.farg = paramSize

				goto FUNC

			case bcReturn:
				rv := e.top0()

				prevcf := e.prevfuncframe()

				// return back to the caller
				pc, prog = e.popfuncframe(prevcf)

				// return value push back to the stack
				e.push(rv)
				break

			case bcToStr:
				top := e.top0()
				str, err := top.ToString()
				if err != nil {
					return e.doErr(prog, pc, err)
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
				if e.LoadVarFn == nil {
					return e.doErrf(prog, pc, "dynamic variable: %s is not fonud", vname)
				}
				val, err := e.LoadVarFn(e, vname)
				if err != nil {
					return e.doErr(prog, pc, err)
				}
				e.push(val)
				break

			case bcStoreVar:
				top := e.top0()
				e.pop()
				vname := prog.idxStr(bc.argument)
				if e.StoreVarFn == nil {
					return e.doErrf(prog, pc, "dynamic variable: %s is not fonud", vname)
				}
				if err := e.StoreVarFn(e, vname, top); err != nil {
					return err
				}
				break

			case bcLoadDollar:
				e.push(NewValStr(event))
				break

			case bcIndex:
				ee := e.top1()
				er := e.top0()
				val, err := ee.Index(er)
				if err != nil {
					return e.doErr(prog, pc, err)
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
					return err
				}
				break

			case bcDot:
				ee := e.top0()
				vname := prog.idxStr(bc.argument)
				val, err := ee.Dot(vname)
				if err != nil {
					return e.doErr(prog, pc, err)
				}
				e.pop()
				e.push(val)
				break

			case bcDotSet:
				recv := e.top1()
				value := e.top0()
				e.popN(2)

				if err := recv.DotSet(prog.idxStr(bc.argument), value); err != nil {
					return err
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
					return e.doErr(prog, pc, err)
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
					return e.doErrf(prog, pc, "@session is not executed")
				} else {
					e.push(e.Session[bc.argument])
				}
				break

			case bcStoreSession:
				if len(e.Session) <= bc.argument {
					return e.doErrf(prog, pc, "@session is not executed")
				} else {
					e.Session[bc.argument] = e.top0()
					e.pop()
				}
				break

			case bcMatch:
				c := e.top0()
				if c.ToBoolean() {
					e.pop()
				} else {
					// leave the frame untouched
					e.popTo(e.curframe.framep + 2)
					break MATCH
				}
				break

			case bcHalt:
				break VM

			default:
				log.Fatalf("invalid unknown bytecode %d", bc.opcode)
			}
		}
	}

	return nil
}

func (e *Evaluator) EvalSession(p *Policy) error {
	if p.g == nil {
		return nil
	}
	glb := []*program{p.g}
	e.Session = nil
	return e.doEval("@session", glb, p)
}

func (e *Evaluator) Eval(event string, p *Policy) error {
	return e.doEval(event, p.p, p)
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
	if err := e.doEval("$expression", p.p, p); err != nil {
		return NewValNull(), err
	}
	return e.top0(), nil
}
