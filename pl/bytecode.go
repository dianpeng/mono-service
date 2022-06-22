package pl

import (
	"bytes"
	"fmt"
	"regexp"
)

const (
	// specifically leave this as a marker to indicate bug that we have during
	// code generation, ie patch code is not filled up
	bcPatch = 0

	// loading a constant into the expression stack
	bcLoadInt   = 1
	bcLoadReal  = 2
	bcLoadStr   = 3
	bcLoadTrue  = 4
	bcLoadFalse = 5
	bcLoadNull  = 6

	// new list
	bcNewList = 7
	bcAddList = 8

	// new map
	bcNewMap = 9
	bcAddMap = 10

	// new pair
	bcNewPair = 11

	// string interpolation
	bcToStr  = 12
	bcConStr = 13

	bcStoreVar   = 14
	bcLoadVar    = 15
	bcDot        = 16
	bcDotSet     = 17
	bcIndex      = 18
	bcIndexSet   = 19
	bcLoadDollar = 20

	// local variable, we use seperate stack to load and store local variables
	// notes, the local variable is been identified without $
	bcLoadLocal    = 21
	bcStoreLocal   = 22
	bcReserveLocal = 23
	bcLoadRegexp   = 24

	bcAction = 30

	// expression
	bcAdd          = 31
	bcSub          = 32
	bcMul          = 33
	bcDiv          = 34
	bcPow          = 35
	bcLt           = 36
	bcLe           = 37
	bcGt           = 38
	bcGe           = 39
	bcEq           = 40
	bcNe           = 41
	bcNot          = 42
	bcNegate       = 43
	bcRegexpMatch  = 44
	bcRegexpNMatch = 45
	bcMod          = 46

	// jump
	// jump is mainly used during conditional expression evaluation, which includes
	// logic and ternary
	//
	// 1. (cond1) || (cond2) : cond1 true --> jump next
	// 2. (cond1) && (cond2) : cond1 false --> jump next

	bcJtrue   = 51
	bcJfalse  = 52
	bcAnd     = 53
	bcOr      = 54
	bcTernary = 55
	bcJump    = 56

	// stack manipulation
	bcSwap = 61
	bcDup1 = 62
	bcDup2 = 63
	bcPop  = 64

	// this instruction can only be generated during the global scope
	bcSetSession = 71
	// this 2 instructions are used during normal policy execution
	bcLoadSession  = 72
	bcStoreSession = 73

	bcSetConst  = 75
	bcLoadConst = 76

	// call
	bcMCall = 82
	bcICall = 83
	bcSCall = 84
	bcVCall = 85 // variable call, ie calling a function that is loaded into
	// a dynamic variable which cannot be resolved
	bcReturn = 89

	// iterator
	bcNewIterator   = 90
	bcHasIterator   = 91
	bcDerefIterator = 92
	bcNextIterator  = 93

	// uvpalue/closure
	bcNewClosure   = 95
	bcLoadUpvalue  = 96
	bcStoreUpvalue = 97

	// exception
	bcPushException = 101
	bcPopException  = 102

	// load the exception object/reason(currently should be just a string) into
	// tos
	bcLoadException = 103

	// config extension part ---------------------------------------------------
	bcConfigPush         = 151
	bcConfigPushWithAttr = 152
	bcConfigPop          = 153

	bcConfigPropertySet         = 155
	bcConfigPropertySetWithAttr = 156

	bcConfigCommand         = 157
	bcConfigCommandWithAttr = 158

	// intrinsic ---------------------------------------------------------------
	// render template
	bcTemplate = 200

	// halt the machine
	bcHalt = 255
)

type bytecode struct {
	opcode   int
	argument int
}

type sourceloc struct {
	source string
	offset int
	line   int
	column int
}

func (s *sourceloc) where() string {
	var start, end int
	if s.offset >= 32 {
		start = s.offset - 32
	} else {
		start = 0
	}

	if s.offset+32 < len(s.source) {
		end = s.offset + 32
	} else {
		end = len(s.source)
	}

	return fmt.Sprintf("around (%d, %d)@(---\n%s\n---)", s.line, s.column, s.source[start:end])
}

type bytecodeList []bytecode
type sourcelocList []sourceloc

const (
	progRule = iota
	progFunc
	progSession
	progExpression
	progConfig
)

type upvalue struct {
	index   int
	onStack bool
}

type program struct {
	policy    *Policy
	name      string
	localSize int
	argSize   int // if this program is a function call, then this indicates
	// size of the arguments required for calling the function

	tbReal     []float64
	tbInt      []int64
	tbStr      []string
	tbTemplate []Template
	tbRegexp   []*regexp.Regexp

	// used for actual interpretation
	bcList   bytecodeList
	dbgList  sourcelocList
	progtype int

	// used when the program is a function, ie for capturing its upvalue
	upvalue []upvalue
}

func newProgram(p *Policy, n string, t int) *program {
	return &program{
		policy:   p,
		name:     n,
		progtype: t,
	}
}

