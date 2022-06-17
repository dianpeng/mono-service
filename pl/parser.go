package pl

import (
	"bytes"
	"fmt"

	"github.com/dianpeng/mono-service/util"
)

// lexical scope representation for scopping rules, store local variables per
// lexical scope, storing scope type and check whether we are inside of the
// loop which means certain statement may become valid, ie break, continue etc..

const (
	scpNormal = iota
	scpLoop
	scpInLoop
)

const (
	entryFunc = iota
	entryRule
	entryVar
	entryExpr
)

const (
	symtConst = iota
	symtVar
)

const (
	// symbol not found or symbol duplicate
	symError        = -1
	symInvalidConst = -2
)

type syminfo struct {
	name string
	dec  int
}

type lexicalScope struct {
	tbl     []syminfo
	eType   int
	scpType int
	parent  *lexicalScope
	isTop   bool

	// patch information for generating loop related information
	brklist []int
	cntlist []int

	// prog field, only used when isTop is true, modify upvalue when performing
	// upvalue collapsing
	prog *program
	// used only when the isTop is true, recording the largest local variable size
	maxLocal int
	offset   int
	next     *lexicalScope

	// caching the top scope of current lexical scope, top can be same as this
	top *lexicalScope
}

func newLexicalScope(t int, p *lexicalScope, isTop bool) *lexicalScope {
	if p != nil && t == scpNormal {
		if p.scpType == scpLoop || p.scpType == scpInLoop {
			t = scpInLoop
		}
	}

	offset := 0
	if p != nil {
		offset = p.nextOffset()
	}

	et := entryRule
	if p != nil {
		et = p.eType
	}

	x := &lexicalScope{
		tbl:     nil,
		eType:   et,
		scpType: t,
		parent:  p,
		isTop:   isTop,
		offset:  offset,
	}

	if p != nil {
		p.next = x
	}

	// always put at last
	if isTop {
		x.top = x
	} else {
		x.top = x.prevTop()
	}
	must(x.top != nil, "every scope should have a top")
	return x
}

func (s *lexicalScope) nextOffset() int {
	return s.offset + len(s.tbl)
}

func (s *lexicalScope) topMaxLocal() int {
	return s.top.maxLocal
}

func (s *lexicalScope) nearestLoop() *lexicalScope {
	x := s
	for x != nil && x.scpType != scpLoop {
		x = x.parent
	}
	return x
}

func (s *lexicalScope) prevTop() *lexicalScope {
	x := s
	for x != nil && !x.isTop {
		x = x.parent
	}
	return x
}

func (s *lexicalScope) nextTop() *lexicalScope {
	x := s.next
	for x != nil && !x.isTop {
		x = x.next
	}
	return x
}

func (s *lexicalScope) localSize() int {
	if s.tbl == nil {
		return 0
	}
	return len(s.tbl)
}

func (s *lexicalScope) vdx(i int) int {
	return s.offset + i
}

func (s *lexicalScope) idx(i int) int {
	return i - s.offset
}

func (s *lexicalScope) add(x string, st int) int {
	idx := s.find(x)
	if idx != symError {
		return idx
	}
	s.tbl = append(s.tbl, syminfo{
		name: x,
		dec:  st,
	})

	s.top.maxLocal++
	return s.vdx(len(s.tbl) - 1)
}

func (s *lexicalScope) addVar(x string) int {
	return s.add(x, symtVar)
}

func (s *lexicalScope) addConst(x string) int {
	return s.add(x, symtConst)
}

func (s *lexicalScope) find(x string) int {
	for idx, xx := range s.tbl {
		if xx.name == x {
			return s.vdx(idx)
		}
	}
	return symError
}

// returns -1 means the symbol is not found
// returns -2 means the symbol is const
func (s *lexicalScope) findVar(x string) int {
	idx := s.find(x)
	if idx == symError {
		return symError
	}
	if s.tbl[s.idx(idx)].dec == symtConst {
		return symInvalidConst
	}
	return idx
}

func (s *lexicalScope) findConst(x string) int {
	return s.find(x)
}

// simulate linker to resolve call types after all the code has been parsed
type callentry struct {
	entryPos int
	callPos  int
	prog     *program
	arg      int
	symbol   string
}

type parser struct {
	l         *lexer
	policy    *Policy
	stbl      *lexicalScope
	constVar  []string
	sessVar   []string
	callPatch []callentry
}

func newParser(input string) *parser {
	return &parser{
		l:      newLexer(input),
		policy: &Policy{},
	}
}

const (
	symSession = iota
	symLocal
	symUpval
	symConst
	symDynamic
)

// lexical scope management
func (p *parser) enterScope(t int, isTop bool) *lexicalScope {
	x := newLexicalScope(t, p.stbl, isTop)
	p.stbl = x
	return x
}

func (p *parser) enterLoopScope() *lexicalScope {
	return p.enterScope(scpLoop, false)
}

func (p *parser) enterScopeTop(et int, prog *program) *lexicalScope {
	pp := p.enterScope(scpNormal, true)
	pp.eType = et
	pp.prog = prog
	return pp
}

func (p *parser) enterNormalScope() *lexicalScope {
	return p.enterScope(scpNormal, false)
}

func (p *parser) leaveScope() *lexicalScope {
	pp := p.stbl.parent
	p.stbl = pp
	if p.stbl != nil {
		p.stbl.next = nil
	}
	return pp
}

func (p *parser) addBreak(label int) {
	must(p.isInLoop(), "must be in loop(addBreak)")
	loop := p.stbl.nearestLoop()
	must(loop != nil, "must have a closed loop")
	loop.brklist = append(loop.brklist, label)
}

func (p *parser) addContinue(label int) {
	must(p.isInLoop(), "must be in loop(addContinue)")
	loop := p.stbl.nearestLoop()
	must(loop != nil, "must have a closed loop")
	loop.cntlist = append(loop.cntlist, label)
}

func (p *parser) patchBreak(prog *program) {
	must(p.isInLoop(), "must be in loop(patchBreak)")
	cur := prog.label()
	for _, x := range p.stbl.brklist {
		prog.emit1At(p.l, x, bcJump, cur)
	}
	p.stbl.brklist = nil
}

func (p *parser) patchContinue(prog *program) {
	must(p.isInLoop(), "must be in loop(patchContinue)")
	cur := prog.label()
	for _, x := range p.stbl.cntlist {
		prog.emit1At(p.l, x, bcJump, cur)
	}
	p.stbl.cntlist = nil
}

func (p *parser) isLoop() bool {
	return p.stbl.scpType == scpLoop
}

func (p *parser) isInLoop() bool {
	return p.stbl.scpType == scpInLoop || p.stbl.scpType == scpLoop
}

func (p *parser) isNormal() bool {
	return p.stbl.scpType == scpNormal
}

func (p *parser) isEntryFunc() bool {
	return p.stbl.eType == entryFunc
}

func (p *parser) isEntryRule() bool {
	return p.stbl.eType == entryRule
}

func (p *parser) isEntryExpr() bool {
	return p.stbl.eType == entryExpr
}

func (p *parser) addLocalVar(x string) int {
	idx := p.stbl.find(x)
	if idx != symError {
		return idx
	}
	return p.stbl.addVar(x)
}

func (p *parser) addLocalConst(x string) int {
	idx := p.stbl.find(x)
	if idx != symError {
		return idx
	}
	return p.stbl.addConst(x)
}

