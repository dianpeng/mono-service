package pl

import (
	"bytes"
)

type Policy struct {
	// initialization part of the policy, if policy does not have initialization
	// code then this part is empty
	g *program

	// all the named rules, evaluation will iterate each program for execution
	// until one is found
	p []*program

	// all the script function. Each function must have its unique name and the
	// name must be unique across all the policy. Notes each program's name becomes
	// function name
	fn []*program
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

func (p *Policy) Dump() string {
	var b bytes.Buffer
	b.WriteString(":function =================================\n")
	for _, p := range p.fn {
		b.WriteString(p.dump())
		b.WriteRune('\n')
	}

	b.WriteString(":rules ====================================\n")
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

	// find out the session variable
	if len(po.p) > 0 {
		p0 := po.p[0]
		if p0.name == "@session" {
			po.p = po.p[1:]
			po.g = p0
		}
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
