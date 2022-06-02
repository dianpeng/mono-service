package pl

import (
	"bytes"
	"fmt"
	"github.com/dianpeng/mono-service/util"
)

type symTable struct {
	tbl []string
}

func newSymTable() *symTable {
	return &symTable{}
}

func (s *symTable) localSize() int {
	if s.tbl == nil {
		return 0
	}
	return len(s.tbl)
}

// if not found, returns -1, otherwise the symbol's index will be returned
func (s *symTable) find(x string) int {
	for idx, xx := range s.tbl {
		if xx == x {
			return idx
		}
	}
	return -1
}

func (s *symTable) add(x string) int {
	idx := s.find(x)
	if idx != -1 {
		return idx
	}
	s.tbl = append(s.tbl, x)
	return len(s.tbl) - 1
}

type parser struct {
	l    *lexer
	pmap []*program
	stbl *symTable
	glb  []string
}

func newParser(input string) *parser {
	return &parser{
		l: newLexer(input),
	}
}

func (p *parser) err(xx string) error {
	if p.l.token == tkError {
		return p.l.toError()
	} else {
		return fmt.Errorf("%s: %s", p.l.position(), xx)
	}
}

func (p *parser) parse() ([]*program, error) {
	p.l.next()

	// only the first statement can be global and it MUST be defined before any
	// other statement
	if p.l.token == tkGlobal {
		if prog, err := p.parseGlobalScope(); err != nil {
			return nil, err
		} else {
			p.pmap = append(p.pmap, prog)
		}
	}

	for {
		tk := p.l.token
		if tk == tkId || tk == tkStr || tk == tkLSqr {
			prog, err := p.parseProg()
			if err != nil {
				return nil, err
			}
			p.pmap = append(p.pmap, prog)
		} else if tk == tkError {
			return nil, p.l.toError()
		} else if tk != tkEof {
			return nil, p.err("unexpected token, cannot be parsed as policy")
		} else {
			break
		}
	}

	return p.pmap, nil
}

func (p *parser) findGlobalIdx(x string) int {
	for idx, n := range p.glb {
		if n == x {
			return idx
		}
	}
	return -1
}

func (p *parser) parseGlobalSet(prog *program) error {
	must(p.l.token == tkGId, "must be gid")
	gname := p.l.valueText
	idx := p.findGlobalIdx(gname)
	if idx == -1 {
		return p.err(fmt.Sprintf("unknown global %s", gname))
	}
	if !p.l.expect(tkAssign) {
		return p.l.toError()
	}
	p.l.next()
	if err := p.parseExpr(prog); err != nil {
		return err
	}
	prog.emit1(bcStoreGlobal, idx)
	return nil
}

func (p *parser) parseGlobalLoad(prog *program) error {
	must(p.l.token == tkGId, "must be gid")
	gname := p.l.valueText
	idx := p.findGlobalIdx(gname)
	if idx == -1 {
		return p.err(fmt.Sprintf("unknown global %s", gname))
	}
	p.l.next()
	prog.emit1(bcLoadGlobal, idx)
	return p.parseSuffix(prog)
}

func (p *parser) parseGlobalScope() (*program, error) {
	if !p.l.expect(tkLBra) {
		return nil, p.l.toError()
	}
	p.l.next()

	prog := newProgram("@global")

	if p.l.token != tkRBra {
		// all the statement resides inside of it will be treated as global statement
		// and will be allocated into a symbol table globally shared
		for {
			if p.l.token != tkGId && p.l.token != tkId {
				return nil, p.err("expect a symbol to serve as global variable name")
			}
			gname := p.l.valueText

			if !p.l.expect(tkAssign) {
				return nil, p.err("expect a '=' to represent global variable assignment")
			}
			p.l.next()

			if err := p.parseExpr(prog); err != nil {
				return nil, err
			}

			p.glb = append(p.glb, gname)
			prog.emit0(bcSetGlobal)

			if p.l.token == tkSemicolon {
				p.l.next()
			}
			if p.l.token == tkRBra {
				break
			}
		}
	}

	// lastly halt the whole execution
	prog.emit0(bcHalt)

	p.l.next()
	return prog, nil
}

