package pl

import (
	"fmt"
)

// script function's value representation. We support first order function, so
// need to have a representation of function written inside of the script
type scriptFunc struct {
	prog *program

	// upvalue capture
	upvalue []Val
}

func (f *scriptFunc) Call(eval *Evaluator, args []Val) (Val, error) {
	return eval.runSFunc(f, args)
}

func (f *scriptFunc) Type() int {
	return ClosureScript
}

func (f *scriptFunc) Id() string {
	return f.prog.name
}

func (f *scriptFunc) Info() string {
	return fmt.Sprintf("[%s:%s]", GetClosureTypeId(f.Type()), f.Id())
}

func (f *scriptFunc) Dump() string {
	return f.prog.dump()
}

func (f *scriptFunc) ToJSON() (Val, error) {
	return NewValStr(f.Id()), nil
}

func (f *scriptFunc) ToString() (string, error) {
	return f.Id(), nil
}

func newScriptFunc(p *program) *scriptFunc {
	return &scriptFunc{
		prog:    p,
		upvalue: make([]Val, 0, p.upvalueSize()),
	}
}
