package pl

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

type globalState struct {
	// global program snippet, ie used for initialization of the module
	globalProgram []*program

	// global variable, should not be used directly, but should be using
	// function to manipulate since it needs to be guarded with lock
	globalVar []Val

	// internally used
	lock sync.RWMutex
}

type symbolInfo struct {
	globalName  []string
	sessionName []string
}

type Module struct {
	// module wise global state object
	global *globalState

	// initialization part of the module, if module does not have initialization
	// code then this part is empty
	session []*program

	// config scope, ie used for configuration initialization etc ...
	config *program

	// dispatch table, ie event matching hash table. Ie each program's event name
	// will become key in this table for finding out which rule should be used
	// NOTES(dpeng): for optimization purpose, this table *MAY* be nil since if
	// our rule limited, linear search is the fastest
	eventMap map[string]*program

	// all the named rules, evaluation will iterate each program for execution
	// until one is found
	p []*program

	// all the script function. Each function must have its unique name and the
	// name must be unique across all the module. Notes each program's name becomes
	// function name
	fn []*program

	// symbol info, used for instrumentation/debugging purpose
	sinfo symbolInfo
}

func newModule() *Module {
	return &Module{
		global:   &globalState{},
		eventMap: make(map[string]*program),
	}
}

func (p *Module) addSessionProgram(
	prog *program,
	nameList []string,
) {
	p.session = append(p.session, prog)
	p.sinfo.sessionName = append(p.sinfo.sessionName, nameList...)
}

func (p *Module) addGlobalProgram(
	prog *program,
	nameList []string,
) {
	p.global.globalProgram = append(p.global.globalProgram, prog)
	p.sinfo.globalName = append(p.sinfo.globalName, nameList...)
}

func (p *Module) GetGlobal(i int) (Val, bool) {
	return p.global.get(i)
}

func (p *Module) StoreGlobal(i int, v Val) bool {
	return p.global.set(i, v)
}

func (p *Module) GlobalSize() int {
	return p.global.size()
}

func (p *Module) nextAnonymousFuncName() string {
	return fmt.Sprintf("[anonymous_function_%d]", len(p.fn))
}

func (p *Module) GetAnonymousFunctionSize() int {
	cnt := 0
	for _, x := range p.fn {
		if strings.HasPrefix(x.name, "[anonymous_function_") {
			cnt++
		}
	}
	return cnt
}

func (p *Module) allAnonymousFunction() []*program {
	o := []*program{}
	for _, x := range p.fn {
		if strings.HasPrefix(x.name, "[anonymous_function_") {
			o = append(o, x)
		}
	}
	return o
}

func (p *Module) getfromlist(name string, l []*program) (*program, int) {
	for idx, pp := range l {
		if pp.name == name {
			return pp, idx
		}
	}
	return nil, -1
}

func (p *Module) getProgram(name string) *program {
	r, _ := p.getfromlist(name, p.p)
	return r
}

// ignore the type of the program
func (p *Module) getFn(name string) *program {
	r, _ := p.getfromlist(name, p.fn)
	return r
}

func (p *Module) getFunction(name string) *program {
	r, _ := p.getfromlist(name, p.fn)
	if r.progtype == progFunc {
		return r
	} else {
		return nil
	}
}

func (p *Module) getIterator(name string) *program {
	r, _ := p.getfromlist(name, p.fn)
	if r.progtype == progIter {
		return r
	} else {
		return nil
	}
}

func (p *Module) getFunctionIndex(name string) int {
	f, i := p.getfromlist(name, p.fn)
	if f != nil {
		if f.progtype == progFunc {
			return i
		}
	}
	return -1
}

func (p *Module) getIteratorIndex(name string) int {
	f, i := p.getfromlist(name, p.fn)
	if f != nil {
		if f.progtype == progIter {
			return i
		}
	}
	return -1
}

func (p *Module) getFnIndex(name string) int {
	_, i := p.getfromlist(name, p.fn)
	return i
}

// Get a script function from the imported module, if the function cannot be
// found then the function returns an null value, otherwise a Closure is
// returned which can be invoked later on
func (p *Module) GetFunction(name string) Val {
	prog := p.getFunction(name)
	if prog != nil {
		return newValScriptFunction(prog)
	} else {
		return NewValNull()
	}
}

func (p *Module) HasSession() bool {
	return len(p.session) != 0
}

func (p *Module) HasConfig() bool {
	return p.config != nil
}

func (p *Module) HasGlobal() bool {
	return len(p.global.globalProgram) != 0
}

func (p *Module) HaveEvent(name string) bool {
	_, ok := p.eventMap[name]
	return ok
}

func (p *Module) addEvent(name string, prog *program) bool {
	if p.HaveEvent(name) {
		return false
	}
	p.eventMap[name] = prog
	return true
}

func (p *Module) findEvent(name string) *program {
	v, ok := p.eventMap[name]
	if !ok {
		return nil
	} else {
		return v
	}
}

func (p *Module) HasEvent(name string) bool {
	_, ok := p.eventMap[name]
	return ok
}

func (p *Module) Dump() string {
	var b bytes.Buffer
	b.WriteString("function> -------------------------------- \n")
	for _, p := range p.fn {
		b.WriteString(p.dump())
		b.WriteRune('\n')
	}

	b.WriteString("rules>    -------------------------------- \n")
	for _, p := range p.p {
		b.WriteString(p.dump())
		b.WriteRune('\n')
	}
	return b.String()
}

// Compile the input string into a Module object
func CompileModule(module string, fs fs.FS) (*Module, error) {
	p := newParser(module, fs)
	po, err := p.parse()
	if err != nil {
		return nil, err
	}
	return po, nil
}

func (g *globalState) size() int {
	g.lock.RLock()
	defer func() {
		g.lock.RUnlock()
	}()
	return len(g.globalVar)
}

func (g *globalState) set(
	i int,
	v Val,
) bool {
	if !v.IsImmutable() {
		return false
	}

	g.lock.Lock()
	defer func() {
		g.lock.Unlock()
	}()
	if len(g.globalVar) <= i {
		return false
	}
	g.globalVar[i] = v
	return true
}

func (g *globalState) get(
	i int,
) (Val, bool) {
	g.lock.RLock()
	defer func() {
		g.lock.RUnlock()
	}()

	if len(g.globalVar) <= i {
		return NewValNull(), false
	}
	return g.globalVar[i], true
}

func (g *globalState) add(
	v Val,
) bool {
	if !v.IsImmutable() {
		return false
	}

	// may not need lock since it is only called during setup
	g.lock.Lock()
	defer func() {
		g.lock.Unlock()
	}()
	g.globalVar = append(g.globalVar, v)
	return true
}