func (p *parser) parseProg() (*program, error) {
	p.stbl = newSymTable()

	var name string

	// parsing the policy name, we accept few different types to ease user's
	// own syntax flavor
	if p.l.token == tkStr || p.l.token == tkId {
		name = p.l.valueText
		p.l.next()
	} else {
		p.l.next()
		if p.l.token != tkStr && p.l.token != tkId {
			return nil, p.err("unexpected token, expect string or identifier for policy name")
		}
		name = p.l.valueText
		if !p.l.expect(tkRSqr) {
			return nil, p.l.toError()
		}
		p.l.next()
	}

	// notes the name must be none empty and also must not start with @, which is
	// builtin event name
	if name == "" {
		return nil, p.err("invalid policy name, cannot be empty string")
	}
	if name[0] == '@' {
		return nil, p.err("invalid policy name, cannot start with @ which is builtin name")
	}

	prog := newProgram(name)

	// check whether we have a conditional when, otherwise generate default match
	// condition
	if p.l.token == tkWhen {
		p.l.next()
		err := p.parseExpr(prog)
		if err != nil {
			return nil, err
		}
		prog.emit0(bcMatch)
	} else {
		// bytecode for compare current event to be the same as the policy name
		// essentially is when event == "policy_name"
		prog.emit0(bcLoadDollar)
		idx := prog.addStr(name)
		prog.emit1(bcLoadStr, idx)
		prog.emit0(bcEq)
		prog.emit0(bcMatch)
	}

	// allow an optional arrow to indicate this is a policy, this is the preferred
	// grammar to indicate the policy definition inside
	if p.l.token == tkArrow {
		p.l.next()
	}

	// now start to parse the body of the policy and start to be awesome
	if p.l.token != tkLPar && p.l.token != tkLBra {
		return nil, p.err("expect a '(' or '{' to start an policy")
	}

	err := p.parseParaList(name, prog)
	if err != nil {
		return nil, err
	}

	if p.l.token == tkSemicolon {
		p.l.next()
	}

	// add a halt regardlessly
	prog.emit0(bcHalt)
	prog.localSize = p.stbl.localSize()
	p.stbl = nil
	return prog, nil
}

func (p *parser) parseParaList(name string, prog *program) error {
	usePar := p.l.token == tkLPar
	ntk := p.l.next()

	if (usePar && ntk == tkRPar) || (!usePar && ntk == tkRBra) {
		p.l.next()
		// empty lists, ie do nothing, just always halt the program
		return nil
	} else {
		for {
			hasSep := false

			switch p.l.token {
			case tkId:
				action := p.l.valueText
				if !p.l.expect(tkAssign) {
					return p.err("expect a '=' after action's name")
				}
				p.l.next()

				if err := p.parseExpr(prog); err != nil {
					return err
				}

				// emit action operations
				{
					idx := prog.addStr(action)
					prog.emit1(bcAction, idx)
				}
				break

			case tkGId:
				if err := p.parseGlobalSet(prog); err != nil {
					return err
				}
				break

			case tkLet:
				if !p.l.expect(tkId) {
					return p.l.toError()
				}

				idx := p.stbl.add(p.l.valueText)

				if !p.l.expect(tkAssign) {
					return p.l.toError()
				}
				p.l.next()

				if err := p.parseExpr(prog); err != nil {
					return err
				}

				prog.emit1(bcStoreLocal, idx)
				break

			case tkIf:
				if err := p.parseBranch(prog); err != nil {
					return err
				}
				prog.emit0(bcPop)

				// allow if to not have a trailing semiclon afterwards, this is really
				// just a hacky way to parse it since we just want to reuse the if
				// expression syntax
				hasSep = true
				break

			case tkDollar:
				// allowing function call directly for simplicity
				if !p.l.expect(tkId) {
					return p.l.toError()
				}
				name := p.l.valueText
				p.l.next()

				if err := p.parseCall(prog, name); err != nil {
					return err
				}

				// statement, clear the stack
				prog.emit0(bcPop)
				break

			default:
				return p.err("expect a identifier to start an action or let to start local variable")
			}

			if p.l.token == tkComma || p.l.token == tkSemicolon {
				hasSep = true
				p.l.next()
			}

			if (usePar && p.l.token == tkRPar) || (!usePar && p.l.token == tkRBra) {
				p.l.next()
				break
			}

			if !hasSep {
				return p.err("expect ','/';' or ')'/'}' after an entry in policy definition")
			}
		}

		return nil
	}
}

