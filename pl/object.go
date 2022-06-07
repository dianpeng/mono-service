package pl

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// sadly go does not have a good sumtype, or at least does not expose it as
// builtin types except using interface{}. Currently our implementation is
// really bad but simple to use, at the cost of memory overhead. We will revisit
// it when we have time

const (
	ValInt = iota
	ValReal
	ValStr
	ValBool
	ValNull
	ValPair
	ValList
	ValMap
	ValRegexp
	ValUsr
)

// index operator for reading
type UValIndex func(interface{}, Val) (Val, error)

type UValIndexSet func(interface{}, Val, Val) error

// dot operator for reading
type UValDot func(interface{}, string) (Val, error)

type UValDotSet func(interface{}, string, Val) error

// method invocation
type UValMethod func(interface{}, string, []Val) (Val, error)

// to string operator
type UValToString func(interface{}) (string, error)

// to json operator
type UValToJSON func(interface{}) (string, error)

// get unique id/type information
type UValId func(interface{}) string

// get a debuggable information
type UValInfo func(interface{}) string

// convert user value back to a go native type, ie interface{}. This function
// will never fail
type UValToNative func(interface{}) interface{}

type UVal struct {
	Context    interface{}
	IndexFn    UValIndex
	IndexSetFn UValIndexSet
	DotFn      UValDot
	DotSetFn   UValDotSet
	MethodFn   UValMethod
	ToStringFn UValToString
	ToJSONFn   UValToJSON
	IdFn       UValId
	InfoFn     UValInfo
	ToNativeFn UValToNative
}

type List struct {
	Data []Val
}

func NewList() *List {
	return &List{
		Data: make([]Val, 0, 0),
	}
}

type Map struct {
	Data map[string]Val
}

func NewMap() *Map {
	return &Map{
		Data: make(map[string]Val),
	}
}

type Pair struct {
	First  Val
	Second Val
}

type Val struct {
	Type   int
	vInt   int64
	vReal   float64
	Bool   bool
	String string
	Regexp *regexp.Regexp
	Pair   *Pair
	List   *List
	Map    *Map
	Usr    *UVal
}

func (v *Val) Int() int64 {
	return v.vInt
}

func (v *Val) Real() float64 {
	return v.vReal
}

func NewValNull() Val {
	return Val{
		Type: ValNull,
	}
}

func NewValInt64(i int64) Val {
	return Val{
		Type: ValInt,
		vInt: i,
	}
}

func NewValInt(i int) Val {
	return Val{
		Type: ValInt,
		vInt: int64(i),
	}
}

func NewValStr(s string) Val {
	return Val{
		Type:   ValStr,
		String: s,
	}
}

func NewValReal(d float64) Val {
	return Val{
		Type: ValReal,
		vReal: d,
	}
}

func NewValBool(b bool) Val {
	return Val{
		Type: ValBool,
		Bool: b,
	}
}

func NewValPair(f Val, s Val) Val {
	return Val{
		Type: ValPair,
		Pair: &Pair{
			First:  f,
			Second: s,
		},
	}
}

func NewValRegexp(r *regexp.Regexp) Val {
	return Val{
		Type:   ValRegexp,
		Regexp: r,
	}
}

func NewValListRaw(d []Val) Val {
	return Val{
		Type: ValList,
		List: &List{
			Data: d,
		},
	}
}

func NewValList() Val {
	return Val{
		Type: ValList,
		List: NewList(),
	}
}

func NewValMap() Val {
	return Val{
		Type: ValMap,
		Map:  NewMap(),
	}
}

func NewValStrList(s []string) Val {
	r := NewValList()
	for _, vv := range s {
		r.AddList(NewValStr(vv))
	}
	return r
}

func NewValIntList(s []int) Val {
	r := NewValList()
	for _, vv := range s {
		r.AddList(NewValInt(vv))
	}
	return r
}

func NewValUsr(
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
	f9 UValToNative) Val {
	return Val{
		Type: ValUsr,
		Usr: &UVal{
			Context:    c,
			IndexFn:    f0,
			IndexSetFn: f1,
			DotFn:      f2,
			DotSetFn:   f3,
			MethodFn:   f4,
			ToStringFn: f5,
			ToJSONFn:   f6,
			IdFn:       f7,
			InfoFn:     f8,
			ToNativeFn: f9,
		},
	}
}

