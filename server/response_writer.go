package server

import (
	"fmt"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/pl"
	"io"
	"net/http"
)

const (
	HttpResponseWriterTypeId = "http.response_writer"
)

type responseWriterWrapper struct {
	handler *serviceHandler
	w       http.ResponseWriter
	status  int
	header  http.Header
	body    io.ReadCloser

	headerDone bool
	bodyDone   bool
	bodyError  error

	// pl.Val field for exposition
	headerVal pl.Val
	bodyVal   pl.Val
}

func ValIsHttpResponseWriter(
	v pl.Val,
) bool {
	return v.Id() == HttpResponseWriterTypeId
}

func newResponseWriterWrapper(
	handler *serviceHandler,
	writer http.ResponseWriter,
) (*responseWriterWrapper, pl.Val) {

	x := &responseWriterWrapper{
		handler: handler,
		w:       writer,
		status:  200,
		header:  make(http.Header),
		body:    hpl.NewEofReadCloser(),
	}

	hdrVal := hpl.NewHeaderVal(
		x.header,
	)
	bodyVal := hpl.NewBodyValFromStream(
		x.body,
	)
	x.headerVal = hdrVal
	x.bodyVal = bodyVal
	val := pl.NewValUsr(x)

	return x, val
}

// -----------------------------------------------------------------------------
// Interface for pl.Usr
func (r *responseWriterWrapper) Index(
	key pl.Val,
) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, http.response_writer's " +
			"component require a string as index")
	}
	switch key.String() {
	case "status":
		return pl.NewValInt(r.status), nil
	case "body":
		return r.bodyVal, nil
	case "header":
		return r.headerVal, nil

	case "headerDone":
		return pl.NewValBool(r.headerDone), nil
	case "bodyDone":
		return pl.NewValBool(r.bodyDone), nil
	case "bodyFlushError":
		if r.bodyError == nil {
			return pl.NewValStr(""), nil
		} else {
			return pl.NewValStr(r.bodyError.Error()), nil
		}

	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("unknown field name: %s", key.String())
}

func (r *responseWriterWrapper) Dot(
	key string,
) (pl.Val, error) {
	return r.Index(pl.NewValStr(key))
}

func (r *responseWriterWrapper) IndexSet(
	key pl.Val,
	val pl.Val,
) error {
	if !key.IsString() {
		return fmt.Errorf("invalid index, http.response_writer's field name " +
			"must be type string")
	}

	switch key.String() {
	case "status":
		if val.Type == pl.ValInt {
			r.status = int(val.Int())
		} else {
			return fmt.Errorf("invalid status type: %s", val.Id())
		}
		break

	case "header":
		hdrVal, err := hpl.NewHeaderValFromVal(val)
		if err != nil {
			return err
		}
		hdr, _ := hdrVal.Usr().(*hpl.Header)
		r.headerVal = val
		r.header = hdr.HttpHeader()
		break

	case "body":
		bodyVal, err := hpl.NewBodyValFromVal(val)
		if err != nil {
			return err
		}
		body, _ := bodyVal.Usr().(*hpl.Body)
		r.bodyVal = bodyVal
		r.body = body.Stream().Stream
		break

	default:
		return fmt.Errorf("invalid field of http.response_writer: %s", key.String())
	}

	return nil
}

func (r *responseWriterWrapper) DotSet(
	key string,
	val pl.Val,
) error {
	return r.IndexSet(pl.NewValStr(key), val)
}

var (
	rwMethodFlushHeader = pl.MustNewFuncProto("http.response_writer.flushHeader", "%0")
	rwMethodFlush       = pl.MustNewFuncProto("http.response_writer.flush", "%0")
)

func (r *responseWriterWrapper) Method(
	name string,
	arg []pl.Val,
) (pl.Val, error) {

	switch name {
	case "flushHeader":
		if _, err := rwMethodFlushHeader.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(r.FlushHeader()), nil

	case "flush":
		if _, err := rwMethodFlush.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(r.Flush()), nil
	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("http.response_writer method: %s is unknown", name)
}

func (r *responseWriterWrapper) ToString() (string, error) {
	return r.Info(), nil
}

func (r *responseWriterWrapper) Id() string {
	return HttpResponseWriterTypeId
}

func (r *responseWriterWrapper) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"type":       HttpResponseWriterTypeId,
			"header":     r.header,
			"status":     r.status,
			"body":       r.body,
			"headerDone": r.headerDone,
			"bodyDone":   r.bodyDone,
		},
	)
}

func (r *responseWriterWrapper) Info() string {
	return HttpResponseWriterTypeId
}

func (r *responseWriterWrapper) ToNative() interface{} {
	return r.w
}

func (r *responseWriterWrapper) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("http.response_writer does not support iterator")
}

func (r *responseWriterWrapper) IsImmutable() bool {
	return false
}

// -----------------------------------------------------------------------------
// Interface for framework.HttpResponseWriter
func (r *responseWriterWrapper) Header() http.Header {
	return r.header
}

func (r *responseWriterWrapper) Status() int {
	return r.status
}

func (r *responseWriterWrapper) WriteStatus(status int) {
	if r.headerDone {
		return
	}
	r.status = status
}

func (r *responseWriterWrapper) FlushHeader() bool {
	if r.headerDone {
		return false
	}
	r.headerDone = true

	// fire a interceptor event
	_ = r.handler.hpl.RunWithContext(
		EventNameResponseHookStatus,
		pl.NewValNull(),
	)

	// write the header and status field out
	for k, v := range r.header {
		r.w.Header()[k] = v
	}
	r.w.WriteHeader(r.status)
	return true
}

func (r *responseWriterWrapper) WriteBody(x io.ReadCloser) {
	if r.bodyDone {
		return
	}
	r.body = x
}

func (r *responseWriterWrapper) GetBody() io.ReadCloser {
	return r.body
}

func (r *responseWriterWrapper) Flush() bool {
	if r.bodyDone {
		return false
	}

	_ = r.handler.hpl.RunWithContext(
		EventNameResponseHookBody,
		pl.NewValNull(),
	)

	r.FlushHeader()

	if r.body != nil {
		_, err := io.Copy(
			r.w,
			r.body,
		)
		r.bodyError = err
	}

	r.bodyDone = true
	r.body = nil
	return true
}

func (r *responseWriterWrapper) IsFlushed() bool {
	return r.bodyDone
}

func (r *responseWriterWrapper) IsHeaderFlushed() bool {
	return r.headerDone
}

// Finalize will finally try to flush the data out if needed and also it will
// run the response hook if needed
func (r *responseWriterWrapper) Finalize() {
	r.Flush()
}

func (r *responseWriterWrapper) ReplyNow(
	status int,
	body string,
) {
	r.WriteStatus(status)
	r.WriteBody(
		hpl.NewReadCloserFromString(body),
	)
	r.Flush()
}

func (r *responseWriterWrapper) ReplyErrorHPL(err error) {
	r.ReplyNow(
		500,
		err.Error(),
	)
}

func (r *responseWriterWrapper) ReplyErrorApp(err error) {
	r.ReplyNow(
		500,
		err.Error(),
	)
}

func (r *responseWriterWrapper) ReplyErrorAppPrepare(err error) {
	r.ReplyNow(
		500,
		err.Error(),
	)
}

func (r *responseWriterWrapper) ReplyErrorCreateService(err error) {
	r.ReplyNow(
		500,
		err.Error(),
	)
}
