package phase

import (
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/hrouter"
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

func GetPhaseName(p int) string {
	switch p {
	case PhaseCreateSessionHandler:
		return ".create_session_handler"
	case PhaseInit:
		return ".init"
	case PhaseHttpAccess:
		return "http.access"
	case PhaseSessionStart:
		return "session.start"
	case PhaseSessionPrepare:
		return "session.prepare"
	case PhaseSessionAccept:
		return "session.accept"
	case PhaseSessionDone:
		return "session.done"
	case PhaseHttpResponse:
		return "http.response"
	case PhaseAccessLog:
		return ".access_log"
	default:
		return "<unknown>"
	}
}

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