func NewValUsrImmutable(
	c interface{},
	f0 UValIndex,
	f2 UValDot,
	f4 UValMethod,
	f5 UValToString,
	f6 UValToJSON,
	f7 UValId,
	f8 UValInfo,
	f9 UValToNative) Val {
	return Val{
		Type: ValUsr,
		Usr: &UVal{
			Context:    c,
			IndexFn:    f0,
			IndexSetFn: nil,
			DotFn:      f2,
			DotSetFn:   nil,
			MethodFn:   f4,
			ToStringFn: f5,
			ToJSONFn:   f6,
			IdFn:       f7,
			InfoFn:     f8,
			ToNativeFn: f9,
		},
	}
}

func (v *Val) IsNumber() bool {
	switch v.Type {
	case ValInt, ValReal:
		return true
	default:
		return false
	}
}

func (v *Val) AddList(vv Val) {
	must(v.Type == ValList, "AddList: must be list")
	v.List.Data = append(v.List.Data, vv)
}

func (v *Val) AddMap(key string, val Val) {
	must(v.Type == ValMap, "AddMap: must be map")
	v.Map.Data[key] = val
}

// never failed
func (v *Val) ToBoolean() bool {
	switch v.Type {
	case ValInt:
		return v.Int() != 0
	case ValReal:
		return v.Real() != 0
	case ValStr:
		return len(v.String) != 0
	case ValBool:
		return v.Bool
	case ValNull:
		return true
	case ValList:
		return len(v.List.Data) != 0
	case ValMap:
		return len(v.Map.Data) != 0
	default:
		return false
	}
}

func (v *Val) ToIndex() (int, error) {
	switch v.Type {
	case ValInt:
		if v.Int() >= 0 {
			return int(v.Int()), nil
		}
		return 0, fmt.Errorf("negative value cannot be index")

	default:
		return 0, fmt.Errorf("none integer type cannot be index")
	}
}

// convert whatever hold in boxed value back to native go type to be used in
// external go envronment. Notes, for user type we just do nothing for now,
// ie just returns a string.
func (v *Val) ToNative() interface{} {
	switch v.Type {
	case ValInt:
		return v.Int()
	case ValReal:
		return v.Real()
	case ValStr:
		return v.String
	case ValBool:
		return v.Bool
	case ValNull:
		return nil
	case ValList:
		var x []interface{}
		for _, xx := range v.List.Data {
			x = append(x, xx.ToNative())
		}
		return x
	case ValMap:
		x := make(map[string]interface{})
		for key, val := range v.Map.Data {
			x[key] = val.ToNative()
		}
		return x
	case ValPair:
		return [2]interface{}{
			v.Pair.First.ToNative(),
			v.Pair.Second.ToNative(),
		}
	case ValRegexp:
		return fmt.Sprintf("[Regexp: %s]", v.Regexp.String())

	default:
		if v.Usr.ToNativeFn != nil {
			return v.Usr.ToNativeFn(v.Usr.Context)
		} else {
			return v.Usr.Context
		}
	}
}

func (v *Val) ToString() (string, error) {
	switch v.Type {
	case ValInt:
		return fmt.Sprintf("%d", v.Int()), nil
	case ValReal:
		return fmt.Sprintf("%f", v.Real()), nil
	case ValBool:
		if v.Bool {
			return "true", nil
		} else {
			return "false", nil
		}
	case ValNull:
		return "", fmt.Errorf("cannot convert Null to string")
	case ValStr:
		return v.String, nil

	case ValRegexp:
		return v.Regexp.String(), nil

	case ValList, ValMap, ValPair:
		return "", fmt.Errorf("cannot convert List/Map/Pair to string")

	default:
		return v.Usr.ToStringFn(v.Usr.Context)
	}
	return "", nil
}

func (v *Val) Index(idx Val) (Val, error) {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull:
		return NewValNull(), fmt.Errorf("cannot index primitive type")

	case ValRegexp:
		return NewValNull(), fmt.Errorf("cannot index regexp")

	case ValStr:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i >= len(v.String) {
			return NewValNull(), fmt.Errorf("index out of range")
		}
		return NewValStr(v.String[i : i+1]), nil

	case ValPair:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i == 0 {
			return v.Pair.First, nil
		}
		if i == 1 {
			return v.Pair.Second, nil
		}

		return NewValNull(), fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")

	case ValList:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i >= len(v.List.Data) {
			return NewValNull(), fmt.Errorf("index out of range")
		}
		return v.List.Data[i], nil

	case ValMap:
		i, err := idx.ToString()
		if err != nil {
			return NewValNull(), err
		}
		if vv, ok := v.Map.Data[i]; !ok {
			return NewValNull(), fmt.Errorf("%s key not found", i)
		} else {
			return vv, nil
		}

	default:
		// User
		return v.Usr.IndexFn(v.Usr.Context, idx)
	}
}

