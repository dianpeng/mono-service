package pl

import (
	"fmt"
	"strings"
)

// native []string's wrapper of pl.Val
type gostrlist struct {
	refl *[]string
}

type gostrlistiter struct {
	x      *[]string
	cursor int
}

const (
	StrListTypeId = ".str_list"
)

func ValIsStrList(x Val) bool {
	return x.Id() == StrListTypeId
}

var (
	mpStrListLength   = MustNewFuncProto(".str_list.length", "%0")
	mpStrListPushBack = MustNewFuncProto(".str_list.push_back", "{%s}{%s%s*}")
	mpStrListPopBack  = MustNewFuncProto(".str_list.pop_Back", "{%d}{%0}")
)

func (g *gostrlist) Length() int {
	return len(*g.refl)
}

func (g *gostrlist) Index(key Val) (Val, error) {
	idx, err := key.ToIndex()
	if err != nil {
		return NewValNull(), fmt.Errorf("string list index: %s", err.Error())
	}

	if idx >= len(*g.refl) {
		idx = len(*g.refl)
	}
	return NewValStr((*g.refl)[idx]), nil
}

func (g *gostrlist) IndexSet(key Val, value Val) error {
	idx, err := key.ToIndex()
	if err != nil {
		return fmt.Errorf("string list index set: %s", err.Error())
	}
	if !value.IsString() {
		return fmt.Errorf("string list index set: " +
			"value cannot be converted to string")
	}
	if idx >= g.Length() {
		return fmt.Errorf("string list index set: index out of bound")
	}

	(*g.refl)[idx] = value.String()
	return nil
}

func (g *gostrlist) Dot(_ string) (Val, error) {
	return NewValNull(), fmt.Errorf("string list dot: unsupported operator")
}

func (g *gostrlist) DotSet(_ string, _ Val) error {
	return fmt.Errorf("string list dot set: unsupported operator")
}

func (g *gostrlist) ToString() (string, error) {
	return strings.Join(
		*g.refl,
		", ",
	), nil
}

func (g *gostrlist) ToJSON() (Val, error) {
	return MarshalVal(
		*g.refl,
	)
}

func (g *gostrlist) Method(name string, args []Val) (Val, error) {
	switch name {
	case "length":
		_, err := mpStrListLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(g.Length()), nil

	case "push_back":
		_, err := mpStrListPushBack.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		for _, x := range args {
			*g.refl = append(*g.refl, x.String())
		}
		break

	case "pop_back":
		alog, err := mpStrListPopBack.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		num := 1
		if alog == 1 {
			num = int(args[0].Int())
			if num < 0 {
				num = 0
			}
		}

		if num < g.Length() {
			*g.refl = (*g.refl)[:g.Length()-num]
		} else {
			*g.refl = make([]string, 0, 0)
		}
		break

	default:
		return NewValNull(), fmt.Errorf("%s metohd: %s is unknown", g.Id(), name)
	}

	return NewValUsr(g), nil
}

func (g *gostrlist) Info() string {
	return StrListTypeId
}

func (g *gostrlist) ToNative() interface{} {
	return *g.refl
}

func (g *gostrlist) Id() string {
	return StrListTypeId
}

func (g *gostrlist) IsImmutable() bool {
	return false
}

func (g *gostrlistiter) Length() int {
	return len(*g.x)
}

func (g *gostrlistiter) SetUp(_ *Evaluator, _ []Val) error {
	return nil
}

func (g *gostrlistiter) Has() bool {
	return g.cursor < g.Length()
}

func (g *gostrlistiter) Next() (bool, error) {
	g.cursor++
	return g.Has(), nil
}

func (g *gostrlistiter) Deref() (Val, Val, error) {
	if g.Has() {
		return NewValInt(g.cursor), NewValStr((*g.x)[g.cursor]), nil
	} else {
		return NewValNull(), NewValNull(), fmt.Errorf("iterator out of bound")
	}
}

func (g *gostrlist) NewIterator() (Iter, error) {
	return &gostrlistiter{
		x:      g.refl,
		cursor: 0,
	}, nil
}

func NewValGoStrList(
	list *[]string,
) Val {
	return NewValUsr(&gostrlist{refl: list})
}
