package phase

import (
	"github.com/dianpeng/mono-service/alog"
	hrouter "github.com/julienschmidt/httprouter"
	"net/http"
)

const (
	PhaseCreateSessionHandler = iota
	PhaseInit

	PhaseHttpAccess
	PhaseHttpRequest

	// session related phases
	PhaseSessionStart
	PhaseSessionPrepare
	PhaseSessionAccept
	PhaseSessionDone

	PhaseHttpResponse

	// lastly log generation
	PhaseAccessLog
)

type PhaseAccess interface {
	Access(*http.Request, hrouter.Params) error
}

type PhaseRequest interface {
	Request(*http.Request, hrouter.Params) error
}

type PhaseResponse interface {
	Response(http.ResponseWriter, *http.Request, hrouter.Params) error
}

type PhaseLog interface {
	Log(*alog.SessionLog)
}