func (v *Val) IndexSet(idx, val Val) error {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull:
		return fmt.Errorf("cannot do subfield set by indexing on primitive type")

	case ValRegexp:
		return fmt.Errorf("cannot do subfield set by indexing on regexp")

	case ValStr:
		i, err := idx.ToIndex()
		if err != nil {
			return err
		}
		if i >= len(v.String) {
			return fmt.Errorf("index out of range")
		}
		if val.Type != ValStr {
			return fmt.Errorf("string subfield setting must be type string")
		}

		b := new(bytes.Buffer)
		b.WriteString(v.String[:i])
		b.WriteString(val.String)
		b.WriteString(v.String[i:])

		v.String = b.String()
		return nil

	case ValPair:
		i, err := idx.ToIndex()
		if err != nil {
			return err
		}
		if i == 0 {
			v.Pair.First = val
			return nil
		}
		if i == 1 {
			v.Pair.Second = val
			return nil
		}

		return fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")

	case ValList:
		i, err := idx.ToIndex()
		if err != nil {
			return err
		}

		if i >= len(v.List.Data) {
			for j := len(v.List.Data); j < i; j++ {
				v.List.Data = append(v.List.Data, NewValNull())
			}
			v.List.Data = append(v.List.Data, val)
		} else {
			v.List.Data[i] = val
		}
		return nil

	case ValMap:
		i, err := idx.ToString()
		if err != nil {
			return err
		}
		v.Map.Data[i] = val
		return nil

	default:
		if v.Usr.IndexSetFn != nil {
			return v.Usr.IndexSetFn(v.Usr.Context, idx, val)
		} else {
			return fmt.Errorf("type: %s does not support index operator mutation", v.Id())
		}
	}
}

func (v *Val) Dot(i string) (Val, error) {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValStr, ValList:
		return NewValNull(), fmt.Errorf("cannot apply dot operator on none Map or User type")

	case ValRegexp:
		return NewValNull(), fmt.Errorf("cannot apply dot operator on regexp")

	case ValMap:
		if vv, ok := v.Map.Data[i]; !ok {
			return NewValNull(), fmt.Errorf("%s key not found", i)
		} else {
			return vv, nil
		}

	case ValPair:
		if i == "first" {
			return v.Pair.First, nil
		}
		if i == "second" {
			return v.Pair.Second, nil
		}

		return NewValNull(), fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")

	default:
		return v.Usr.DotFn(v.Usr.Context, i)
	}
}

func (v *Val) DotSet(i string, val Val) error {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValStr, ValList:
		return fmt.Errorf("cannot apply dot operator on none Map or User type")

	case ValRegexp:
		return fmt.Errorf("cannot apply dot operator on regexp")

	case ValMap:
		v.Map.Data[i] = val
		return nil

	case ValPair:
		if i == "first" {
			v.Pair.First = val
			return nil
		}
		if i == "second" {
			v.Pair.Second = val
			return nil
		}

		return fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")

	default:
		if v.Usr.DotSetFn != nil {
			return v.Usr.DotSetFn(v.Usr.Context, i, val)
		} else {
			return fmt.Errorf("type: %s does not support dot operator mutation", v.Id())
		}
	}
}

func (v *Val) methodInt(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: int:to_string must have 0 arguments")
		}
		return NewValStr(fmt.Sprintf("%d", v.Int())), nil
	default:
		return NewValNull(), fmt.Errorf("method: int:%s is unknown", name)
	}
}

