package pl

import (
	"regexp"
)

// heck, go's regexp library has too many crap

func initModRegexp() {
	addrefMF(
		"regexp",
		"new",
		"",
		"%s",
		regexp.Compile,
	)
	addrefMF(
		"regexp",
		"find",
		"",
		"%r%s",
		func(r *regexp.Regexp, b string) string {
			return string(r.Find([]byte(b)))
		},
	)

	valFromIAA := func(iaa [][]int) Val {
		r := NewValList()
		for _, y := range iaa {
			rr := NewValIntList(y)
			r.AddList(rr)
		}
		return r
	}

	valFromISS := func(iss [][]string) Val {
		r := NewValList()
		for _, y := range iss {
			r.AddList(NewValStrList(y))
		}
		return r
	}

	addrefMF(
		"regexp",
		"find_all",
		"",
		"%r%s%d",
		func(r *regexp.Regexp, b string, n int) Val {
			x := r.FindAll([]byte(b), n)
			rr := NewValList()
			for _, y := range x {
				rr.AddList(NewValStr(string(y)))
			}
			return rr
		},
	)

	addrefMF(
		"regexp",
		"find_all_index",
		"",
		"%r%s%d",
		func(r *regexp.Regexp, b string, n int) Val {
			x := r.FindAllIndex([]byte(b), n)
			return valFromIAA(x)
		},
	)

	addrefMF(
		"regexp",
		"find_all_string",
		"",
		"%r%s%d",
		func(r *regexp.Regexp, b string, n int) Val {
			x := r.FindAllString(b, n)
			return NewValStrList(x)
		},
	)

	addrefMF(
		"regexp",
		"find_all_string_index",
		"",
		"%r%s%d",
		func(r *regexp.Regexp, b string, n int) Val {
			x := r.FindAllStringIndex(b, n)
			return valFromIAA(x)
		},
	)

	addrefMF(
		"regexp",
		"find_all_string_submatch",
		"",
		"%r%s%d",
		func(r *regexp.Regexp, b string, n int) Val {
			return valFromISS(r.FindAllStringSubmatch(b, n))
		},
	)

	addrefMF(
		"regexp",
		"find_all_string_submatch_index",
		"",
		"%r%s%d",
		func(r *regexp.Regexp, b string, n int) Val {
			return valFromIAA(r.FindAllStringSubmatchIndex(b, n))
		},
	)

	addrefMF(
		"regexp",
		"find_string",
		"",
		"%r%s",
		func(r *regexp.Regexp, b string) string {
			return r.FindString(b)
		},
	)

	addrefMF(
		"regexp",
		"find_string_index",
		"",
		"%r%s",
		func(r *regexp.Regexp, b string) Val {
			return NewValIntList(r.FindStringIndex(b))
		},
	)

	addrefMF(
		"regexp",
		"match",
		"",
		"%r%s",
		func(r *regexp.Regexp, b string) bool {
			return r.Match([]byte(b))
		},
	)

	addrefMF(
		"regexp",
		"match_string",
		"",
		"%r%s",
		func(r *regexp.Regexp, b string) bool {
			return r.MatchString(b)
		},
	)

	addrefMF(
		"regexp",
		"replace_all",
		"",
		"%r%s%s",
		func(r *regexp.Regexp, a, b string) string {
			return string(r.ReplaceAll([]byte(a), []byte(b)))
		},
	)
}
