package pl

import (
	"bytes"
	"fmt"
	"regexp"
)

const (
	bcAction = 0

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
	bcLoadLocal  = 21
	bcStoreLocal = 22

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

	bcJfalse  = 52
	bcAnd     = 53
	bcOr      = 54
	bcTernary = 55
	bcJump    = 56

	// stack manipulation
	bcPop = 70

	// this instruction can only be generated during the global scope
	bcSetSession = 71

	// this 2 instructions are used during normal policy execution
	bcLoadSession  = 72
	bcStoreSession = 73

	// call
	bcCall  = 81
	bcMCall = 82

	// special functions
	// render template
	bcTemplate = 100

	// extensions
	bcLoadRegexp = 200

	// halt the machine
	bcMatch = 254
	bcHalt  = 255
)

type bytecode struct {
	opcode   int
	argument int
}

type bytecodeList []bytecode

type program struct {
	name       string
	localSize  int
	tbReal     []float64
	tbInt      []int64
	tbStr      []string
	tbTemplate []Template
	tbRegexp   []*regexp.Regexp

	// used for actual interpretation
	bcList bytecodeList
}

func newProgram(n string) *program {
	return &program{
		name: n,
	}
}

func newProgramEmpty(n string) *program {
	pp := newProgram(n)
	pp.bcList = append(pp.bcList, bytecode{
		opcode:   bcHalt,
		argument: 0,
	})
	return pp
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

func (p *program) emit0(opcode int) {
	p.bcList = append(p.bcList, bytecode{
		opcode: opcode,
	})
}

func (p *program) emit1(opcode int, argument int) {
	p.bcList = append(p.bcList, bytecode{
		opcode:   opcode,
		argument: argument,
	})
}

func (p *program) popLast() bytecode {
	must(len(p.bcList) > 0, "not empty")
	sz := len(p.bcList)
	last := p.bcList[sz-1]
	// pop the last instruction
	p.bcList = p.bcList[:sz-1]
	return last
}

// patching
func (p *program) label() int {
	return len(p.bcList)
}

func (p *program) patch() int {
	idx := len(p.bcList)
	p.bcList = append(p.bcList, bytecode{})
	return idx
}

func (p *program) emit0At(idx int, opcode int) {
	p.bcList[idx] = bytecode{
		opcode: opcode,
	}
}

func (p *program) emit1At(idx int, opcode int, argument int) {
	p.bcList[idx] = bytecode{
		opcode:   opcode,
		argument: argument,
	}
}

func (p *program) dump() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf(":program (%s)\n", p.name))

	for idx, x := range p.bcList {
		name := getBytecodeName(x.opcode)
		arg := x.argument
		switch x.opcode {
		case bcLoadInt:
			b.WriteString(fmt.Sprintf("%d. %s(%d:%d)", idx+1, name, arg, p.idxInt(arg)))
			break

		case bcLoadReal:
			b.WriteString(fmt.Sprintf("%d. %s(%d:%f)", idx+1, name, arg, p.idxReal(arg)))
			break

		case bcLoadStr, bcAction, bcDot:
			b.WriteString(fmt.Sprintf("%d. %s(%d:%s)", idx+1, name, arg, p.idxStr(arg)))
			break

		case bcLoadRegexp:
			b.WriteString(fmt.Sprintf("%d. %s(%d:%s)", idx+1, name, arg, p.idxRegexp(arg).String()))

		case bcAddList,
			bcAddMap,
			bcCall,
			bcMCall,
			bcConStr,
			bcLoadLocal,
			bcStoreLocal,
			bcOr,
			bcAnd,
			bcJfalse,
			bcJump,
			bcTernary:

			b.WriteString(fmt.Sprintf("%d. %s(%d)", idx+1, name, arg))
			break

		default:
			b.WriteString(fmt.Sprintf("%d. %s", idx+1, name))
		}
		b.WriteRune('\n')
	}
	return b.String()
}

func getBytecodeName(bc int) string {
	switch bc {
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
	case bcCall:
		return "call"
	case bcMCall:
		return "mcall"
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
	case bcJump:
		return "jump"
	case bcAnd:
		return "and"
	case bcOr:
		return "or"
	case bcTernary:
		return "ternary"
	case bcPop:
		return "pop"
	case bcSetSession:
		return "set-session"
	case bcLoadSession:
		return "load-session"
	case bcStoreSession:
		return "store-session"
	case bcLoadRegexp:
		return "load-regexp"

	// special functions
	case bcTemplate:
		return "template"

	case bcMatch:
		return "match"
	case bcHalt:
		return "halt"
	default:
		return fmt.Sprintf("<unknown: %d>", bc)
	}
}
