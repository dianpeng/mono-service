package pl

import "fmt"

type DynamicVariable struct {
	Key   string
	Value Val
}

func must(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("must: %s", msg))
	}
}

func musterr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err.Error()))
	}
}

func unreachable(msg string) { panic(fmt.Sprintf("unreachable: %s", msg)) }

// module function name
func modFuncName(m string, f string) string {
	return fmt.Sprintf("%s::%s", m, f)
}
