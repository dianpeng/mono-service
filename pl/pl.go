package pl

import "fmt"

func must(cond bool, msg string, a ...interface{}) {
	if !cond {
		f := fmt.Sprintf("must: %s", msg)
		panic(fmt.Sprintf(f, a...))
	}
}

func musterr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err.Error()))
	}
}

func unreachable(msg string) { panic(fmt.Sprintf("unreachable: %s", msg)) }

// module function name
const (
	modSep = "::"
)

func modSymbolName(m string, s string) string {
	return fmt.Sprintf("%s::%s", m, s)
}

func modFuncName(m string, f string) string {
	return modSymbolName(m, f)
}

// the VM's returned argument is transient and violatile. User must duplicate it
// if they wish to store it, otherwise it will gone since the stack will be
// modified accordingly
func Dup(x []Val) []Val {
	xx := make([]Val, 0, len(x))
	for _, v := range x {
		xx = append(xx, v)
	}
	return xx
}
