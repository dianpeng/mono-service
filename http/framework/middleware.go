package framework

import (
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"

	"fmt"
	"io"
	"net/http"
)

// We cannot use net/http.ResponseWriter since it is not composable. We need a
// way to represent http response body in a memory efficient way and at the
// same time allow the response middleware to modify its body along the road,
// ie it must be a value type instead of a Write([]byte) interface. Therefore,
// we choose to allow user to send us a io.ReadCloser as response body which
// is a value and can be composed
type HttpResponseWriter interface {
	// Get the current status code
	Status() int
	WriteStatus(int)
	Header() http.Header

	// If body is not set for this response, then it returns nil
	GetBody() io.ReadCloser
	WriteBody(io.ReadCloser)

	Flush() bool
	FlushHeader() bool

	IsFlushed() bool
	IsHeaderFlushed() bool

	// Flush response and close the transaction, mainly used to report error
	// in various middleware stage
	ReplyNow(
		int,
		string,
	)

	// Categorized response APIs, which should be preferred
	ReplyError(
		string,
		int,
		error,
	)
}

type Middleware interface {
	Accept(
		*http.Request,
		hrouter.Params,
		HttpResponseWriter,
		ServiceContext,
	) bool

	Name() string
}

type MiddlewareFactory interface {
	Create([]pl.Val) (Middleware, error)
	Name() string
	Comment() string
}

// main interface for middleware is the middleware list since it exposes the
// middleware as composable object
type middlewareFactoryEntry struct {
	m      MiddlewareFactory
	config []pl.Val
	name   string
}

type MiddlewareFactoryList struct {
	l       []middlewareFactoryEntry
	name    string
	comment string
}

// MiddlewareFactoryList will create a middlewareCompose which has Middleware's
// interface implementation
type middlewareCompose struct {
	l    []Middleware
	name string
}

func (m *middlewareCompose) Accept(
	h *http.Request,
	r hrouter.Params,
	resp HttpResponseWriter,
	ctx ServiceContext,
) bool {

	for _, x := range m.l {
		if !x.Accept(
			h,
			r,
			resp,
			ctx,
		) {
			return false
		}

		// the response has been generated, so break out the middleware chain
		if resp.IsFlushed() {
			return false
		}
	}
	return true
}

func (m *middlewareCompose) Name() string {
	return m.name
}

func NewMiddlewareFactoryList(n string, c string) *MiddlewareFactoryList {
	return &MiddlewareFactoryList{
		name:    n,
		comment: c,
	}
}

func (f *MiddlewareFactoryList) AddRequest(
	name string,
	config []pl.Val,
) error {
	x := GetRequestFactory(name)
	if x == nil {
		return fmt.Errorf("middleware(request): %s is not found", name)
	}
	f.l = append(f.l, middlewareFactoryEntry{
		m:      x,
		config: config,
	})
	return nil
}

func (f *MiddlewareFactoryList) AddResponse(
	name string,
	config []pl.Val,
) error {
	x := GetResponseFactory(name)
	if x == nil {
		return fmt.Errorf("middleware(response): %s is not found", name)
	}
	f.l = append(f.l, middlewareFactoryEntry{
		m:      x,
		config: config,
	})
	return nil
}

func (f *MiddlewareFactoryList) Name() string {
	return f.name
}

func (f *MiddlewareFactoryList) Comment() string {
	return f.comment
}

func (f *MiddlewareFactoryList) Create(_ []pl.Val) (Middleware, error) {
	l := []Middleware{}

	for _, x := range f.l {
		m, err := x.m.Create(x.config)
		if err != nil {
			return nil, err
		}
		l = append(l, m)
	}
	return &middlewareCompose{
		l:    l,
		name: f.name,
	}, nil
}

type middlewarefactorymap struct {
	m map[string]MiddlewareFactory
}

func newmiddlewarefactorymap() middlewarefactorymap {
	return middlewarefactorymap{
		m: make(map[string]MiddlewareFactory),
	}
}

func (m *middlewarefactorymap) add(name string, f MiddlewareFactory) {
	m.m[name] = f
}

func (m *middlewarefactorymap) get(name string) MiddlewareFactory {
	v, ok := m.m[name]
	if ok {
		return v
	} else {
		return nil
	}
}
