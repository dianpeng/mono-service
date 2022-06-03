package pl

import (
	"bytes"
	"fmt"
	"log"
	"math"
)

type Policy struct {
	// can be null
	g *program
	p []*program
}

func (p *Policy) getProgram(name string) *program {
	for _, xx := range p.p {
		if xx.name == name {
			return xx
		}
	}
	return nil
}

func (p *Policy) Dump() string {
	var b bytes.Buffer
	for _, p := range p.p {
		b.WriteString(p.dump())
		b.WriteRune('\n')
	}
	return b.String()
}

// Compile the input string into a Policy object
func CompilePolicy(policy string) (*Policy, error) {
	p := newParser(policy)
	pp, err := p.parse()
	if err != nil {
		return nil, err
	}

	var glb *program

	// find out the session variable
	if len(pp) > 0 {
		p0 := pp[0]
		if p0.name == "@session" {
			glb = p0
			pp = pp[1:]
		}
	}

	return &Policy{
		g: glb,
		p: pp,
	}, nil
}

// Evaluation part of the VM
type EvalLoadVar func(*Evaluator, string) (Val, error)
type EvalCall func(*Evaluator, string, []Val) (Val, error)
type EvalAction func(*Evaluator, string, Val) error

type Evaluator struct {
	Stack     []Val
	Session   []Val
	Local     []Val
	LoadVarFn EvalLoadVar
	CallFn    EvalCall
	ActionFn  EvalAction
}

func NewEvaluatorSimple() *Evaluator {
	return &Evaluator{}
}

func NewEvaluator(
	f0 EvalLoadVar,
	f1 EvalCall,
	f2 EvalAction) *Evaluator {

	return &Evaluator{
		Stack:     nil,
		Session:   nil,
		LoadVarFn: f0,
		CallFn:    f1,
		ActionFn:  f2,
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

func (e *Evaluator) newLocal(size int) {
	if len(e.Local) < size {
		e.Local = make([]Val, size, size)
		for i := 0; i < size; i++ {
			e.Local[i] = NewValNull()
		}
	}
}

func mustReal(x Val) float64 {
	if x.Type == ValInt {
		return float64(x.Int)
	} else {
		return x.Real
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
				return NewValInt64(lhs.Int - rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real - rhs.Real), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) - mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for -")

	case bcMul:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(lhs.Int * rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real * rhs.Real), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) * mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcPow:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(powI(lhs.Int, rhs.Int)), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(math.Pow(lhs.Real, rhs.Real)), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(math.Pow(mustReal(lhs), mustReal(rhs))), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcMod:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				if rhs.Int == 0 {
					return NewValNull(), fmt.Errorf("divide zero")
				}
				return NewValInt64(lhs.Int % rhs.Int), nil
			}
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcDiv:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				if rhs.Int == 0 {
					return NewValNull(), fmt.Errorf("divide zero")
				}
				return NewValInt64(lhs.Int / rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real / rhs.Real), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValReal(mustReal(lhs) / mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for *")

	case bcAdd:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValInt64(lhs.Int + rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValReal(lhs.Real + rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValStr(lhs.String + rhs.String), nil
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
				return NewValBool(lhs.Int == rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real == rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String == rhs.String), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) == mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for ==")

	case bcNe:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int != rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real != rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String != rhs.String), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) != mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for !=")

	case bcLt:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int < rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real < rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String < rhs.String), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) < mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for <")

	case bcLe:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int <= rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real <= rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String <= rhs.String), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) <= mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for <=")

	case bcGt:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int > rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real > rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String > rhs.String), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) > mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for >")

	case bcGe:
		if lhs.Type == rhs.Type {
			if lhs.Type == ValInt {
				return NewValBool(lhs.Int >= rhs.Int), nil
			}
			if lhs.Type == ValReal {
				return NewValBool(lhs.Real >= rhs.Real), nil
			}
			if lhs.Type == ValStr {
				return NewValBool(lhs.String >= rhs.String), nil
			}
		} else if lhs.IsNumber() && rhs.IsNumber() {
			return NewValBool(mustReal(lhs) >= mustReal(rhs)), nil
		}
		return NewValNull(), fmt.Errorf("invalid operand for >=")

	case bcRegexpMatch:
		if lhs.Type == ValStr && rhs.Type == ValRegexp {
			r := rhs.Regexp.Match([]byte(lhs.String))
			return NewValBool(r), nil
		} else {
			return NewValNull(), fmt.Errorf("regexp operator ~ must be applied on string and regexp")
		}

	case bcRegexpNMatch:
		if lhs.Type == ValStr && rhs.Type == ValRegexp {
			r := rhs.Regexp.Match([]byte(lhs.String))
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
		return NewValInt64(-v.Int), nil
	case ValReal:
		return NewValReal(-v.Real), nil
	default:
		return NewValNull(), fmt.Errorf("invalid operand for !")
	}
}

