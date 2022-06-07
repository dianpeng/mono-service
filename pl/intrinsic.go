package pl

import (
	"fmt"
	"reflect"
	"regexp"
)

type anyfunc interface{}

type intrinsicinfo struct {
	cname    string
	pname    string
	argproto *FuncProto

	// must be prototype of EvalCall, this is the entry of the vm interpreter
	entry EvalCall

	// used when need reflection to wrap the actual function
	rawfunc   anyfunc
	funcvalue *reflect.Value
}

var intrinsicFunc []*intrinsicinfo

func visnil(i reflect.Value) bool {
	switch i.Kind() {
	case reflect.Func,
		reflect.Chan,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return i.IsNil()
	default:
		return false
	}
}

// special helper function to unpack reflection call's return argument into
// acceptable format (Val, error). Notes, we paniced when the return value
// of function is not acceptable. The allowed native format of a call's return
// must be (any, error) or any
func unpack(i reflect.Value) (Val, error) {
	switch i.Kind() {
	case reflect.Int, reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return NewValInt64(int64(i.Int())), nil

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return NewValInt64(int64(i.Uint())), nil

	case reflect.Float32, reflect.Float64:
		return NewValReal(float64(i.Float())), nil

	case reflect.String:
		return NewValStr(i.String()), nil

	case reflect.Bool:
		return NewValBool(i.Bool()), nil

	default:
		if visnil(i) {
			return NewValNull(), nil
		}

		// this will never fail
		ivalue := i.Interface()

		// -------------------------------------------------------------------------
		// special cases that we can recognize and can perform the conversion

		// 1. []byte, convert to string
		if i.Kind() == reflect.Slice {
			ev := i.Elem()
			if ev.Kind() == reflect.Uint8 {
				// just a byte slice, then perform the conversion to string
				barray, ok := ivalue.([]byte)
				must(ok, "must be convertable")
				return NewValStr(string(barray)), nil
			}
		}

		// 2. check whether it is regex, if so we convert it back to Val wrapper
		{
			if v, ok := ivalue.(*regexp.Regexp); ok {
				return NewValRegexp(v), nil
			}
		}

		// N. check whether the value is essentially just a Val, if so just unbox
		//    it and return it back
		{
			if v, ok := ivalue.(Val); ok {
				return v, nil
			}
		}

		break
	}

	return NewValNull(), fmt.Errorf("unsupported conversion from %s to Val", i.Type().String())
}

// the first error is the reflection's value's error and the second is just the
// error about the unpack process
func unpackError(i reflect.Value) (error, error) {
	v, ok := i.Interface().(error)
	if !ok {
		return nil, fmt.Errorf("cannot convert %s to error", i.Type().String())
	} else {
		return v, nil
	}
}

func unpackReturn(r []reflect.Value) (Val, error) {
	rlen := len(r)

	switch rlen {
	case 0:
		return NewValNull(), nil

	case 1:
		a0 := r[0]
		r0, err := unpack(a0)
		if err != nil {
			panic("invalid return value type from reflection call")
		}
		return r0, nil

	case 2:
		r0, err := unpack(r[0])
		if err != nil {
			panic("invalid return value type from reflection call")
		}
		r1, err := unpackError(r[1])
		if err != nil {
			panic("invalid return value type from reflection call, must be error")
		}
		return r0, r1
	default:
		panic(fmt.Sprintf("invalid return format, expect only 1 or 2 arguments, got %d", rlen))
	}
}

// for simplicity, we uses reflect to expose most of the functions back to the
// script environment. In case, when certain function becomes performance
// bottleneck, then we rebind them with hande crafted go code to save reflect
// performance overhead. Using reflect makes us easy to work the binding go
// function into the script.
func funcWrapper(
	info *intrinsicinfo, // info
	args []Val, // function arguments from script side
) (the_value Val, the_error error) {
	must(info.funcvalue != nil, "the reflection entry should be initialized")

	// okay, we will have to recover from the panic of the reflect.Call since it
	// does not wrap error into a "error" object but just panic. We gonna have to
	// recover the panic when the function call is incorrect or invalid

	argswrapper, err := info.argproto.Pack(args)
	if err != nil {
		return NewValNull(), err
	}

	defer func() {
		if err := recover(); err != nil {
			// we should recover the error
			the_error = fmt.Errorf("call %s is incorrect %+v", info.cname, err)
		}
	}()

	retwrapper := info.funcvalue.Call(argswrapper)

	// now unpack the retwrapper accordingly
	return unpackReturn(retwrapper)
}

type IntrinsicCall = func(*intrinsicinfo, *Evaluator, string, []Val) (Val, error)

func newiinfoEntry(
	cname string,
	pname string,
	arg string,
	entry IntrinsicCall) (*intrinsicinfo, error) {

	x := &intrinsicinfo{
		cname: cname,
		pname: pname,
	}

	proto, err := NewFuncProto(cname, arg)
	if err != nil {
		return nil, err
	}
	x.argproto = proto
	x.entry = func(a *Evaluator, b string, c []Val) (Val, error) {
		return entry(x, a, b, c)
	}
	return x, nil
}

func newiinfoReflect(
	cname string,
	pname string,
	arg string,
	f anyfunc) (*intrinsicinfo, error) {

	x := &intrinsicinfo{
		cname: cname,
		pname: pname,
	}

	proto, err := NewFuncProto(cname, arg)
	if err != nil {
		return nil, err
	}
	x.argproto = proto
	x.rawfunc = f
	fn := reflect.ValueOf(f)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("newiinfoReflect's entry is not a function")
	}

	x.funcvalue = &fn
	x.entry = func(_ *Evaluator, _ string, args []Val) (Val, error) {
		return funcWrapper(x, args)
	}
	return x, nil
}

func addF(cn string, pn string, p string, entry IntrinsicCall) {
	x, err := newiinfoEntry(
		cn,
		pn,
		p,
		entry,
	)

	musterr(cn, err)
	intrinsicFunc = append(intrinsicFunc, x)
}

func addMF(m string, f string, pn string, p string, entry IntrinsicCall) {
	mname := modFuncName(m, f)
	x, err := newiinfoEntry(
		mname,
		pn,
		p,
		entry,
	)

	musterr(mname, err)
	intrinsicFunc = append(intrinsicFunc, x)
}

func addrefF(
	cn string,
	pn string,
	p string,
	f anyfunc,
) {
	x, err := newiinfoReflect(
		cn,
		pn,
		p,
		f,
	)
	musterr(cn, err)
	intrinsicFunc = append(intrinsicFunc, x)
}

func addrefMF(
	m string,
	fn string,
	pn string,
	p string,
	f anyfunc,
) {
	mname := modFuncName(m, fn)
	x, err := newiinfoReflect(
		mname,
		pn,
		p,
		f,
	)

	musterr(mname, err)
	intrinsicFunc = append(intrinsicFunc, x)
}

// used by the compiler to generate ICall instructions
func indexIntrinsic(name string) int {
	for idx, v := range intrinsicFunc {
		if v.cname == name {
			return idx
		}
	}
	return -1
}

func init() {
	initModBasic()
	initModTime()
	initModCodec()
	initModStr()
	initModRandom()
}
