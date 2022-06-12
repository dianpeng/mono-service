//go:build !go1.18
// +build !go1.18

package pl

import (
	"strings"
)

func strcut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func initModStrCut() {
	addrefMF(
		"str",
		"cut",
		"",
		"%s%s",
		func(a, b string) Val {
			before, after, found := strcut(a, b)
			if found {
				return NewValPair(NewValStr(before), NewValStr(after))
			} else {
				return NewValNull()
			}
		},
	)
}
