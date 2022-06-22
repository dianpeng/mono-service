package framework

import (
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
)

// builtin middleware

type event struct {
	args []pl.Val
}

func (e *event) Name() string {
	return "event"
}

func (e *event) Accept(
	_ *http.Request,
	_ hrouter.Params,
	w HttpResponseWriter,
	ctx ServiceContext,
) bool {
	cfg := NewPLConfig(ctx, e.args)
	eventName := ""
	if err := cfg.GetStr(0, &eventName); err != nil {
		w.ReplyErrorHPL(err)
		return false
	}

	// run the event
	if err := ctx.Hpl().Run(eventName); err != nil {
		w.ReplyErrorHPL(err)
		return false
	}

	return true
}

type eventfactory struct{}

func (_ *eventfactory) Create(x []pl.Val) (Middleware, error) {
	return &event{args: x}, nil
}

func (_ *eventfactory) Name() string {
	return "event"
}

func (_ *eventfactory) Comment() string {
	return "event a specific event and run corresponding PL entry synchronously"
}

func init() {
	AddResponseFactory(
		"event",
		&eventfactory{},
	)
}
