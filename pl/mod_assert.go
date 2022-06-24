package pl

import (
	"fmt"
)

func assertVeq(lhs Val, rhs Val) (bool, string) {
	if lhs.Type == rhs.Type {
		if IsValueType(lhs.Type) {
			switch lhs.Type {
			case ValNull:
				return true, fmt.Sprintf("lhs: null; rhs: null")

			case ValInt:
				l := lhs.Int()
				r := rhs.Int()
				return l == r, fmt.Sprintf("lhs: %d; rhs: %d", l, r)

			case ValReal:
				l := lhs.Real()
				r := rhs.Real()
				return l == r, fmt.Sprintf("lhs: %f; rhs: %f", l, r)

			case ValStr:
				l := lhs.String()
				r := rhs.String()
				return l == r, fmt.Sprintf("lhs: %s; rhs: %s", l, r)

			case ValBool:
				l := lhs.Bool()
				r := rhs.Bool()
				return l == r, fmt.Sprintf("lhs: %t; rhs: %t", l, r)

			case ValPair:
				lp := lhs.Pair()
				rp := rhs.Pair()

				lok, _ := assertVeq(lp.First, rp.First)
				rok, _ := assertVeq(lp.Second, rp.Second)

				return lok && rok, fmt.Sprintf("lhs: %s; rhs: %s", lp.Info(), rp.Info())

			case ValList:
				ll := lhs.List()
				rl := rhs.List()
				info := fmt.Sprintf("lhs: %s; rhs: %s", ll.Info(), rl.Info())

				if len(ll.Data) == len(rl.Data) {
					for i := 0; i < len(ll.Data); i++ {
						if ok, _ := assertVeq(ll.Data[i], rl.Data[i]); !ok {
							return false, info
						}
					}
					return true, info
				}
				return false, info

			case ValMap:
				lm := lhs.Map()
				rm := rhs.Map()
				info := fmt.Sprintf("lhs: %s; rhs: %s", lm.Info(), rm.Info())
				if lm.Length() == rm.Length() {
					cnt := lm.Foreach(
						func(k string, v Val) bool {
							vv, ok := rm.Get(k)
							if !ok {
								return false
							}
							if ok, _ := assertVeq(v, vv); !ok {
								return false
							} else {
								return true
							}
						},
					)
					if cnt != lm.Length() {
						return false, info
					} else {
						return true, info
					}
				}

				return false, info

			default:
				must(lhs.Type == ValRegexp, "must be regexp")
				l := lhs.Regexp()
				r := rhs.Regexp()
				info := fmt.Sprintf("lhs: [regexp %s]; rhs: [regexp %s]", l.String(), r.String())
				return l.String() == r.String(), info
			}
		}
	}

	return false, fmt.Sprintf("lhs: %s; rhs: %s", lhs.Info(), rhs.Info())
}

func init() {
	addMF(
		"assert",
		"yes",
		"",
		"{%a}{%a%s}",
		func(info *IntrinsicInfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			alog, err := info.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			result := args[0].ToBoolean()
			msg := "<none>"
			if alog == 2 {
				msg = args[1].String()
			}

			if !result {
				return NewValNull(), fmt.Errorf("assert::yes failed]: %s", msg)
			} else {
				return NewValNull(), nil
			}
		},
	)

	addMF(
		"assert",
		"no",
		"",
		"{%a}{%a%s}",
		func(info *IntrinsicInfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			alog, err := info.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			result := args[0].ToBoolean()
			msg := "<none>"
			if alog == 2 {
				msg = args[1].String()
			}

			if result {
				return NewValNull(), fmt.Errorf("assert::no failed]: %s", msg)
			} else {
				return NewValNull(), nil
			}
		},
	)

	addMF(
		"assert",
		"eq",
		"",
		"{%a%a}{%a%a%s}",
		func(info *IntrinsicInfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			alog, err := info.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			msg := "<none>"
			if alog == 3 {
				msg = args[2].String()
			}

			lhs := args[0]
			rhs := args[1]

			if ok, info := assertVeq(lhs, rhs); !ok {
				return NewValNull(), fmt.Errorf("assert::eq failed]: %s; message: %s", info, msg)
			} else {
				return NewValNull(), nil
			}
		},
	)

	addMF(
		"assert",
		"ne",
		"",
		"{%a%a}{%a%a%s}",
		func(info *IntrinsicInfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			alog, err := info.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			msg := "<none>"
			if alog == 3 {
				msg = args[2].String()
			}

			lhs := args[0]
			rhs := args[1]

			if ok, info := assertVeq(lhs, rhs); ok {
				return NewValNull(), fmt.Errorf("assert::ne failed]: %s; message: %s", info, msg)
			} else {
				return NewValNull(), nil
			}
		},
	)

	throwFunc := func(
		info *IntrinsicInfo,
		e *Evaluator,
		args []Val,
	) (string, error) {
		alog, err := info.Check(args)
		if err != nil {
			return "", err
		}
		msg := "<none>"
		if alog == 2 {
			msg = args[1].String()
		}

		_, err = args[0].Closure().Call(
			e,
			nil,
		)
		return msg, err
	}

	addMF(
		"assert",
		"pass",
		"",
		"{%c}{%c%s}",
		func(info *IntrinsicInfo, e *Evaluator, _ string, args []Val) (Val, error) {
			msg, err := throwFunc(info, e, args)
			if err != nil {
				return NewValNull(), fmt.Errorf("assert.pass failed]: %s; message: %s",
					msg,
					err.Error(),
				)
			} else {
				return NewValNull(), nil
			}
		},
	)

	addMF(
		"assert",
		"throw",
		"",
		"{%c}{%c%s}",
		func(info *IntrinsicInfo, e *Evaluator, _ string, args []Val) (Val, error) {
			msg, err := throwFunc(info, e, args)
			if err == nil {
				return NewValNull(), fmt.Errorf("assert.throw failed]: message: %s",
					msg,
				)
			} else {
				return NewValNull(), nil
			}
		},
	)
}