func newProgramEmpty(p *Policy, n string, t int) *program {
	pp := newProgram(p, n, t)
	pp.bcList = append(pp.bcList, bytecode{
		opcode:   bcHalt,
		argument: 0,
	})
	return pp
}

func (p *program) freeCall() bool {
	return len(p.upvalue) == 0
}

func (p *program) upvalueSize() int {
	return len(p.upvalue)
}

// constant table manipulation
func (p *program) addInt(i int64) int {
	for idx, xx := range p.tbInt {
		if xx == i {
			return int(idx)
		}
	}
	idx := len(p.tbInt)
	p.tbInt = append(p.tbInt, i)
	return int(idx)
}

func (p *program) addReal(i float64) int {
	for idx, xx := range p.tbReal {
		if xx == i {
			return int(idx)
		}
	}
	idx := len(p.tbReal)
	p.tbReal = append(p.tbReal, i)
	return int(idx)
}

func (p *program) addStr(i string) int {
	for idx, xx := range p.tbStr {
		if xx == i {
			return int(idx)
		}
	}
	idx := len(p.tbStr)
	p.tbStr = append(p.tbStr, i)
	return int(idx)
}

func (p *program) idxInt(i int) int64 {
	must(i < len(p.tbInt), "invalid index(int)")
	return p.tbInt[i]
}

func (p *program) idxReal(i int) float64 {
	must(i < len(p.tbReal), "invalid index(real)")
	return p.tbReal[i]
}

func (p *program) idxStr(i int) string {
	must(i < len(p.tbStr), "invalid index(str)")
	return p.tbStr[i]
}

func (p *program) idxTemplate(i int) Template {
	must(i < len(p.tbTemplate), "invalid index(template)")
	return p.tbTemplate[i]
}

func (p *program) idxRegexp(i int) *regexp.Regexp {
	must(i < len(p.tbRegexp), "invalid index(regexp)")
	return p.tbRegexp[i]
}

func (p *program) addTemplate(t string, c string, opt Val) (int, error) {
	temp := newTemplate(t)
	if temp == nil {
		return -1, fmt.Errorf("unsupported template type %s", t)
	}

	idx := len(p.tbTemplate)
	err := temp.Compile(fmt.Sprintf("temp-%d", idx), c, opt)
	if err != nil {
		return 0, err
	}
	p.tbTemplate = append(p.tbTemplate, temp)
	return idx, nil
}

func (p *program) addRegexp(r string) (int, error) {
	rexp, err := regexp.Compile(r)
	if err != nil {
		return 0, err
	}
	idx := len(p.tbRegexp)
	p.tbRegexp = append(p.tbRegexp, rexp)
	return idx, nil
}

func (p *program) emit0(l *lexer, opcode int) {
	p.bcList = append(p.bcList, bytecode{
		opcode: opcode,
	})
	p.dbgList = append(p.dbgList, l.dbg())
}

func (p *program) emit1(l *lexer, opcode int, argument int) {
	p.bcList = append(p.bcList, bytecode{
		opcode:   opcode,
		argument: argument,
	})
	p.dbgList = append(p.dbgList, l.dbg())
}

func (p *program) popLast() bytecode {
	must(len(p.bcList) > 0, "not empty")
	sz := len(p.bcList)
	last := p.bcList[sz-1]
	// pop the last instruction
	p.bcList = p.bcList[:sz-1]
	p.dbgList = p.dbgList[:sz-1]

	return last
}

// patching
func (p *program) label() int {
	return len(p.bcList)
}

func (p *program) patch(l *lexer) int {
	idx := len(p.bcList)
	p.bcList = append(p.bcList, bytecode{})
	p.dbgList = append(p.dbgList, l.dbg())
	return idx
}

func (p *program) emit0At(_ *lexer, idx int, opcode int) {
	p.bcList[idx] = bytecode{
		opcode: opcode,
	}
}

func (p *program) emit1At(_ *lexer, idx int, opcode int, argument int) {
	p.bcList[idx] = bytecode{
		opcode:   opcode,
		argument: argument,
	}
}

func (p *program) dump() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf(":program (%s)\n", p.name))
	for idx, x := range p.bcList {
		b.WriteString(fmt.Sprintf("%d> %s\n", idx, x.dumpWithProgram(p)))
	}
	return b.String()
}

func (p *program) typeName() string {
	switch p.progtype {
	case progRule:
		return "rule"
	case progFunc:
		return "function"
	case progSession:
		return "session"
	default:
		return ""
	}
}

func (x *bytecode) dumpWithProgram(p *program) string {
	resolver := func(op int, arg int) string {
		switch op {
		case bcLoadReal:
			return fmt.Sprintf("%f", p.idxReal(arg))
		case bcLoadStr, bcAction, bcDot:
			return p.idxStr(arg)
		case bcLoadRegexp:
			return p.idxRegexp(arg).String()
		case bcTemplate:
			return "[template]"
		default:
			return "<unknown>"
		}
	}
	return x.dump(resolver)
}

