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

type mapval struct {
	val   Val
	index int // point back to the iterator key slice which is like an asshole
}

type mapkey struct {
	key string
	use bool // used to indicate this one is just a tombstone
}

type Map struct {
	data map[string]mapval

	// in order to make map iterable, we will have to keep a list of keys that
	// has been inserted into the map
	key []mapkey
}

type MapIter struct {
	m     *Map
	index int
	size  int
}

func (m *MapIter) init() {
	l := m.m.Length()

	for m.index < l {
		k := m.m.key[m.index]
		if k.use {
			break
		}
		m.index++
	}
}

func (m *MapIter) Has() bool {
	return m.index < m.m.Length()
}

func (m *MapIter) Next() bool {
	l := m.m.Length()
	m.index++

	for m.index < l {
		k := m.m.key[m.index]
		if k.use {
			break
		}
		m.index++
	}

	return m.Has()
}

func (m *MapIter) iterandmod() (Val, Val, error) {
	return NewValNull(), NewValNull(), fmt.Errorf("map is been modified during iteration")
}

func (m *MapIter) Deref() (Val, Val, error) {
	if m.size != m.m.Length() {
		return m.iterandmod()
	}
	if !m.Has() {
		return NewValNull(), NewValNull(), fmt.Errorf("iterator out of bound")
	}
	if !m.m.key[m.index].use {
		return m.iterandmod()
	}

	k := m.m.key[m.index].key
	v, ok := m.m.data[k]
	if !ok {
		return m.iterandmod()
	}

	return NewValStr(k), v.val, nil
}

func NewMap() *Map {
	return &Map{
		data: make(map[string]mapval),
	}
}

func (m *Map) NewIter() Iter {
	x := &MapIter{
		m:     m,
		index: 0,
		size:  m.Length(),
	}
	x.init()
	return x
}

func (m *Map) Foreach(f func(string, Val) bool) int {
	cnt := 0
	for k, v := range m.data {
		if !f(k, v.val) {
			break
		}
		cnt++
	}
	return cnt
}

func (m *Map) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *Map) Set(key string, v Val) {
	old, ok := m.data[key]
	ki := 0

	if ok {
		ki = old.index
	} else {
		ki = len(m.key)
		m.key = append(m.key, mapkey{
			key: key,
			use: true,
		})
	}

	m.data[key] = mapval{
		val:   v,
		index: ki,
	}
}

func (m *Map) tryKeyGC() {
	if m.Length()*2 < cap(m.key) {
		nkey := make([]mapkey, len(m.key), len(m.key))
		for k, v := range m.data {
			m.data[k] = mapval{
				val:   v.val,
				index: len(nkey),
			}
			nkey = append(nkey, mapkey{
				key: k,
				use: true,
			})
		}

		m.key = nkey
	}
}

func (m *Map) Del(key string) bool {
	x, ok := m.data[key]
	if ok {
		m.key[x.index].use = false
		m.key[x.index].key = ""
		m.tryKeyGC()
		delete(m.data, key)
	}
	return ok
}

func (m *Map) Get(key string) (Val, bool) {
	v, ok := m.data[key]
	if ok {
		return v.val, true
	} else {
		return NewValNull(), false
	}
}

func (m *Map) Length() int {
	return len(m.data)
}

func (m *Map) ToNative() interface{} {
	x := make(map[string]interface{})
	for key, val := range m.data {
		x[key] = val.val.ToNative()
	}
	return x
}

func (m *Map) Index(idx Val) (Val, error) {
	i, err := idx.ToString()
	if err != nil {
		return NewValNull(), err
	}
	if vv, ok := m.Get(i); !ok {
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
	m.Set(i, val)
	return nil
}

func (m *Map) Dot(i string) (Val, error) {
	if vv, ok := m.Get(i); !ok {
		return NewValNull(), fmt.Errorf("%s key not found", i)
	} else {
		return vv, nil
	}
}

func (m *Map) DotSet(i string, val Val) error {
	m.Set(i, val)
	return nil
}

func (m *Map) Method(name string, args []Val) (Val, error) {
	switch name {
	case "length":
		_, err := mpMapLength.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		return NewValInt(m.Length()), nil

	case "set":
		_, err := mpMapSet.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		m.Set(args[0].String(), args[1])
		return NewValNull(), nil

	case "del":
		_, err := mpMapDel.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		m.Del(args[0].String())
		return NewValNull(), nil

	case "get":
		_, err := mpMapGet.Check(args)
		if err != nil {
			return NewValNull(), err
		}
		v, ok := m.Get(args[0].String())
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
		ok := m.Has(args[0].String())
		return NewValBool(ok), nil

	default:
		return NewValNull(), fmt.Errorf("method: list:%s is unknown", name)
	}
}

func (m *Map) Info() string {
	return fmt.Sprintf("[map: %d]", m.Length())
}