func (p *parser) mustAddLocalVar(x string) int {
	idx := p.addLocalVar(x)
	must(idx != symError, x)
	return idx
}

// local variable name resolution
func (p *parser) findLocalAt(n string, scp *lexicalScope) (int, *lexicalScope) {
	x := scp

	for x != nil {
		idx := x.find(n)
		if idx != symError {
			return idx, x
		}
		if x.isTop {
			break
		}
		x = x.parent
	}
	return symError, x
}

func (p *parser) findLocalVarAt(n string, scp *lexicalScope) (int, *lexicalScope) {
	x := scp
	for x != nil {
		idx := x.findVar(n)
		if idx != symError {
			return idx, x
		}
		if x.isTop {
			break
		}
		x = x.parent
	}
	return symError, x
}

func (p *parser) findLocal(n string) int {
	x, _ := p.findLocalAt(n, p.stbl)
	return x
}

func (p *parser) findLocalVar(n string) int {
	x, _ := p.findLocalVarAt(n, p.stbl)
	return x
}

func (p *parser) findNearestUpcall(fr *lexicalScope) (*lexicalScope, *lexicalScope) {
	x := fr.prevTop()
	return x.parent, x
}

// upvalue name resolution
func (p *parser) findUpval(n string) int {
	start, currentTop := p.findNearestUpcall(p.stbl)
	cur := start
	idx := -1

	for cur != nil {
		i, top := p.findLocalAt(n, cur)
		if i >= 0 {
			idx = i
			break
		}
		cur = top.parent
	}

	if idx == -1 {
		return -1
	}

	// collapse the upvalue from multiple function's upvalue stack

	patchScope := cur.nextTop()
	must(patchScope != nil, "must have a preceeding top scope")
	onStack := true

	for {
		patchScope.prog.upvalue = append(patchScope.prog.upvalue,
			upvalue{
				index:   idx,
				onStack: onStack,
			},
		)

		// only first scope is on stack
		onStack = false

		// update the index
		idx = len(patchScope.prog.upvalue) - 1

		if patchScope == currentTop {
			break
		} else {
			patchScope = patchScope.nextTop()
		}
	}

	return idx
}

// session variable name resolution
func (p *parser) findSessionIdx(x string) int {
	for idx, n := range p.sessVar {
		if n == x {
			return idx
		}
	}
	return symError
}

func (p *parser) findConstIdx(x string) int {
	for idx, n := range p.constVar {
		if n == x {
			return idx
		}
	}
	return symError
}

// symbol resolution, ie binding
func (p *parser) resolveSymbol(xx string, ignoreConst bool, forWrite bool) (int, int) {
	// local symbol table, based on for-read/for-write to handle local cases, since
	// local symbol can be const value
	if forWrite {
		local := p.findLocalVar(xx)
		if local == symInvalidConst || local >= 0 {
			return symLocal, local
		}
	} else {
		local := p.findLocal(xx)
		if local != symError {
			return symLocal, local
		}
	}

	// notes, upvalue capture are all none constant. Constant is just syntax sugar
	// for local variables
	upval := p.findUpval(xx)
	if upval >= 0 {
		return symUpval, upval
	}

	// session symbol table
	sessVar := p.findSessionIdx(xx)
	if sessVar != symError {
		return symSession, sessVar
	}

	if !ignoreConst {
		// const symbol table
		constVar := p.findConstIdx(xx)
		if constVar != symError {
			return symConst, constVar
		}
	}

	// lastly, it must be dynamic variable. Notes, we don't have closure for now
	return symDynamic, symError
}

func (p *parser) err(xx string) error {
	if p.l.token == tkError {
		return p.l.toError()
	} else {
		return fmt.Errorf("%s: %s", p.l.position(), xx)
	}
}

func (p *parser) errf(f string, a ...interface{}) error {
	return p.err(fmt.Sprintf(f, a...))
}

func (p *parser) parseExpression() (*Policy, error) {
	p.l.next()
	prog := newProgram("$expression", progExpression)
	p.enterScopeTop(entryExpr, prog)
	defer func() {
		p.leaveScope()
	}()

	// load a null on top of the stack in case the expression does nothing
	prog.emit0(p.l, bcLoadNull)

	// now parsing the expression
	if err := p.parseExpr(prog); err != nil {
		return nil, err
	}
	prog.emit0(p.l, bcHalt)

	p.policy.p = append(p.policy.p, prog)

	p.patchAllCall()

	return p.policy, nil
}

func (p *parser) parse() (*Policy, error) {
	p.l.next()

	// top of the program allow user to write 2 types of special scope
	//
	// 1) Const scope, ie definition of const variable which is initialized once
	//    the program is been parsed, ie right inside of the parser
	//
	// 2) Session Scope, which should be reinitialized once the session is done.
	//    The definition of session is defined by user
	if p.l.token == tkConst {
		if err := p.parseConstScope(); err != nil {
			return nil, err
		}
	}

	if p.l.token == tkSession {
		if err := p.parseSessionScope(); err != nil {
			return nil, err
		}
	}

LOOP:
	for {
		tk := p.l.token
		switch tk {
		case tkRule, tkId, tkStr, tkLSqr:
			if err := p.parseRule(); err != nil {
				return nil, err
			}
			break

		case tkFunction:
			// eat the fn token, parseFunction expect after the fn keyword
			p.l.next()
			if _, err := p.parseFunction(false); err != nil {
				return nil, err
			}
			break

		case tkError:
			return nil, p.l.toError()

		default:
			if tk != tkEof {
				// print a little bit better diagnostic information, well, in general
				// we should invest more to error reporting
				switch tk {
				case tkSession:
					return nil, p.errf("session section should be put at the top, after const scope")
				case tkConst:
					return nil, p.errf("const section should be put at the top")
				default:
					return nil, p.errf("unknown token: %s, unrecognized statement", p.l.tokenName())
				}
			} else {
				break LOOP
			}
		}
	}

	p.patchAllCall()

	return p.policy, nil
}

func (p *parser) parseSessionLoad(prog *program) error {
	must(p.l.token == tkSId, "must be sesion id")

	gname := p.l.valueText
	idx := p.findSessionIdx(gname)
	if idx == symError {
		return p.errf("unknown session %s", gname)
	}
	p.l.next()
	prog.emit1(p.l, bcLoadSession, idx)
	return p.parseSuffix(prog)
}

func (p *parser) parseSessionScope() error {
	x, list, err := p.parseVarScope("@session",
		func(_ string, prog *program, p *parser) {
			prog.emit0(p.l, bcSetSession)
		},
	)

	if err != nil {
		return err
	}
	p.sessVar = list
	p.policy.session = x
	return nil
}

func (p *parser) parseConstScope() error {
	x, list, err := p.parseVarScope("@const",
		func(_ string, prog *program, p *parser) {
			prog.emit0(p.l, bcSetConst)
		},
	)

	if err != nil {
		return err
	}
	p.constVar = list
	p.policy.constProgram = x
	return nil
}

