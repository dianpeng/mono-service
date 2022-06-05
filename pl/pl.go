package pl

import "fmt"

func must(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("must: %s", msg))
	}
}

func unreachable(msg string) { panic(fmt.Sprintf("unreachable: %s", msg)) }

// module function name
func modFuncName(m string, f string) string {
	return fmt.Sprintf("%s$%s", m, f)
}