func (e *Evaluator) builtinCallRegexpMatch(args []Val) (Val, error) {
	if len(args) != 2 {
		return NewValNull(), fmt.Errorf("regexp_match's argument must be 2")
	}
	if args[0].Type != ValStr || args[1].Type != ValRegexp {
		return NewValNull(), fmt.Errorf("regexp_match's argument's type invalid")
	}

	r := args[1].Regexp.Match([]byte(args[0].String))
	return NewValBool(r), nil
}

func (e *Evaluator) builtinCallRegexpContain(args []Val) (Val, error) {
	if len(args) != 2 {
		return NewValNull(), fmt.Errorf("regexp_contain's argument must be 2")
	}
	if args[0].Type != ValStr || args[1].Type != ValRegexp {
		return NewValNull(), fmt.Errorf("regexp_contain's argument's type invalid")
	}

	r := args[1].Regexp.MatchString(args[0].String)
	return NewValBool(r), nil
}

func (e *Evaluator) builtinCall(name string, args []Val) (Val, error, bool) {
	switch name {
	case "regexp_match":
		a, b := e.builtinCallRegexpMatch(args)
		return a, b, true

	case "regexp_contain":
		a, b := e.builtinCallRegexpContain(args)
		return a, b, true

	default:
		break
	}

	return NewValNull(), nil, false
}

func (e *Evaluator) doEval(event string, pp []*program) error {

	// VM execution
VM:
	for _, prog := range pp {
		e.newLocal(prog.localSize)

		// actual loop to interpret the internal policy, the policy selection code
		// is been fused inside of the bytecode body and terminated by bcMatch bc
	MATCH:
		for pc := 0; ; pc++ {
			bc := prog.bcList[pc]

			switch bc.opcode {
			case bcAction:
				actName := prog.idxStr(bc.argument)
				val := e.top0()
				if e.ActionFn == nil {
					return fmt.Errorf("action callback is not set")
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
					return err
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
					return err
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
					m.AddMap(name.String, val)
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

				r, err, found := e.builtinCall(funcName.String, arg)
				if !found {
					if e.CallFn == nil {
						return fmt.Errorf("call callback is not set")
					}
					r, err = e.CallFn(e, funcName.String, arg)
				}
				if err != nil {
					return err
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
				ret, err := recv.Method(methodName.String, arg)
				if err != nil {
					return err
				}

				e.popN(paramSize + 2)
				e.push(ret)
				break

			case bcToStr:
				top := e.top0()
				str, err := top.ToString()
				if err != nil {
					return err
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
					b.WriteString(v.String)
				}
				e.popN(sz)
				e.push(NewValStr(b.String()))
				break

			case bcLoadVar:
				vname := prog.idxStr(bc.argument)
				if e.LoadVarFn == nil {
					return fmt.Errorf("load_var callback is not set")
				}
				val, err := e.LoadVarFn(e, vname)
				if err != nil {
					return err
				}
				e.push(val)
				break

			case bcLoadDollar:
				e.push(NewValStr(event))
				break

			case bcIndex:
				ee := e.top1()
				er := e.top0()
				val, err := ee.Index(er)
				if err != nil {
					return err
				}
				e.popN(2)
				e.push(val)
				break

			case bcDot:
				ee := e.top0()
				vname := prog.idxStr(bc.argument)
				val, err := ee.Dot(NewValStr(vname))
				if err != nil {
					return err
				}
				e.pop()
				e.push(val)
				break

			case bcStoreLocal:
				e.Local[bc.argument] = e.top0()
				e.pop()
				break

			case bcLoadLocal:
				e.push(e.Local[bc.argument])
				break

			// special functions, ie intrinsic
			case bcTemplate:
				ctx := e.top0()
				e.pop()
				tmp := prog.idxTemplate(bc.argument)
				data, err := tmp.Execute(ctx)
				if err != nil {
					return err
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
					return fmt.Errorf("@session is not executed, or not initialized")
				} else {
					e.push(e.Session[bc.argument])
				}
				break

			case bcStoreSession:
				if len(e.Session) <= bc.argument {
					return fmt.Errorf("@session is not executed, or not initialized")
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
	return e.doEval("@session", glb)
}

func (e *Evaluator) Eval(event string, p *Policy) error {
	return e.doEval(event, p.p)
}