func (x *bytecode) dump(resolver func(int, int) string) string {
	var b bytes.Buffer
	name := getBytecodeName(x.opcode)
	arg := x.argument

	wrapper := func(bc int, arg int) string {
		if resolver != nil {
			return fmt.Sprintf(":%s", resolver(bc, arg))
		} else {
			return ""
		}
	}

	switch x.opcode {

	case bcLoadInt,
		bcLoadReal,
		bcLoadStr,
		bcAction,
		bcDot,
		bcLoadRegexp,
		bcTemplate:

		b.WriteString(fmt.Sprintf("%s(%d%s)", name, arg, wrapper(x.opcode, arg)))
		break

	case bcAddList,
		bcAddMap,
		bcICall,
		bcMCall,
		bcSCall,
		bcReturn,
		bcConStr,

		bcLoadLocal,
		bcStoreLocal,
		bcReserveLocal,

		bcLoadVar,
		bcStoreVar,
		bcDotSet,

		bcAnd,
		bcOr,
		bcJfalse,
		bcJtrue,
		bcJump,
		bcTernary:

		b.WriteString(fmt.Sprintf("%s(%d)", name, arg))
		break

	default:
		b.WriteString(fmt.Sprintf("%s[%d]", name, arg))
	}

	return b.String()
}

func getBytecodeName(bc int) string {
	switch bc {
	case bcPatch:
		return "<patch>"
	case bcAction:
		return "action"
	case bcLoadInt:
		return "load-int"
	case bcLoadReal:
		return "load-real"
	case bcLoadStr:
		return "load-str"
	case bcLoadTrue:
		return "load-true"
	case bcLoadFalse:
		return "load-false"
	case bcLoadNull:
		return "load-null"
	case bcNewList:
		return "new-list"
	case bcAddList:
		return "add-list"
	case bcNewMap:
		return "new-map"
	case bcAddMap:
		return "add-map"
	case bcNewPair:
		return "new-pair"
	case bcICall:
		return "icall"
	case bcMCall:
		return "mcall"
	case bcSCall:
		return "scall"
	case bcReturn:
		return "return"
	case bcNewIterator:
		return "new-iterator"
	case bcHasIterator:
		return "has-iterator"
	case bcDerefIterator:
		return "deref-iterator"
	case bcNextIterator:
		return "next-iterator"
	case bcToStr:
		return "to-str"
	case bcConStr:
		return "con-str"
	case bcLoadVar:
		return "load-var"
	case bcStoreVar:
		return "store-var"
	case bcDot:
		return "dot"
	case bcIndex:
		return "index"
	case bcLoadDollar:
		return "load-dollar"
	case bcLoadLocal:
		return "load-local"
	case bcStoreLocal:
		return "store-local"
	case bcReserveLocal:
		return "reserve-local"
	case bcAdd:
		return "add"
	case bcSub:
		return "sub"
	case bcMul:
		return "mul"
	case bcDiv:
		return "div"
	case bcMod:
		return "mod"
	case bcPow:
		return "pow"
	case bcLt:
		return "lt"
	case bcLe:
		return "le"
	case bcGt:
		return "gt"
	case bcGe:
		return "ge"
	case bcEq:
		return "eq"
	case bcNe:
		return "ne"
	case bcRegexpMatch:
		return "regexp-match"
	case bcRegexpNMatch:
		return "regexp-nmatch"
	case bcNot:
		return "not"
	case bcNegate:
		return "negate"
	case bcJfalse:
		return "jfalse"
	case bcJtrue:
		return "jtrue"
	case bcJump:
		return "jump"
	case bcAnd:
		return "and"
	case bcOr:
		return "or"
	case bcTernary:
		return "ternary"
	case bcSwap:
		return "swap"
	case bcPop:
		return "pop"
	case bcDup1:
		return "dup1"
	case bcDup2:
		return "dup2"
	case bcSetSession:
		return "set-session"
	case bcLoadSession:
		return "load-session"
	case bcStoreSession:
		return "store-session"
	case bcSetConst:
		return "set-const"
	case bcLoadConst:
		return "load-const"
	case bcLoadRegexp:
		return "load-regexp"
	case bcPushException:
		return "push-exception"
	case bcPopException:
		return "pop-exception"
	case bcLoadException:
		return "load-exception"

	// config
	case bcConfigPush:
		return "config-push"
	case bcConfigPushWithAttr:
		return "config-push-with-attr"
	case bcConfigPop:
		return "config-pop"
	case bcConfigPropertySet:
		return "config-property-set"
	case bcConfigPropertySetWithAttr:
		return "config-property-set-with-attr"
	case bcConfigCommand:
		return "config-command"
	case bcConfigCommandWithAttr:
		return "config-command-with-attr"

		// closure
	case bcNewClosure:
		return "new-closure"
	case bcStoreUpvalue:
		return "store-upvalue"
	case bcLoadUpvalue:
		return "load-upvalue"

	// special functions
	case bcTemplate:
		return "template"

	case bcHalt:
		return "halt"
	default:
		return fmt.Sprintf("<unknown: %d>", bc)
	}
}