func (p *parser) parseVarScope(rulename string,
	gen func(string, *program, *parser)) (*program, []string, error) {
	if !p.l.expect(tkLBra) {
		return nil, nil, p.l.toError()
	}
	p.l.next()

	prog := newProgram(rulename, progSession)

	p.enterScopeTop(entryVar, prog)
	defer func() {
		p.leaveScope()
	}()

	localR := prog.patch(p.l)
	nlist := []string{}

	if p.l.token != tkRBra {
		// all the statement resides inside of it will be treated as session statement
		// and will be allocated into a symbol table session shared
		for {
			if p.l.token != tkId {
				return nil, nil, p.err("expect a symbol as variable name")
			}
			gname := p.l.valueText

			if !p.l.expect(tkAssign) {
				return nil, nil, p.err("expect a '=' for variable assignment")
			}
			p.l.next()

			if err := p.parseExpr(prog); err != nil {
				return nil, nil, err
			}

			nlist = append(nlist, gname)
			gen(gname, prog, p)

			if !p.l.expectCurrent(tkSemicolon) {
				return nil, nil, p.l.toError()
			}
			p.l.next()

			if p.l.token == tkRBra {
				break
			}
		}
	}

	// lastly halt the whole execution
	prog.emit0(p.l, bcHalt)
	prog.emit1At(p.l, localR, bcReserveLocal, 0)
	p.l.next()

	return prog, nlist, nil
}

// parsing a user defined script function
// function name'(' arg-list ')'
// function's argument are been treated just like local variables, since they
// will be put into the stack by the caller
func (p *parser) parseFunction(anony bool) (_ string, the_error error) {
	funcName := ""

	// if it is not an anonymous function
	if !anony {
		if !p.l.expectCurrent(tkId) {
			return "", p.l.toError()
		}
		funcName = p.l.valueText
		p.l.next()
	} else {
		funcName = p.policy.nextAnonymousFuncName()
	}
	if !p.l.expectCurrent(tkLPar) {
		return "", p.l.toError()
	}

	prog := newProgram(funcName, progFunc)
	p.enterScopeTop(entryFunc, prog)
	defer func() {
		p.leaveScope()
	}()

	localR := prog.patch(p.l)
	p.policy.fn = append(p.policy.fn, prog)
	defer func() {
		if the_error != nil {
			sz := len(p.policy.fn)
			p.policy.fn = p.policy.fn[:sz-1]
		}
	}()

	// parsing the function's argument and just treat them as
	if p.l.next() != tkRPar {
		argcnt := 0
		for {
			if !p.l.expectCurrent(tkId) {
				return "", p.l.toError()
			}
			idx := p.addLocalVar(p.l.valueText)
			if idx == symError {
				return "", p.err("argument name duplicated")
			}
			p.l.next()
			argcnt++

			if p.l.token == tkComma {
				p.l.next()
			} else if p.l.token == tkRPar {
				break
			} else {
				return "", p.err("expect a ')' or ',' inside of function argument list")
			}
		}

		prog.argSize = argcnt
	}
	p.l.next()

	// reserve a frame slot
	p.mustAddLocalVar("#frame")

	// now parsing the function body
	if err := p.parseBody(prog); err != nil {
		return "", err
	}

	// always generate a return nil at the end of the function
	prog.emit0(p.l, bcLoadNull)
	prog.emit0(p.l, bcReturn)
	prog.localSize = p.stbl.topMaxLocal() - 1
	prog.emit1At(p.l, localR, bcReserveLocal, p.stbl.topMaxLocal()-1)

	return funcName, nil
}

func (p *parser) parseRule() error {
	if p.l.token == tkRule {
		p.l.next()
	}

	var name string

	// parsing the policy name, we accept few different types to ease user's
	// own syntax flavor
	if p.l.token == tkStr || p.l.token == tkId {
		name = p.l.valueText
		p.l.next()
	} else if p.l.token == tkLSqr {
		p.l.next()
		if p.l.token != tkStr && p.l.token != tkId {
			return p.err("unexpected token, expect string or identifier for rule name")
		}
		name = p.l.valueText
		if !p.l.expect(tkRSqr) {
			return p.l.toError()
		}
		p.l.next()
	} else {
		return p.err("unexpected token here, expect a string/identifier or a [id] to " +
			"represent rule name")
	}

	// notes the name must be none empty and also must not start with @, which is
	// builtin event name
	if name == "" {
		return p.err("invalid rule name, cannot be empty string")
	}
	if name[0] == '@' {
		return p.err("invalid rule name, cannot start with @ which is builtin name")
	}

	prog := newProgram(name, progRule)
	p.enterScopeTop(entryRule, prog)
	p.mustAddLocalVar("#frame")
	defer func() {
		p.leaveScope()
	}()

	localR := prog.patch(p.l)

	// check whether we have a conditional when, otherwise generate default match
	// condition
	if p.l.token == tkWhen {
		p.l.next()
		err := p.parseExpr(prog)
		if err != nil {
			return err
		}
		prog.emit0(p.l, bcMatch)
	} else {
		// bytecode for compare current event to be the same as the rule name
		// essentially is when event == "rule_name"
		prog.emit0(p.l, bcLoadDollar)
		idx := prog.addStr(name)
		prog.emit1(p.l, bcLoadStr, idx)
		prog.emit0(p.l, bcEq)
		prog.emit0(p.l, bcMatch)
	}

	// allow an optional arrow to indicate this is a rule, this is the preferred
	// grammar to indicate the rule definition inside
	if p.l.token == tkArrow {
		p.l.next()
	}

	// parse the rule's body
	if err := p.parseBody(prog); err != nil {
		return err
	}

	// always generate a halt at the end of the rule
	prog.emit0(p.l, bcHalt)
	prog.localSize = p.stbl.topMaxLocal() - 1
	prog.emit1At(p.l, localR, bcReserveLocal, p.stbl.topMaxLocal()-1)
	p.policy.p = append(p.policy.p, prog)

	return nil
}

// parse basic statement, ie

// 1) a function call
// 2) a method call
// 3) a suffix expression
// 4) an assignment
// 5) an action
// 6) a session variable suffix expression

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

func (p *parser) parseSymbolAssign(prog *program,
	symName string, /* symbol name */
	symType int, /* symbol type */
	symIdx int, /* symbol index, if applicable */
) error {
	loadVar := func() {
		switch symType {
		case symSession:
			must(symIdx >= 0, "must be valid index")
			prog.emit1(p.l, bcLoadSession, symIdx)
			break

		case symLocal:
			must(symIdx >= 0, "must be valid index")
			prog.emit1(p.l, bcLoadLocal, symIdx)
			break

		case symUpval:
			must(symIdx >= 0, "must be valid index")
			prog.emit1(p.l, bcLoadUpvalue, symIdx)
			break

		default:
			prog.emit1(p.l, bcLoadVar, prog.addStr(symName))
			break
		}
	}

	// current assignment token
	assign := p.l.token

	// If the token is an agg assignment then we need to load the variable
	// on tos and then perform the corresponding operation then perform the
	// store, ie desugar the syntax little bit
	switch assign {
	case tkAssign:
		// do nothing
		break

	// we do not have extra bytecode for inc and dec for simplicity, but just
	// desugar it here. The inc and dec is basically been transformed as this:
	// x++ -> x+=1 -> x=x+1.
	case tkAddAssign,
		tkSubAssign,
		tkMulAssign,
		tkDivAssign,
		tkPowAssign,
		tkModAssign,
		tkInc,
		tkDec:

		loadVar()
		break

	default:
		unreachable("invalid token")
		break
	}

	p.l.next() // the previous assignment token

	// now we should expect a expression if the token is not Inc/Dec
	switch assign {
	case tkInc:
		prog.emit1(p.l, bcLoadInt, prog.addInt(1))
		prog.emit0(p.l, bcAdd)
		break

	case tkDec:
		prog.emit1(p.l, bcLoadInt, prog.addInt(1))
		prog.emit0(p.l, bcSub)
		break

	default:
		if err := p.parseExpr(prog); err != nil {
			return err
		}

		switch assign {
		case tkAddAssign:
			prog.emit0(p.l, bcAdd)
			break

		case tkSubAssign:
			prog.emit0(p.l, bcSub)
			break

		case tkMulAssign:
			prog.emit0(p.l, bcMul)
			break

		case tkDivAssign:
			prog.emit0(p.l, bcDiv)
			break

		case tkModAssign:
			prog.emit0(p.l, bcMod)
			break

		case tkPowAssign:
			prog.emit0(p.l, bcPow)
			break

		default:
			// tkAssign
			break
		}
		break
	}

	// now the TOS should have the valid value to be used as storage

	switch symType {
	case symSession:
		prog.emit1(p.l, bcStoreSession, symIdx)
		break

	case symLocal:
		prog.emit1(p.l, bcStoreLocal, symIdx)
		break

	case symUpval:
		prog.emit1(p.l, bcStoreUpvalue, symIdx)
		break

	default:
		prog.emit1(p.l, bcStoreVar, prog.addStr(symName))
		break
	}

	return nil
}

