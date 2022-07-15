package pl

import (
	"fmt"
)

var (
	mpListLength   = MustNewFuncProto("list.length", "%0")
	mpListPushBack = MustNewFuncProto("list.push_back", "{%a}{%a%a*}")
	mpListPopBack  = MustNewFuncProto("list.pop_back", "{%d}{%0}")
	mpListExtend   = MustNewFuncProto("list.extend", "%l")
	mpListSlice    = MustNewFuncProto("list.slice", "{%d}{%d%d}")
)

type List struct {
	Data []Val
}

func NewList() *List {
	return &List{
		Data: make([]Val, 0, 0),
	}
}

type ListIter struct {
	l   *List
	cur int
}

func (li *ListIter) SetUp(_ *Evaluator, _ []Val) error {
	return nil
}

func (li *ListIter) Has() bool {
	return li.cur < li.l.Length()
}

func (li *ListIter) Next() (bool, error) {
	li.cur++
	return li.Has(), nil
}

func (li *ListIter) Deref() (Val, Val, error) {
	if li.Has() {
		return NewValInt(li.cur), li.l.Data[li.cur], nil
	} else {
		return NewValNull(), NewValNull(), fmt.Errorf("iterator out of bound")
	}
}

func NewListIter(l *List) *ListIter {
	return &ListIter{
		l:   l,
		cur: 0,
	}
}

func (l *List) NewIter() Iter {
	return &ListIter{
		l:   l,
		cur: 0,
	}
}

func (l *List) Length() int {
	return len(l.Data)
}

func (l *List) Append(x Val) {
	l.Data = append(l.Data, x)
}

func (l *List) At(i int) Val {
	return l.Data[i]
}

func (l *List) ToNative() interface{} {
	var x []interface{}
	for _, xx := range l.Data {
		x = append(x, xx.ToNative())
	}
	return x
}

func (l *List) Info() string {
	return fmt.Sprintf("[list: %d]", len(l.Data))
}

func (l *List) Index(idx Val) (Val, error) {
	i, err := idx.ToIndex()
	if err != nil {
		return NewValNull(), err
	}
	if i >= len(l.Data) {
		return NewValNull(), fmt.Errorf("index out of range")
	}
	return l.Data[i], nil
}

func (l *List) IndexSet(idx Val, val Val) error {
	i, err := idx.ToIndex()
	if err != nil {
		return err
	}

	if i >= len(l.Data) {
		for j := len(l.Data); j < i; j++ {
			l.Append(NewValNull())
		}
		l.Append(val)
	} else {
		l.Data[i] = val
	}
	return nil
}

func (l *List) Method(name string, args []Val) (Val, error) {
	switch name {
	case "length":
		_, err := mpListLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(len(l.Data)), nil

	case "push_back":
		_, err := mpListPushBack.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		for _, x := range args {
			l.Append(x)
		}
		return NewValListFromList(l), nil

	case "pop_back":
		alog, err := mpListPopBack.Check(args)
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

		if num < len(l.Data) {
			l.Data = l.Data[0 : len(l.Data)-num]
		} else {
			l.Data = make([]Val, 0, 0)
		}

		return NewValListFromList(l), nil

	case "extend":
		_, err := mpListExtend.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		for _, x := range args[0].List().Data {
			l.Append(x)
		}
		return NewValListFromList(l), nil

	case "slice":
		alog, err := mpListSlice.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		var ret []Val

		length := l.Length()
		start := int(args[0].Int())
		end := length

		if alog == 2 {
			end = int(args[1].Int())
			if end > length {
				end = length
			}
		}

		if start >= length {
			start = length
		}

		ret = l.Data[start:end]
		return NewValListRaw(ret), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}
