package hpl

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"regexp"
)

// sum type of the evaluator
const (
	ValInt = iota
	ValReal
	ValStr
	ValBool
	ValNull
	ValPair
	ValList
	ValMap
	ValRegexp

	// special one, cannot be interpreted unless user allows us to do so
	ValUsr
)

type UValIndex func(interface{}, Val) (Val, error)
type UValDot func(interface{}, string) (Val, error)
type UValToString func(interface{}) (string, error)
type UValToJSON func(interface{}) (string, error)

// returns an unique id to represent the type of the user defined type
type UValId func(interface{}) string

type UVal struct {
	Context    interface{}
	IndexFn    UValIndex
	DotFn      UValDot
	ToStringFn UValToString
	ToJSONFn   UValToJSON
	IdFn       UValId
}

type List struct {
	Data []Val
}

func NewList() *List {
	return &List{
		Data: make([]Val, 0, 0),
	}
}

type Map struct {
	Data map[string]Val
}

func NewMap() *Map {
	return &Map{
		Data: make(map[string]Val),
	}
}

type Pair struct {
	First  Val
	Second Val
}

type Val struct {
	Type   int
	Int    int64
	Real   float64
	Bool   bool
	String string
	Regexp *regexp.Regexp
	Pair   *Pair
	List   *List
	Map    *Map
	Usr    *UVal
}

func NewValNull() Val {
	return Val{
		Type: ValNull,
	}
}

func NewValInt64(i int64) Val {
	return Val{
		Type: ValInt,
		Int:  i,
	}
}

func NewValInt(i int) Val {
	return Val{
		Type: ValInt,
		Int:  int64(i),
	}
}

func NewValStr(s string) Val {
	return Val{
		Type:   ValStr,
		String: s,
	}
}

func NewValReal(d float64) Val {
	return Val{
		Type: ValReal,
		Real: d,
	}
}

func NewValBool(b bool) Val {
	return Val{
		Type: ValBool,
		Bool: b,
	}
}

func NewValPair(f Val, s Val) Val {
	return Val{
		Type: ValPair,
		Pair: &Pair{
			First:  f,
			Second: s,
		},
	}
}

func NewValRegexp(r *regexp.Regexp) Val {
	return Val{
		Type:   ValRegexp,
		Regexp: r,
	}
}

func NewValList() Val {
	return Val{
		Type: ValList,
		List: NewList(),
	}
}

func NewValMap() Val {
	return Val{
		Type: ValMap,
		Map:  NewMap(),
	}
}

func NewValUsr(
	c interface{},
	f0 UValIndex,
	f1 UValDot,
	f2 UValToString,
	f3 UValToJSON,
	f4 UValId) Val {
	return Val{
		Type: ValUsr,
		Usr: &UVal{
			Context:    c,
			IndexFn:    f0,
			DotFn:      f1,
			ToStringFn: f2,
			ToJSONFn:   f3,
			IdFn:       f4,
		},
	}
}

func (v *Val) IsNumber() bool {
	switch v.Type {
	case ValInt, ValReal:
		return true
	default:
		return false
	}
}

func (v *Val) AddList(vv Val) {
	must(v.Type == ValList, "AddList: must be list")
	v.List.Data = append(v.List.Data, vv)
}

func (v *Val) AddMap(key string, val Val) {
	must(v.Type == ValMap, "AddMap: must be map")
	v.Map.Data[key] = val
}

// never failed
func (v *Val) ToBoolean() bool {
	switch v.Type {
	case ValInt:
		return v.Int != 0
	case ValReal:
		return v.Real != 0
	case ValStr:
		return len(v.String) != 0
	case ValBool:
		return v.Bool
	case ValNull:
		return true
	case ValList:
		return len(v.List.Data) != 0
	case ValMap:
		return len(v.Map.Data) != 0
	default:
		return false
	}
}

// convert whatever hold in boxed value back to native go type to be used in
// external go envronment. Notes, for user type we just do nothing for now,
// ie just returns a string.
func (v *Val) ToNative() interface{} {
	switch v.Type {
	case ValInt:
		return v.Int
	case ValReal:
		return v.Real
	case ValStr:
		return v.String
	case ValBool:
		return v.Bool
	case ValNull:
		return nil
	case ValList:
		var x []interface{}
		for _, xx := range v.List.Data {
			x = append(x, xx.ToNative())
		}
		return x
	case ValMap:
		x := make(map[string]interface{})
		for key, val := range v.Map.Data {
			x[key] = val.ToNative()
		}
		return x
	case ValPair:
		return [2]interface{}{
			v.Pair.First.ToNative(),
			v.Pair.Second.ToNative(),
		}
	case ValRegexp:
		return fmt.Sprintf("[Regexp: %s]", v.Regexp.String())

	default:
		return fmt.Sprintf("[User: %s]", v.Id())
	}
}

