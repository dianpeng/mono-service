package pl

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"

	"github.com/dianpeng/mono-service/util"
)

// lexical scope representation for scopping rules, store local variables per
// lexical scope, storing scope type and check whether we are inside of the
// loop which means certain statement may become valid, ie break, continue etc..

const (
	varPlaceholder = "_"
)

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
	entryConfig
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
	if !isTop && p != nil {
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
	must(s.find(x) == symError, "forget to put variable check?")

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
	swapPos  int
	callPos  int
	prog     *program
	arg      int
	symbol   string
}

type modParser struct {
	path    string // path to the module that is been parsing
	modName string // module's prefix name

	// these fields indicates the offset inside of the symbol table when we
	// start to parse the module. We will need to patch the symbol name once
	// the module parsing is done
	gIndex int
	sIndex int
	fIndex int
}

type importInfo struct {
	path   string
	prefix string
}

type parser struct {
	l      *lexer
	module *Module
	stbl   *lexicalScope

	globalVar []string
	sessVar   []string
	callPatch []callentry

	counter uint64 // random counter used to generate unique id etc ...

	// module parsing related. The module is essentially just like include in
	// C/C++ style language. We don't really have compiled global module but
	// instead we import each module like copy paste the code right here but
	// with symbol manipulation. All the import code will not result in different
	// Module object but will be merged into the same module with different
	// symbol name for external binding.
	mlist      []*modParser
	importList []importInfo

	// intermediate results
	modModName    string
	modImportPath string

	// helpers
	fs fs.FS
}

func newParser(input string, fs fs.FS) *parser {
	return &parser{
		l:      newLexer(input),
		module: newModule(),
		fs:     fs,
	}
}

func (p *parser) parsingMod() bool {
	return len(p.mlist) != 0
}

func (p *parser) curMod() *modParser {
	return p.mlist[len(p.mlist)-1]
}

const (
	symSession = iota
	symLocal
	symUpval
	symFunc
	symGlobal
	symDynamic
)

func (p *parser) uid() string {
	p.counter++
	return fmt.Sprintf("@uid_%d", p.counter)
}

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

func (p *parser) varName(x string) string {
	// placeholder name, then just randomly generate a unique name and return
	if x == varPlaceholder {
		return p.uid()
	}
	return x
}

func (p *parser) defLocalVar(x string) int {
	x = p.varName(x)

	idx := p.stbl.find(x)
	if idx != symError {
		return symError
	}
	return p.stbl.addVar(x)
}

func (p *parser) defLocalConst(x string) int {
	x = p.varName(x)

	idx := p.stbl.find(x)
	if idx != symError {
		return symError
	}
	return p.stbl.addConst(x)
}

