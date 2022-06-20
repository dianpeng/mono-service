package pl

type nativeFunc struct {
	id    string
	entry func([]Val) (Val, error)
}

func (f *nativeFunc) Type() int {
	return ClosureNative
}

func (f *nativeFunc) Info() string {
	return f.Id()
}

func (f *nativeFunc) Id() string {
	return f.id
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
