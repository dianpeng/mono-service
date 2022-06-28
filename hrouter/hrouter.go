package hrouter

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

type parlist map[string]string

type Params struct {
	p parlist
}

type KV struct {
	Key   string
	Value string
}

func NewParams(r *http.Request) Params {
	return Params{
		p: mux.Vars(r),
	}
}

func NewEmptyParams() Params {
	return Params{
		p: make(parlist),
	}
}

func (p *Params) ToList() []KV {
	x := []KV{}
	for k, v := range p.p {
		x = append(x, KV{
			Key:   k,
			Value: v,
		})
	}
	return x
}

func (p *Params) ByName(id string) string {
	return p.p[id]
}

func (p *Params) Set(k, v string) {
	p.p[k] = v
}

func (p *Params) Length() int {
	return len(p.p)
}

func (p *Params) String() string {
	b := new(bytes.Buffer)
	for k, v := range p.p {
		b.WriteString(fmt.Sprintf("%s=>%s ", k, v))
	}
	return b.String()
}
