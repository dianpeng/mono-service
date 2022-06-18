package pl

import (
	"fmt"
	"reflect"
)

// quick go interface{} to pl.Val style
func MarshalVal(v interface{}) (Val, error) {
	vv := reflect.ValueOf(v)
	return marshalValue(vv)
}

func marshalValue(value reflect.Value) (Val, error) {
	// 1. nil
	if visnil(value) {
		return NewValNull(), nil
	}

	// 2. byte array
	if value.Kind() == reflect.Slice {
		ev := value.Elem()
		ifc := value.Interface()
		if ev.Kind() == reflect.Uint8 {
			barray, ok := ifc.([]byte)
			must(ok, "must be convertable")
			return NewValStr(string(barray)), nil
		}
	}

	switch value.Kind() {
	case reflect.Bool:
		return NewValBool(value.Bool()), nil

	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return NewValInt64(int64(value.Int())), nil

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return NewValInt64(int64(value.Uint())), nil

	case reflect.Float32, reflect.Float64:
		return NewValReal(float64(value.Float())), nil

	case reflect.String:
		return NewValStr(value.String()), nil

	case reflect.Complex64, reflect.Complex128:
		return NewValNull(), fmt.Errorf("complex value cannot be converted to Val")

	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return NewValNull(), fmt.Errorf("internal type cannot be converted to Val")

	case reflect.Slice:
		return marshalSlice(value)

	case reflect.Array:
		return marshalArray(value)

	case reflect.Interface:
		return marshalValue(value.Elem())

	case reflect.Map:
		return marshalMap(value)

	case reflect.Struct:
		return marshalStruct(value)

	default:
		return NewValNull(), fmt.Errorf("unknwon type")
	}
}

func marshalSlice(v reflect.Value) (Val, error) {
	o := NewValList()
	l := v.Len()
	for i := 0; i < l; i++ {
		vv, err := marshalValue(v.Index(i))
		if err != nil {
			return NewValNull(), err
		}
		o.AddList(vv)
	}
	return o, nil
}

func marshalArray(v reflect.Value) (Val, error) {
	return marshalSlice(v)
}

func marshalMap(v reflect.Value) (Val, error) {
	m := NewValMap()
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		kval, err := marshalValue(k)
		if err != nil {
			return NewValNull(), err
		}
		if !kval.IsString() {
			return NewValNull(), fmt.Errorf("JSON object's key must be string")
		}

		v := iter.Value()
		vval, err := marshalValue(v)
		if err != nil {
			return NewValNull(), err
		}

		m.AddMap(kval.String(), vval)
	}
	return m, nil
}

func marshalStruct(v reflect.Value) (Val, error) {
	m := NewValMap()
	n := v.NumField()
	t := v.Type()
	for i := 0; i < n; i++ {
		name := t.Field(i).Name
		v, err := marshalValue(v.Index(i))
		if err != nil {
			return NewValNull(), err
		}
		m.AddMap(name, v)
	}
	return m, nil
}
