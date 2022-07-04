package hpl

import (
	"bytes"
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
	"strings"
)

type Header struct {
	header http.Header
}

func ValIsHttpHeader(v pl.Val) bool {
	return v.Id() == HttpHeaderTypeId
}

func (h *Header) HttpHeader() http.Header {
	return h.header
}

func (h *Header) Has(key string) bool {
	_, ok := h.header[key]
	return ok
}

func (h *Header) GetFirst(key string) string {
	return h.header.Get(key)
}

func (h *Header) Delete(key string) {
	h.header.Del(key)
}

func (h *Header) DeleteByFilter(key pl.Val) int {
	return httpHeaderFilter(
		h.header,
		key,
		func(key string, _ []string, hdr http.Header) bool {
			hdr.Del(key)
			return true
		},
	)
}

func (h *Header) GetByFilter(key pl.Val, d string) string {
	b := []string{}
	bp := &b

	httpHeaderFilter(
		h.header,
		key,
		func(_ string, val []string, _ http.Header) bool {
			*bp = append(*bp, val...)
			return true
		},
	)

	return strings.Join(b, d)
}

func (h *Header) Get(key string, d string) string {
	return strings.Join(h.header.Values(key), d)
}

func (h *Header) Set(key string, d string) {
	h.header.Set(key, d)
}

func (h *Header) Add(key string, d string) {
	h.header.Add(key, d)
}

func (h *Header) Length() int {
	cnt := 0
	for _, x := range h.header {
		for _, _ = range x {
			cnt++
		}
	}
	return cnt
}

func (h *Header) Index(key pl.Val) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index for http.header")
	}
	return pl.NewValStr(h.header.Get(key.String())), nil
}

func (h *Header) Dot(key string) (pl.Val, error) {
	return h.Index(pl.NewValStr(key))
}

func (h *Header) IndexSet(key pl.Val, val pl.Val) error {
	if key.Type == pl.ValStr {
		return h.DotSet(key.String(), val)
	}
	return fmt.Errorf("invalid index type for http.header")
}

func (h *Header) DotSet(key string, val pl.Val) error {
	str, err := val.ToString()
	if err != nil {
		return fmt.Errorf("http.header value cannot be converted to string: %s", err.Error())
	}
	h.header.Set(key, str)
	return nil
}

func (h *Header) ToString() (string, error) {
	var b bytes.Buffer
	for key, val := range h.header {
		b.WriteString(fmt.Sprintf("%s => %s\n", key, val))
	}
	return b.String(), nil
}

func (h *Header) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(h.header)
}

var (
	methodProtoHeaderHas            = pl.MustNewFuncProto("http.header.has", "%s")
	methodProtoHeaderGetFirst       = pl.MustNewFuncProto("http.header.getFirst", "%s")
	methodProtoHeaderGet            = pl.MustNewFuncProto("http.header.get", "%s%s")
	methodProtoHeaderDelete         = pl.MustNewFuncProto("http.header.delete", "%s")
	methodProtoHeaderDeleteByFilter = pl.MustNewFuncProto("http.header.deleteByFilter", "(%s|%r)")
	methodProtoHeaderGetByFilter    = pl.MustNewFuncProto("http.header.getByFilter", "%s(%s|%r)")
	methodProtoHeaderSet            = pl.MustNewFuncProto("http.header.set", "%s%s")
	methodProtoHeaderAdd            = pl.MustNewFuncProto("http.header.add", "%s%s")
	methodProtoHeaderLength         = pl.MustNewFuncProto("http.header.length", "%0")
)

func (h *Header) Method(name string, arg []pl.Val) (pl.Val, error) {
	switch name {
	case "has":
		if _, err := methodProtoHeaderHas.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(h.Has(arg[0].String())), nil

	case "getFirst":
		if _, err := methodProtoHeaderGetFirst.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValStr(h.GetFirst(arg[0].String())), nil

	case "get":
		if _, err := methodProtoHeaderGet.Check(arg); err != nil {
			return pl.NewValNull(), err
		}

		return pl.NewValStr(h.Get(arg[0].String(), arg[1].String())), nil

	case "getByFilter":
		if _, err := methodProtoHeaderGetByFilter.Check(arg); err != nil {
			return pl.NewValNull(), err
		}

		return pl.NewValStr(h.GetByFilter(arg[0], arg[1].String())), nil

	case "delete":
		if _, err := methodProtoHeaderDelete.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		h.Delete(arg[0].String())
		return pl.NewValNull(), nil

	case "deleteByFilter":
		if _, err := methodProtoHeaderDeleteByFilter.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValInt(h.DeleteByFilter(arg[0])), nil

	case "set":
		if _, err := methodProtoHeaderSet.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		h.Set(arg[0].String(), arg[1].String())
		return pl.NewValNull(), nil

	case "add":
		if _, err := methodProtoHeaderAdd.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		h.Add(arg[0].String(), arg[1].String())
		return pl.NewValNull(), nil

	case "length":
		if _, err := methodProtoHeaderLength.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValInt(h.Length()), nil
	default:
		break
	}
	return pl.NewValNull(), fmt.Errorf("method: http.header:%s is unknown", name)
}

func (h *Header) Info() string {
	return HttpHeaderTypeId
}

func (h *Header) ToNative() interface{} {
	return h.header
}

func (h *Header) Id() string {
	return HttpHeaderTypeId
}

func (h *Header) IsImmutable() bool {
	return false
}

// iterator of http header
type headerentry struct {
	key   string
	value string
}
type headeriter struct {
	kv     []headerentry
	cursor int
}

func newheaderiter(
	h http.Header,
) *headeriter {
	kv := []headerentry{}
	for k, vlist := range h {
		for _, v := range vlist {
			kv = append(kv, headerentry{
				key:   k,
				value: v,
			})
		}
	}
	return &headeriter{kv: kv}
}

func (h *headeriter) Has() bool {
	return h.cursor < len(h.kv)
}

func (h *headeriter) Next() (bool, error) {
	h.cursor++
	return h.Has(), nil
}

func (h *headeriter) SetUp(_ *pl.Evaluator, _ []pl.Val) error {
	return nil
}

func (h *headeriter) Deref() (pl.Val, pl.Val, error) {
	if !h.Has() {
		return pl.NewValNull(), pl.NewValNull(), fmt.Errorf("iterator out of bound")
	}
	return pl.NewValStr(h.kv[h.cursor].key), pl.NewValStr(h.kv[h.cursor].value), nil
}

func (h *Header) NewIterator() (pl.Iter, error) {
	return newheaderiter(h.header), nil
}

func NewHeaderVal(header http.Header) pl.Val {
	x := &Header{
		header: header,
	}

	return pl.NewValUsr(x)
}

func NewHeaderValFromVal(v pl.Val) (pl.Val, error) {
	if v.Id() == HttpHeaderTypeId {
		hdr, _ := v.Usr().(*Header)
		return NewHeaderVal(hdr.header), nil
	}

	hdr := make(http.Header)

	if !foreachHeaderKV(
		v,
		func(k, v string) {
			hdr.Add(k, v)
		},
	) {
		return pl.NewValNull(), fmt.Errorf("unknown value type to initialize http.header", v.Id())
	}

	return NewHeaderVal(hdr), nil
}
