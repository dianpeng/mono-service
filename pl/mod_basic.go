package pl

import (
	"bytes"
	"fmt"
	"strconv"
)

// all the builtin basic functions
func initModBasic() {
	addF(
		"dprint",
		"",
		"%a*",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			buf := new(bytes.Buffer)
			for _, x := range args {
				native := x.ToNative()
				buf.WriteString(fmt.Sprintf("%T[%+v]", native, native))
				buf.WriteString("; ")
			}
			fmt.Println(buf.String())
			return NewValNull(), nil
		},
	)

  addF(
    "print",
    "",
    "%a*",
    func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
      var buf []string
      for _, x := range args {
        str, err := x.ToString()
        if err != nil {
          str = fmt.Sprintf("[%s:unknown]", x.Id())
        }
        buf.WriteString(str)
      }
      fmt.Println(buf.String())
      return NewValNull(), nil
    },
  )

	addF(
		"to_string",
		"",
		"%a",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			s, err := args[0].ToString()
			if err != nil {
				return NewValNull(), err
			}
			return NewValStr(s), nil
		},
	)

	addF(
		"to_int",
		"",
		"{%a}{%a%d}",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			alen, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			i, err := strconv.ParseInt(args[0].String(), 10, 64)

			if alen == 1 {
				if err != nil {
					return NewValNull(), err
				}
				return NewValInt64(i), nil
			} else {
				if err != nil {
					return args[1], nil
				} else {
					return NewValInt64(i), nil
				}
			}
		},
	)

	addF(
		"to_real",
		"",
		"{%a}{%a%f}",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			alen, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			i, err := strconv.ParseFloat(args[0].String(), 64)

			if alen == 1 {
				if err != nil {
					return NewValNull(), err
				}
				return NewValReal(i), nil
			} else {
				if err != nil {
					return args[1], nil
				} else {
					return NewValReal(i), nil
				}
			}
		},
	)

	addF(
		"to_bool",
		"",
		"{%a}",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			return NewValBool(args[0].ToBoolean()), nil
		},
	)

	addF(
		"type",
		"",
		"{%a}",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			return NewValStr(args[0].Id()), nil
		},
	)

	addF(
		"len",
		"",
		"{%a}",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			a := args[0]
			switch args[0].Type {
			case ValStr:
				return NewValInt(len(a.String())), nil
			case ValRegexp, ValUsr, ValInt, ValReal, ValBool, ValNull:
				return NewValInt(0), nil
			case ValPair:
				return NewValInt(2), nil
			case ValList:
				return NewValInt(len(a.List().Data)), nil
			default:
				return NewValInt(len(a.Map().Data)), nil
			}
		},
	)

	addF(
		"empty",
		"",
		"{%a}",
		func(info *intrinsicinfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.argproto.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			a := args[0]
			switch args[0].Type {
			case ValStr:
				return NewValBool(len(a.String()) == 0), nil
			case ValRegexp, ValUsr, ValInt, ValReal, ValBool, ValNull:
				return NewValBool(true), nil
			case ValPair:
				return NewValBool(true), nil
			case ValList:
				return NewValBool(len(a.List().Data) == 0), nil
			default:
				return NewValBool(len(a.Map().Data) == 0), nil
			}
		},
	)
}
