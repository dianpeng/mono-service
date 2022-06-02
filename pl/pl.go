package pl

import "fmt"

func must(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("must: %s", msg))
	}
}

func unreachable(msg string) { panic(fmt.Sprintf("unreachable: %s", msg)) }