func (p *parser) parseBasicStmt(prog *program, lexeme lexeme) error {

	switch p.l.token {
	// assignment, agg-assignment, inc, dec
	case tkAssign,
		tkAddAssign,
		tkSubAssign,
		tkMulAssign,
		tkDivAssign,
		tkPowAssign,
		tkModAssign,
		tkInc, tkDec:

		switch lexeme.token {
		// unbounded token/symbol, require resolution
		case tkId:
			sym, idx := p.resolveSymbol(lexeme.sval, true, true)
			must(sym != symConst, "must not be const")

			if sym == symLocal && idx == symInvalidConst {
				return p.errf("local variable %s is const, cannot be assigned to", lexeme.sval)
			}

			return p.parseSymbolAssign(prog,
				lexeme.sval,
				sym,
				idx,
			)

			// session store
		case tkSId:
			idx := p.findSessionIdx(lexeme.sval)
			if idx == symError {
				return p.errf("session %s variable not existed", lexeme.sval)
			}

			return p.parseSymbolAssign(prog,
				lexeme.sval,
				symSession,
				idx,
			)

			// const store
		case tkCId:
			return p.errf("const scope variable cannot be used for storing")

			// dynamic variable store
		case tkDId:
			return p.parseSymbolAssign(prog,
				lexeme.sval,
				symDynamic,
				symError,
			)

		default:
			return p.err("assignment's lhs must be identifier or a session identifier")
		}
		return nil

	default:
		// other prefix expression
		break
	}

	// Suffix expression assignment, agg-assignment, inc and dec -----------------

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
	if isassign(p.l.token) {
		assign := p.l.token

		// the assignment will be desugar as following, example:

		// a.b.c.d.e += 3
		// the bytecode will generate the expression evaluation until a.b.c.d
		// and then there's a bcDup on the stack to duplicate TOS, then we generate
		// the last instructino back to the output

		var lastbc bytecode

		switch st {
		case suffixDot, suffixIndex:
			lastbc = prog.popLast()
			break

		default:
			return p.err("invalid assignment expression, the field assignment " +
				"can only apply to [] or '.' operators")
		}

		if isaggassign(assign) {
			if st == suffixIndex {
				prog.emit0(p.l, bcDup2)
			} else {
				prog.emit0(p.l, bcDup1)
			}
			prog.emit1(p.l, lastbc.opcode, lastbc.argument)
		}

		// now we generate the expression following the assignment operators
		p.l.next()

		switch assign {
		case tkInc:
			prog.emit1(p.l, bcLoadInt, prog.addInt(1))
			prog.emit0(p.l, bcAdd)
			break

		case tkDec:
			prog.emit1(p.l, bcLoadInt, prog.addInt(1))
			prog.emit0(p.l, bcSub)
			break

		default:
			if err := p.parseExpr(prog); err != nil {
				return err
			}

			switch assign {
			case tkAddAssign:
				prog.emit0(p.l, bcAdd)
				break

			case tkSubAssign:
				prog.emit0(p.l, bcSub)
				break

			case tkMulAssign:
				prog.emit0(p.l, bcMul)
				break

			case tkDivAssign:
				prog.emit0(p.l, bcDiv)
				break

			case tkModAssign:
				prog.emit0(p.l, bcMod)
				break

			case tkPowAssign:
				prog.emit0(p.l, bcPow)
				break

			default:
				// tk assign
				break
			}
		}

		// lasty based on the st to decide how to load the value from tos back to
		// the component
		switch st {
		case suffixDot:
			must(lastbc.opcode == bcDot, "must be dot")
			prog.emit1(p.l, bcDotSet, lastbc.argument)
			break

		default:
			must(lastbc.opcode == bcIndex, "must be index")
			prog.emit0(p.l, bcIndexSet)
			break
		}
	}

	return nil
}

func (p *parser) parseStmt(prog *program) error {
	// save the current token and do not generate anything
	lexeme := p.lexeme()

	// lookahead
	p.l.next()

	// for rule we have action operation which is only available inside of the rule
	if p.l.token == tkArrow {
		if !p.isEntryRule() {
			return p.err("action can only be applied inside of the rule")
		}

		p.l.next()
		if err := p.parseExpr(prog); err != nil {
			return err
		}
		if lexeme.token != tkId {
			return p.err("action's lhs must be an identifier")
		}
		prog.emit1(p.l, bcAction, prog.addStr(lexeme.sval))
		return nil
	} else {
		// normal statement operation, ie assignment, agg assignment inc and dec
		return p.parseBasicStmt(prog, lexeme)
	}
}

// exception handling
// we have 2 types of exception use cases,
// 1) expression try else,
//    ie 'try foo() else 10', when foo() has error, then the 10 will be returned
//
// 2) statement try else style
//    ie
//    try {
//      ...
//    } else [let] id {
//    }

func (p *parser) parseTry(prog *program,
	tryGen func(*program) error,
	elseGen func(*program) error,
) error {
	must(p.l.token == tkTry, "must be try")

	// add a frame of exception
	pushexp := prog.patch(p.l)

	// eat the try
	p.l.next()

	// generate try body
	if err := tryGen(prog); err != nil {
		return err
	}

	// patch the enter exception
	prog.emit1At(p.l, pushexp, bcPushException, prog.label())

	// exception region finished, and jump out of the else branch
	popexp := prog.patch(p.l)

	// else branch
	if !p.l.expectCurrent(tkElse) {
		return p.l.toError()
	}
	p.l.next()

	// optionally let, which allow user to capture the exception/error
	if p.l.token == tkLet || p.l.token == tkId {
		var idx int
		if p.l.token == tkLet {
			if !p.l.expect(tkId) {
				return p.l.toError()
			}
			idx = p.addLocalVar(p.l.valueText)
			if idx == symError {
				return p.errf("duplicate local variable: %s", p.l.valueText)
			}
			p.l.next()
		} else {
			idx = p.findLocalVar(p.l.valueText)
			if idx == symError {
				return p.errf("local variable not found: %s", p.l.valueText)
			} else if idx == symConst {
				return p.errf("local variable is const: %s", p.l.valueText)
			}
			p.l.next()
		}
		must(idx >= 0, "must be valid index")

		prog.emit0(p.l, bcLoadException)
		prog.emit1(p.l, bcStoreLocal, idx)
	}

	if err := elseGen(prog); err != nil {
		return err
	}

	prog.emit1At(p.l, popexp, bcPopException, prog.label())
	return nil
}

