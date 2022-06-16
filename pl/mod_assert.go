package pl

import (
	"fmt"
)

func assertVeq(lhs Val, rhs Val) bool {
	if lhs.Type != rhs.Type {
		return false
	} else {
		if !IsValueType(lhs.Type) {
			return false
		} else {
			switch lhs.Type {
			case ValNull:
				return true

			case ValInt:
				return lhs.Int() == rhs.Int()

			case ValReal:
				return lhs.Real() == rhs.Real()

			case ValStr:
				return lhs.String() == rhs.String()

			case ValBool:
				return lhs.Bool() == rhs.Bool()

			case ValPair:
				lp := lhs.Pair()
				rp := rhs.Pair()
				return assertVeq(lp.First, rp.First) && assertVeq(lp.Second, rp.Second)

			case ValList:
				ll := lhs.List()
				rl := rhs.List()
				if len(ll.Data) == len(rl.Data) {
					for i := 0; i < len(ll.Data); i++ {
						if !assertVeq(ll.Data[i], rl.Data[i]) {
							return false
						}
					}
					return true
				}
				return false

			case ValMap:
				lm := lhs.Map()
				rm := rhs.Map()

				if len(lm.Data) == len(rm.Data) {
					for k, v := range lm.Data {
						vv, ok := rm.Data[k]
						if !ok {
							return false
						}
						if !assertVeq(v, vv) {
							return false
						}
					}
					return true
				}

				return false

			default:
				must(lhs.Type == ValRegexp, "must be regexp")
				return lhs.Regexp().String() == rhs.Regexp().String()
			}
		}
	}

	return false
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
				return NewValNull(), fmt.Errorf("assert::yes failed: %s", msg)
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
				return NewValNull(), fmt.Errorf("assert::no failed: %s", msg)
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

			if !assertVeq(lhs, rhs) {
				return NewValNull(), fmt.Errorf("assert::eq failed: %s", msg)
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

			if assertVeq(lhs, rhs) {
				return NewValNull(), fmt.Errorf("assert::ne failed: %s", msg)
			} else {
				return NewValNull(), nil
			}
		},
	)
}
