package pl

import (
	"fmt"
)

const (
	PlaceholderTypeId = ".placeholder"
)

func IsValPlaceholder(v Val) bool {
	return v.Id() == PlaceholderTypeId
}

// binding method implementation. This builtin provides a cool feature called
// curry in functional programming language. It allows user to take a function
// and generate yet another function out based on calling function argument
// manipulation.

// type placeholder, used to serve as a placeholder inside of the bind function
// call to allow curry of function
type placeholder struct{}

func (p *placeholder) Index(_ Val) (Val, error) {
	return NewValNull(), fmt.Errorf("type: %s does not support index", p.Id())
}

func (p *placeholder) IndexSet(_ Val, _ Val) error {
	return fmt.Errorf("type: %s does not support index set", p.Id())
}

func (p *placeholder) Dot(_ string) (Val, error) {
	return NewValNull(), fmt.Errorf("type: %s does not support dot", p.Id())
}

func (p *placeholder) DotSet(_ string, _ Val) error {
	return fmt.Errorf("type: %s does not support dot set", p.Id())
}

func (p *placeholder) Method(_ string, _ []Val) (Val, error) {
	return NewValNull(), fmt.Errorf("type: %s does not method call", p.Id())
}

func (p *placeholder) ToString() (string, error) {
	return ".placeholder", nil
}

func (p *placeholder) ToJSON() (Val, error) {
	return MarshalVal(
		map[string]interface{}{
			"type": p.Id(),
		},
	)
}

func (p *placeholder) Id() string {
	return PlaceholderTypeId
}

func (p *placeholder) Info() string {
	return p.Id()
}

func (p *placeholder) ToNative() interface{} {
	return p
}

func (p *placeholder) IsImmutable() bool {
	return true
}

func (p *placeholder) NewIterator() (Iter, error) {
	return nil, fmt.Errorf("type: %s does not support iterator", p.Id())
}

type bindArg struct {
	arg         Val
	placeholder bool
}

func fnBind(
	evaluator *Evaluator,
	callback Closure,
	args []Val,
) (Val, error) {
	alist := []bindArg{}
	for _, x := range args {
		if IsValPlaceholder(x) {
			alist = append(alist, bindArg{
				arg:         NewValNull(),
				placeholder: true,
			})
		} else {
			alist = append(alist, bindArg{
				arg:         x,
				placeholder: false,
			})
		}
	}

	funcName := fmt.Sprintf("bind(%s)", callback.Info())

	return NewValNativeFunction(
		funcName,
		func(nargs []Val) (Val, error) {
			nidx := 0
			args := []Val{}

			for _, a := range alist {
				if a.placeholder {
					if nidx == len(nargs) {
						return NewValNull(), fmt.Errorf("binded function: %s "+
							"lacking input argument for placeholder",
							funcName)
					}
					narg := nargs[nidx]
					nidx++
					args = append(args, narg)
				} else {
					args = append(args, a.arg)
				}
			}

			if nidx != len(nargs) {
				return NewValNull(), fmt.Errorf("binded function: %s invalid argument number", funcName)
			}

			must(len(args) == len(alist), "argument size must match")
			return callback.Call(evaluator, args)
		},
	), nil
}

func init() {
	addF(
		"new_placeholder",
		"",
		"%0",
		func(info *IntrinsicInfo, _ *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			return NewValUsr(&placeholder{}), nil
		},
	)

	addF(
		"bind",
		"",
		"{%c}{%c%a*}",
		func(info *IntrinsicInfo, evaluator *Evaluator, _ string, args []Val) (Val, error) {
			_, err := info.Check(args)
			if err != nil {
				return NewValNull(), err
			}
			return fnBind(
				evaluator,
				args[0].Closure(),
				args[1:],
			)
		},
	)
}