func (v *Val) methodReal(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:to_string must have 0 arguments")
		}
		return NewValStr(fmt.Sprintf("%f", v.Real())), nil
	case "is_nan":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:is_ana must have 0 arguments")
		}
		return NewValBool(math.IsNaN(v.Real())), nil
	case "is_inf":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:is_inf must have 0 arguments")
		}
		return NewValBool(math.IsInf(v.Real(), 0)), nil
	case "is_ninf":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:is_ninf must have 0 arguments")
		}
		return NewValBool(math.IsInf(v.Real(), -1)), nil
	case "is_pinf":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:is_pinf must have 0 arguments")
		}
		return NewValBool(math.IsInf(v.Real(), 1)), nil
	case "cell":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:cell must have 0 arguments")
		}
		return NewValReal(math.Ceil(v.Real())), nil
	case "floor":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: real:floor must have 0 arguments")
		}
		return NewValReal(math.Floor(v.Real())), nil

	default:
		return NewValNull(), fmt.Errorf("method: real:%s is unknown", name)
	}
}

func (v *Val) methodBool(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: bool:to_string must have 0 arguments")
		}
		var r string
		if v.Bool {
			r = "true"
		} else {
			r = "false"
		}
		return NewValStr(r), nil
	default:
		return NewValNull(), fmt.Errorf("method: bool:%s is unknown", name)
	}
}

func (v *Val) methodNull(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: null:to_string must have 0 arguments")
		}
		return NewValStr("null"), nil
	default:
		return NewValNull(), fmt.Errorf("method: null:%s is unknown", name)
	}
}

func (v *Val) methodStr(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: str:to_string must have 0 arguments")
		}
		return *v, nil

	case "len":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: str:len must have 0 arguments")
		}
		return NewValInt(len(v.String)), nil

	case "upper":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: str:upper must have 0 arguments")
		}
		v.String = strings.ToUpper(v.String)
		return *v, nil

	case "lower":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: str:lower must have 0 arguments")
		}
		v.String = strings.ToLower(v.String)
		return *v, nil

	case "substr":
		if len(args) != 2 && len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: str:substr must have 1 or 2 arguments")
		}
		if args[0].Type != ValInt || (len(args) == 2 && args[1].Type != ValInt) {
			return NewValNull(), fmt.Errorf("method: str:substr argument must be integer")
		}
		var ret string
		if len(args) == 2 {
			ret = v.String[args[0].Int():args[1].Int()]
		} else {
			ret = v.String[args[0].Int():]
		}
		return NewValStr(ret), nil

	case "index":
		if len(args) != 2 && len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: str:index must have 1 or 2 arguments")
		}
		if args[0].Type != ValStr || (len(args) == 2 && args[1].Type != ValInt) {
			return NewValNull(), fmt.Errorf("method: str:index argument must be one string and one int")
		}
		if len(args) == 1 {
			return NewValInt(strings.Index(v.String, args[0].String)), nil
		} else {
			sub := v.String[args[1].Int():]
			where := strings.Index(sub, args[0].String)
			if where == -1 {
				return NewValInt(-1), nil
			} else {
				return NewValInt64(int64(where) + args[1].Int()), nil
			}
		}

	default:
		return NewValNull(), fmt.Errorf("method: str:%s is unknown", name)
	}
}

func (v *Val) methodList(name string, args []Val) (Val, error) {
	switch name {
	case "len":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: list:len must have 0 arguments")
		}
		return NewValInt(len(v.List.Data)), nil

	case "push_back":
		if len(args) == 0 {
			return NewValNull(), fmt.Errorf("method: list:push_back must at least have 1 argument")
		}
		for _, x := range args {
			v.AddList(x)
		}
		return NewValNull(), nil

	case "pop_back":
		if len(args) != 0 && len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: list:pop_back must have 0 or 1 argument")
		}
		num := 1
		if len(args) == 1 {
			if args[0].Type != ValInt || args[0].Int() < 0 {
				return NewValNull(), fmt.Errorf("method: list:pop_back invalid argument")
			}
			num = int(args[0].Int())
		}

		if num < len(v.List.Data) {
			v.List.Data = v.List.Data[0 : len(v.List.Data)-num]
		} else {
			v.List.Data = make([]Val, 0, 0)
		}

		return NewValNull(), nil

	case "extend":
		if len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: list:extend must have 1 argument")
		}
		if args[0].Type != ValList {
			return NewValNull(), fmt.Errorf("method: list:extend invalid argument")
		}
		for _, x := range args[0].List.Data {
			v.AddList(x)
		}
		return NewValNull(), nil

	case "slice":
		if len(args) != 2 && len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: list:slice must have 1 or 2 arguments")
		}
		if args[0].Type != ValInt || (len(args) == 2 && args[1].Type != ValInt) {
			return NewValNull(), fmt.Errorf("method: list:slice argument must be integer")
		}
		var ret []Val
		if len(args) == 2 {
			ret = v.List.Data[args[0].Int():args[1].Int()]
		} else {
			ret = v.List.Data[args[0].Int():]
		}
		return NewValListRaw(ret), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}

