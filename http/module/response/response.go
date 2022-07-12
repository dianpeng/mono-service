package response

import (
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
)

type response struct {
	args []pl.Val
}

func (r *response) Name() string {
	return "response.response"
}

func (r *response) Accept(
	_ *http.Request,
	p hrouter.Params,
	w framework.HttpResponseWriter,
	ctx framework.ServiceContext,
) bool {
	cfg := hpl.NewPLConfig(
		ctx.Runtime().Eval,
		r.args,
	)

	status := 200
	flush := false
	body := ""
	headerVal := pl.NewValNull()

	cfg.TryGetInt(
		0,
		&status,
		200,
	)

	cfg.TryGetStr(
		1,
		&body,
		"",
	)

	cfg.TryGet(
		2,
		&headerVal,
		pl.NewValNull(),
	)

	cfg.TryGetBool(
		3,
		&flush,
		false,
	)

	if !headerVal.IsNull() {
		header, err := hpl.NewHeaderValFromVal(headerVal)
		if err != nil {
			w.ReplyError(
				"response.response",
				500,
				err,
			)
			return false
		}

		hdr := header.Usr().(*hpl.Header).HttpHeader()
		w.SetHeader(hdr)
	}

	w.WriteStatus(status)
	w.WriteBody(
		hpl.NewReadCloserFromString(body),
	)

	if flush {
		w.Flush()
	}
	return true
}

type responsefactory struct{}

func (r *responsefactory) Name() string {
	return "response.response"
}

func (r *responsefactory) Comment() string {
	return "generate a response based on user configuration"
}

func (r *responsefactory) Create(x []pl.Val) (framework.Middleware, error) {
	return &response{
		args: x,
	}, nil
}

func init() {
	framework.AddResponseFactory(
		"response",
		&responsefactory{},
	)
}
