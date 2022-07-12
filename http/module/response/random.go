package response

import (
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/util"
	"net/http"
)

type random struct {
	args []pl.Val
}

func (e *random) Name() string {
	return "response.random"
}

func (e *random) Accept(
	r *http.Request,
	p hrouter.Params,
	w framework.HttpResponseWriter,
	ctx framework.ServiceContext,
) bool {
	cfg := hpl.NewPLConfig(
		ctx.Runtime().Eval,
		e.args,
	)

	status := 0
	size := 0
	flush := false

	cfg.TryGetInt(
		0,
		&status,
		200,
	)

	cfg.TryGetInt(
		1,
		&size,
		1024,
	)

	cfg.TryGetBool(
		2,
		&flush,
		false,
	)

	w.WriteStatus(status)
	w.WriteBody(hpl.NewReadCloserFromString(util.RandomString(size)))

	if flush {
		w.Flush()
	}
	return true
}

type randomfactory struct{}

func (r *randomfactory) Name() string {
	return "response.random"
}

func (r *randomfactory) Comment() string {
	return "generate a random string as response"
}

func (r *randomfactory) Create(x []pl.Val) (framework.Middleware, error) {
	return &random{
		args: x,
	}, nil
}

func init() {
	framework.AddResponseFactory(
		"random",
		&randomfactory{},
	)
}
