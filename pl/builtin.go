package pl

/*
// when using reflection, the following conversion rules are maintained:
//   1. %i -> int
//   2. %i1 -> int8
//   3. %i2 -> int16
//   4. %i4 -> int32
//   5. %i8 -> int64

import (
	"reflect"

	// builtins library functions
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"strings"
	"time"
)

const (
	// net module
	bFnNetB64Encode = iota
	bFnNetB64Decode

	bFnNetURLEncode
	bFnNetURLDecode

	bFnNetPathEscape
	bFnNetPathUnescape

	// string module
	bFnToUpper
	bFnToLower
	bFnSplit
	bFnTrim
	bFnLTrim
	bFnRTrim

	// <<global>>
	bFnToString
	bFnToNumber
	bFnToBoolean
	bFnType

	bFnNow
	bFnHttpDate
	bFnHttpDateNano
	bFnUUID
	bFnEcho
)

type anyfunc interface{}

type builtininfo struct {
	index int
	cname string
	pname string
	argproto *FuncProto
  retproto *FuncReturnProto

  // must be prototype of EvalCall, this is the entry of the vm interpreter
  entry EvalCall

  // used when need reflection to wrap the actual function
  rawfunc anyfunc
  funcvalue *reflect.Value
}

var bFunc []*builtininfo

// for simplicity, we uses reflect to expose most of the functions back to the
// script environment. In case, when certain function becomes performance
// bottleneck, then we rebind them with hande crafted go code to save reflect
// performance overhead. Using reflect makes us easy to work the binding go
// function into the script.
func funcWrapper(
  info *builtininfo, // info
	args []Val,        // function arguments from script side
) (Val, error) {
  must(f.Kind() == reflect.Func, "must be a function")

  // okay, we will have to recover from the panic of the reflect.Call since it
  // does not wrap error into a "error" object but just panic. We gonna have to
  // recover the panic when the function call is incorrect or invalid

  var argswrapper []reflect.Value
  for _, v := range args {
    argswrapper = append(argswrapper, v.ToReflect())
  }
}

func addF(i int, cn string, pn string, p string) *builtininfo {
	proto, err := NewFuncProto(cn, p)
	must(err == nil, fmt.Sprintf("function %s proto %s is invalid: %s", cn, p, err.Error()))

	bFunc = append(bFunc, builtininfo{
		index: i,
		cname: cn,
		pname: pn,
		proto: proto,
	})
}

func addMF(i int, m string, f string, pn string, p string) *builtininfo {
	proto, err := NewModFuncProto(m, f, p)
	must(err == nil,
		fmt.Sprintf("module %s's function %s's proto %s is invalid: %s", m, f, p, err.Error()))

	bFunc = append(bFunc, builtininfo{
		index: i,
		cname: modFuncName(m, f),
		pname: pn,
		proto: proto,
	})
}

// initialize all the builtin functions
func init() {
}

*/
