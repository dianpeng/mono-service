package hpl

import (
	"fmt"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
)

type RouterParams struct {
	params hrouter.Params
}

func (h *RouterParams) Index(name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, http.router's field must be string")
	}

	return pl.NewValStr(h.params.ByName(name.String())), nil
}

func (h *RouterParams) Dot(name string) (pl.Val, error) {
	return h.Index(pl.NewValStr(name))
}

func (h *RouterParams) IndexSet(name pl.Val, value pl.Val) error {
	if name.Type != pl.ValStr {
		return fmt.Errorf("invalid index, http.router.params'key field must be string")
	}
	return h.DotSet(name.String(), value)
}

func (h *RouterParams) DotSet(name string, value pl.Val) error {
	str, err := value.ToString()
	if err != nil {
		return fmt.Errorf("invalid index, http.router.param's key set value must be string")
	}
	h.params.Set(name, str)
	return nil
}

func (h *RouterParams) ToString() (string, error) {
	return h.params.String(), nil
}

func (h *RouterParams) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(h.params)
}

func (h *RouterParams) Method(name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.router.params %s is unknown", name)
}

func (h *RouterParams) Info() string {
	return HttpRouterParamsTypeId
}

func (h *RouterParams) ToNative() interface{} {
	return h.params
}

func (h *RouterParams) Id() string {
	return HttpRouterParamsTypeId
}

func (h *RouterParams) IsImmutable() bool {
	return false
}

type routerpariter struct {
	kv     []hrouter.KV
	cursor int
}

func (h *routerpariter) Has() bool {
	return h.cursor < len(h.kv)
}

func (h *routerpariter) Next() bool {
	h.cursor++
	return h.Has()
}

func newrouterpariter(
	p hrouter.Params,
) *routerpariter {
	return &routerpariter{
		kv: p.ToList(),
	}
}

func (h *routerpariter) Deref() (pl.Val, pl.Val, error) {
	if !h.Has() {
		return pl.NewValNull(), pl.NewValNull(), fmt.Errorf("iterator out of bound")
	}
	return pl.NewValStr(h.kv[h.cursor].Key), pl.NewValStr(h.kv[h.cursor].Value), nil
}

func (h *RouterParams) NewIterator() (pl.Iter, error) {
	return newrouterpariter(h.params), nil
}

func NewRouterParamsVal(r hrouter.Params) pl.Val {
	x := &RouterParams{
		params: r,
	}
	return pl.NewValUsr(x)
}
