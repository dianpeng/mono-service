package util

import (
	"strings"
)

type Matcher func(string, string) bool

func ToMatcher(
	pattern string,
) Matcher {
	// special case, wildcard for all
	if pattern == "*" {
		return func(_, _ string) bool {
			return true
		}
	}

	// prefix and suffix cases
	prefixWildcard := strings.HasPrefix(pattern, "*")
	suffixWildcard := strings.HasSuffix(pattern, "*")

	if prefixWildcard && suffixWildcard {
		return strings.Contains
	} else if prefixWildcard {
		return strings.HasPrefix
	} else if suffixWildcard {
		return strings.HasSuffix
	} else {
		// exact matching
		return func(a, b string) bool {
			return a == b
		}
	}
}
