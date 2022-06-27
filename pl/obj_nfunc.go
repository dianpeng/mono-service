package pl

import (
	"fmt"
)

type nativeFunc struct {
	id    string
	entry func([]Val) (Val, error)
}

func (f *nativeFunc) Call(e *Evaluator, args []Val) (Val, error) {
	return e.runNFunc(f, args)
}

func (f *nativeFunc) Type() int {
	return ClosureNative
}

func (f *nativeFunc) Id() string {
	return f.id
}

func (f *nativeFunc) Info() string {
	return fmt.Sprintf("[%s: %s]", GetClosureTypeId(f.Type()), f.Id())
}

func (f *nativeFunc) ToJSON() (Val, error) {
	return NewValStr(f.Id()), nil
}

func (f *nativeFunc) ToString() (string, error) {
	return f.Id(), nil
}

func (f *nativeFunc) Dump() string {
	return "[native]"
}

func newNativeFunc(id string, e func([]Val) (Val, error)) *nativeFunc {
	return &nativeFunc{
		id:    id,
		entry: e,
	}
}

func NewNativeFunction(id string, e func([]Val) (Val, error)) Closure {
	return newNativeFunc(
		id,
		e,
	)
}
