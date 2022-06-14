package pl

import "strings"

func init() {

	addrefMF(
		"str",
		"cmp",
		"",
		"%s%s",
		strings.Compare,
	)

	addrefMF(
		"str",
		"contains",
		"",
		"%s%s",
		strings.Contains,
	)

	addrefMF(
		"str",
		"contains_any",
		"",
		"%s%s",
		strings.ContainsAny,
	)

	addrefMF(
		"str",
		"contains_char",
		"",
		"%s%s",
		func(a string, b string) bool {
			if len(b) == 0 {
				return false
			}
			return strings.ContainsRune(a, []rune(b)[0])
		},
	)

	addrefMF(
		"str",
		"count",
		"",
		"%s%s",
		strings.Count,
	)

	initModStrCut()

	addrefMF(
		"str",
		"caseless_eq",
		"",
		"%s%s",
		strings.EqualFold,
	)

	addrefMF(
		"str",
		"has_prefix",
		"",
		"%s%s",
		strings.HasPrefix,
	)

	addrefMF(
		"str",
		"has_suffix",
		"",
		"%s%s",
		strings.HasSuffix,
	)

	addrefMF(
		"str",
		"index",
		"",
		"%s%s",
		strings.Index,
	)
	addrefMF(
		"str",
		"index_any",
		"",
		"%s%s",
		strings.IndexAny,
	)
	addrefMF(
		"str",
		"index_byte",
		"",
		"%s%s",
		func(a, b string) int {
			if len(b) == 0 {
				return -1
			}
			return strings.IndexByte(a, []byte(b)[0])
		},
	)
	addMF(
		"str",
		"join",
		"",
		"%l%s",
		func(info *IntrinsicInfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			var slist []string
			for _, v := range args[0].List().Data {
				if v.Type == ValStr {
					slist = append(slist, v.String())
				}
			}
			return NewValStr(strings.Join(slist, args[1].String())), nil
		},
	)

	addrefMF(
		"str",
		"last_index",
		"",
		"%s%s",
		strings.LastIndex,
	)
	addrefMF(
		"str",
		"last_index_any",
		"",
		"%s%s",
		strings.LastIndexAny,
	)
	addrefMF(
		"str",
		"last_index_byte",
		"",
		"%s%s",
		func(a, b string) int {
			if len(b) == 0 {
				return -1
			}
			return strings.LastIndexByte(a, []byte(b)[0])
		},
	)

	addrefMF(
		"str",
		"repeat",
		"",
		"%s%d",
		strings.Repeat,
	)

	addrefMF(
		"str",
		"replace",
		"",
		"%s%s%s%d",
		strings.Replace,
	)

	{

		addrefMF(
			"str",
			"split",
			"",
			"%s%s",
			func(a, b string) Val {
				return NewValStrList(strings.Split(a, b))
			},
		)

		addrefMF(
			"str",
			"split_after",
			"",
			"%s%s",
			func(a, b string) Val {
				return NewValStrList(strings.SplitAfter(a, b))
			},
		)

		addrefMF(
			"str",
			"split_after_n",
			"",
			"%s%s%d",
			func(a, b string, n int) Val {
				return NewValStrList(strings.SplitAfterN(a, b, n))
			},
		)
	}

	addrefMF(
		"str",
		"to_lower",
		"",
		"%s",
		strings.ToLower,
	)

	addrefMF(
		"str",
		"to_upper",
		"",
		"%s",
		strings.ToUpper,
	)

	addrefMF(
		"str",
		"to_title",
		"",
		"%s",
		strings.ToTitle,
	)

	addrefMF(
		"str",
		"trim",
		"",
		"%s%s",
		strings.Trim,
	)

	addrefMF(
		"str",
		"trim_left",
		"",
		"%s%s",
		strings.TrimLeft,
	)

	addrefMF(
		"str",
		"trim_right",
		"",
		"%s%s",
		strings.TrimRight,
	)

	addrefMF(
		"str",
		"trim_prefix",
		"",
		"%s%s",
		strings.TrimPrefix,
	)

	addrefMF(
		"str",
		"trim_suffix",
		"",
		"%s%s",
		strings.TrimSuffix,
	)

	addrefMF(
		"str",
		"trim_space",
		"",
		"%s%s",
		strings.TrimSpace,
	)
}