func (p *parser) mustAddLocalVar(x string) int {
	idx := p.defLocalVar(x)
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

func (p *parser) findGlobalIdx(x string) int {
	for idx, n := range p.globalVar {
		if n == x {
			return idx
		}
	}
	return symError
}

// symbol resolution, ie binding
func (p *parser) resolveSymbol(xx string, forWrite bool) (int, int) {
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

	if !forWrite {
		// finding out the global function with matching symbol name
		funcIdx := p.module.getFunctionIndex(xx)
		if funcIdx != -1 {
			return symFunc, funcIdx
		}
	}

	// session symbol table
	sessVar := p.findSessionIdx(xx)
	if sessVar != symError {
		return symSession, sessVar
	}

	// global symbol table
	globalVar := p.findGlobalIdx(xx)
	if globalVar != symError {
		return symGlobal, globalVar
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

func (p *parser) parseExpression() (*Module, error) {
	p.l.next()
	prog := newProgram(p.module, "$expression", progExpression)
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

	p.module.p = append(p.module.p, prog)

	p.patchAllCall()

	return p.module, nil
}

func (p *parser) parse() (*Module, error) {
	p.l.next()
	if err := p.parseEntry(); err != nil {
		return nil, err
	}
	return p.module, nil
}

func (p *parser) parseTopVarScope() error {
	// top of the program allow user to write 2 types of special scope
	//
	// 1) Global scope, ie definition of global variable which is initialized once
	//    the program is been parsed, ie right inside of the parser
	//
	// 2) Session Scope, which should be reinitialized once the session is done.
	//    The definition of session is defined by user
	if p.l.token == tkGlobal {
		if err := p.parseGlobalScope(); err != nil {
			return err
		}
	} else if p.l.token == tkConfig {
		p.l.next()
		if err := p.parseConfig(); err != nil {
			return err
		}
	}

	if p.l.token == tkSession {
		if err := p.parseSessionScope(); err != nil {
			return err
		}
	} else if p.l.token == tkConfig {
		p.l.next()
		if err := p.parseConfig(); err != nil {
			return err
		}
	}

	return nil
}

// ----------------------------------------------------------------------------
// Module parsing
// Module in PL is just a syntax sugar. It is almost the same as paste the code
// from the import tree right inside of our source tree and parse them just as
// if the symbol is in the file. Except the parser will rename the symbol with
// module syntax to allow them to be used.
//
// The naming convention of module is as following. Each module will have a
// module name as namespace. By default each module's symbol can only be visiable
// by using mod_name::symbol_name style module naming. Additionally, all import
// module's symbol, as long as the symbol is exporting, will be exported to the
// include module. It means if root file import module A and module A import
// module B. Then root file will aslo have module B's symbol. This is just for
// simplicity
//
// If multiple duplicates import is included, then the parser will automatically
// ignore the same name (same path) module by default.

func (p *parser) isImported(
	path string,
) bool {
	for _, x := range p.importList {
		if x.path == path {
			return true
		}
	}
	return false
}

func (p *parser) modPrefixExist(
	name string,
) bool {
	for _, x := range p.importList {
		if x.prefix == name {
			return true
		}
	}
	return false
}

func (p *parser) addImportInfo(
	path string,
	prefix string,
) {
	p.importList = append(p.importList,
		importInfo{
			path:   path,
			prefix: prefix,
		},
	)
}

func (p *parser) parseModDec() error {
	if p.l.token != tkModule {
		fmt.Printf("xx: %s\n", p.l.tokenName())
		return p.errf("parsing imported module: %s, first statement must be "+
			"module declaration", p.modImportPath)
	}
	if !p.l.expect(tkId) {
		return p.l.toError()
	}

	prefix := p.l.valueText
	p.l.next()

	if mname, err := p.parseModSymbol(
		prefix,
	); err != nil {
		return err
	} else {
		p.modModName = mname.fullname()

		if p.modPrefixExist(p.modModName) {
			return p.errf("module from file %s, its module name %s is already existed",
				p.modImportPath,
				p.modModName,
			)
		}

		p.addImportInfo(
			p.modImportPath,
			p.modModName,
		)

		// checking whether we have module prefix collision or not, if we have
		// then just reporting the error

		return nil
	}
}

func (p *parser) importLoop() error {
	for _, v := range p.mlist {
		if v.path == p.modImportPath {
			return p.errf("the module %s has been imported already, cycle detected",
				p.modImportPath,
			)
		}
	}

	return nil
}

func (p *parser) readFile(path string) (string, error) {
	if p.fs != nil {
		x, err := fs.ReadFile(p.fs, p.modImportPath)
		if err != nil {
			return "", err
		} else {
			return string(x), nil
		}
	} else {
		return util.LoadFile(path)
	}
}

func (p *parser) doImport() error {
	// try to check whether we have a cycle
	if err := p.importLoop(); err != nil {
		return err
	}

	// try to check wether the import path has already been done
	if p.isImported(p.modImportPath) {
		// skip this module since it has been imported already
		return nil
	}

	// now start to perform the import operation
	data, err := p.readFile(p.modImportPath)
	if err != nil {
		return p.errf("cannot load import file from path %s: %s", p.modImportPath, err.Error())
	}

	// save the lexer
	savedL := p.l
	defer func() {
		p.l = savedL
	}()

	p.l = newLexer(data)
	p.l.next()

	// start to parse the imported module
	if err := p.parseModule(); err != nil {
		return err
	}

	return nil
}

func (p *parser) parseImportStmt() error {
	must(p.l.token == tkImport, "must be import")
	p.l.next()
	if p.l.token == tkStr {
		p.modImportPath = p.l.valueText
		p.l.next()

		return p.doImport()
	} else if p.l.token == tkLPar {
		p.l.next()
		if p.l.token == tkRPar {
			return p.err("nothing has been imported, do you forget to write import path ?")
		}

		for {
			if !p.l.expectCurrent(tkStr) {
				return p.l.toError()
			}
			p.modImportPath = p.l.valueText
			p.l.next()

			if err := p.doImport(); err != nil {
				return err
			}

			if p.l.token == tkRPar {
				p.l.next()
				break
			}
		}

		return nil
	} else {
		return p.err("expect a string as module path or '(' to start a group of " +
			"paths after import")
	}
}

func (p *parser) parseImport() error {
	for p.l.token == tkImport {
		if err := p.parseImportStmt(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) startModule() error {
	p.mlist = append(
		p.mlist,
		&modParser{
			path:    p.modImportPath,
			modName: p.modModName,
			gIndex:  len(p.globalVar),
			sIndex:  len(p.sessVar),
			fIndex:  len(p.module.fn),
		},
	)

	return nil
}

func (p *parser) endModule() error {
	cmod := p.curMod()

	// (0) patch all the global variable symbol name to include the module prefix
	{
		sz := len(p.globalVar)
		for i := cmod.gIndex; i < sz; i++ {
			p.globalVar[i] = modSymbolName(
				cmod.modName,
				p.globalVar[i],
			)
		}
	}

	// (1) patch all the session variable symbol name to include the module prefix
	{
		sz := len(p.sessVar)
		for i := cmod.sIndex; i < sz; i++ {
			p.sessVar[i] = modSymbolName(
				cmod.modName,
				p.sessVar[i],
			)
		}
	}

	// (2) patch all the function symbol name to include the module prefix
	{
		sz := len(p.module.fn)
		for i := cmod.fIndex; i < sz; i++ {
			oldName := p.module.fn[i].name
			p.module.fn[i].name = modSymbolName(
				cmod.modName,
				oldName,
			)
		}
	}

	p.mlist = p.mlist[:len(p.mlist)-1]
	return nil
}

func (p *parser) parseModule() error {
	// handle first module declaration to get module prefix name
	if err := p.parseModDec(); err != nil {
		return err
	}

	// push the module object
	if err := p.startModule(); err != nil {
		return err
	}

	// handle import
	if err := p.parseImport(); err != nil {
		return err
	}

	// handle top var scope
	if err := p.parseTopVarScope(); err != nil {
		return err
	}

	// handle rest of the top level statement
LOOP:
	for {
		tk := p.l.token
		switch tk {
		case tkFunction:
			// eat the fn token, parseFunction expect after the fn keyword
			p.l.next()
			if _, err := p.parseFunction(false); err != nil {
				return err
			}
			break

		case tkError:
			return p.l.toError()

		default:
			if tk != tkEof {
				// print a little bit better diagnostic information, well, in general
				// we should invest more to error reporting
				switch tk {
				case tkSession:
					return p.errf("session section should be put at the top, after global scope")
				case tkGlobal:
					return p.errf("global section should be put at the top")
				default:
					return p.errf("unknown token: %s, unrecognized statement", p.l.tokenName())
				}
			} else {
				break LOOP
			}
		}
	}

	// patch all the pending call instructions
	p.patchAllCall()

	// lastly patch all the module symbol
	if err := p.endModule(); err != nil {
		return err
	}

	// done
	return nil
}

func (p *parser) parseEntry() error {
	if err := p.parseImport(); err != nil {
		return err
	}

	if err := p.parseTopVarScope(); err != nil {
		return err
	}

	// top most statement allowed ------------------------------------------------
LOOP:
	for {
		tk := p.l.token
		switch tk {
		case tkRule, tkId, tkStr, tkLSqr:
			if err := p.parseRule(); err != nil {
				return err
			}
			break

		case tkConfig:
			// try to parse the config section, which will be merged into @config
			// rule for internal use cases.
			p.l.next()
			if err := p.parseConfig(); err != nil {
				return err
			}
			break

		case tkFunction:
			// eat the fn token, parseFunction expect after the fn keyword
			p.l.next()
			if _, err := p.parseFunction(false); err != nil {
				return err
			}
			break

		case tkError:
			return p.l.toError()

		default:
			if tk != tkEof {
				// print a little bit better diagnostic information, well, in general
				// we should invest more to error reporting
				switch tk {
				case tkSession:
					return p.errf("session section should be put at the top, after global scope")
				case tkGlobal:
					return p.errf("global section should be put at the top")
				default:
					return p.errf("unknown token: %s, unrecognized statement", p.l.tokenName())
				}
			} else {
				break LOOP
			}
		}
	}

	p.patchAllCall()

	// lastly check whether we have config scope, if so, we need to patch config
	// to include a bcHalt otherwise will break our VM
	if p.module.config != nil {
		p.module.config.emit0(p.l, bcHalt)
	}

	return nil
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
	x, list, err := p.parseVarScope(ConfigRule,
		func(_ string, prog *program, p *parser) {
			prog.emit0(p.l, bcSetSession)
		},
	)

	if err != nil {
		return err
	}

	p.sessVar = append(p.sessVar, list...)
	p.module.addSessionProgram(x, list)
	return nil
}

func (p *parser) parseGlobalScope() error {
	x, list, err := p.parseVarScope(GlobalRule,
		func(_ string, prog *program, p *parser) {
			prog.emit0(p.l, bcSetGlobal)
		},
	)

	if err != nil {
		return err
	}

	p.globalVar = append(p.globalVar, list...)
	p.module.addGlobalProgram(x, list)
	return nil
}

func (p *parser) parseVarScope(rulename string,
	gen func(string, *program, *parser)) (*program, []string, error) {
	if !p.l.expect(tkLBra) {
		return nil, nil, p.l.toError()
	}
	p.l.next()

	prog := newProgram(p.module, rulename, progSession)

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

			// allow placeholder inside of the var scope
			gname := p.varName(p.l.valueText)

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

// checking function name collision
func (p *parser) functionExists(
	fname string,
) bool {
	offset := 0
	if p.parsingMod() {
		offset = p.curMod().fIndex
	}
	for i := offset; i < len(p.module.fn); i++ {
		if p.module.fn[i].name == fname {
			return true
		}
	}
	return false
}

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
		funcName = p.module.nextAnonymousFuncName()
	}

	if p.functionExists(funcName) {
		return "", p.errf("function %s is already existed", funcName)
	}

	prog := newProgram(p.module, funcName, progFunc)
	p.enterScopeTop(entryFunc, prog)
	defer func() {
		p.leaveScope()
	}()

	localR := prog.patch(p.l)
	p.module.fn = append(p.module.fn, prog)
	defer func() {
		if the_error != nil {
			sz := len(p.module.fn)
			p.module.fn = p.module.fn[:sz-1]
		}
	}()

	// (0) Parsing function's argument list section

	if !p.l.expectCurrent(tkLPar) {
		return "", p.l.toError()
	}

	// parsing the function's argument and just treat them as
	if p.l.next() != tkRPar {
		argcnt := 0
		for {
			if !p.l.expectCurrent(tkId) {
				return "", p.l.toError()
			}
			if p.l.valueText == varPlaceholder {
				return "", p.err("argument name cannot be defined as placeholder")
			}

			idx := p.defLocalVar(p.l.valueText)
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

	// After all the argument been reserved, lastly reserve a frame slot to maintain
	// ABI
	p.mustAddLocalVar("#frame")

	// (1) Parsing function body

	// We currently support 2 types of function style, one is one line short
	// function definition and the other is full function definition

	if p.l.token == tkColon {
		p.l.next()
		// one liner function will have to return an expression
		if err := p.parseExpr(prog); err != nil {
			return "", err
		}
		prog.emit0(p.l, bcReturn)
	} else {
		// now parsing the function body
		if err := p.parseBody(prog); err != nil {
			return "", err
		}
		// always generate a return nil at the end of the function
		prog.emit0(p.l, bcLoadNull)
		prog.emit0(p.l, bcReturn)
	}

	// (2) Finish the function population
	prog.localSize = p.stbl.topMaxLocal() - 1
	prog.emit1At(p.l, localR, bcReserveLocal, p.stbl.topMaxLocal()-1)

	return funcName, nil
}

func (p *parser) parseRule() error {
	if p.l.token == tkRule {
		p.l.next()
	}

	var name string

	// parsing the module name, we accept few different types to ease user's
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
		return p.err("unexpected token here! A rule *MUST* have an identifier " +
			"to represent event name, can be a string literal, an " +
			"identifier or a '[' id ']' for documentation")
	}

	// notes the name must be none empty and also must not start with @, which is
	// builtin event name
	if name == "" {
		return p.err("invalid rule name, cannot be empty string")
	}
	if name[0] == '@' {
		return p.err("invalid rule name, cannot start with @ which is builtin name")
	}

	prog := newProgram(p.module, name, progRule)
	p.enterScopeTop(entryRule, prog)

	p.mustAddLocalVar("#event")
	p.mustAddLocalVar("#frame")
	defer func() {
		p.leaveScope()
	}()

	localR := prog.patch(p.l)

	if !p.module.addEvent(name, prog) {
		return p.err("invalid rule name, the rule has already been existed")
	}

	// Allow an optional arrow to indicate this is a rule, this is the preferred
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
	p.module.p = append(p.module.p, prog)

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

		case symGlobal:
			must(symIdx >= 0, "must be valid index")
			prog.emit1(p.l, bcLoadGlobal, symIdx)
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

	case symGlobal:
		prog.emit1(p.l, bcStoreGlobal, symIdx)
		break

	default:
		prog.emit1(p.l, bcStoreVar, prog.addStr(symName))
		break
	}

	return nil
}

func (p *parser) parseBasicStmt(prog *program, lexeme lexeme, popExpr bool) (bool, error) {

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
			sym, idx := p.resolveSymbol(lexeme.sval, true)
			must(sym != symFunc, "cannot be function index")

			if sym == symLocal && idx == symInvalidConst {
				return true, p.errf("local variable %s is const, cannot be assigned to", lexeme.sval)
			}

			return true, p.parseSymbolAssign(prog,
				lexeme.sval,
				sym,
				idx,
			)

			// session store
		case tkSId:
			idx := p.findSessionIdx(lexeme.sval)
			if idx == symError {
				return true, p.errf("session %s variable not existed", lexeme.sval)
			}

			return true, p.parseSymbolAssign(prog,
				lexeme.sval,
				symSession,
				idx,
			)

			// const store
		case tkGId:
			return true, p.errf("global scope variable cannot be used for storing")

			// dynamic variable store
		case tkDId:
			return true, p.parseSymbolAssign(prog,
				lexeme.sval,
				symDynamic,
				symError,
			)

		default:
			return true, p.err("assignment's lhs must be identifier or a session identifier")
		}
		return true, nil

	default:
		// other prefix expression
		break
	}

	// Suffix expression assignment, agg-assignment, inc and dec -----------------

	// now try to generate the previous saved lexeme and it could be anything
	if err := p.parsePrimary(prog, lexeme); err != nil {
		return false, err
	}

	// then try to parse any suffix expression if applicable
	var st int
	if err := p.parseSuffixImpl(prog, &st); err != nil {
		return false, err
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
			return true, p.err("invalid assignment expression, the field assignment " +
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
				return true, err
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
		return true, nil
	} else {
		// try to consume rest as expression like grammar until we cannot eat any
		// tokens
		if err := p.parseBasicStmtExprRest(prog); err != nil {
			return false, err
		}

		// pop the last expression just been evaluated
		if popExpr {
			prog.emit0(p.l, bcPop)
		}
		return false, nil
	}
}

// this is used to catch up cases when the statement is not a statement, ie
// not an assignment statement but really an expression. This is hard for
// our case since it requires multiple look aheads to decide and we are one
// pass generator/compiler which does not yield AST sorts of things. We cannot
// turn back. The following parser will try to pick up where we left and continue
// as an expression and generate expression code. It is relative complicated
func (p *parser) parseBasicStmtExprRest(prog *program) error {

	// (0) try to get the operator afterwards to see whether it is a bin
	if prec := p.binPrecedence(p.l.token); prec != -1 {
		// need to pick up until we hit a none binary operators
		for prec != -1 {
			if err := p.parseBinaryRest(prog, prec); err != nil {
				return err
			}
			prec = p.binPrecedence(p.l.token)
		}
	}

	// (1) after the section (0) we know that we have consumed all the possible
	//     binary operators, now we should try the ternary dangling, which is
	//     if like statement
	if err := p.parseTernaryRest(prog); err != nil {
		return err
	}

	// now we are done to consume up all the expression like statement and we
	// should not see any other valid dangling, theoretically :)

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
		_, err := p.parseBasicStmt(prog, lexeme, true)
		return err
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
	// add a frame of exception
	pushexp := prog.patch(p.l)

	// generate try body
	if err := tryGen(prog); err != nil {
		return err
	}

	// exception region finished, and jump out of the else branch
	popexp := prog.patch(p.l)

	// patch the enter exception
	prog.emit1At(p.l, pushexp, bcPushException, prog.label())

	// else branch
	if !p.l.expectCurrent(tkElse) {
		return p.l.toError()
	}
	p.l.next()

	// optionally let, which allow user to capture the exception/error
	if p.l.token == tkLet {
		if !p.l.expect(tkId) {
			return p.l.toError()
		}
		idx := p.defLocalVar(p.l.valueText)
		if idx == symError {
			return p.errf("duplicate local variable: %s", p.l.valueText)
		}
		p.l.next()
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

	p.l.next()
	return p.parseTry(prog, parseChunk, parseChunk)
}

func (p *parser) parseBranch(prog *program,
	bodyGen func(*program) error, /* invoked whenever a branch body is needed */
	dangling func(*program) error, /* invoked whenever a branch does have dangling,
	   notes, could be elif or else */
) error {
	var jump_out []int
	var prev_jmp int

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
	p.l.next()
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

	// make sure to add the variable before parsing/compiling its rhs expression
	// to allow recursive anonymous function call
	var idx int
	if symt == symtVar {
		idx = p.defLocalVar(p.l.valueText)
	} else {
		idx = p.defLocalConst(p.l.valueText)
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
func (p *parser) parseForeverLoop(prog *program, bodyGen func(*program) error) error {
	p.enterLoopScope()

	loopHeader := prog.label()

	if err := bodyGen(prog); err != nil {
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
func (p *parser) parseIteratorLoop(prog *program, key string, bodyGen func(*program) error) error {
	must(p.l.token == tkComma, "must be comma")

	// 1. parsing the value local variable name
	if !p.l.expect(tkId) {
		return p.l.toError()
	}
	val := p.l.valueText

	// 2. parsing the iterator expression
	if !p.l.expect(tkAssign) {
		return p.l.toError()
	}
	p.l.next()

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
	valIdx := p.defLocalVar(val)

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
	if err := bodyGen(prog); err != nil {
		return err
	}

	// patch continue to jump at the end of the loop
	p.patchContinue(prog)

	// 4.3 loop tail check, if the iterator can move forward, then just move forwad
	//     and return true, otherwise return false
	prog.emit0(p.l, bcNextIterator)
	prog.emit1(p.l, bcJtrue, headerPos)

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
func (p *parser) parseTripCountLoopRest(prog *program, bodyGen func(*program) error) error {
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

		if _, err := p.parseBasicStmt(prog, lexeme, true); err != nil {
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

	if err := bodyGen(prog); err != nil {
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
func (p *parser) parseTripCountLoopFull(prog *program, key string, bodyGen func(*program) error) error {
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
	return p.parseTripCountLoopRest(prog, bodyGen)
}

func (p *parser) parseTripCountLoopNoInit(prog *program, bodyGen func(*program) error) error {
	must(p.l.token == tkSemicolon, "must be ;")
	p.enterLoopScope()
	defer func() {
		p.leaveScope()
	}()

	return p.parseTripCountLoopRest(prog, bodyGen)
}

func (p *parser) parseIteratorOrTripLoop(prog *program, bodyGen func(*program) error) error {
	must(p.l.token == tkLet, "must be let")

	if !p.l.expect(tkId) {
		return p.l.toError()
	}
	key := p.l.valueText

	p.l.next()

	if p.l.token == tkComma {
		return p.parseIteratorLoop(prog, key, bodyGen)
	} else if p.l.token == tkAssign {
		return p.parseTripCountLoopFull(prog, key, bodyGen)
	} else {
		return p.err("invalid token, expect a ',' for a range for loop or " +
			"a '=' to start a trip count foreach's initial statement")
	}
}

// 3) condition loop
func (p *parser) parseConditionLoop(prog *program, bodyGen func(*program) error) error {

	loopHeader := prog.label()

	// Get the condition on the TOS
	if err := p.parseExpr(prog); err != nil {
		return err
	}

	// jump out of the loop if needed, via bcJfalse
	jumpOut := prog.patch(p.l)

	// lastly, parse the body of the loop
	p.enterLoopScope()
	if err := bodyGen(prog); err != nil {
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

func (p *parser) parseForImpl(prog *program, bodyGen func(*program) error) error {
	p.l.next()

	switch p.l.token {

	case tkLBra:
		// forever loop,
		// for {
		//   ...
		// }
		return p.parseForeverLoop(prog, bodyGen)

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
		return p.parseIteratorOrTripLoop(prog, bodyGen)

	case tkSemicolon:
		// trip count loop, but ignore the initial statement // ie
		// for ; condition; step {
		// }
		return p.parseTripCountLoopNoInit(prog, bodyGen)

	default:
		// condition loop, ie the expression been setup is been treated as a
		// while loop condition:
		// for condition {
		//   ..
		// }
		return p.parseConditionLoop(prog, bodyGen)
	}
}

func (p *parser) parseFor(prog *program) error {
	return p.parseForImpl(
		prog,
		p.parseBody,
	)
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

// parsing emit statement.
func (p *parser) parseEmit(prog *program) error {
	must(p.l.token == tkEmit, "must be emit")

	if !p.isEntryRule() {
		return p.err("emit statement is only allowed inside of rule body")
	}

	p.l.next()

	eventName := ""

	switch p.l.token {
	case tkId, tkStr:
		eventName = p.l.valueText
		break

	default:
		break
	}
	p.l.next()

	// emit a local variable contain an eventName string
	prog.emit1(p.l, bcLoadStr, prog.addStr(eventName))

	// optionally we may have a ',' to indicate there's an arugment
	if p.l.token == tkComma {
		p.l.next()
		if err := p.parseExpr(prog); err != nil {
			return err
		}
	} else {
		prog.emit0(p.l, bcLoadNull)
	}

	// this is just for convinience, emit should not carry argument, but put a 1
	// there to indicate one argument is easy for us to do simulation
	prog.emit1(p.l, bcEmit, 1)
	return nil
}

func (p *parser) parseBodyStmt(prog *program) (bool, error) {
	hasSep := true

	switch p.l.token {
	case tkSemicolon:
		break

	case tkLBra:
		p.enterNormalScope()
		if err := p.parseBody(prog); err != nil {
			return false, err
		}
		p.leaveScope()
		hasSep = false
		break

	case tkLet:
		if err := p.parseVarDecl(prog, symtVar); err != nil {
			return false, err
		}
		break

	case tkConst:
		if err := p.parseVarDecl(prog, symtConst); err != nil {
			return false, err
		}
		break

	case tkIf:
		if err := p.parseBranchStmt(prog); err != nil {
			return false, err
		}
		hasSep = false
		break

	case tkTry:
		if err := p.parseTryStmt(prog); err != nil {
			return false, err
		}
		hasSep = false
		break

	case tkFor:
		if err := p.parseFor(prog); err != nil {
			return false, err
		}
		hasSep = false
		break

	case tkBreak:
		if err := p.parseBreak(prog); err != nil {
			return false, err
		}
		break

	case tkContinue:
		if err := p.parseContinue(prog); err != nil {
			return false, err
		}
		break

	case tkReturn:
		if err := p.parseReturn(prog); err != nil {
			return false, err
		}
		break

	case tkEmit:
		if err := p.parseEmit(prog); err != nil {
			return false, err
		}
		break

	default:
		if err := p.parseStmt(prog); err != nil {
			return false, err
		}
		break
	}

	return hasSep, nil
}

func (p *parser) parseBodyImpl(prog *program,
	stmt func(*program) (bool, error)) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}
	p.l.next()

	if p.l.token == tkRBra {
		p.l.next()
		return nil
	}

	for {
		hasSep, err := stmt(prog)
		if err != nil {
			return err
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

func (p *parser) parseBody(prog *program) error {
	return p.parseBodyImpl(
		prog,
		p.parseBodyStmt,
	)
}

// Expression parsing, using simple precedence climbing style way
func (p *parser) parseExpr(prog *program) error {
	return p.parseTernary(prog)
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
func (p *parser) parseExprStmt(prog *program) (bool, bool, error) {
	switch p.l.token {
	case tkLet:
		return true, false, p.parseVarDecl(prog, symtVar)
	case tkConst:
		return true, false, p.parseVarDecl(prog, symtConst)
	case tkFor:
		return false, false, p.parseFor(prog)
	case tkSemicolon:
		return true, false, nil

	default:
		lexeme := p.lexeme()
		p.l.next()

		isStmt, err := p.parseBasicStmt(prog, lexeme, false)
		if err != nil {
			return true, true, err
		} else {
			return true, !isStmt, nil
		}
	}
}

func (p *parser) parseExprBody(prog *program) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}
	p.l.next()

	popVal := false
	needSep := true

	p.enterNormalScope()

	if s, p, err := p.parseExprStmt(prog); err != nil {
		return err
	} else {
		popVal = p
		needSep = s
	}
	if needSep {
		if !p.l.expectCurrent(tkSemicolon) {
			return p.l.toError()
		} else {
			p.l.next()
		}
	}

	for {
		// okay now it is possible that it is a statement or an expression
		if p.l.token == tkRBra {
			break
		}

		// it must follow an expression, so we should just pop the previous
		// generated expression here
		if popVal {
			prog.emit0(p.l, bcPop)
		}

		if s, p, err := p.parseExprStmt(prog); err != nil {
			return err
		} else {
			popVal = p
			needSep = s
		}

		if needSep {
			if !p.l.expectCurrent(tkSemicolon) {
				return p.l.toError()
			} else {
				p.l.next()
			}
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
	return p.parseTernaryRest(prog)
}

func (p *parser) parseTernaryRest(prog *program) error {
	ntk := p.l.token
	if ntk == tkIf {
		p.l.next()

		// this is the if condition
		if err := p.parseBin(prog); err != nil {
			return err
		}

		// position to insert bcTernary instruction
		patch_point := prog.patch(p.l)

		if !p.l.expectCurrent(tkElse) {
			return p.l.toError()
		}
		p.l.next()

		// else branch generated value, when bcTernary evaluates to false, it will
		// jump here
		if err := p.parseBin(prog); err != nil {
			return err
		}

		// if condition evaluation to be true, then jumpping here, otherwise it
		// fallthrough to else branch experssion evaluation
		prog.emit1At(p.l, patch_point, bcTernary, prog.label())

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

	return p.parseBinaryRest(prog, prec)
}

func (p *parser) parseBinaryRest(prog *program, prec int) error {
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

func (p *parser) parseExprScope(prog *program) error {
	p.enterNormalScope()
	defer func() {
		p.leaveScope()
	}()

	isExpr := false

	if p.l.token != tkRBra {
		needSemicolon, ise, err := p.parseExprStmt(prog)
		if err != nil {
			return err
		}
		isExpr = ise

		// if the expression expects a ; afterwards, then we need to check it
		if needSemicolon {
			if !p.l.expectCurrent(tkSemicolon) {
				return p.l.toError()
			}
			p.l.next()
		}
	}

	for {
		if p.l.token == tkRBra {
			break
		}

		// pop previous expression if applicable
		if isExpr {
			// pop the expression out of the stack in case it is an expression and we
			// have following statement. If this is the last expression then we will
			// undo it at very last when we are
			prog.emit0(p.l, bcPop)
		}

		// next statement
		needSemicolon, ise, err := p.parseExprStmt(prog)
		if err != nil {
			return err
		}
		isExpr = ise

		// if the expression expects a ; afterwards, then we need to check it
		if needSemicolon {
			if !p.l.expectCurrent(tkSemicolon) {
				return p.l.toError()
			}
			p.l.next()
		}
	}

	if !isExpr {
		// lastone isnot an expression, then report error
		return p.err("expression scope's last statement must be an expression " +
			"which can generate a value. It cannot be things like " +
			"for loop, let/const declarations!")
	}

	// eat the last }
	must(p.l.token == tkRBra, "must be }")
	p.l.next()
	return nil
}

func (p *parser) parsePrimary(prog *program, l lexeme) error {
	tk := l.token
	switch tk {
	case tkIf:
		return p.parseBranchExpr(prog)

	case tkTry:
		return p.parseTryExpr(prog)

	case tkInt:
		idx := prog.addInt(l.ival)
		prog.emit1(p.l, bcLoadInt, idx)
		break

	case tkReal:
		idx := prog.addReal(l.rval)
		prog.emit1(p.l, bcLoadReal, idx)
		break

	case tkStr:
		strV := l.sval
		return p.parseStrInterpolation(prog, strV)

	// long string does not support interpolation
	case tkMStr:
		strV := l.sval
		prog.emit1(p.l, bcLoadStr, prog.addStr(strV))
		break

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
		scallIdx := p.module.getFunctionIndex(funcName)
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

	case tkDollar, tkId, tkGId, tkSId, tkDId:
		if err := p.parsePrefixExpr(prog, tk, l.sval); err != nil {
			return err
		}
		break

	case tkLExprBra:
		// experssion scope value, same as if true { ... }
		if err := p.parseExprScope(prog); err != nil {
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
func (p *parser) parseVCall(prog *program) error {
	pcnt, err := p.parseCallArgs(prog)
	if err != nil {
		return err
	}
	prog.emit1(p.l, bcVCall, pcnt)
	return nil
}

// module symbols
type modSymbol struct {
	prefix string
	rest   []string
}

func (m *modSymbol) fullname() string {
	if len(m.rest) == 0 {
		return m.prefix
	}
	x := []string{m.prefix}
	x = append(x, m.rest...)
	return strings.Join(x, modSep)
}

func (m *modSymbol) isModule() bool {
	return len(m.rest) > 0
}

// If the symbol is a::b::c, then this function returns a::b
func (m *modSymbol) modulePrefix() string {
	if len(m.rest) == 0 {
		return ""
	}
	return strings.Join(m.rest, modSep)
}

func (m *modSymbol) topModuleName() string {
	if len(m.rest) == 0 {
		return ""
	}
	return m.rest[0]
}

// Optionally parsing the module symbol
func (p *parser) parseModSymbol(
	prefix string,
) (modSymbol, error) {
	m := modSymbol{
		prefix: prefix,
	}

	if p.l.token == tkScope {
		p.l.next()

		for {
			if !p.l.expectCurrent(tkId) {
				return m, p.l.toError()
			}
			m.rest = append(m.rest, p.l.valueText)

			if p.l.next() != tkScope {
				break
			}
			p.l.next()
		}
	}

	return m, nil
}

// Rules for unbounded call name resolution

// The unbounded call's name resolution is little bit different from the normal
// symbol resolution. The difference is that when we don't know a name for
// variable loading/storing, we issue bcLoadVar/bcStoreVar; for function call
// we issue bcVCall.
// In general, the name of unbounded function call is as following:
// 1) try to resolve it as known symbol
//    ie local, session or const
// 2) If 1) failed, then defer the call when all parsing/compilation is done
// 3) Once all compilation is done, we try to lookup the name as following :
//   3.1) If it is an intrinsic function call, then just issue it
//   3.2) If it is a script function call, then just issue it
//   3.3) Otherwise, try to load the function name as dynamic variable on to
//        the stack and then issue the bcVCall
//
// NOTES(dpeng): bcCall is depracated

func (p *parser) parseUnboundCall(prog *program,
	modName modSymbol,
	argAdd int,
	isPipe bool,
) error {
	name := modName.fullname()

	entryIndex := prog.patch(p.l)
	swapPos := -1

	// if it is a pipecall, then we should swap the entry and its previous args
	if isPipe {
		swapPos = prog.patch(p.l)
	}

	// function call argument
	pcnt := 0

	if !isPipe || (isPipe && p.l.token == tkLPar) {
		pc, err := p.parseCallArgs(prog)
		if err != nil {
			return err
		}
		pcnt = pc
	}

	// now try to resolve the name via variable lookup in case it is just a vcall
	symtype, symindex := p.resolveSymbol(
		name,
		false,
	)

	switch symtype {
	case symSession:
		prog.emit1At(p.l, entryIndex, bcLoadSession, symindex)
		break

	case symLocal:
		prog.emit1At(p.l, entryIndex, bcLoadLocal, symindex)
		break

	case symFunc:
		prog.emit1At(p.l, entryIndex, bcNewClosure, symindex)
		break

	case symUpval:
		prog.emit1At(p.l, entryIndex, bcLoadUpvalue, symindex)
		break

	case symGlobal:
		prog.emit1At(p.l, entryIndex, bcLoadGlobal, symindex)
		break

	default:
		// notes if it is dynamic variable then just dispatch it with bcVCall later
		// on instead of eargly issue it
		p.callPatch = append(p.callPatch, callentry{
			entryPos: entryIndex,
			swapPos:  swapPos,
			callPos:  prog.patch(p.l),
			prog:     prog,
			arg:      pcnt + argAdd,
			symbol:   name,
		})
		return nil
	}

	if swapPos != -1 {
		prog.emit0At(p.l, swapPos, bcSwap)
	}

	// emit the final vcall
	prog.emit1(p.l, bcVCall, pcnt+argAdd)
	return nil
}

func (p *parser) patchAllCall() {
	for _, e := range p.callPatch {
		// whether we should perform swap due to the pipe call
		if e.swapPos != -1 {
			e.prog.emit0At(p.l, e.swapPos, bcSwap)
		}

		// The call entry in current stage is the call entry that its name cannot
		// be resolved during variable lookup, ie try to dispatch it via VCall

		// 1) try to index it as intrinsic function

		// 2) try to index it as script function which has an external name and not
		//    be bounded as local variable name

		// 3) try to dispatch it as dynamic function call, ie upcall

		intrinsicIdx := indexIntrinsic(e.symbol)
		if intrinsicIdx != -1 {
			idx := e.prog.addInt(int64(intrinsicIdx))
			e.prog.emit1At(p.l, e.entryPos, bcLoadInt, idx)

			e.prog.emit1At(p.l, e.callPos, bcICall, e.arg)
		} else {
			scallIdx := p.module.getFunctionIndex(e.symbol)
			if scallIdx != -1 {
				idx := e.prog.addInt(int64(scallIdx))
				e.prog.emit1At(p.l, e.entryPos, bcLoadInt, idx)
				e.prog.emit1At(p.l, e.callPos, bcSCall, e.arg)
			} else {
				// okay, the variable here is unknow to us, now let's just issue it
				// as dynamic variable loading and then call on this dynamic variable
				idx := e.prog.addStr(e.symbol)
				e.prog.emit1At(p.l, e.entryPos, bcLoadVar, idx)
				e.prog.emit1At(p.l, e.callPos, bcVCall, e.arg)
			}
		}
	}
}

const (
	suffixUnknown = iota
	suffixDot
	suffixIndex
	suffixCall
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
			*lastType = suffixMethod

			if !p.l.expect(tkId) {
				return p.l.toError()
			}
			method := p.l.valueText
			p.l.next()

			prog.emit1(p.l, bcLoadMethod, prog.addStr(method))
			break

		case tkLPar:
			*lastType = suffixCall

			if err := p.parseVCall(prog); err != nil {
				return err
			}
			break

		case tkPipe:
			*lastType = suffixCall

			// Pipe style expression
			// Pipe is used to invoke free/global function in a compact format,
			// case 1> a | b[(args)] => b(a, [args])
			// case 2> a | b => b(a)

			if !p.l.expect(tkId) {
				return p.l.toError()
			}

			name := p.l.valueText
			p.l.next()

			// optionally module name
			mname, err := p.parseModSymbol(name)
			if err != nil {
				return err
			}
			if err := p.parseUnboundCall(prog, mname, 1, true); err != nil {
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
func (p *parser) parsePrefixExpr(prog *program, tk int, name string) error {
	mname, err := p.parseModSymbol(name)
	if err != nil {
		return err
	}

	name = mname.fullname()

	switch tk {
	case tkId:
		// identifier leading types
		switch p.l.token {
		case tkLPar:
			if err := p.parseUnboundCall(prog, mname, 0, false); err != nil {
				return err
			}
			break

		default:
			// Resolve the identifier here, notes the symbol can be following types
			// 1. session variable
			// 2. local variable
			// 3. dynmaic variable
			symT, symIdx := p.resolveSymbol(name, false)
			must(symIdx != symInvalidConst, "should never return invalid constant")

			switch symT {
			case symSession:
				prog.emit1(p.l, bcLoadSession, symIdx)
				break

			case symFunc:
				prog.emit1(p.l, bcNewClosure, symIdx)
				break

			case symLocal:
				prog.emit1(p.l, bcLoadLocal, symIdx)
				break

			case symUpval:
				prog.emit1(p.l, bcLoadUpvalue, symIdx)
				break

			case symGlobal:
				prog.emit1(p.l, bcLoadGlobal, symIdx)

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
		if p.isEntryRule() {
			prog.emit0(p.l, bcLoadDollar)
		} else {
			return p.err("only rule body can use $ as variable")
		}
		break

	case tkSId:
		// session symbol table
		idx := p.findSessionIdx(name)
		if idx == -1 {
			return p.errf("session variable %s is unknown", name)
		}
		prog.emit1(p.l, bcLoadSession, idx)
		break

	case tkGId:
		idx := p.findGlobalIdx(name)
		if idx == -1 {
			return p.errf("global variable %s is unknown", name)
		}
		prog.emit1(p.l, bcLoadGlobal, idx)
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

			switch p.l.token {
			case tkStr, tkId:
				idx := prog.addStr(p.l.valueText)
				prog.emit1(p.l, bcLoadStr, idx)
				p.l.next()
				break

			case tkLSqr:
				p.l.next()
				if err := p.parseExpr(prog); err != nil {
					return err
				}
				if !p.l.expectCurrent(tkRSqr) {
					return p.l.toError()
				}
				p.l.next()

			default:
				return p.err("unexpected token, expect a quoted string or identifier as map key")
			}

			if !p.l.expectCurrent(tkColon) {
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
	pp := newParser(strV[offset:], p.fs)

	// duplicate parent parser's states, FIXME(dpeng): make the following function
	// into a separate function to allow derived parser from parent parser
	pp.stbl = p.stbl
	pp.globalVar = p.globalVar
	pp.sessVar = p.sessVar
	pp.module = p.module

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

// -----------------------------------------------------------------------------
// Configuration section parsing. The configuration section is mostly a unqiue
// feature provide by us. Instead of allowing another freaking configuration
// file to be used by user, we try to embed configuration, normal yaml, json,
// xml into our code. Thus we can blur the boundary between configuration and
// script evaluation. Additionally, this makes the code much more compact and
// we do not need to extend the existed configuration format, or even invent
// our own configuration style to support dynamic expression/code, which is
// typical headache for many users.
func (p *parser) parseConfig() error {
	// try to find an existed config scope
	prog := p.module.config
	if prog == nil {
		prog = newProgram(p.module, ConfigRule, progConfig)
		p.module.config = prog
	}
	return p.parseConfigScope(prog)
}

// config scope related if branch statement ------------------------------------
func (p *parser) parseConfigScopeBranchStmtBody(prog *program) error {
	if !p.l.expectCurrent(tkLBra) {
		return p.l.toError()
	}
	{
		p.enterNormalScope()
		if err := p.parseConfigScopeBody(prog); err != nil {
			return err
		}
		p.leaveScope()
	}

	return nil
}

func (p *parser) parseConfigScopeBranchStmt(prog *program) error {
	p.l.next()
	return p.parseBranch(
		prog,
		p.parseConfigScopeBranchStmtBody,
		func(_ *program) error {
			return nil
		},
	)
}

// config scope related try statement ------------------------------------------
func (p *parser) parseConfigScopeTryStmt(prog *program) error {
	parseChunk := func(prog *program) error {
		p.enterNormalScope()
		if err := p.parseConfigScopeBody(prog); err != nil {
			return err
		}
		p.leaveScope()
		return nil
	}

	p.l.next()
	return p.parseTry(prog, parseChunk, parseChunk)
}

// config scope related loop statement ------------------------------------------
func (p *parser) parseConfigScopeFor(prog *program) error {
	return p.parseForImpl(
		prog,
		p.parseConfigScopeBody,
	)
}

// this syntax is desigend to distinguish normal statement from configuration
// statement and also keep the bare minimum syntax needs to be learnt from.
func (p *parser) parseConfigScopeCmd(prog *program) error {
	if p.l.token == tkDot {
		if !p.l.expect(tkId) {
			return p.l.toError()
		}
		prog.emit1(p.l, bcLoadStr, prog.addStr(p.l.valueText))
		p.l.next()
	} else if p.l.token == tkLSqr {
		p.l.next()
		if err := p.parseExpr(prog); err != nil {
			return err
		}

		// ] following the expression
		if !p.l.expectCurrent(tkRSqr) {
			return p.l.toError()
		}
		p.l.next()
	} else if p.l.token == tkId {
		prog.emit1(p.l, bcLoadStr, prog.addStr(p.l.valueText))
		p.l.next()
	} else {
		return p.err("unknown token for config command instruction")
	}

	switch p.l.token {
	case tkAssign:
		p.l.next()
		if err := p.parseExpr(prog); err != nil {
			return err
		}

		prog.emit0(
			p.l,
			bcConfigPropertySet,
		)
		break

	case tkLPar:
		// command invocation
		if cnt, err := p.parseCallArgs(prog); err != nil {
			return err
		} else {
			prog.emit1(
				p.l,
				bcConfigCommand,
				cnt,
			)
		}
		break

	default:
		return p.err("unknown statement inside of config scope, " +
			"you are trying to set a config item, but with " +
			"invalid operator. A config name, like .foo or [\"bar\"] " +
			"should follow a '=' to set a property or a '(' argument ')' " +
			"set a config command")
	}
	return nil
}

func (p *parser) parseConfigScopeBasicStmt(prog *program) (bool, error) {
	lexeme := p.lexeme()
	p.l.next()

	if lexeme.token == tkId {
		if p.l.token == tkLBra {
			// we specifically treat any UNKNOWN identifier suffixed with a { as a
			// config scope statement, ie getting config scope statement
			return false, p.parseConfigScopeInner(prog, lexeme.sval)
		} else if p.l.token == tkId || p.l.token == tkLSqr || p.l.token == tkDot {
			p.enterNormalScope()
			prog.emit1(p.l, bcConfigPush, prog.addStr(lexeme.sval))

			// 1. xxx name(...)
			// 2. xxx .name(...)
			// 3. xxx [name](...)
			if err := p.parseConfigScopeCmd(prog); err != nil {
				return false, err
			}

			prog.emit0(p.l, bcConfigPop)
			p.leaveScope()

			// notes this format of syntax require a semicolon afterwards
			return true, nil
		}
	}

	_, err := p.parseBasicStmt(prog, lexeme, true)
	return true, err
}

func (p *parser) parseConfigScopeBodyStmt(prog *program) (bool, error) {
	hasSep := true

	switch p.l.token {
	case tkSemicolon:
		break

	case tkLBra:
		p.enterNormalScope()
		if err := p.parseConfigScopeBody(prog); err != nil {
			return false, err
		}
		p.leaveScope()
		hasSep = false
		break

	case tkLet:
		if err := p.parseVarDecl(prog, symtVar); err != nil {
			return false, err
		}
		break

	case tkConst:
		if err := p.parseVarDecl(prog, symtConst); err != nil {
			return false, err
		}
		break

	case tkIf:
		if err := p.parseConfigScopeBranchStmt(prog); err != nil {
			return false, err
		}
		hasSep = false
		break

	case tkTry:
		if err := p.parseConfigScopeTryStmt(prog); err != nil {
			return false, err
		}
		hasSep = false
		break

	case tkFor:
		if err := p.parseConfigScopeFor(prog); err != nil {
			return false, err
		}
		hasSep = false
		break

	case tkBreak:
		if err := p.parseBreak(prog); err != nil {
			return false, err
		}
		break

	case tkContinue:
		if err := p.parseContinue(prog); err != nil {
			return false, err
		}
		break

	// notes inside of the configuration, user are not allowed to return from
	// the process since this will just break the configuration process. If
	// user want to terminate the process user needs to just raise an error
	// via other statement, like abort

	// setting the configuration section's own property and its call, etc ...
	case tkDot, tkLSqr:
		if err := p.parseConfigScopeCmd(prog); err != nil {
			return false, err
		}
		break

	default:
		if x, err := p.parseConfigScopeBasicStmt(prog); err != nil {
			return false, err
		} else {
			hasSep = x
		}
		break
	}

	return hasSep, nil
}

func (p *parser) parseConfigScopeBody(prog *program) error {
	return p.parseBodyImpl(
		prog,
		p.parseConfigScopeBodyStmt,
	)
}

func (p *parser) parseConfigScopeInner(prog *program, name string) error {
	p.enterNormalScope()
	prog.emit1(p.l, bcConfigPush, prog.addStr(name))
	if err := p.parseConfigScopeBody(prog); err != nil {
		return err
	}
	prog.emit0(p.l, bcConfigPop)
	p.leaveScope()
	return nil
}

func (p *parser) parseConfigScope(prog *program) error {
	p.enterScopeTop(entryConfig, prog)
	defer func() {
		p.leaveScope()
	}()

	if !p.l.expectCurrent(tkId) {
		return p.l.toError()
	}

	// push the current config scope into the stack for evaluation
	svcName := p.l.valueText
	p.l.next()
	return p.parseConfigScopeInner(prog, svcName)
}
