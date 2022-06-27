package pl

import (
	"fmt"
)

type methodFunc struct {
	recv  Val
	name  string
	entry MethodFn
}

func (f *methodFunc) Call(
	_ *Evaluator,
	args []Val,
) (Val, error) {
	return f.entry(f.name, args)
}

func (f *methodFunc) Type() int {
	return ClosureMethod
}

func (f *methodFunc) Id() string {
	return f.name
}

func (f *methodFunc) Info() string {
	return fmt.Sprintf("[%s: %s]", GetClosureTypeId(f.Type()), f.Id())
}

func (f *methodFunc) Dump() string {
	return "[method]"
}

func (f *methodFunc) ToJSON() (Val, error) {
	return NewValStr(f.Id()), nil
}

func (f *methodFunc) ToString() (string, error) {
	return f.Id(), nil
}

func newMethodFunc(
	entry MethodFn,
	name string,
) *methodFunc {
	return &methodFunc{
		entry: entry,
		name:  name,
	}
}
