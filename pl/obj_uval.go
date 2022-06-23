package pl

import (
	"fmt"
)

type UValIndex func(interface{}, Val) (Val, error)
type UValIndexSet func(interface{}, Val, Val) error
type UValDot func(interface{}, string) (Val, error)
type UValDotSet func(interface{}, string, Val) error
type UValMethod func(interface{}, string, []Val) (Val, error)
type UValToString func(interface{}) (string, error)
type UValToJSON func(interface{}) (Val, error)
type UValId func(interface{}) string
type UValInfo func(interface{}) string
type UValToNative func(interface{}) interface{}
type UValIter func(interface{}) (Iter, error)
type UValIsImmutable func(interface{}) bool

type UVal struct {
	context     interface{}
	indexFn     UValIndex
	indexSetFn  UValIndexSet
	dotFn       UValDot
	dotSetFn    UValDotSet
	methodFn    UValMethod
	toStringFn  UValToString
	toJSONFn    UValToJSON
	idFn        UValId
	infoFn      UValInfo
	toNativeFn  UValToNative
	iterFn      UValIter
	immutableFn UValIsImmutable
}

func (u *UVal) Context() interface{} {
	return u.context
}

func (u *UVal) Index(b Val) (Val, error) {
	if u.indexFn != nil {
		return u.indexFn(u.context, b)
	} else {
		return NewValNull(), fmt.Errorf("user type does not support *index* operation")
	}
}

func (u *UVal) IndexSet(b Val, c Val) error {
	if u.indexSetFn != nil {
		return u.indexSetFn(u.context, b, c)
	} else {
		return fmt.Errorf("user type does not support *index set* operator")
	}
}

func (u *UVal) Dot(b string) (Val, error) {
	if u.dotFn != nil {
		return u.dotFn(u.context, b)
	} else {
		return NewValNull(), fmt.Errorf("user type does not support *dot* operator")
	}
}

func (u *UVal) DotSet(b string, c Val) error {
	if u.dotSetFn != nil {
		return u.dotSetFn(u.context, b, c)
	} else {
		return fmt.Errorf("user type does not support *dot set* operator")
	}
}

func (u *UVal) Method(b string, c []Val) (Val, error) {
	if u.methodFn != nil {
		return u.methodFn(u.context, b, c)
	} else {
		return NewValNull(), fmt.Errorf("user type does not support *method* operator")
	}
}

func (u *UVal) ToString() (string, error) {
	if u.toStringFn != nil {
		return u.toStringFn(u.context)
	} else {
		return "", fmt.Errorf("user type does not support *to string* operator")
	}
}

func (u *UVal) ToJSON() (Val, error) {
	if u.toJSONFn != nil {
		return u.toJSONFn(u.context)
	} else {
		return NewValNull(), fmt.Errorf("user type does not support *to json* operator")
	}
}

func (u *UVal) Id() string {
	if u.idFn != nil {
		return u.idFn(u.context)
	} else {
		return "[user]"
	}
}

func (u *UVal) Info() string {
	if u.infoFn != nil {
		return u.infoFn(u.context)
	} else {
		return "[user]"
	}
}

func (u *UVal) ToNative() interface{} {
	if u.toNativeFn != nil {
		return u.toNativeFn(u.context)
	} else {
		return u.context
	}
}

func (u *UVal) NewIterator() (Iter, error) {
	if u.iterFn != nil {
		return u.iterFn(u.context)
	} else {
		return nil, fmt.Errorf("user type does not support iterator")
	}
}

func (u *UVal) IsImmutable() bool {
	if u.immutableFn != nil {
		return false
	} else {
		return u.immutableFn(u.context)
	}
}

func NewUVal(
	c interface{},
	f0 UValIndex,
	f1 UValIndexSet,
	f2 UValDot,
	f3 UValDotSet,

	f4 UValMethod,
	f5 UValToString,
	f6 UValToJSON,
	f7 UValId,
	f8 UValInfo,
	f9 UValToNative,
	f10 UValIter,
	f11 UValIsImmutable,
) *UVal {
	return &UVal{
		context:     c,
		indexFn:     f0,
		indexSetFn:  f1,
		dotFn:       f2,
		dotSetFn:    f3,
		methodFn:    f4,
		toStringFn:  f5,
		toJSONFn:    f6,
		idFn:        f7,
		infoFn:      f8,
		toNativeFn:  f9,
		iterFn:      f10,
		immutableFn: f11,
	}
}

func NewUValData(
	c interface{},
) *UVal {
	return &UVal{
		context: c,
	}
}