func (p *parser) parseTryStmt(prog *program) error {
	parseChunk := func(prog *program) error {
		p.enterNormalScope()
		if err := p.parseBody(prog); err != nil {
			return err
		}
		p.leaveScope()
		return nil
	}
	return p.parseTry(prog, parseChunk, parseChunk)
}

func (p *parser) parseBranch(prog *program,
	bodyGen func(*program) error, /* invoked whenever a branch body is needed */
	dangling func(*program) error, /* invoked whenever a branch does have dangling,
	   notes, could be elif or else */
) error {
	must(p.l.token == tkIf, "must be if")

	var jump_out []int
	var prev_jmp int

	// skip if
	p.l.next()

	// (0) Parse the leading If, which does not allow nested if
	if err := p.parseTernary(prog); err != nil {
		return err
	}
	prev_jmp = prog.patch(p.l)
	if err := bodyGen(prog); err != nil {
		return err
	}
	jump_out = append(jump_out, prog.patch(p.l))

	// (1) Parse the rest of the Elif chain
	for {
		if p.l.token != tkElif {
			break
		}
		p.l.next()

		// previous condition failure target position
		prog.emit1At(p.l, prev_jmp, bcJfalse, prog.label())

		if err := p.parseExpr(prog); err != nil {
			return err
		}

		prev_jmp = prog.patch(p.l)

		if err := bodyGen(prog); err != nil {
			return err
		}

		jump_out = append(jump_out, prog.patch(p.l))
	}

	// (2) Lastly the Else branch
	//     notes if there's no else, the else branch can be assumed to return a
	//     null

	prog.emit1At(p.l, prev_jmp, bcJfalse, prog.label())
	if p.l.token == tkElse {
		p.l.next()
		if err := bodyGen(prog); err != nil {
			return err
		}
	} else {
		// notes we need to detect whether we have dangling branch or not. when
		// expression generation is needed, the dangling branch should be taken
		// care of otherwise the if expression statement does not have any value
		// when the leading branch is failed,
		// example:

		// if cond1 {
		// }
		// ** dangling **
		//
		// The above if statement's first condition missed, the whole expression will
		// be just dangling which will result in no expression been evaluated on the
		// stack. Instead, we can either emit an error or automatically generate a
		// null value

		// saimilar example:
		//
		// if cond1 {
		// } elif cond2 {
		// } elif .... {
		// }
		//
		// the above example lacks the last else which means the if statement has
		// dangling value which will result in nothing emitted

		if err := dangling(prog); err != nil {
			return err
		}
	}

	// Patch all the jump
	for _, pos := range jump_out {
		prog.emit1At(p.l, pos, bcJump, prog.label())
	}

	return nil
}

func (p *parser) parseBranchStmtBody(prog *program) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}

	{
		p.enterNormalScope()

		if err := p.parseBody(prog); err != nil {
			return err
		}

		p.leaveScope()
	}

	return nil
}

func (p *parser) parseBranchStmt(prog *program) error {
	return p.parseBranch(
		prog,
		p.parseBranchStmtBody,
		func(_ *program) error {
			return nil
		},
	)
}

func (p *parser) parseVarDecl(prog *program, symt int) error {
	if !p.l.expect(tkId) {
		return p.l.toError()
	}

	var idx int
	if symt == symtVar {
		idx = p.addLocalVar(p.l.valueText)
	} else {
		idx = p.addLocalConst(p.l.valueText)
	}
	if idx == symError {
		return p.err("duplicate local variable via let")
	}
	p.l.next()

	if p.l.token == tkAssign {
		p.l.next()
		if err := p.parseExpr(prog); err != nil {
			return err
		}
	} else {
		// default value to be null
		prog.emit0(p.l, bcLoadNull)
	}

	prog.emit1(p.l, bcStoreLocal, idx)
	return nil
}

func (p *parser) parseReturn(prog *program) error {
	if !p.isEntryFunc() {
		return p.err("return is only allowed inside of function body")
	}
	p.l.next()
	if err := p.parseExpr(prog); err != nil {
		return err
	}
	prog.emit0(p.l, bcReturn)
	return nil
}

// 1) froever loop, just jump back to where it starts
func (p *parser) parseForeverLoop(prog *program) error {
	p.enterLoopScope()

	loopHeader := prog.label()

	if err := p.parseBody(prog); err != nil {
		return err
	}

	// patch continue to jump at the last jump which jumps back to the loop header
	p.patchContinue(prog)

	prog.emit1(p.l, bcJump, loopHeader)

	// out of the loop, patch break to jump here
	p.patchBreak(prog)

	p.leaveScope()
	return nil
}

// -------------------------------------------------------------------
// 2) iterator/trip loop
// -------------------------------------------------------------------
//
// this function will be called when we see the comma after key, ie
//        let key, val = ...
//               ^
// -------------------------------------------------------------------
func (p *parser) parseIteratorLoop(prog *program, key string) error {
	must(p.l.token == tkComma, "must be comma")
	p.l.next()

	// 1. parsing the value local variable name
	if !p.l.expect(tkId) {
		return p.l.toError()
	}
	val := p.l.valueText

	// 2. parsing the iterator expression
	if !p.l.expect(tkAssign) {
		return p.l.toError()
	}

	if err := p.parseExpr(prog); err != nil {
		return err
	}

	// 3. Convert the TOS into iterator value
	prog.emit0(p.l, bcNewIterator)

	// 4. Enter into the loop body, the key value local variable will be swapped
	//    into the loop body and the iterator protocol will be materialized inside
	//    of the loop body
	p.enterLoopScope()

	keyIdx := p.mustAddLocalVar(key)
	valIdx := p.addLocalVar(val)

	if valIdx == symError {
		return p.err("duplicate local variable name in iterator style loop")
	}

	// 4.1 Perform the iterator testing, whether we should done for the loop or
	//     not
	prog.emit0(p.l, bcHasIterator)

	// header testify jumping
	headerTestify := prog.patch(p.l)

	// get the real header label, which indicate a full cycle of a single loop
	// iteration
	headerPos := prog.label()

	// tos has key and value
	prog.emit0(p.l, bcDerefIterator)

	// notes tos = val, tos - 1 = key
	prog.emit1(p.l, bcStoreLocal, valIdx)
	prog.emit1(p.l, bcStoreLocal, keyIdx)

	// 4.2 loop body parsing
	if err := p.parseBody(prog); err != nil {
		return err
	}

	// patch continue to jump at the end of the loop
	p.patchContinue(prog)

	// 4.3 loop tail check, if the iterator can move forward, then just move forwad
	//     and return true, otherwise return false
	prog.emit0(p.l, bcNextIterator)
	prog.emit1(p.l, bcJfalse, headerPos)

	// header testify should be patched to jump at current position, which is
	// out of the loop body
	prog.emit1At(p.l, headerTestify, bcJfalse, prog.label())

	// patch all the break to jump here which cleans up the loop stack
	p.patchBreak(prog)

	// 4.4 now we are out of the loop body, pop the iterator left on the stack
	prog.emit0(p.l, bcPop)

	// pop the lexical scope out of the stack
	p.leaveScope()

	return nil
}

