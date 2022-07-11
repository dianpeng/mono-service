package phase

const (
	PhaseCreateService = iota
	PhaseInit

	PhaseHttpRequest

	// application related phases
	PhaseApplicationPrepare
	PhaseApplicationAccept
	PhaseApplicationEvent
	PhaseApplicationDone

	PhaseHttpResponse
	PhaseHttpResponseFinalize

	// lastly log generation
	PhaseAccessLog

	// unknown phase, used for background execution
	PhaseBackground
)

func GetPhaseName(p int) string {
	switch p {
	case PhaseCreateService:
		return "create_service"
	case PhaseInit:
		return "init"
	case PhaseApplicationPrepare:
		return "application.prepare"
	case PhaseApplicationAccept:
		return "application.accept"
	case PhaseApplicationEvent:
		return "application.event"
	case PhaseApplicationDone:
		return "application.done"
	case PhaseHttpResponse:
		return "http.response"
	case PhaseHttpResponseFinalize:
		return "http.response_finalize"
	case PhaseAccessLog:
		return "access_log"
	case PhaseBackground:
		return "backgronud"
	default:
		return "<unknown>"
	}
}
