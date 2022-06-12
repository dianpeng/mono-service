//go:build go1.18
// +build go1.18

package pl

import (
	"reflect"
)

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
