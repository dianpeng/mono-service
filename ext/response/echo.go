package response

// echoing whatever has been received back if we have a body

import (
	"github.com/dianpeng/mono-service/framework"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
)

type echo struct {
	args []pl.Val
}

func (e *echo) Name() string {
	return "response.echo"
}

func (e *echo) Accept(
	r *http.Request,
	p hrouter.Params,
	w framework.HttpResponseWriter,
	ctx framework.ServiceContext,
) bool {
	cfg := framework.NewPLConfig(
		ctx,
		e.args,
	)

	status := 200
	flush := false

	cfg.TryGetInt(
		0,
		&status,
		200,
	)

	cfg.TryGetBool(
		1,
		&flush,
		false,
	)

	w.WriteStatus(status)
	w.WriteBody(
		r.Body,
	)

	if flush {
		w.Flush()
	}

	return true
}

type echofactory struct{}

func (e *echofactory) Create(x []pl.Val) (framework.Middleware, error) {
	return &echo{
		args: x,
	}, nil
}

func (e *echofactory) Name() string {
	return "response.echo"
}

func (e *echofactory) Comment() string {
	return "echo request's body back as response"
}

func init() {
	framework.AddResponseFactory(
		"echo",
		&echofactory{},
	)
}