// notes, this assumes the trip count's initial statement has been finished
func (p *parser) parseTripCountLoopRest(prog *program) error {
	must(p.l.token == tkSemicolon, "must be ;")
	must(p.isLoop(), "loop must already setup")

	// the trip count for loop is kind of hard to do a one pass code generation
	// since the trip addition statement after each iteration should be placed
	// after the for loop body is done but syntax wise it is at the top. To address
	// this issue, we use some fancy jump to resolve the issue.
	//
	// Conceptually, the generated code is layout as following:
	//
	// [ initial statement ]  1)
	//
	// [    condition      ]  2)
	//
	// [  trip statement   ]  3)
	//
	// [       body        ]  4)

	// the condition 2) will always directly jump into the body and after the
	// trip statement 3) it will jump to 2)
	//
	// so the first iteration will be 1) 2) 4)
	// and rest iteration will be 4) 3) 2) 4) which forms a cycle
	//
	// and the layout of each block is syntacally the same order as the loop syntax
	// organization

	// notes if condition is left to be empty, which means true, then we still
	// generate a load true at the tos for simpilicity, though at the cost of
	// performance overhead

	// skip the leading ; which is left by the previous initial statement
	p.l.next()
	condPos := prog.label()

	// 1) Condition parsing/code generation
	if p.l.token == tkSemicolon {
		// empty condition
		prog.emit0(p.l, bcLoadTrue)

		// eat the semicolon
		p.l.next()
	} else {
		// does have an expression, then just evaluate the expression
		if err := p.parseExpr(prog); err != nil {
			return err
		}
		if !p.l.expectCurrent(tkSemicolon) {
			return p.l.toError()
		}
		p.l.next()
	}

	// generate condition check code, notes this jump is a jcc based on the tos
	// value evaluation's result
	condEvalJump := prog.patch(p.l)

	// after the condition we set up a jump that directly jumps into the loop body
	condJump := prog.patch(p.l)

	// 2) Trip statement code generation
	tripPos := prog.label()

	if p.l.token != tkLBra {
		lexeme := p.lexeme()
		p.l.next()

		if err := p.parseBasicStmt(prog, lexeme); err != nil {
			return err
		}
	}

	// finish the trip statement part, now it should jump into the condition
	// check
	prog.emit1(p.l, bcJump, condPos)

	// 3) Loop body
	if p.l.token != tkLBra {
		return p.err("expect '{' to start loop's body")
	}

	// after condition jump to the loop body
	prog.emit1At(p.l, condJump, bcJump, prog.label())

	if err := p.parseBody(prog); err != nil {
		return err
	}

	// patch the continue statement to jump here
	p.patchContinue(prog)

	// after the loop body, jump back to the trip statement
	prog.emit1(p.l, bcJump, tripPos)

	// patch the conditional evaluation jump to jump here when it is false
	prog.emit1At(p.l, condEvalJump, bcJfalse, prog.label())

	// since we are out of the loop, now let's just patch all the break to jump
	// here
	p.patchBreak(prog)

	return nil
}

// full entry, ie start with let key
func (p *parser) parseTripCountLoopFull(prog *program, key string) error {
	must(p.l.token == tkAssign, "must be =")
	p.enterLoopScope()
	defer func() {
		p.leaveScope()
	}()

	p.l.next()
	if err := p.parseExpr(prog); err != nil {
		return err
	}

	keyIdx := p.mustAddLocalVar(key)
	prog.emit1(p.l, bcStoreLocal, keyIdx)

	if !p.l.expectCurrent(tkSemicolon) {
		return p.l.toError()
	}
	return p.parseTripCountLoopRest(prog)
}

func (p *parser) parseTripCountLoopNoInit(prog *program) error {
	must(p.l.token == tkSemicolon, "must be ;")
	p.enterLoopScope()
	defer func() {
		p.leaveScope()
	}()

	return p.parseTripCountLoopRest(prog)
}

func (p *parser) parseIteratorOrTripLoop(prog *program) error {
	must(p.l.token == tkLet, "must be let")

	if !p.l.expect(tkId) {
		return p.l.toError()
	}
	key := p.l.valueText

	p.l.next()

	if p.l.token == tkComma {
		return p.parseIteratorLoop(prog, key)
	} else if p.l.token == tkAssign {
		return p.parseTripCountLoopFull(prog, key)
	} else {
		return p.err("invalid token, expect a ',' for a range for loop or " +
			"a '=' to start a trip count foreach's initial statement")
	}
}

// 3) condition loop
func (p *parser) parseConditionLoop(prog *program) error {

	loopHeader := prog.label()

	// Get the condition on the TOS
	if err := p.parseExpr(prog); err != nil {
		return err
	}

	// jump out of the loop if needed, via bcJfalse
	jumpOut := prog.patch(p.l)

	// lastly, parse the body of the loop
	p.enterLoopScope()
	if err := p.parseBody(prog); err != nil {
		return err
	}

	p.patchContinue(prog)

	// jump back to reevaluate the loop condition
	prog.emit1(p.l, bcJump, loopHeader)

	// out of the loop body, now patch the conditional testify jump to jump
	// to current position
	prog.emit1At(p.l, jumpOut, bcJfalse, prog.label())

	// patch all the break statement
	p.patchBreak(prog)

	p.leaveScope()
	return nil
}

func (p *parser) parseFor(prog *program) error {
	p.l.next()

	switch p.l.token {

	case tkLBra:
		// forever loop,
		// for {
		//   ...
		// }
		return p.parseForeverLoop(prog)

	case tkLet:
		// for let loop, ie iterator or trip style

		// Iterator style: notes the let k, v before the '='
		// for let k, v = expression {
		//   k is the iterator index
		//   v is the iterator value
		// }

		// Trip style: only one induction variable
		// for let i = expression; i < 100; i = i + 1 {
		// }
		return p.parseIteratorOrTripLoop(prog)

	case tkSemicolon:
		// trip count loop, but ignore the initial statement // ie
		// for ; condition; step {
		// }
		return p.parseTripCountLoopNoInit(prog)

	default:
		// condition loop, ie the expression been setup is been treated as a
		// while loop condition:
		// for condition {
		//   ..
		// }
		return p.parseConditionLoop(prog)
	}
}

// loop control
func (p *parser) parseBreak(prog *program) error {
	if !p.isInLoop() {
		return p.err("break can only work inside of the loop body")
	}
	p.l.next()
	p.addBreak(prog.patch(p.l))
	return nil
}

func (p *parser) parseContinue(prog *program) error {
	if !p.isInLoop() {
		return p.err("continue can only work inside of the loop body")
	}
	p.l.next()
	p.addContinue(prog.patch(p.l))
	return nil
}

