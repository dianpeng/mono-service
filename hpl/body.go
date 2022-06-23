package hpl

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"io"
)

type Body struct {
	streamVal pl.Val
	stream    *ReadableStream
}

func ValIsHttpBody(v pl.Val) bool {
	return v.Id() == HttpBodyTypeId
}

func (h *Body) Reader() io.Reader {
	return h.stream.ToStream()
}

func (h *Body) Stream() *ReadableStream {
	return h.stream
}

func (h *Body) StreamVal() pl.Val {
	return h.streamVal
}

func (h *Body) SetStream(stream io.ReadCloser) {
	h.stream.SetStream(stream)
}

func (h *Body) SetString(data string) {
	h.stream.SetString(data)
}

func (h *Body) SetBuffer(data []byte) {
	h.stream.SetBuffer(data)
}

func (h *Body) Index(name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("http.body invalid field index")
	}

	switch name.String() {
	case "stream":
		return h.streamVal, nil
	default:
		break
	}
	return pl.NewValNull(), fmt.Errorf("http.body unknown field name %s", name.String())
}

func (h *Body) Dot(name string) (pl.Val, error) {
	return h.Index(pl.NewValStr(name))
}

func (h *Body) IndexSet(name pl.Val, val pl.Val) error {
	if name.Type != pl.ValStr {
		return fmt.Errorf("http.body invalid field index")
	}

	switch name.String() {
	case "stream":
		if ValIsReadableStream(val) {
			h.streamVal = val
			s, ok := val.Usr().(*ReadableStream)
			must(ok, "must be readablestream")
			h.stream = s
			return nil
		}

		if val.Type == pl.ValStr {
			h.streamVal = NewReadableStreamValFromString(val.String())
			s, ok := h.streamVal.Usr().(*ReadableStream)
			must(ok, "must be readablestream")
			h.stream = s
			return nil
		}

		return fmt.Errorf("http.body.stream assignment value type invalid")
	default:
		break
	}
	return fmt.Errorf("http.body component assign unknown field %s", name)
}

func (h *Body) DotSet(name string, val pl.Val) error {
	return h.IndexSet(pl.NewValStr(name), val)
}

func (h *Body) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"type": HttpBodyTypeId,
		},
	)
}

func (h *Body) ToString() (string, error) {
	return HttpBodyTypeId, nil
}

var (
	methodProtoString = pl.MustNewFuncProto("http.body.string", "%0")
)

func (h *Body) Method(name string, arg []pl.Val) (pl.Val, error) {
	switch name {
	case "string":
		_, err := methodProtoString.Check(arg)
		if err != nil {
			return pl.NewValNull(), err
		}
		str, err := h.stream.ConsumeAsString()
		if err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValStr(str), nil

	default:
		break
	}
	return pl.NewValNull(), fmt.Errorf("http.body unknown method %s", name)
}

func (h *Body) Info() string {
	return HttpBodyTypeId
}

func (h *Body) ToNative() interface{} {
	return h.stream
}

func (h *Body) Id() string {
	return HttpBodyTypeId
}

func (h *Body) IsImmutable() bool {
	return false
}

func (h *Body) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("http.body does not support iterator")
}

func newBodyValFromReadableStream(rsVal pl.Val, rs *ReadableStream) pl.Val {
	x := &Body{
		streamVal: rsVal,
		stream:    rs,
	}

	return pl.NewValUsr(x)
}

func NewBodyValFromStream(rawStream io.ReadCloser) pl.Val {
	streamVal := NewReadableStreamValFromStream(rawStream)
	stream, ok := streamVal.Usr().(*ReadableStream)
	must(ok, "must be readablestream")

	return newBodyValFromReadableStream(streamVal, stream)
}

func NewBodyValFromString(data string) pl.Val {
	streamVal := NewReadableStreamValFromString(data)
	stream, ok := streamVal.Usr().(*ReadableStream)
	must(ok, "must be readablestream")

	return newBodyValFromReadableStream(streamVal, stream)
}

func NewBodyValFromBuffer(data []byte) pl.Val {
	streamVal := NewReadableStreamValFromBuffer(data)
	stream, ok := streamVal.Usr().(*ReadableStream)
	must(ok, "must be readablestream")

	return newBodyValFromReadableStream(streamVal, stream)
}

func NewBodyValFromVal(v pl.Val) (pl.Val, error) {
	switch v.Type {
	case pl.ValStr:
		return NewBodyValFromString(v.String()), nil
	default:
		if ValIsReadableStream(v) {
			x, _ := v.Usr().(*ReadableStream)
			return NewBodyValFromStream(x.Stream), nil
		}
		if ValIsHttpBody(v) {
			x, _ := v.Usr().(*Body)
			return NewBodyValFromStream(x.Stream().Stream), nil
		}
		break
	}

	return pl.NewValNull(), fmt.Errorf("cannot create http.body from type: %s", v.Id())
}