func (v *Val) methodMap(name string, args []Val) (Val, error) {
	switch name {
	case "len":
		if len(args) != 0 {
			return NewValNull(), fmt.Errorf("method: map:len must have 0 arguments")
		}
		return NewValInt(len(v.Map.Data)), nil

	case "set":
		if len(args) != 2 {
			return NewValNull(), fmt.Errorf("method: map:set must have 2 arguments")
		}
		if args[0].Type != ValStr {
			return NewValNull(), fmt.Errorf("method: map:set invalid argument")
		}
		v.Map.Data[args[0].String] = args[1]
		return NewValNull(), nil

	case "del":
		if len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: map:del must have 1 argument")
		}
		if args[0].Type != ValStr {
			return NewValNull(), fmt.Errorf("method: map:del invalid argument")
		}
		delete(v.Map.Data, args[0].String)
		return NewValNull(), nil

	case "get":
		if len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: map:get must have 2 argument")
		}
		if args[0].Type != ValStr {
			return NewValNull(), fmt.Errorf("method: map:get invalid argument")
		}
		v, ok := v.Map.Data[args[0].String]
		if !ok {
			return args[1], nil
		} else {
			return v, nil
		}

	case "has":
		if len(args) != 1 {
			return NewValNull(), fmt.Errorf("method: map:has must have 1 argument")
		}
		if args[0].Type != ValStr {
			return NewValNull(), fmt.Errorf("method: map:has invalid argument")
		}
		_, ok := v.Map.Data[args[0].String]
		return NewValBool(ok), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}

func (v *Val) methodUsr(name string, args []Val) (Val, error) {
	if v.Usr.MethodFn != nil {
		return v.Usr.MethodFn(v.Usr.Context, name, args)
	} else {
		return NewValNull(), fmt.Errorf("%s:%s is unknown", v.Id(), name)
	}
}

func (v *Val) Method(name string, args []Val) (Val, error) {
	switch v.Type {
	case ValInt:
		return v.methodInt(name, args)

	case ValReal:
		return v.methodReal(name, args)

	case ValBool:
		return v.methodBool(name, args)

	case ValNull:
		return v.methodNull(name, args)

	case ValStr:
		return v.methodStr(name, args)

	case ValList:
		return v.methodList(name, args)

	case ValMap:
		return v.methodMap(name, args)

	case ValRegexp:
		break

	default:
		return v.methodUsr(name, args)
	}

	return NewValNull(), fmt.Errorf("%s:%s is unknown", v.Id(), name)
}

func (v *Val) Id() string {
	switch v.Type {
	case ValInt:
		return "int"
	case ValReal:
		return "real"
	case ValBool:
		return "bool"
	case ValNull:
		return "null"
	case ValStr:
		return "str"
	case ValList:
		return "list"
	case ValMap:
		return "map"
	case ValPair:
		return "pair"
	case ValRegexp:
		return "regexp"
	default:
		if v.Usr.IdFn != nil {
			return v.Usr.IdFn(v.Usr.Context)
		} else {
			return "user"
		}
	}
}

func (v *Val) Info() string {
	switch v.Type {
	case ValInt:
		return fmt.Sprintf("[int: %d]", v.Int())
	case ValReal:
		return fmt.Sprintf("[real: %f]", v.Real())
	case ValBool:
		if v.Bool {
			return "[bool: true]"
		} else {
			return "[bool: false]"
		}
	case ValNull:
		return "[null]"
	case ValStr:
		return fmt.Sprintf("[string: %s]", v.String)
	case ValList:
		return fmt.Sprintf("[list: %d]", len(v.List.Data))
	case ValMap:
		return fmt.Sprintf("[map: %d]", len(v.Map.Data))
	case ValPair:
		return fmt.Sprintf("[pair: %s=>%s]", v.Pair.First.Info(), v.Pair.Second.Info())
	case ValRegexp:
		return fmt.Sprintf("[regexp: %s]", v.Regexp.String())
	default:
		if v.Usr.InfoFn != nil {
			return v.Usr.InfoFn(v.Usr.Context)
		} else {
			return fmt.Sprintf("[user: %s]", v.Id())
		}
	}
}
