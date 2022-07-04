package hpl

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"net/url"
	"strings"
)

type UrlSearchKV struct {
	Key   string
	Value string
}

type UrlSearch struct {
	search []UrlSearchKV
}

type urlsearchiter struct {
	search []UrlSearchKV
	cursor int
}

func (i *urlsearchiter) SetUp(_ *pl.Evaluator, _ []pl.Val) error {
	return nil
}

func (i *urlsearchiter) Has() bool {
	return i.cursor < len(i.search)
}

func (i *urlsearchiter) Deref() (pl.Val, pl.Val, error) {
	if !i.Has() {
		return pl.NewValNull(), pl.NewValNull(), fmt.Errorf("iterator out of bound")
	}
	return pl.NewValStr(i.search[i.cursor].Key), pl.NewValStr(i.search[i.cursor].Value), nil
}

func (i *urlsearchiter) Next() (bool, error) {
	i.cursor++
	return i.Has(), nil
}

func newurlsearchiter(search *UrlSearch) *urlsearchiter {
	return &urlsearchiter{
		search: search.search,
	}
}

func newUrlSearch(q url.Values) UrlSearch {
	l := []UrlSearchKV{}
	for k, vlist := range q {
		for _, v := range vlist {
			l = append(l, UrlSearchKV{
				Key:   k,
				Value: v,
			})
		}
	}

	return UrlSearch{
		search: l,
	}
}

func (h *UrlSearch) IsImmutable() bool {
	return false
}

func (h *UrlSearch) String() string {
	b := []string{}
	for _, kv := range h.search {
		b = append(b, fmt.Sprintf("%s=%s"), kv.Key, url.QueryEscape(kv.Value))
	}
	return strings.Join(b, "&")
}

func (h *UrlSearch) Get(key string) []string {
	o := []string{}
	for _, kv := range h.search {
		if kv.Key == key {
			o = append(o, kv.Value)
		}
	}
	return o
}

func (h *UrlSearch) GetFirst(key string) (string, bool) {
	for _, kv := range h.search {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

func (h *UrlSearch) Delete(key string) int {
	top := 0
	bot := len(h.search)
	predicate := func(kv *UrlSearchKV) bool {
		return kv.Key == key
	}

	for top < bot {
		// finding a unqualified top
		for ; top < bot; top++ {
			if !predicate(&h.search[top]) {
				break
			}
		}
		if top == bot {
			break
		}

		// finding a qualified bot
		for bot--; bot > top; bot-- {
			if predicate(&h.search[bot]) {
				break
			}
		}
		if top == bot {
			break
		}

		// swap bot and top if needed
		if bot != top {
			// swap the top and bot
			t := h.search[bot]
			h.search[bot] = h.search[top]
			h.search[top] = t
			top++
		}
	}

	cnt := len(h.search) - top
	h.search = h.search[:top]
	return cnt
}

func (h *UrlSearch) Set(key string, value string) {
	h.Delete(key)
	h.search = append(h.search, UrlSearchKV{
		Key:   key,
		Value: value,
	})
}

func (h *UrlSearch) Append(key string, value string) {
	h.search = append(h.search, UrlSearchKV{
		Key:   key,
		Value: value,
	})
}

func (h *UrlSearch) Index(key pl.Val) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, UrlSearch index " +
			"key must be string")
	}

	x, ok := h.GetFirst(key.String())
	if ok {
		return pl.NewValStr(x), nil
	} else {
		return pl.NewValNull(), nil
	}
}

func (h *UrlSearch) IndexSet(key pl.Val, val pl.Val) error {
	if key.IsString() && val.IsString() {
		h.Append(key.String(), val.String())
		return nil
	} else {
		return fmt.Errorf("invalid index, UrlSearch index set key and value " +
			"must be string")
	}
}

func (h *UrlSearch) Dot(name string) (pl.Val, error) {
	return h.Index(pl.NewValStr(name))
}

func (h *UrlSearch) DotSet(key string, val pl.Val) error {
	if !val.IsString() {
		return fmt.Errorf("invalid dot set value, UrlSearch expect value to be string")
	}

	h.Set(key, val.String())
	return nil
}

func (h *UrlSearch) ToString() (string, error) {
	return h.String(), nil
}

func (h *UrlSearch) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"list":   h.search,
			"string": h.String(),
		},
	)
}

var (
	methodProtoUrlSearchGet    = pl.MustNewFuncProto(".urlsearch.get", "%s")
	methodProtoUrlSearchGetAll = pl.MustNewFuncProto(".urlsearch.getAll", "%s")
	methodProtoUrlSearchDelete = pl.MustNewFuncProto(".urlsearch.delete", "%s")
	methodProtoUrlSearchSet    = pl.MustNewFuncProto(".urlsearch.set", "%s%s")
	methodProtoUrlSearchAppend = pl.MustNewFuncProto(".urlsearch.append", "%s%s")
	methodProtoUrlSearchString = pl.MustNewFuncProto(".urlsearch.string", "%0")
)

func (h *UrlSearch) Method(name string, args []pl.Val) (pl.Val, error) {
	switch name {
	case "get":
		if _, err := methodProtoUrlSearchGet.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		v, ok := h.GetFirst(args[0].String())
		if !ok {
			return pl.NewValNull(), nil
		} else {
			return pl.NewValStr(v), nil
		}
	case "getAll":
		if _, err := methodProtoUrlSearchGetAll.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		x := h.Get(args[0].String())
		return pl.NewValStrList(x), nil

	case "delete":
		if _, err := methodProtoUrlSearchDelete.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValInt(h.Delete(args[0].String())), nil
	case "set":
		if _, err := methodProtoUrlSearchSet.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		h.Set(args[0].String(), args[1].String())
		return pl.NewValNull(), nil
	case "append":
		if _, err := methodProtoUrlSearchAppend.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		h.Append(args[0].String(), args[1].String())
		return pl.NewValNull(), nil

	case "string":
		if _, err := methodProtoUrlSearchString.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValStr(h.String()), nil
	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("method: .urlsearch's %s method is unknown", name)
}

func (h *UrlSearch) Info() string {
	return fmt.Sprintf(".urlsearch[]")
}

func (h *UrlSearch) Id() string {
	return UrlSearchTypeId
}

func (h *UrlSearch) ToNative() interface{} {
	return h.search
}

func (h *UrlSearch) NewIterator() (pl.Iter, error) {
	return newurlsearchiter(h), nil
}

func NewUrlSearchVal(s *UrlSearch) pl.Val {
	return pl.NewValUsr(s)
}

func NewUrlSearchValFromValues(v url.Values) pl.Val {
	search := newUrlSearch(v)
	return pl.NewValUsr(&search)
}

func NewUrlSearchValFromString(input string) (pl.Val, error) {
	v, err := url.ParseQuery(input)
	if err != nil {
		return pl.NewValNull(), err
	}
	return NewUrlSearchValFromValues(v), nil
}
