package pl

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strings"
)

const (
	ValNull = iota
	ValInt
	ValReal
	ValStr
	ValBool
	ValPair
	ValList
	ValMap
	ValRegexp
	ValIter
	ValClosure
	ValUsr

	// should not be visiable, and only be used by internal evaluator
	valFrame
)

const (
	MetaMethod = "__@method"

	MetaIndex    = "__@index"
	MetaIndexSet = "__@index_set"
	MetaDot      = "__@dot"
	MetaDotSet   = "__@dot_set"

	MetaToString = "__@to_string"
	MetaToJSON   = "__@to_json"
	MetaToIter   = "__@to_iter"
)

const (
	ClosureScript = iota
	ClosureNative
)

func GetClosureTypeId(id int) string {
	switch id {
	case ClosureScript:
		return "closure_script"
	case ClosureNative:
		return "closure_native"
	default:
		unreachable("invalid closure type")
		return ""
	}
}

// Closure interface, ie wrapping around the function calls
type Closure interface {
	Call([]Val) (Val, error)
	Type() int
	Id() string
}

// Iterator interface, matching against the bytecode semantic
type Iter interface {
	// pure function, returns whether the current iterator is valid for deference
	// or not.
	Has() bool

	// if applicable move the iterator to next position and returns whether the
	// status of iterator is valid *AFTER* the next
	Next() bool

	// get the value of the iterator, the first is index and second is the value
	Deref() (Val, Val, error)
}

// Internally, all the user extending type will be defined just as a UsrVal
// object which can be used to extending to user's own needs
type Usr interface {
	// Meta function supported by the runtime object model
	Index(Val) (Val, error)
	IndexSet(Val, Val) error
	Dot(string) (Val, error)
	DotSet(string, Val) error
	Method(string, []Val) (Val, error)
	ToString() (string, error)
	ToJSON() (Val, error)
	Id() string
	Info() string
	ToNative() interface{}

	// return a iterator
	NewIterator() (Iter, error)

	// support invocation operations, ie calling a user type
}

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

type UVal struct {
	context    interface{}
	indexFn    UValIndex
	indexSetFn UValIndexSet
	dotFn      UValDot
	dotSetFn   UValDotSet
	methodFn   UValMethod
	toStringFn UValToString
	toJSONFn   UValToJSON
	idFn       UValId
	infoFn     UValInfo
	toNativeFn UValToNative
	iterFn     UValIter
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
) *UVal {
	return &UVal{
		context:    c,
		indexFn:    f0,
		indexSetFn: f1,
		dotFn:      f2,
		dotSetFn:   f3,
		methodFn:   f4,
		toStringFn: f5,
		toJSONFn:   f6,
		idFn:       f7,
		infoFn:     f8,
		toNativeFn: f9,
		iterFn:     f10,
	}
}

func NewUValData(
	c interface{},
) *UVal {
	return &UVal{
		context: c,
	}
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
	Type int

	vData interface{}
}

func (v *Val) Int() int64 {
	x, ok := v.vData.(int64)
	must(ok, "must be int")
	return x
}

func (v *Val) SetInt(i int64) {
	v.Type = ValInt
	v.vData = i
}

func (v *Val) Real() float64 {
	x, ok := v.vData.(float64)
	must(ok, "must be real")
	return x
}

func (v *Val) SetReal(vv float64) {
	v.Type = ValReal
	v.vData = vv
}

func (v *Val) Bool() bool {
	x, ok := v.vData.(bool)
	must(ok, "must be bool")
	return x
}

func (v *Val) SetBool(vv bool) {
	v.Type = ValBool
	v.vData = vv
}

func (v *Val) String() string {
	x, ok := v.vData.(string)
	must(ok, "must be string")
	return x
}

func (v *Val) SetString(vv string) {
	v.Type = ValStr
	v.vData = vv
}

func (v *Val) Regexp() *regexp.Regexp {
	x, ok := v.vData.(*regexp.Regexp)
	must(ok, "must be regexp")
	return x
}

func (v *Val) SetRegexp(vv *regexp.Regexp) {
	v.Type = ValRegexp
	v.vData = vv
}