// expression parsing, using simple precedence climbing style way

func (p *parser) parseExpr(prog *program) error {
	switch p.l.token {
	case tkIf:
		return p.parseBranch(prog)
	default:
		return p.parseTernary(prog)
	}
}

// a slightly better way to organize conditional expression

// notes, for easier code arrangment, we only allow condition to be ternary
// instead of expression to avoid nested if in if's condition. Since the if
// body has mandatory bracket, it means we can nest if inside and it is still
// fine to maintain

// branch := if elif-list else
// if := IF ternary '{' expr-list '}'
// elif-list := elif*
// elif := ELIF ternary '{' expr-list'}'
// else := ELSE '{' expr-list '}'
// expr-list := expr (';' expr)* ';'?

func (p *parser) parseBranchBody(prog *program) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}
	p.l.next()

	err := p.parseExpr(prog)
	if err != nil {
		return err
	}

	for {
		// if the previous seen expression does not have semicolon then it MUST be
		// an expression instead of statement
		if p.l.token != tkSemicolon {
			break
		}
		p.l.next()

		// okay now it is possible that it is a statement or an expression
		if p.l.token == tkRBra {
			break
		}

		// it must follow an expression, so we should just pop the previous
		// generated expression here
		prog.emit0(bcPop)

		if err := p.parseExpr(prog); err != nil {
			return err
		}
	}

	if !p.l.expectCurrent(tkRBra) {
		return p.l.toError()
	}
	p.l.next()
	return nil
}

func (p *parser) parseBranch(prog *program) error {
	must(p.l.token == tkIf, "must be if")

	var jump_out []int
	var prev_jmp int

	// (0) If

	// skip if
	p.l.next()

	// parse condition
	if err := p.parseTernary(prog); err != nil {
		return err
	}
	prev_jmp = prog.patch()
	if err := p.parseBranchBody(prog); err != nil {
		return err
	}
	jump_out = append(jump_out, prog.patch())

	// (1) Elif
	for {
		if p.l.token != tkElif {
			break
		}
		p.l.next()

		// previous condition failure target position
		prog.emit1At(prev_jmp, bcJfalse, prog.label())

		if err := p.parseExpr(prog); err != nil {
			return err
		}

		prev_jmp = prog.patch()

		if err := p.parseBranchBody(prog); err != nil {
			return err
		}

		jump_out = append(jump_out, prog.patch())
	}

	// (2) Dangling else
	//     notes if there's no else, the else branch can be assumed to return a
	//     null

	prog.emit1At(prev_jmp, bcJfalse, prog.label())
	if p.l.token == tkElse {
		p.l.next()
		if err := p.parseBranchBody(prog); err != nil {
			return err
		}
	} else {
		prog.emit0(bcLoadNull)
	}

	// lastly, patch all the jump
	for _, p := range jump_out {
		prog.emit1At(p, bcJump, prog.label())
	}

	return nil
}

func (p *parser) parseTernary(prog *program) error {
	if err := p.parseBin(prog); err != nil {
		return err
	}
	ntk := p.l.token

	if ntk == tkQuest {
		patch_point := prog.patch()
		p.l.next()

		if err := p.parseBin(prog); err != nil {
			return err
		}

		if !p.l.expectCurrent(tkColon) {
			return p.l.toError()
		}
		p.l.next()

		jump_out := prog.patch()

		// ternary operator, jump when false
		prog.emit1At(patch_point, bcTernary, prog.label())

		if err := p.parseBin(prog); err != nil {
			return err
		}

		// lastly, patch jump out to here
		prog.emit1At(jump_out, bcJump, prog.label())
		return nil
	} else {
		return nil
	}
}

