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
	l       *lexer
	pmap    []*program
	stbl    *symTable
	sessVar []string
}

func newParser(input string) *parser {
	return &parser{
		l: newLexer(input),
	}
}

const (
	symSession = iota
	symLocal
	symDynamic
)

func (p *parser) findSessionIdx(x string) int {
	for idx, n := range p.sessVar {
		if n == x {
			return idx
		}
	}
	return -1
}

func (p *parser) resolveSymbol(xx string) (int, int) {
	// local symbol table
	local := p.stbl.find(xx)
	if local != -1 {
		return symLocal, local
	}

	// session symbol table
	sessVar := p.findSessionIdx(xx)
	if sessVar != -1 {
		return symSession, sessVar
	}

	// lastly, it must be dynamic variable. Notes, we don't have closure for now
	return symDynamic, -1
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

	// only the first statement can be session and it MUST be defined before any
	// other statement
	if p.l.token == tkSession {
		if prog, err := p.parseSessionScope(); err != nil {
			return nil, err
		} else {
			p.pmap = append(p.pmap, prog)
		}
	}

	for {
		tk := p.l.token
		if tk == tkId || tk == tkStr || tk == tkLSqr {
			prog, err := p.parseRule()
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

func (p *parser) parseSessionSet(prog *program) error {
	must(p.l.token == tkGId, "must be gid")
	gname := p.l.valueText
	idx := p.findSessionIdx(gname)
	if idx == -1 {
		return p.err(fmt.Sprintf("unknown session %s", gname))
	}
	if !p.l.expect(tkAssign) {
		return p.l.toError()
	}
	p.l.next()
	if err := p.parseExpr(prog); err != nil {
		return err
	}
	prog.emit1(bcStoreSession, idx)
	return nil
}

func (p *parser) parseSessionLoad(prog *program) error {
	must(p.l.token == tkGId, "must be gid")
	gname := p.l.valueText
	idx := p.findSessionIdx(gname)
	if idx == -1 {
		return p.err(fmt.Sprintf("unknown session %s", gname))
	}
	p.l.next()
	prog.emit1(bcLoadSession, idx)
	return p.parseSuffix(prog)
}

func (p *parser) parseSessionScope() (*program, error) {
	if !p.l.expect(tkLBra) {
		return nil, p.l.toError()
	}
	p.l.next()

	prog := newProgram("@session")

	if p.l.token != tkRBra {
		// all the statement resides inside of it will be treated as session statement
		// and will be allocated into a symbol table session shared
		for {
			if p.l.token != tkGId && p.l.token != tkId {
				return nil, p.err("expect a symbol to serve as session variable name")
			}
			gname := p.l.valueText

			if !p.l.expect(tkAssign) {
				return nil, p.err("expect a '=' to represent session variable assignment")
			}
			p.l.next()

			if err := p.parseExpr(prog); err != nil {
				return nil, err
			}

			p.sessVar = append(p.sessVar, gname)
			prog.emit0(bcSetSession)

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

func (p *parser) parseRule() (*program, error) {
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

	err := p.parseStmt(name, prog)
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

// parse basic statement, ie

// 1) a function call
// 2) a method call
// 3) a suffix expression
// 4) an assignment
// 5) an action
// 6) a global variable suffix expression (starting with special GId token)

type lexeme struct {
	token int
	ival  int64
	rval  float64
	sval  string
}

func (p *parser) lexeme() lexeme {
	return lexeme{
		token: p.l.token,
		ival:  p.l.valueInt,
		rval:  p.l.valueReal,
		sval:  p.l.valueText,
	}
}

func (p *parser) parseBasicStmt(prog *program) error {
	// save the current token and do not generate anything
	lexeme := p.lexeme()

	// lookahead
	p.l.next()

	switch p.l.token {
	case tkArrow:

		p.l.next()
		if err := p.parseExpr(prog); err != nil {
			return err
		}
		if lexeme.token != tkId {
			return p.err("action rule's lhs must be an identifier")
		}
		prog.emit1(bcAction, prog.addStr(lexeme.sval))
		return nil

	case tkAssign:
		switch lexeme.token {
		case tkId:
			p.l.next()
			if err := p.parseExpr(prog); err != nil {
				return err
			}

			sym, idx := p.resolveSymbol(lexeme.sval)
			switch sym {
			case symSession:
				prog.emit1(bcStoreSession, idx)
				break
			case symLocal:
				prog.emit1(bcStoreLocal, idx)
				break
			default:
				prog.emit1(bcStoreVar, prog.addStr(lexeme.sval))
				break
			}
			break

		case tkGId:
			p.l.next()
			if err := p.parseExpr(prog); err != nil {
				return err
			}

			// session symbol table
			sessVar := p.findSessionIdx(lexeme.sval)
			if sessVar != -1 {
				return p.err(fmt.Sprintf("session variable %s is not existed", lexeme.sval))
			}
			prog.emit1(bcStoreSession, sessVar)
			break

		default:
			return p.err("assignment's lhs must be identifier or a session identifier")
		}
		return nil

	default:
		break
	}

	// now try to generate the previous saved lexeme and it could be anything
	if err := p.parsePrimary(prog, lexeme); err != nil {
		return err
	}

	// then try to parse any suffix expression if applicable
	var st int
	if err := p.parseSuffixImpl(prog, &st); err != nil {
		return err
	}

	// handle the dangling assignment if needed. The assignment operation is been
	// handled by parser via patching the last bytecode been emitted
	if p.l.token == tkAssign {
		p.l.next()

		// generate assignment
		switch st {
		case suffixDot:
			lastIns := prog.popLast()
			must(lastIns.opcode == bcDot, "must be dot")

			if err := p.parseExpr(prog); err != nil {
				return err
			}

			prog.emit1(bcDotSet, lastIns.argument)
			break

		case suffixIndex:
			lastIns := prog.popLast()
			must(lastIns.opcode == bcDot, "must be index")

			if err := p.parseExpr(prog); err != nil {
				return err
			}
			prog.emit0(bcIndexSet)
			break

		default:
			return p.err("invalid assignment expression, the component assignment " +
				"can only apply to [] or '.' operators")
		}
	}

	return nil
}

func (p *parser) parseStmt(name string, prog *program) error {
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

			default:
				if err := p.parseBasicStmt(prog); err != nil {
					return err
				}
				break
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
				return p.err("expect ','/';' or ')'/'}' after an entry in rule definition")
			}
		}

		return nil
	}
}

// Expression parsing, using simple precedence climbing style way
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

		// notes we need to use comma instead of colon, since clone is already used
		// as method call, otherwise the parser has ambigiuty which needs extra
		// rules to resolve
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

func (p *parser) parsePrimary(prog *program, l lexeme) error {
	tk := l.token
	switch tk {
	case tkInt:
		idx := prog.addInt(l.ival)
		prog.emit1(bcLoadInt, idx)
		break

	case tkReal:
		idx := prog.addReal(l.rval)
		prog.emit1(bcLoadReal, idx)
		break

	case tkStr, tkMStr:
		strV := l.sval
		return p.parseStrInterpolation(prog, strV)

	case tkTrue:
		prog.emit0(bcLoadTrue)
		break

	case tkFalse:
		prog.emit0(bcLoadFalse)
		break

	case tkNull:
		prog.emit0(bcLoadNull)
		break

	case tkRegex:
		idx, err := prog.addRegexp(l.sval)
		if err != nil {
			return err
		}
		prog.emit1(bcLoadRegexp, idx)
		break

	case tkLPar:
		if err := p.parsePairOrSubexpr(prog); err != nil {
			return err
		}
		break

	case tkLSqr:
		if err := p.parseList(prog); err != nil {
			return err
		}
		break

	case tkLBra:
		if err := p.parseMap(prog); err != nil {
			return err
		}
		break

	case tkDollar, tkId, tkGId:
		if err := p.parsePExpr(prog, tk, l.sval); err != nil {
			return err
		}
		break

	// intrinsic related expressions
	case tkTemplate:
		return p.parseTemplate(prog)

	default:
		return p.err("unexpected token during expression parsing")
	}
	return nil
}

func (p *parser) parseAtomic(prog *program) error {
	l := p.lexeme()
	p.l.next()
	if err := p.parsePrimary(prog, l); err != nil {
		return err
	}

	return p.parseSuffix(prog)
}

func (p *parser) parsePairOrSubexpr(prog *program) error {
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

func (p *parser) parseCallArgs(prog *program) (int, error) {
	must(p.l.token == tkLPar, "expect '(' for function call")

	p.l.next()

	// then the parameter list
	pcnt := 0
	if p.l.token != tkRPar {
		for {
			if err := p.parseExpr(prog); err != nil {
				return -1, nil
			}
			pcnt++
			if p.l.token == tkComma {
				p.l.next()
			} else if p.l.token == tkRPar {
				break
			} else {
				return -1, p.err("invalid token, expect ',' or ')' in function argument list")
			}
		}
	}
	p.l.next()
	return pcnt, nil
}

func (p *parser) parseCall(prog *program, name string, bc int) error {
	{
		idx := prog.addStr(name)
		prog.emit1(bcLoadStr, idx)
	}

	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}
	prog.emit1(bc, pcnt)
	return nil
}

func (p *parser) parseICall(prog *program, index int) error {
	{
		idx := prog.addInt(int64(index))
		prog.emit1(bcLoadInt, idx)
	}

	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}
	prog.emit1(bcICall, pcnt)
	return nil
}

func (p *parser) parseNCall(prog *program, name string) error {
	intrinsicIdx := indexIntrinsic(name)
	if intrinsicIdx != -1 {
		return p.parseICall(prog, intrinsicIdx)
	} else {
		return p.parseCall(prog, name, bcCall)
	}
}

func (p *parser) parseMCall(prog *program, name string) error {
	return p.parseCall(prog, name, bcMCall)
}

const (
	suffixDot = iota
	suffixIndex
	suffixMethod
)

func (p *parser) parseSuffixImpl(prog *program, lastType *int) error {
	// suffix expression
SUFFIX:
	for {
		switch p.l.token {
		case tkDot:
			*lastType = suffixDot
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
			*lastType = suffixIndex
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

		case tkSharp:
			*lastType = suffixMethod
			// method invocation
			if !p.l.expect(tkId) {
				return p.l.toError()
			}

			methodName := p.l.valueText
			if !p.l.expect(tkLPar) {
				return p.l.toError()
			}

			if err := p.parseMCall(prog, methodName); err != nil {
				return err
			}
			break

		default:
			break SUFFIX
		}
	}
	return nil
}

func (p *parser) parseSuffix(prog *program) error {
	var t int
	return p.parseSuffixImpl(prog, &t)
}

// any types of expression which is prefixed with an "ID" token
// foo()
// foo.bar
// foo["bar"]
// foo:bar()
func (p *parser) parsePExpr(prog *program, tk int, name string) error {

	switch tk {
	case tkId:
		// identifier leading types
		switch p.l.token {
		case tkLPar:
			if err := p.parseNCall(prog, name); err != nil {
				return err
			}
			break

		case tkScope:
			// module call, mod::function_name(....)
			modName := name
			if !p.l.expect(tkId) {
				return p.l.toError()
			}
			funcName := p.l.valueText
			if !p.l.expect(tkLPar) {
				return p.l.toError()
			}
			if err := p.parseNCall(prog, modFuncName(modName, funcName)); err != nil {
				return err
			}
			break

		default:
			// Resolve the identifier here, notes the symbol can be following types
			// 1. session variable
			// 2. local variable
			// 3. dynmaic variable
			symT, symIdx := p.resolveSymbol(name)
			switch symT {
			case symSession:
				prog.emit1(bcLoadSession, symIdx)
				break

			case symLocal:
				prog.emit1(bcLoadLocal, symIdx)
				break

			default:
				prog.emit1(bcLoadVar, prog.addStr(name))
				break
			}
			break
		}

		// check whether we have suffix experssion
		switch p.l.token {
		case tkDot, tkLSqr, tkSharp:
			break

		default:
			return nil
		}
		break

	case tkDollar:
		prog.emit0(bcLoadDollar)
		break

	default:
		must(tk == tkGId, "must be GId")

		gname := name

		// session symbol table
		idx := p.findSessionIdx(gname)
		if idx == -1 {
			return p.err(fmt.Sprintf("global variable %s is unknown", gname))
		}
		prog.emit1(bcLoadSession, idx)
		break
	}

	return nil
}

// list literal
func (p *parser) parseList(prog *program) error {
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

func (p *parser) parseTemplate(prog *program) error {

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