func (v *Val) List() *List {
	x, ok := v.vData.(*List)
	must(ok, "must be list")
	return x
}

func (v *Val) SetList(vv *List) {
	v.Type = ValList
	v.vData = vv
}

func (v *Val) Map() *Map {
	x, ok := v.vData.(*Map)
	must(ok, "must be map")
	return x
}

func (v *Val) SetMap(vv *Map) {
	v.Type = ValMap
	v.vData = vv
}

func (v *Val) UVal() *UVal {
	x, ok := v.vData.(*UVal)
	must(ok, "must be uval")
	return x
}

func (v *Val) SetUVal(u *UVal) {
	v.Type = ValUsr
	v.vData = u
}

func (v *Val) Usr() Usr {
	x, ok := v.vData.(Usr)
	must(ok, "must be user")
	return x
}

func (v *Val) SetUsr(vv Usr) {
	v.Type = ValUsr
	v.vData = vv
}

func (v *Val) Pair() *Pair {
	x, ok := v.vData.(*Pair)
	must(ok, "must be pair")
	return x
}

func (v *Val) SetPair(vv *Pair) {
	v.Type = ValPair
	v.vData = vv
}

func (v *Val) SetPairKV(first Val, second Val) {
	v.SetPair(&Pair{
		First:  first,
		Second: second,
	},
	)
}

func (v *Val) SetIter(iter Iter) {
	v.Type = ValIter
	v.vData = iter
}

func (v *Val) Iter() Iter {
	x, ok := v.vData.(Iter)
	must(ok, "must be iterator")
	return x
}

func (v *Val) SetClosure(closure Closure) {
	v.Type = ValClosure
	v.vData = closure
}

func (v *Val) Closure() Closure {
	x, ok := v.vData.(Closure)
	must(ok, "must be closure")
	return x
}

func (v *Val) IsNumber() bool {
	switch v.Type {
	case ValInt, ValReal:
		return true
	default:
		return false
	}
}

func (v *Val) IsInt() bool {
	return v.Type == ValInt
}

func (v *Val) IsReal() bool {
	return v.Type == ValReal
}

func (v *Val) IsBool() bool {
	return v.Type == ValBool
}

func (v *Val) IsNull() bool {
	return v.Type == ValNull
}

func (v *Val) IsString() bool {
	return v.Type == ValStr
}

func (v *Val) IsPair() bool {
	return v.Type == ValPair
}

func (v *Val) IsList() bool {
	return v.Type == ValList
}

func (v *Val) IsMap() bool {
	return v.Type == ValMap
}

func (v *Val) IsRegexp() bool {
	return v.Type == ValRegexp
}

func (v *Val) IsIter() bool {
	return v.Type == ValIter
}

func (v *Val) IsUsr() bool {
	return v.Type == ValUsr
}

func (v *Val) IsClosure() bool {
	return v.Type == ValClosure
}

// val frame helper
func (v *Val) isFrame() bool {
	return v.Type == valFrame
}

func (v *Val) setFrame(f interface{}) {
	v.Type = valFrame
	v.vData = f
}

func (v *Val) frame() interface{} {
	must(v.Type == valFrame, "must be frame")
	return v.vData
}

func newValFrame(f interface{}) Val {
	return Val{
		Type:  valFrame,
		vData: f,
	}
}

// New function ----------------------------------------------------------------
func NewValNull() Val {
	return Val{
		Type: ValNull,
	}
}

func NewValInt64(i int64) Val {
	return Val{
		Type:  ValInt,
		vData: i,
	}
}

func NewValInt(i int) Val {
	return Val{
		Type:  ValInt,
		vData: int64(i),
	}
}

func NewValStr(s string) Val {
	return Val{
		Type:  ValStr,
		vData: s,
	}
}

func NewValReal(d float64) Val {
	return Val{
		Type:  ValReal,
		vData: d,
	}
}

func NewValBool(b bool) Val {
	return Val{
		Type:  ValBool,
		vData: b,
	}
}