func (p *parser) binPrecedence(tk int) int {
	switch tk {
	case tkOr:
		return 0
	case tkAnd:
		return 1
	case tkEq, tkNe, tkRegexpMatch, tkRegexpNMatch:
		return 2
	case tkLt, tkLe, tkGt, tkGe:
		return 3
	case tkAdd, tkSub:
		return 4
	case tkMul, tkDiv, tkMod, tkPow:
		return 5

	default:
		return -1
	}
}

const maxOperatorPrecedence = 6

func (p *parser) parseBin(prog *program) error {
	return p.parseBinary(prog, 0)
}

func (p *parser) parseBinary(prog *program, prec int) error {
	if prec == maxOperatorPrecedence {
		return p.parseUnary(prog)
	}

	if err := p.parseUnary(prog); err != nil {
		return err
	}

	for {
		tk := p.l.token

		// checking whether the precedence is allowed to
		nextPrec := p.binPrecedence(tk)

		// special case, we cannot deal with it since the next token is not a ops
		if nextPrec == -1 {
			break
		}

		if nextPrec < prec {
			// next precedence is smaller than the one we currently see, so we cannot
			// process it here, and the caller frame has some one who can deal with it
			break
		}

		jump_pos := -1

		switch tk {
		case tkAnd, tkOr:
			jump_pos = prog.patch()
		default:
			break
		}

		p.l.next()

		err := p.parseBinary(prog, nextPrec+1)
		if err != nil {
			return err
		}

		// based on the token, generate bytecode
		switch tk {
		case tkAdd:
			prog.emit0(bcAdd)
			break

		case tkSub:
			prog.emit0(bcSub)
			break

		case tkMul:
			prog.emit0(bcMul)
			break

		case tkDiv:
			prog.emit0(bcDiv)
			break

		case tkMod:
			prog.emit0(bcMod)
			break

		case tkPow:
			prog.emit0(bcPow)
			break

		case tkLt:
			prog.emit0(bcLt)
			break

		case tkLe:
			prog.emit0(bcLe)
			break

		case tkGt:
			prog.emit0(bcGt)
			break

		case tkGe:
			prog.emit0(bcGe)
			break

		case tkEq:
			prog.emit0(bcEq)
			break

		case tkNe:
			prog.emit0(bcNe)
			break

		case tkRegexpMatch:
			prog.emit0(bcRegexpMatch)
			break

		case tkRegexpNMatch:
			prog.emit0(bcRegexpNMatch)
			break

		case tkAnd:
			must(jump_pos >= 0, "and")
			prog.emit1At(jump_pos, bcAnd, prog.label())
			break

		case tkOr:
			must(jump_pos >= 0, "or")
			prog.emit1At(jump_pos, bcOr, prog.label())
			break

		default:
			unreachable("parseBinary")
			break
		}
	}

	return nil
}

func (p *parser) parseUnary(prog *program) error {
	tk := p.l.token
	switch tk {
	case tkAdd, tkSub, tkNot:
		p.l.next()
		if err := p.parseUnary(prog); err != nil {
			return err
		}
		break

	default:
		return p.parseAtomic(prog)
	}

	switch tk {
	case tkSub:
		prog.emit0(bcNegate)
		break

	case tkNot:
		prog.emit0(bcNot)
		break

	default:
		break
	}

	return nil
}

func (p *parser) parseAtomic(prog *program) error {
	tk := p.l.token
	switch tk {
	case tkInt:
		idx := prog.addInt(p.l.valueInt)
		prog.emit1(bcLoadInt, idx)
		p.l.next()
		break

	case tkReal:
		idx := prog.addReal(p.l.valueReal)
		prog.emit1(bcLoadReal, idx)
		p.l.next()
		break

	case tkStr, tkMStr:
		strV := p.l.valueText
		p.l.next()
		return p.parseStrInterpolation(prog, strV)

	case tkTrue:
		prog.emit0(bcLoadTrue)
		p.l.next()
		break

	case tkFalse:
		prog.emit0(bcLoadFalse)
		p.l.next()
		break

	case tkNull:
		prog.emit0(bcLoadNull)
		p.l.next()
		break

	case tkRegex:
		idx, err := prog.addRegexp(p.l.valueText)
		if err != nil {
			return err
		}
		prog.emit1(bcLoadRegexp, idx)
		p.l.next()
		break

	case tkLPar:
		return p.parsePairOrSubexpr(prog)

	case tkLSqr:
		return p.parseList(prog)

	case tkLBra:
		return p.parseMap(prog)

	case tkDollar:
		return p.parseDExpr(prog)

	case tkId:
		return p.parseLocal(prog)

	case tkGId:
		return p.parseGlobalLoad(prog)

	// intrinsic related expressions
	case tkRender:
		return p.parseRender(prog)

	default:
		return p.err("unexpected token during expression parsing")
	}

	return nil
}

