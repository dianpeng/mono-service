//go:build go1.18
// +build go1.18

package pl

import (
	"strings"
)

func initModStrCut() {
	addrefMF(
		"str",
		"cut",
		"",
		"%s%s",
		func(a, b string) Val {
			before, after, found := strings.Cut(a, b)
			if found {
				return NewValPair(NewValStr(before), NewValStr(after))
			} else {
				return NewValNull()
			}
		},
	)
}