func NewValPair(f Val, s Val) Val {
	return Val{
		Type: ValPair,
		vData: &Pair{
			First:  f,
			Second: s,
		},
	}
}

func NewValRegexp(r *regexp.Regexp) Val {
	return Val{
		Type:  ValRegexp,
		vData: r,
	}
}

func NewValListRaw(d []Val) Val {
	return Val{
		Type: ValList,
		vData: &List{
			Data: d,
		},
	}
}

func NewValList() Val {
	return Val{
		Type:  ValList,
		vData: NewList(),
	}
}

func NewValMap() Val {
	return Val{
		Type:  ValMap,
		vData: NewMap(),
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

func NewValUsr(u Usr) Val {
	return Val{
		Type:  ValUsr,
		vData: u,
	}
}

func NewValUValData(
	c interface{},
) Val {
	return Val{
		Type:  ValUsr,
		vData: NewUValData(c),
	}
}

func NewValUVal(
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
) Val {
	return Val{
		Type: ValUsr,
		vData: NewUVal(
			c,
			f0,
			f1,
			f2,
			f3,
			f4,
			f5,
			f6,
			f7,
			f8,
			f9,
			f10,
		),
	}
}

func NewValUValImmutable(
	c interface{},
	f0 UValIndex,
	f2 UValDot,
	f4 UValMethod,
	f5 UValToString,
	f6 UValToJSON,
	f7 UValId,
	f8 UValInfo,
	f9 UValToNative,
	f10 UValIter,
) Val {
	return Val{
		Type: ValUsr,
		vData: NewUVal(
			c,
			f0,
			nil,
			f2,
			nil,
			f4,
			f5,
			f6,
			f7,
			f8,
			f9,
			f10,
		),
	}
}

func (v *Val) AddList(vv Val) {
	must(v.Type == ValList, "AddList: must be list")
	v.List().Data = append(v.List().Data, vv)
}

func (v *Val) AddMap(key string, val Val) {
	must(v.Type == ValMap, "AddMap: must be map")
	v.Map().Data[key] = val
}

// never failed
func (v *Val) ToBoolean() bool {
	switch v.Type {
	case ValInt:
		return v.Int() != 0
	case ValReal:
		return v.Real() != 0
	case ValStr:
		return len(v.String()) != 0
	case ValBool:
		return v.Bool()
	case ValNull:
		return true
	case ValList:
		return len(v.List().Data) != 0
	case ValMap:
		return len(v.Map().Data) != 0
	case ValIter:
		return v.Iter().Has()
	case ValClosure:
		return true
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
		return v.String()
	case ValBool:
		return v.Bool()
	case ValNull:
		return nil
	case ValList:
		var x []interface{}
		for _, xx := range v.List().Data {
			x = append(x, xx.ToNative())
		}
		return x

	case ValMap:
		x := make(map[string]interface{})
		for key, val := range v.Map().Data {
			x[key] = val.ToNative()
		}
		return x

	case ValPair:
		return [2]interface{}{
			v.Pair().First.ToNative(),
			v.Pair().Second.ToNative(),
		}

	case ValRegexp:
		return fmt.Sprintf("[Regexp: %s]", v.Regexp().String())

	case ValIter:
		return v.Iter()

	case ValClosure:
		return v.Closure()

	case valFrame:
		return nil

	default:
		return v.Usr().ToNative()
	}
}

func (v *Val) ToString() (string, error) {
	switch v.Type {
	case ValInt:
		return fmt.Sprintf("%d", v.Int()), nil
	case ValReal:
		return fmt.Sprintf("%f", v.Real()), nil
	case ValBool:
		if v.Bool() {
			return "true", nil
		} else {
			return "false", nil
		}
	case ValNull:
		return "null", nil
	case ValStr:
		return v.String(), nil

	case ValRegexp:
		return v.Regexp().String(), nil

	case ValList, ValMap, ValPair:
		return "", fmt.Errorf("cannot convert List/Map/Pair to string")

	case ValIter:
		return "iter", nil

	case ValClosure:
		return "closure", nil

	case valFrame:
		return "", nil

	default:
		return v.Usr().ToString()
	}
}

func (v *Val) Index(idx Val) (Val, error) {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValIter, ValClosure:
		return NewValNull(), fmt.Errorf("cannot index type: %s", v.Id())

	case ValRegexp:
		return NewValNull(), fmt.Errorf("cannot index regexp")

	case ValStr:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i >= len(v.String()) {
			return NewValNull(), fmt.Errorf("index out of range")
		}
		return NewValStr(v.String()[i : i+1]), nil

	case ValPair:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i == 0 {
			return v.Pair().First, nil
		}
		if i == 1 {
			return v.Pair().Second, nil
		}

		return NewValNull(), fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")

	case ValList:
		i, err := idx.ToIndex()
		if err != nil {
			return NewValNull(), err
		}
		if i >= len(v.List().Data) {
			return NewValNull(), fmt.Errorf("index out of range")
		}
		return v.List().Data[i], nil

	case ValMap:
		i, err := idx.ToString()
		if err != nil {
			return NewValNull(), err
		}
		if vv, ok := v.Map().Data[i]; !ok {
			return NewValNull(), fmt.Errorf("%s key not found", i)
		} else {
			return vv, nil
		}

	case valFrame:
		return NewValNull(), nil

	default:
		return v.Usr().Index(idx)
	}
}

func (v *Val) IndexSet(idx, val Val) error {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValIter, ValClosure:
		return fmt.Errorf("cannot do index set on type: %s", v.Id())

	case ValRegexp:
		return fmt.Errorf("cannot do subfield set by indexing on regexp")

	case ValStr:
		i, err := idx.ToIndex()
		if err != nil {
			return err
		}
		if i >= len(v.String()) {
			return fmt.Errorf("index out of range")
		}
		if val.Type != ValStr {
			return fmt.Errorf("string subfield setting must be type string")
		}

		b := new(bytes.Buffer)
		b.WriteString(v.String()[:i])
		b.WriteString(val.String())
		b.WriteString(v.String()[i:])
		v.SetString(b.String())
		return nil

	case ValPair:
		i, err := idx.ToIndex()
		if err != nil {
			return err
		}
		if i == 0 {
			v.Pair().First = val
			return nil
		}
		if i == 1 {
			v.Pair().Second = val
			return nil
		}

		return fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")

	case ValList:
		i, err := idx.ToIndex()
		if err != nil {
			return err
		}

		if i >= len(v.List().Data) {
			for j := len(v.List().Data); j < i; j++ {
				v.List().Data = append(v.List().Data, NewValNull())
			}
			v.List().Data = append(v.List().Data, val)
		} else {
			v.List().Data[i] = val
		}
		return nil

	case ValMap:
		i, err := idx.ToString()
		if err != nil {
			return err
		}
		v.Map().Data[i] = val
		return nil

	case valFrame:
		return nil

	default:
		return v.Usr().IndexSet(idx, val)
	}
}