func (v *Val) ToIndex() (int, error) {
	switch v.Type {
	case ValInt:
		if v.Int >= 0 {
			return int(v.Int), nil
		}
		return 0, fmt.Errorf("negative value cannot be index")

	default:
		return 0, fmt.Errorf("none integer type cannot be index")
	}
}

func (v *Val) ToString() (string, error) {
	switch v.Type {
	case ValInt:
		return fmt.Sprintf("%d", v.Int), nil
	case ValReal:
		return fmt.Sprintf("%f", v.Real), nil
	case ValBool:
		if v.Bool {
			return "true", nil
		} else {
			return "false", nil
		}
	case ValNull:
		return "", fmt.Errorf("cannot convert Null to string")
	case ValStr:
		return v.String, nil

	case ValRegexp:
		return v.Regexp.String(), nil

	case ValList, ValMap, ValPair:
		return "", fmt.Errorf("cannot convert List/Map/Pair to string")

	default:
		return v.Usr.ToStringFn(v.Usr.Context)
	}
	return "", nil
}

func (v *Val) Index(idx Val) (Val, error) {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull:
		return NewValNull(), fmt.Errorf("cannot index primitive type")

	case ValRegexp:
		return NewValNull(), fmt.Errorf("cannot index regexp")

	case ValStr:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i >= len(v.String) {
			return NewValNull(), fmt.Errorf("index out of range")
		}
		return NewValStr(v.String[i : i+1]), nil

	case ValPair:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i == 0 {
			return v.Pair.First, nil
		}
		if i == 1 {
			return v.Pair.Second, nil
		}

		return NewValNull(), fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")

	case ValList:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i >= len(v.List.Data) {
			return NewValNull(), fmt.Errorf("index out of range")
		}
		return v.List.Data[i], nil

	case ValMap:
		i, err := idx.ToString()
		if err != nil {
			return NewValNull(), err
		}
		if vv, ok := v.Map.Data[i]; !ok {
			return NewValNull(), fmt.Errorf("%s key not found", i)
		} else {
			return vv, nil
		}

	default:
		// User
		return v.Usr.IndexFn(v.Usr.Context, idx)
	}
}

func (v *Val) Dot(idx Val) (Val, error) {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValStr, ValList:
		return NewValNull(), fmt.Errorf("cannot apply dot operator on none Map or User type")

	case ValRegexp:
		return NewValNull(), fmt.Errorf("cannot apply dot operator on regexp")

	case ValMap:
		i, err := idx.ToString()
		if err != nil {
			return NewValNull(), err
		}
		if vv, ok := v.Map.Data[i]; !ok {
			return NewValNull(), fmt.Errorf("%s key not found", i)
		} else {
			return vv, nil
		}

	case ValPair:
		i, err := idx.ToString()
		if err != nil {
			return NewValNull(), err
		}
		if i == "first" {
			return v.Pair.First, nil
		}
		if i == "second" {
			return v.Pair.Second, nil
		}

		return NewValNull(), fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")

	default:
		i, err := idx.ToString()
		if err != nil {
			return NewValNull(), err
		}
		return v.Usr.DotFn(v.Usr.Context, i)
	}
}

func (v *Val) Id() string {
	switch v.Type {
	case ValInt:
		return "int"
	case ValReal:
		return "real"
	case ValBool:
		return "bool"
	case ValNull:
		return "null"
	case ValStr:
		return "str"
	case ValList:
		return "list"
	case ValMap:
		return "map"
	case ValPair:
		return "pair"
	case ValRegexp:
		return "regexp"
	default:
		return v.Usr.IdFn(v.Usr.Context)
	}
}

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

	// find out the global
	if len(pp) > 0 {
		p0 := pp[0]
		if p0.name == "@global" {
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
	Global    []Val
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
		Global:    nil,
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

			// globals
			case bcSetGlobal:
				ctx := e.top0()
				e.pop()
				e.Global = append(e.Global, ctx)
				break

			case bcLoadGlobal:
				if len(e.Global) <= bc.argument {
					return fmt.Errorf("@global is not executed, or not initialized")
				} else {
					e.push(e.Global[bc.argument])
				}
				break

			case bcStoreGlobal:
				if len(e.Global) <= bc.argument {
					return fmt.Errorf("@global is not executed, or not initialized")
				} else {
					e.Global[bc.argument] = e.top0()
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

func (e *Evaluator) EvalGlobal(p *Policy) error {
	if p.g == nil {
		return nil
	}
	glb := []*program{p.g}
	e.Global = nil
	return e.doEval("@global", glb)
}

func (e *Evaluator) Eval(event string, p *Policy) error {
	return e.doEval(event, p.p)
}
