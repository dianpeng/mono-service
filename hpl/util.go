package hpl

import (
	"github.com/dianpeng/mono-service/pl"
	"net/http"
	"strings"
)

// http header filter operation. Based on the http header key pattern. The input
// key pattern can be 2 types as following

// 1) a string pattern
//      which can be a exact key or just a key pattern with simple semantic
//      meta char current as *XXX or XXX*

// 2) a regex type
//      which will be used to perform Match operation against header key to filter
//      out what we need

type matcher func(string, string) bool

func metaType(key string) matcher {
	prefixWildcard := strings.HasPrefix(key, "*")
	suffixWildcard := strings.HasSuffix(key, "*")
	if prefixWildcard && suffixWildcard {
		return strings.Contains
	} else if prefixWildcard {
		return strings.HasPrefix
	} else if suffixWildcard {
		return strings.HasSuffix
	} else {
		return func(_, _ string) bool {
			return true
		}
	}
}

type httpHeaderFilterFunc = func(string, []string, http.Header) bool

func httpHeaderFilter(hdr http.Header, key_pattern pl.Val, filter httpHeaderFilterFunc) int {
	if key_pattern.Type == pl.ValStr {
		k := key_pattern.String()
		m := metaType(k)
		cnt := 0

		for key, val := range hdr {
			lkey := strings.ToLower(key)
			if m(lkey, k) {
				cnt++
				if !filter(key, val, hdr) {
					break
				}
			}
		}

		return cnt
	} else if key_pattern.Type == pl.ValRegexp {
		cnt := 0
		for key, val := range hdr {
			lkey := strings.ToLower(key)
			if key_pattern.Regexp().MatchString(lkey) {
				cnt++
				if !filter(key, val, hdr) {
					break
				}
			}
		}

		return cnt
	} else {
		return 0
	}
}

func httpHeaderDelete(hdr http.Header, key_pattern pl.Val) int {
	return httpHeaderFilter(
		hdr,
		key_pattern,
		func(key string, val []string, hdr http.Header) bool {
			hdr.Del(key)
			return true
		},
	)
}

func foreachHeaderKV(arg pl.Val, fn func(key string, val string)) bool {
	switch arg.Type {
	case pl.ValList:
		for _, v := range arg.List().Data {
			if v.Type == pl.ValPair && v.Pair().First.Type == pl.ValStr && v.Pair().Second.Type == pl.ValStr {
				fn(v.Pair().First.String(), v.Pair().Second.String())
			}
		}
		return true

	case pl.ValPair:
		if arg.Pair().First.Type == pl.ValStr && arg.Pair().Second.Type == pl.ValStr {
			fn(arg.Pair().First.String(), arg.Pair().Second.String())
		}
		return true

	case pl.ValMap:
		arg.Map().Foreach(
			func(k string, v pl.Val) bool {
				if v.Type == pl.ValStr {
					fn(k, v.String())
				}
				return true
			},
		)
		return true

	default:
		if ValIsHttpHeader(arg) {
			// known special user type to us, then just foreach the header
			hdr := arg.Usr().(*Header)
			for k, v := range hdr.header {
				for _, vv := range v {
					fn(k, vv)
				}
			}
		}
		break
	}

	return false
}
