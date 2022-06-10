package hpl

import (
	"encoding/json"
	"fmt"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
)

type RouterParams struct {
	params hrouter.Params
}

func (h *RouterParams) Index(_ interface{}, name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, http.router's field must be string")
	}

	return pl.NewValStr(h.params.ByName(name.String())), nil
}

func (h *RouterParams) Dot(x interface{}, name string) (pl.Val, error) {
	return h.Index(x, pl.NewValStr(name))
}

func (h *RouterParams) IndexSet(x interface{}, name pl.Val, value pl.Val) error {
	if name.Type != pl.ValStr {
		return fmt.Errorf("invalid index, http.router.params'key field must be string")
	}
	return h.DotSet(x, name.String(), value)
}

func (h *RouterParams) DotSet(_ interface{}, name string, value pl.Val) error {
	str, err := value.ToString()
	if err != nil {
		return fmt.Errorf("invalid index, http.router.param's key set value must be string")
	}
	h.params.Set(name, str)
	return nil
}

func (h *RouterParams) ToString(_ interface{}) (string, error) {
	return h.params.String(), nil
}

func (h *RouterParams) ToJSON(_ interface{}) (string, error) {
	blob, _ := json.Marshal(h.params)
	return string(blob), nil
}

func (h *RouterParams) method(_ interface{}, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("method: http.router.params %s is unknown", name)
}

func (h *RouterParams) Info(_ interface{}) string {
	return HttpRouterParamsTypeId
}

func (h *RouterParams) ToNative(_ interface{}) interface{} {
	return h.params
}

func NewRouterParamsVal(r hrouter.Params) pl.Val {
	x := &RouterParams{
		params: r,
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
			return HttpRouterParamsTypeId
		},
		x.Info,
		x.ToNative,
	)
}
