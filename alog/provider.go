package alog

// Provider abstract out what the subsystem should provide to the access log
// pipeline for logging out
type Provider interface {
	FormatStartTime(
		string,
	) string

	ReqHeaderBytes() (int64, bool)
	BytesReceived() (int64, bool)
	ResponseHeadersBytes() (int64, bool)
	ResponseTrailersBytes() (int64, bool)
	BytesSent() (int64, bool)

	Duration() (int64, bool)
	RequestDuration() (int64, bool)
	ResponseDuration() (int64, bool)

	ConnectionTerminationDetails() (string, bool)

	ConnectionId() (string, bool)
	VirtualHost() (string, bool)
	RouterInfo(FormatParam) (string, bool)
	Req(FormatParam) (string, bool)
	Resp(FormatParam) (string, bool)
	URI(FormatParam) (string, bool)
	Trailer(FormatParam) (string, bool)
	ResponseCode() (int64, bool)

	ResponseCodeDetail() (string, bool)

	RequestMiddleware(FormatParam) (string, bool)
	ResponseMiddleware(FormatParam) (string, bool)
	ApplicationMiddleware(FormatParam) (string, bool)
	Host() (string, bool)
	ServiceName() (string, bool)
	ClientIp() (string, bool)
	Protocol() (string, bool)
	Scheme() (string, bool)
}
