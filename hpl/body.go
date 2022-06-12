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

func (h *Body) Index(_ interface{}, name pl.Val) (pl.Val, error) {
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

func (h *Body) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *Body) IndexSet(_ interface{}, name pl.Val, val pl.Val) error {
	if name.Type != pl.ValStr {
		return fmt.Errorf("http.body invalid field index")
	}

	switch name.String() {
	case "stream":
		if ValIsReadableStream(val) {
			h.streamVal = val
			s, ok := val.Usr().Context.(*ReadableStream)
			must(ok, "must be readablestream")
			h.stream = s
			return nil
		}

		if val.Type == pl.ValStr {
			h.streamVal = NewReadableStreamValFromString(val.String())
			s, ok := h.streamVal.Usr().Context.(*ReadableStream)
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

func (h *Body) DotSet(x interface{}, name string, val pl.Val) error {
	return h.IndexSet(x, pl.NewValStr(name), val)
}

func (h *Body) ToJSON(_ interface{}) (string, error) {
	return fmt.Sprintf("{\"type\": \"%s\"}", HttpBodyTypeId), nil
}

func (h *Body) ToString(_ interface{}) (string, error) {
	return HttpBodyTypeId, nil
}

func (h *Body) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("http.body unknown method %s", name)
}

func (h *Body) Info(_ interface{}) string {
	return HttpBodyTypeId
}

func (h *Body) ToNative(_ interface{}) interface{} {
	return h.stream
}

func newBodyValFromReadableStream(rsVal pl.Val, rs *ReadableStream) pl.Val {
	x := &Body{
		streamVal: rsVal,
		stream:    rs,
	}

  return pl.NewValUsr(
		x,
		x.Index,
		x.IndexSet,
		x.Dot,
		x.DotSet,
		x.method,
		x.ToString,
		x.ToJSON,
		func(_ interface{}) string {
			return HttpBodyTypeId
		},
		x.Info,
		x.ToNative,
	)
}

func NewBodyValFromStream(rawStream io.ReadCloser) pl.Val {
	streamVal := NewReadableStreamValFromStream(rawStream)
	stream, ok := streamVal.Usr().Context.(*ReadableStream)
	must(ok, "must be readablestream")

	return newBodyValFromReadableStream(streamVal, stream)
}

func NewBodyValFromString(data string) pl.Val {
	streamVal := NewReadableStreamValFromString(data)
	stream, ok := streamVal.Usr().Context.(*ReadableStream)
	must(ok, "must be readablestream")

	return newBodyValFromReadableStream(streamVal, stream)
}

func NewBodyValFromBuffer(data []byte) pl.Val {
	streamVal := NewReadableStreamValFromBuffer(data)
	stream, ok := streamVal.Usr().Context.(*ReadableStream)
	must(ok, "must be readablestream")

	return newBodyValFromReadableStream(streamVal, stream)
}