func (p *parser) parseBody(prog *program) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}
	p.l.next()

	if p.l.token == tkRBra {
		p.l.next()
		return nil
	}

	for {
		hasSep := true

		switch p.l.token {
		case tkSemicolon:
			break

		case tkLBra:
			p.enterNormalScope()
			if err := p.parseBody(prog); err != nil {
				return err
			}
			p.leaveScope()
			hasSep = false
			break

		case tkLet:
			if err := p.parseVarDecl(prog, symtVar); err != nil {
				return err
			}
			break

		case tkConst:
			if err := p.parseVarDecl(prog, symtConst); err != nil {
				return err
			}
			break

		case tkIf:
			if err := p.parseBranchStmt(prog); err != nil {
				return err
			}
			hasSep = false
			break

		case tkTry:
			if err := p.parseTryStmt(prog); err != nil {
				return err
			}
			hasSep = false
			break

		case tkFor:
			if err := p.parseFor(prog); err != nil {
				return err
			}
			hasSep = false
			break

		case tkBreak:
			if err := p.parseBreak(prog); err != nil {
				return err
			}
			break

		case tkContinue:
			if err := p.parseContinue(prog); err != nil {
				return err
			}
			break

		case tkReturn:
			if err := p.parseReturn(prog); err != nil {
				return err
			}
			break

		default:
			if err := p.parseStmt(prog); err != nil {
				return err
			}
			break
		}

		if hasSep {
			if !p.l.expectCurrent(tkSemicolon) {
				return p.l.toError()
			}
			p.l.next()
		}

		if p.l.token == tkRBra {
			p.l.next()
			break
		}
	}

	return nil
}

// Expression parsing, using simple precedence climbing style way
func (p *parser) parseExpr(prog *program) error {
	switch p.l.token {
	case tkIf:
		return p.parseBranchExpr(prog)
	case tkTry:
		return p.parseTryExpr(prog)
	default:
		return p.parseTernary(prog)
	}
}

func (p *parser) parseTryExpr(prog *program) error {
	parseChunk := func(prog *program) error {
		if p.l.token == tkLBra {
			return p.parseExprBody(prog)
		} else {
			err := p.parseExpr(prog)
			return err
		}
	}
	return p.parseTry(prog, parseChunk, parseChunk)
}

func (p *parser) parseBranchExpr(prog *program) error {
	return p.parseBranch(prog,
		p.parseExprBody,
		func(prog *program) error {
			prog.emit0(p.l, bcLoadNull)
			return nil
		},
	)
}

// a slightly better way to organize conditional expression

// notes, for easier code arrangment, we only allow condition to be ternary
// instead of expression to avoid nested if in if's condition. Since the if
// body has mandatory bracket, it means we can nest if inside and it is still
// fine to maintain
func (p *parser) parseExprStmt(prog *program) (bool, error) {
	switch p.l.token {
	case tkLet:
		return false, p.parseVarDecl(prog, symtVar)
	case tkConst:
		return false, p.parseVarDecl(prog, symtConst)
	default:
		return true, p.parseExpr(prog)
	}
}

func (p *parser) parseExprBody(prog *program) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}
	p.l.next()

	popVal := false

	p.enterNormalScope()

	if p, err := p.parseExprStmt(prog); err != nil {
		return err
	} else {
		popVal = p
	}
	if !p.l.expectCurrent(tkSemicolon) {
		return p.l.toError()
	}

	for {
		p.l.next()

		// okay now it is possible that it is a statement or an expression
		if p.l.token == tkRBra {
			break
		}

		// it must follow an expression, so we should just pop the previous
		// generated expression here
		if popVal {
			prog.emit0(p.l, bcPop)
		}

		if p, err := p.parseExprStmt(prog); err != nil {
			return err
		} else {
			popVal = p
		}

		if !p.l.expectCurrent(tkSemicolon) {
			return p.l.toError()
		}
	}

	p.leaveScope()

	// make sure that the if statement does generate some value instead of nothing
	if !popVal {
		return p.err("the if expression scope does not generate any value, " +
			"please make sure the last statement inside if scope is a " +
			"expression instead of other statement like, let ")
	}

	if !p.l.expectCurrent(tkRBra) {
		return p.l.toError()
	}
	p.l.next()
	return nil
}

// notes, we use a new ternary expression, compact if expression statement,
// that uses if suffix/postfix like python
// expression if condition else value
func (p *parser) parseTernary(prog *program) error {
	if err := p.parseBin(prog); err != nil {
		return err
	}
	ntk := p.l.token

	if ntk == tkIf {
		patch_point := prog.patch(p.l)
		p.l.next()

		if err := p.parseBin(prog); err != nil {
			return err
		}

		// notes we need to use comma instead of colon, since clone is already used
		// as method call, otherwise the parser has ambigiuty which needs extra
		// rules to resolve
		if !p.l.expectCurrent(tkElse) {
			return p.l.toError()
		}
		p.l.next()

		jump_out := prog.patch(p.l)

		// ternary operator, jump when false
		prog.emit1At(p.l, patch_point, bcTernary, prog.label())

		if err := p.parseBin(prog); err != nil {
			return err
		}

		// lastly, patch jump out to here
		prog.emit1At(p.l, jump_out, bcJump, prog.label())
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
			jump_pos = prog.patch(p.l)
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
			prog.emit0(p.l, bcAdd)
			break

		case tkSub:
			prog.emit0(p.l, bcSub)
			break

		case tkMul:
			prog.emit0(p.l, bcMul)
			break

		case tkDiv:
			prog.emit0(p.l, bcDiv)
			break

		case tkMod:
			prog.emit0(p.l, bcMod)
			break

		case tkPow:
			prog.emit0(p.l, bcPow)
			break

		case tkLt:
			prog.emit0(p.l, bcLt)
			break

		case tkLe:
			prog.emit0(p.l, bcLe)
			break

		case tkGt:
			prog.emit0(p.l, bcGt)
			break

		case tkGe:
			prog.emit0(p.l, bcGe)
			break

		case tkEq:
			prog.emit0(p.l, bcEq)
			break

		case tkNe:
			prog.emit0(p.l, bcNe)
			break

		case tkRegexpMatch:
			prog.emit0(p.l, bcRegexpMatch)
			break

		case tkRegexpNMatch:
			prog.emit0(p.l, bcRegexpNMatch)
			break

		case tkAnd:
			must(jump_pos >= 0, "and")
			prog.emit1At(p.l, jump_pos, bcAnd, prog.label())
			break

		case tkOr:
			must(jump_pos >= 0, "or")
			prog.emit1At(p.l, jump_pos, bcOr, prog.label())
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
		prog.emit0(p.l, bcNegate)
		break

	case tkNot:
		prog.emit0(p.l, bcNot)
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
		prog.emit1(p.l, bcLoadInt, idx)
		break

	case tkReal:
		idx := prog.addReal(l.rval)
		prog.emit1(p.l, bcLoadReal, idx)
		break

	case tkStr, tkMStr:
		strV := l.sval
		return p.parseStrInterpolation(prog, strV)

	case tkTrue:
		prog.emit0(p.l, bcLoadTrue)
		break

	case tkFalse:
		prog.emit0(p.l, bcLoadFalse)
		break

	case tkNull:
		prog.emit0(p.l, bcLoadNull)
		break

	case tkFunction:
		funcName, err := p.parseFunction(true)
		if err != nil {
			return err
		}
		scallIdx := p.policy.getFunctionIndex(funcName)
		must(scallIdx != -1, "the function(anonymouse) must be existed")
		prog.emit1(p.l, bcNewClosure, scallIdx)
		break

	case tkRegex:
		idx, err := prog.addRegexp(l.sval)
		if err != nil {
			return err
		}
		prog.emit1(p.l, bcLoadRegexp, idx)
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

	case tkDollar, tkId, tkCId, tkSId, tkDId:
		if err := p.parsePExpr(prog, tk, l.sval); err != nil {
			return err
		}
		break

	// intrinsic related expressions
	case tkTemplate:
		return p.parseTemplate(prog)

	default:
		return p.errf("unexpected token %s in expression parsing", p.l.tokenName())
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

		prog.emit0(p.l, bcNewPair)
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
				return -1, err
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
		prog.emit1(p.l, bcLoadStr, idx)
	}

	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}
	prog.emit1(p.l, bc, pcnt)
	return nil
}