func (v *Val) Dot(i string) (Val, error) {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValStr, ValList, ValIter, ValClosure:
		return NewValNull(), fmt.Errorf("cannot do dot on type: %s", v.Id())

	case ValRegexp:
		return NewValNull(), fmt.Errorf("cannot apply dot operator on regexp")

	case ValMap:
		if vv, ok := v.Map().Data[i]; !ok {
			return NewValNull(), fmt.Errorf("%s key not found", i)
		} else {
			return vv, nil
		}

	case ValPair:
		if i == "first" {
			return v.Pair().First, nil
		}
		if i == "second" {
			return v.Pair().Second, nil
		}

		return NewValNull(), fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")

	case valFrame:
		return NewValNull(), nil

	default:
		return v.Usr().Dot(i)
	}
}

func (v *Val) DotSet(i string, val Val) error {
	switch v.Type {
	case ValInt, ValReal, ValBool, ValNull, ValStr, ValList, ValIter, ValClosure:
		return fmt.Errorf("cannot do dot set on type: %s", v.Id())

	case ValRegexp:
		return fmt.Errorf("cannot apply dot operator on regexp")

	case ValMap:
		v.Map().Data[i] = val
		return nil

	case ValPair:
		if i == "first" {
			v.Pair().First = val
			return nil
		}
		if i == "second" {
			v.Pair().Second = val
			return nil
		}

		return fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")

	case valFrame:
		return nil

	default:
		return v.Usr().DotSet(i, val)
	}
}