func (p *parser) parsePairOrSubexpr(prog *program) error {
	must(p.l.token == tkLPar, "expect '(' for pair")
	p.l.next()

	// first component
	if err := p.parseExpr(prog); err != nil {
		return err
	}

	switch p.l.token {
	case tkComma:
		p.l.next()

		if err := p.parseExpr(prog); err != nil {
			return err
		}

		if p.l.token != tkRPar {
			return p.err("expect ')' to close the pair")
		}
		p.l.next()

		prog.emit0(bcNewPair)
		return nil

	case tkRPar:
		// subexpression
		p.l.next()
		return nil

	default:
		return p.err("expect ')' or ',' for subexpression or pair")
	}
}

// dollar expression, ie dExpr
func (p *parser) parseCall(prog *program, name string) error {
	must(p.l.token == tkLPar, "expect '(' for function call")

	// function invocation ----------------------------------------------------
	// the calling convention is as following, we push function name into the
	// stack at the top of the stack and then we push all the function argument
	// on to the stack. Lastly a call instruction is emitted with argument size
	{
		idx := prog.addStr(name)
		prog.emit1(bcLoadStr, idx)
	}

	p.l.next()

	// then the parameter list
	pcnt := 0
	if p.l.token != tkRPar {
		for {
			if err := p.parseExpr(prog); err != nil {
				return nil
			}
			pcnt++
			if p.l.token == tkComma {
				p.l.next()
			} else if p.l.token == tkRPar {
				break
			} else {
				return p.err("invalid token, expect ',' or ')' in function argument list")
			}
		}
	}

	p.l.next()
	prog.emit1(bcCall, pcnt)
	return nil
}

func (p *parser) parseSuffix(prog *program) error {
	// suffix expression
SUFFIX:
	for {
		switch p.l.token {
		case tkDot:
			ntk := p.l.next()
			if ntk == tkId || ntk == tkStr {
				idx := prog.addStr(p.l.valueText)
				prog.emit1(bcDot, idx)
				p.l.next()
			} else {
				return p.err("invalid expresion, expect id or string after '.'")
			}
			break

		case tkLSqr:
			p.l.next()
			if err := p.parseExpr(prog); err != nil {
				return err
			}
			prog.emit0(bcIndex)
			if p.l.token != tkRSqr {
				return p.err("invalid expression, expect ] to close index")
			}
			p.l.next()
			break
		default:
			break SUFFIX
		}
	}
	return nil
}

func (p *parser) parseLocal(prog *program) error {
	must(p.l.token == tkId, "expect a identifier")
	idx := p.stbl.find(p.l.valueText)
	if idx == -1 {
		return p.err(fmt.Sprintf("unknown local %s", p.l.valueText))
	}
	prog.emit1(bcLoadLocal, idx)
	p.l.next()
	return p.parseSuffix(prog)
}

func (p *parser) parseDExpr(prog *program) error {
	must(p.l.token == tkDollar, "expect a $")
	tk := p.l.next()

	if tk == tkId {
		name := p.l.valueText
		ntk := p.l.next()
		if ntk == tkLPar {
			if err := p.parseCall(prog, name); err != nil {
				return err
			}
		} else {
			// still a suffix value, potentially, we should generate a variable
			// load bytecode on the stack
			idx := prog.addStr(name)
			prog.emit1(bcLoadVar, idx)
		}
	} else {
		// loading the dollar variable into the expression stack
		prog.emit0(bcLoadDollar)
	}
	return p.parseSuffix(prog)
}