func (p *parser) parseICall(prog *program, index int) error {
	{
		idx := prog.addInt(int64(index))
		prog.emit1(p.l, bcLoadInt, idx)
	}

	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}
	prog.emit1(p.l, bcICall, pcnt)
	return nil
}

// Scripting call frame is as following
// [argN]            <-- tos
// [argN-1]
// [arg1]
// [functionIndex]
func (p *parser) parseMCall(prog *program, name string) error {
	if !p.l.expectCurrent(tkLPar) {
		return p.l.toError()
	}
	return p.parseCall(prog, name, bcMCall)
}

func (p *parser) parseVCall(prog *program) error {
	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}
	prog.emit1(p.l, bcVCall, pcnt)
	return nil
}

func (p *parser) parseFreeCall(prog *program, name string) error {
	entryIndex := prog.patch(p.l)

	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}

	p.callPatch = append(p.callPatch, callentry{
		entryPos: entryIndex,
		callPos:  prog.patch(p.l),
		prog:     prog,
		arg:      pcnt,
		symbol:   name,
	})
	return nil
}

func (p *parser) patchAllCall() {
	for _, e := range p.callPatch {
		intrinsicIdx := indexIntrinsic(e.symbol)
		if intrinsicIdx != -1 {
			idx := e.prog.addInt(int64(intrinsicIdx))
			e.prog.emit1At(p.l, e.entryPos, bcLoadInt, idx)
			e.prog.emit1At(p.l, e.callPos, bcICall, e.arg)
		} else {
			scallIdx := p.policy.getFunctionIndex(e.symbol)
			if scallIdx != -1 {
				idx := e.prog.addInt(int64(scallIdx))
				e.prog.emit1At(p.l, e.entryPos, bcLoadInt, idx)
				e.prog.emit1At(p.l, e.callPos, bcSCall, e.arg)
			} else {
				idx := e.prog.addStr(e.symbol)
				e.prog.emit1At(p.l, e.entryPos, bcLoadStr, idx)
				e.prog.emit1At(p.l, e.callPos, bcCall, e.arg)
			}
		}
	}
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
			ntk := p.l.next()
			if ntk == tkId || ntk == tkStr {
				name := p.l.valueText
				*lastType = suffixDot
				idx := prog.addStr(name)
				prog.emit1(p.l, bcDot, idx)
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
			prog.emit0(p.l, bcIndex)
			if p.l.token != tkRSqr {
				return p.err("invalid expression, expect ] to close index")
			}
			p.l.next()
			break

		case tkColon:
			if !p.l.expect(tkId) {
				return p.l.toError()
			}
			p.l.next()

			method := p.l.valueText
			*lastType = suffixMethod
			if err := p.parseMCall(prog, method); err != nil {
				return err
			}
			break

		case tkLPar:
			if err := p.parseVCall(prog); err != nil {
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
			if err := p.parseFreeCall(prog, name); err != nil {
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
			if err := p.parseFreeCall(prog, modFuncName(modName, funcName)); err != nil {
				return err
			}
			break

		default:
			// Resolve the identifier here, notes the symbol can be following types
			// 1. session variable
			// 2. local variable
			// 3. dynmaic variable
			symT, symIdx := p.resolveSymbol(name, false, false)
			must(symIdx != symInvalidConst, "should never return invalid constant")

			switch symT {
			case symSession:
				prog.emit1(p.l, bcLoadSession, symIdx)
				break

			case symLocal:
				prog.emit1(p.l, bcLoadLocal, symIdx)
				break

			case symUpval:
				prog.emit1(p.l, bcLoadUpvalue, symIdx)
				break

			case symConst:
				prog.emit1(p.l, bcLoadConst, symIdx)

			default:
				prog.emit1(p.l, bcLoadVar, prog.addStr(name))
				break
			}
			break
		}

		// check whether we have suffix experssion
		switch p.l.token {
		case tkDot, tkLSqr, tkLPar:
			break

		default:
			return nil
		}
		break

	case tkDollar:
		prog.emit0(p.l, bcLoadDollar)
		break

	case tkSId:
		// session symbol table
		idx := p.findSessionIdx(name)
		if idx == -1 {
			return p.errf("session variable %s is unknown", name)
		}
		prog.emit1(p.l, bcLoadSession, idx)
		break

	case tkCId:
		idx := p.findConstIdx(name)
		if idx == -1 {
			return p.errf("const variable %s is unknown", name)
		}
		prog.emit1(p.l, bcLoadConst, idx)
		break

	default:
		gname := name

		idx := prog.addStr(gname)
		prog.emit1(p.l, bcLoadVar, idx)
		break
	}

	return nil
}

// list literal
func (p *parser) parseList(prog *program) error {
	prog.emit0(p.l, bcNewList)
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
		prog.emit1(p.l, bcAddList, cnt)
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

		prog.emit1(p.l, bcTemplate, idx)
		return nil
	}
}

func (p *parser) parseMap(prog *program) error {
	prog.emit0(p.l, bcNewMap)
	if p.l.token == tkRBra {
		p.l.next()
	} else {
		cnt := 0
		for {
			cnt++

			if p.l.token == tkStr || p.l.token == tkId {
				idx := prog.addStr(p.l.valueText)
				prog.emit1(p.l, bcLoadStr, idx)
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

		prog.emit1(p.l, bcAddMap, cnt)
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

	// duplicate parent parser's states, FIXME(dpeng): make the following function
	// into a separate function to allow derived parser from parent parser
	pp.stbl = p.stbl
	pp.constVar = p.constVar
	pp.sessVar = p.sessVar
	pp.policy = p.policy

	// allow DRBra to be parsed as symbol, otherwise will never be able to met the
	// end of the expression
	pp.l.allowDRBra = true

	// notes, we should start the lexer here, otherwise it will not work --------
	pp.l.next()

	if err := pp.parseExpr(prog); err != nil {
		return 0, p.errf("string interpolation error: %s", err.Error())
	}

	// the parser should always stop at the token DRBra, otherwise it means we
	// have dangling text cannot be consumed by the parser
	if pp.l.token != tkDRBra {
		return 0, p.err("invalid string interpolation part, expect }}")
	}

	p.callPatch = append(p.callPatch, pp.callPatch...)

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
		prog.emit1(p.l, bcLoadStr, idx)
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
			prog.emit0(p.l, bcToStr)
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
					prog.emit1(p.l, bcLoadStr, sidx)
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
		prog.emit1(p.l, bcLoadStr, sidx)
		strCnt++
	}

	if strCnt > 1 {
		prog.emit1(p.l, bcConStr, strCnt)
	}

	return nil
}