var (
	// int#method
	mpIntToString = MustNewFuncProto("int.to_string", "%0")

	// real#method
	mpRealToString = MustNewFuncProto("real.to_string", "%0")
	mpRealIsNaN    = MustNewFuncProto("real.is_nan", "%0")
	mpRealIsInf    = MustNewFuncProto("real.is_inf", "%0")
	mpRealIsNInf   = MustNewFuncProto("real.is_ninf", "%0")
	mpRealIsPInf   = MustNewFuncProto("real.is_pinf", "%0")
	mpRealCell     = MustNewFuncProto("real.cell", "%0")
	mpRealFloor    = MustNewFuncProto("real.floor", "%0")

	// bool#method
	mpBoolToString = MustNewFuncProto("bool.to_string", "%0")

	// null#method
	mpNullToString = MustNewFuncProto("null.to_string", "%0")

	// str#method
	mpStrToString = MustNewFuncProto("str.to_string", "%0")
	mpStrLength   = MustNewFuncProto("str.length", "%0")
	mpStrToUpper  = MustNewFuncProto("str.to_upper", "%0")
	mpStrToLower  = MustNewFuncProto("str.to_lower", "%0")
	mpStrSubStr   = MustNewFuncProto("str.substr", "{%d}{%d%d}")
	mpStrIndex    = MustNewFuncProto("str.index", "{%d}{%d%d}")

	// list#method
	mpListLength   = MustNewFuncProto("list.length", "%0")
	mpListPushBack = MustNewFuncProto("list.push_back", "%a")
	mpListPopBack  = MustNewFuncProto("list.pop_back", "%d")
	mpListExtend   = MustNewFuncProto("list.extend", "%l")
	mpListSlice    = MustNewFuncProto("list.slice", "{%d}{%d%d}")

	// map#method
	mpMapLength = MustNewFuncProto("map.length", "%0")
	mpMapSet    = MustNewFuncProto("map.set", "%s%a")
	mpMapDel    = MustNewFuncProto("map.del", "%s")
	mpMapGet    = MustNewFuncProto("map.get", "%s%a")
	mpMapHas    = MustNewFuncProto("map.has", "%s")
)

func (v *Val) methodInt(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		_, err := mpIntToString.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValStr(fmt.Sprintf("%d", v.Int())), nil
	default:
		return NewValNull(), fmt.Errorf("method: int:%s is unknown", name)
	}
}

func (v *Val) methodReal(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		_, err := mpRealToString.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValStr(fmt.Sprintf("%f", v.Real())), nil
	case "is_nan":
		_, err := mpRealIsNaN.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValBool(math.IsNaN(v.Real())), nil
	case "is_inf":
		_, err := mpRealIsInf.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValBool(math.IsInf(v.Real(), 0)), nil
	case "is_ninf":
		_, err := mpRealIsNInf.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValBool(math.IsInf(v.Real(), -1)), nil
	case "is_pinf":
		_, err := mpRealIsPInf.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValBool(math.IsInf(v.Real(), 1)), nil
	case "cell":
		_, err := mpRealCell.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValReal(math.Ceil(v.Real())), nil
	case "floor":
		_, err := mpRealFloor.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValReal(math.Floor(v.Real())), nil

	default:
		return NewValNull(), fmt.Errorf("method: real:%s is unknown", name)
	}
}

func (v *Val) methodBool(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		_, err := mpBoolToString.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		var r string
		if v.Bool() {
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
		_, err := mpNullToString.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValStr("null"), nil
	default:
		return NewValNull(), fmt.Errorf("method: null:%s is unknown", name)
	}
}