// list literal
func (p *parser) parseList(prog *program) error {
	must(p.l.token == tkLSqr, "expect a [")
	p.l.next()

	prog.emit0(bcNewList)
	if p.l.token == tkRSqr {
		p.l.next()
		return nil
	} else {
		cnt := 0
		for {
			cnt++
			if err := p.parseExpr(prog); err != nil {
				return err
			}
			if p.l.token == tkComma {
				p.l.next()
			} else if p.l.token == tkRSqr {
				p.l.next()
				break
			} else {
				return p.err("unexpected token, expect ',' or ']' to close a list literal")
			}
		}
		prog.emit1(bcAddList, cnt)
		return nil
	}
}

// string blob is represented as following
// @(path-to-file) or string
func (p *parser) parseStringBlob() (string, error) {
	var content string

	switch p.l.token {
	case tkRId:
		x, err := util.LoadFile(p.l.valueText)
		if err != nil {
			return "", err
		}
		content = x
		p.l.next()
		break

	case tkStr, tkMStr:
		content = p.l.valueText
		p.l.next()
		break

	default:
		return "", p.err("expect a string blob, start with @ or strings")
	}

	return content, nil
}

// parsing template selector along with its needed options
// go[key=value; key=value]
func (p *parser) parseTemplateSelector(x string) (string, Val, error) {
	lexer := newLexer(x)
	if !lexer.expect(tkId) {
		return "", NewValNull(), p.err("template name must be valid identifier")
	}
	selector := lexer.valueText
	lexer.next()

	if lexer.token == tkEof {
		return selector, NewValNull(), nil
	} else if lexer.token == tkLSqr {
		opts := NewValMap()

		if lexer.next() != tkRSqr {
		LOOP:
			for {
				if !lexer.expectCurrent(tkId) {
					return "", NewValNull(), p.err("template option name must be identifier")
				}
				name := lexer.valueText

				if !lexer.expect(tkAssign) {
					return "", NewValNull(), p.err("expect a '=' after option name")
				}
				lexer.next()

				switch lexer.token {
				case tkStr:
					opts.AddMap(name, NewValStr(lexer.valueText))
					break
				case tkInt:
					opts.AddMap(name, NewValInt64(lexer.valueInt))
					break
				case tkReal:
					opts.AddMap(name, NewValReal(lexer.valueReal))
					break
				case tkTrue:
					opts.AddMap(name, NewValBool(true))
					break
				case tkFalse:
					opts.AddMap(name, NewValBool(false))
					break
				case tkNull:
					opts.AddMap(name, NewValNull())
					break
				default:
					return "", NewValNull(), p.err("invalid template compile option type")
				}

				switch lexer.next() {
				case tkComma:
					lexer.next()
					break
				case tkRSqr:
					lexer.next()
					break LOOP
				default:
					return "", NewValNull(), p.err("unexpected token in template selector")
				}
			}
		}

		return selector, opts, nil
	} else {
		return "", NewValNull(), p.err("unexpected token in template selector")
	}
}

func (p *parser) parseRender(prog *program) error {
	must(p.l.token == tkRender, "must be render")
	p.l.next()

	// (1) template type
	if !p.l.expectCurrent(tkStr) {
		return p.l.toError()
	}

	templateType, templateOpt, err := p.parseTemplateSelector(p.l.valueText)
	if err != nil {
		return err
	}
	if !p.l.expect(tkComma) {
		return p.l.toError()
	}
	p.l.next()

	// (2) template expression
	if err := p.parseExpr(prog); err != nil {
		return err
	}
	if !p.l.expectCurrent(tkComma) {
		return p.l.toError()
	}
	p.l.next()

	// (3) string blob
	if content, err := p.parseStringBlob(); err != nil {
		return err
	} else {
		idx, err := prog.addTemplate(templateType, content, templateOpt)
		if err != nil {
			return err
		}

		prog.emit1(bcTemplate, idx)
		return nil
	}
}

