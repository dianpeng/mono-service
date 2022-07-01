package g

const (
	MaxSessionCacheSize = 200
	VHostRejectStatus   = 403
	VHostRejectBody     = "Service Disallowed"

	VHostErrorStatus = 500
	VHostErrorBody   = "Service Error"

	VHostHttpClientPoolMaxSize      = 256
	VHostHttpClientPoolTimeout      = 30
	VHostHttpClientPoolMaxDrainSize = 4096

	VHostLogFormat = "{{ServiceName}}" +
		"{{ClientIp}}" +
		"{{ReqScheme}}" +
		"{{ReqURI}}" +
		"{{ReqHeaderHost}}" +
		"{ReqHeaderUA}}" +
		"{{ReqHeaderVia}}" +
		"{{ReqSummary}}" +
		"{{RespStatus}}"
)