func (v *Val) methodStr(name string, args []Val) (Val, error) {
	switch name {
	case "to_string":
		_, err := mpStrToString.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return *v, nil

	case "length":
		_, err := mpStrLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(len(v.String())), nil

	case "to_upper":
		_, err := mpStrToUpper.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValStr(strings.ToUpper(v.String())), nil

	case "to_lower":
		_, err := mpStrToLower.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValStr(strings.ToLower(v.String())), nil

	case "substr":
		alog, err := mpStrSubStr.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		var ret string
		if alog == 2 {
			ret = v.String()[args[0].Int():args[1].Int()]
		} else {
			ret = v.String()[args[0].Int():]
		}
		return NewValStr(ret), nil

	case "index":
		alog, err := mpStrIndex.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		if alog == 1 {
			return NewValInt(strings.Index(v.String(), args[0].String())), nil
		} else {
			sub := v.String()[args[1].Int():]
			where := strings.Index(sub, args[0].String())
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
	case "length":
		_, err := mpListLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(len(v.List().Data)), nil

	case "push_back":
		_, err := mpListPushBack.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		for _, x := range args {
			v.AddList(x)
		}
		return NewValNull(), nil

	case "pop_back":
		_, err := mpListPopBack.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		num := int(args[0].Int())
		if num < len(v.List().Data) {
			v.List().Data = v.List().Data[0 : len(v.List().Data)-num]
		} else {
			v.List().Data = make([]Val, 0, 0)
		}

		return NewValNull(), nil

	case "extend":
		_, err := mpListExtend.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		for _, x := range args[0].List().Data {
			v.AddList(x)
		}
		return NewValNull(), nil

	case "slice":
		alog, err := mpListSlice.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		var ret []Val
		if alog == 2 {
			ret = v.List().Data[args[0].Int():args[1].Int()]
		} else {
			ret = v.List().Data[args[0].Int():]
		}
		return NewValListRaw(ret), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}

func (v *Val) methodMap(name string, args []Val) (Val, error) {
	switch name {
	case "length":
		_, err := mpMapLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(len(v.Map().Data)), nil

	case "set":
		_, err := mpMapSet.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		v.Map().Data[args[0].String()] = args[1]
		return NewValNull(), nil

	case "del":
		_, err := mpMapDel.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		delete(v.Map().Data, args[0].String())
		return NewValNull(), nil

	case "get":
		_, err := mpMapGet.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		v, ok := v.Map().Data[args[0].String()]
		if !ok {
			return args[1], nil
		} else {
			return v, nil
		}

	case "has":
		_, err := mpMapHas.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		_, ok := v.Map().Data[args[0].String()]
		return NewValBool(ok), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}

func (v *Val) methodUsr(name string, args []Val) (Val, error) {
	return v.Usr().Method(name, args)
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

	case ValRegexp, ValIter, ValClosure:
		break

	case valFrame:
		return NewValNull(), nil

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
	case ValIter:
		return "iter"
	case ValClosure:
		return "closure"
	case valFrame:
		return "#frame"
	default:
		return v.Usr().Id()
	}
}

func (v *Val) Info() string {
	switch v.Type {
	case ValInt:
		return fmt.Sprintf("[int: %d]", v.Int())
	case ValReal:
		return fmt.Sprintf("[real: %f]", v.Real())
	case ValBool:
		if v.Bool() {
			return "[bool: true]"
		} else {
			return "[bool: false]"
		}
	case ValNull:
		return "[null]"
	case ValStr:
		return fmt.Sprintf("[string: %s]", v.String())
	case ValList:
		return fmt.Sprintf("[list: %d]", len(v.List().Data))
	case ValMap:
		return fmt.Sprintf("[map: %d]", len(v.Map().Data))
	case ValPair:
		return fmt.Sprintf("[pair: %s=>%s]", v.Pair().First.Info(), v.Pair().Second.Info())
	case ValRegexp:
		return fmt.Sprintf("[regexp: %s]", v.Regexp().String())
	case ValIter:
		return "iter"
	case ValClosure:
		return "closure"
	case valFrame:
		return "#frame"
	default:
		return v.Usr().Info()
	}
}

func init() {
	// testing whether UVal is convertable to Usr
	uval := &UVal{}
	var u Usr
	u = uval
	_ = u
}