func (p *parser) parseMap(prog *program) error {
	must(p.l.token == tkLBra, "expect a {")
	p.l.next()

	prog.emit0(bcNewMap)
	if p.l.token == tkRBra {
		p.l.next()
	} else {
		cnt := 0
		for {
			cnt++

			if p.l.token == tkStr || p.l.token == tkId {
				idx := prog.addStr(p.l.valueText)
				prog.emit1(bcLoadStr, idx)
			} else {
				return p.err("unexpected token, expect a quoted string or identifier as map key")
			}

			if !p.l.expect(tkColon) {
				return p.l.toError()
			}
			p.l.next()

			if err := p.parseExpr(prog); err != nil {
				return err
			}

			if p.l.token == tkComma {
				p.l.next()
				continue
			} else if p.l.token == tkRBra {
				p.l.next()
				break
			} else {
				return p.err("unexpected token, expect '}' or ',' after a map entry")
			}
		}

		prog.emit1(bcAddMap, cnt)
	}
	return nil
}

// string interpolation
const (
	sInter0 = 0
	sInter1 = 1
)

func (p *parser) parseStrInterpolationExpr(strV string, offset int, prog *program) (int, error) {
	// we create a new parser here just to parse the interpolation part in the
	// string
	pp := newParser(strV[offset:])

	// share the same local variables
	pp.stbl = p.stbl

	// notes, we should start the lexer here, otherwise it will not work --------
	pp.l.next()

	if err := pp.parseExpr(prog); err != nil {
		return 0, p.err(fmt.Sprintf("string interpolation error: %s", err.Error()))
	}

	// the parser should always stop at the token DRBra, otherwise it means we
	// have dangling text cannot be consumed by the parser
	if pp.l.token != tkDRBra {
		return 0, p.err("invalid string interpolation part, expect }}")
	}
	return pp.l.cursor + offset, nil
}

// helper function to get as much as { from current sequences and returns the
// # of {
func maxLBra(rlist []rune, offset int) int {
	cnt := 0
	for ; offset < len(rlist); offset++ {
		c := rlist[offset]
		if c == '{' {
			cnt++
		} else {
			break
		}
	}
	return cnt
}

func (p *parser) parseStrInterpolation(prog *program, strV string) error {
	if len(strV) == 0 {
		// empty string
		idx := prog.addStr("")
		prog.emit1(bcLoadStr, idx)
		return nil
	}

	// parsing strV's internal structure with strings pieces and concate them
	// together notes.
	var b bytes.Buffer

	rlist := []rune(strV)
	size := len(rlist)
	strCnt := 0
	st := sInter0

	for idx := 0; idx < size; idx++ {
		c := rlist[idx]

		// 0) State for parsing the embedded scriptting and convert them into the
		//    strings for future concatenation

		// performing string scriptting parsing
		if st == sInter1 {
			newPos, err := p.parseStrInterpolationExpr(strV, idx, prog)
			if err != nil {
				return err
			}

			// lastly, convert whatever on the evaluation stack to be valid string
			prog.emit0(bcToStr)
			strCnt++
			idx = newPos - 1
			st = sInter0

			continue
		}

		// 1) State for parsing the string pieces and push them on to the evaluation
		//    stack for future concatenation

		// otherwise checking wether we should enter into the state for parsing the
		// string pieces and push them back to the evaluation stack for future
		// string concatenation and evaluation
		if c == '{' {
			cnt := maxLBra(rlist, idx)
			if cnt == 2 {
				piece := b.String()
				b.Reset()
				if piece != "" {
					sidx := prog.addStr(piece)
					prog.emit1(bcLoadStr, sidx)
					strCnt++
				}

				idx++
				st = sInter1
				continue
			} else {
				// Special rules for escapping, any consequtive { will result in cases
				// as following :
				// 1. just one {, then push it as literal
				// 2. start a interpolation sequences
				// 3. any sequences has longer than 2, then we push (# - 1)
				//    corresponding {
				if cnt == 1 {
					b.WriteRune('{')
				} else {
					for i := 0; i < cnt-1; i++ {
						b.WriteRune('{')
					}
					idx += cnt - 1
				}
				continue
			}
		}

		b.WriteRune(c)
	}

	if xx := b.String(); xx != "" {
		sidx := prog.addStr(xx)
		prog.emit1(bcLoadStr, sidx)
		strCnt++
	}

	if strCnt > 1 {
		prog.emit1(bcConStr, strCnt)
	}

	return nil
}
