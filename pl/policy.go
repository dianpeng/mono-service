package pl

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

type globalState struct {
	// global program snippet, ie used for initialization of the policy
	globalProgram *program

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

type Policy struct {
	// Policy wise global state object
	global *globalState

	// initialization part of the policy, if policy does not have initialization
	// code then this part is empty
	session *program

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
	// name must be unique across all the policy. Notes each program's name becomes
	// function name
	fn []*program

	// symbol info, used for instrumentation/debugging purpose
	sinfo symbolInfo
}

// Currently, due to the synchronization issue of underlying object/value. The
// global variable only supports storing immutable data structure, for now it
// is just primitive type.
func IsAllowedGlobalValueType(v Val) bool {
	switch v.Type {
	// currently only immutable data type, ie primitive type is allowed
	case ValInt, ValReal, ValNull, ValStr, ValBool:
		return true
	case ValUsr:
		return v.Usr().IsImmutable()
	default:
		return false
	}
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
	if !IsAllowedGlobalValueType(v) {
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
	if !IsAllowedGlobalValueType(v) {
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

func newPolicy() *Policy {
	return &Policy{
		global:   &globalState{},
		eventMap: make(map[string]*program),
	}
}

func (p *Policy) GetGlobal(i int) (Val, bool) {
	return p.global.get(i)
}

func (p *Policy) StoreGlobal(i int, v Val) bool {
	return p.global.set(i, v)
}

func (p *Policy) GlobalSize() int {
	return p.global.size()
}

func (p *Policy) nextAnonymousFuncName() string {
	return fmt.Sprintf("[anonymous_function_%d]", len(p.fn))
}

func (p *Policy) GetAnonymousFunctionSize() int {
	cnt := 0
	for _, x := range p.fn {
		if strings.HasPrefix(x.name, "[anonymous_function_") {
			cnt++
		}
	}
	return cnt
}

func (p *Policy) allAnonymousFunction() []*program {
	o := []*program{}
	for _, x := range p.fn {
		if strings.HasPrefix(x.name, "[anonymous_function_") {
			o = append(o, x)
		}
	}
	return o
}

func (p *Policy) getfromlist(name string, l []*program) (*program, int) {
	for idx, pp := range l {
		if pp.name == name {
			return pp, idx
		}
	}
	return nil, -1
}

func (p *Policy) getProgram(name string) *program {
	r, _ := p.getfromlist(name, p.p)
	return r
}

func (p *Policy) getFunction(name string) *program {
	r, _ := p.getfromlist(name, p.fn)
	return r
}

func (p *Policy) getFunctionIndex(name string) int {
	_, i := p.getfromlist(name, p.fn)
	return i
}

// Get a script function from the imported policy, if the function cannot be
// found then the function returns an null value, otherwise a Closure is
// returned which can be invoked later on
func (p *Policy) GetFunction(name string) Val {
	prog := p.getFunction(name)
	if prog != nil {
		return newValScriptFunction(prog)
	} else {
		return NewValNull()
	}
}

func (p *Policy) HasSession() bool {
	return p.session != nil
}

func (p *Policy) HasConfig() bool {
	return p.config != nil
}

func (p *Policy) HasGlobal() bool {
	return p.global.globalProgram != nil
}

func (p *Policy) HaveEvent(name string) bool {
	_, ok := p.eventMap[name]
	return ok
}

func (p *Policy) addEvent(name string, prog *program) bool {
	if p.HaveEvent(name) {
		return false
	}
	p.eventMap[name] = prog
	return true
}

func (p *Policy) findEvent(name string) *program {
	v, ok := p.eventMap[name]
	if !ok {
		return nil
	} else {
		return v
	}
}

func (p *Policy) Dump() string {
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

// Compile the input string into a Policy object
func CompilePolicy(policy string) (*Policy, error) {
	p := newParser(policy)
	po, err := p.parse()
	if err != nil {
		return nil, err
	}
	return po, nil
}

func CompilePolicyAsExpression(expr string) (*Policy, error) {
	p := newParser(expr)
	po, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return po, nil
}
