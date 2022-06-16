package pl

import (
	"fmt"
)

var (
	// map#method
	mpMapLength = MustNewFuncProto("map.length", "%0")
	mpMapSet    = MustNewFuncProto("map.set", "%s%a")
	mpMapDel    = MustNewFuncProto("map.del", "%s")
	mpMapGet    = MustNewFuncProto("map.get", "%s%a")
	mpMapHas    = MustNewFuncProto("map.has", "%s")
)

type Map struct {
	Data map[string]Val
}

func NewMap() *Map {
	return &Map{
		Data: make(map[string]Val),
	}
}

func (m *Map) ToNative() interface{} {
	x := make(map[string]interface{})
	for key, val := range m.Data {
		x[key] = val.ToNative()
	}
	return x
}

func (m *Map) Index(idx Val) (Val, error) {
	i, err := idx.ToString()
	if err != nil {
		return NewValNull(), err
	}
	if vv, ok := m.Data[i]; !ok {
		return NewValNull(), fmt.Errorf("%s key not found", i)
	} else {
		return vv, nil
	}
}

func (m *Map) IndexSet(idx, val Val) error {
	i, err := idx.ToString()
	if err != nil {
		return err
	}
	m.Data[i] = val
	return nil
}

func (m *Map) Dot(i string) (Val, error) {
	if vv, ok := m.Data[i]; !ok {
		return NewValNull(), fmt.Errorf("%s key not found", i)
	} else {
		return vv, nil
	}
}

func (m *Map) DotSet(i string, val Val) error {
	m.Data[i] = val
	return nil
}

func (m *Map) Method(name string, args []Val) (Val, error) {
	switch name {
	case "length":
		_, err := mpMapLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(len(m.Data)), nil

	case "set":
		_, err := mpMapSet.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		m.Data[args[0].String()] = args[1]
		return NewValNull(), nil

	case "del":
		_, err := mpMapDel.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		delete(m.Data, args[0].String())
		return NewValNull(), nil

	case "get":
		_, err := mpMapGet.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		v, ok := m.Data[args[0].String()]
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
		_, ok := m.Data[args[0].String()]
		return NewValBool(ok), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}

func (m *Map) Info() string {
	return fmt.Sprintf("[map: %d]", len(m.Data))
}
