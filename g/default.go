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

	VHostLogFormat = "" +
		"%START_TIME%" +
		"%SERVICE_NAME%" +
		"%REQ(:METHOD)%" +
		"%REQ(:STATUS_CODE)%" +
		"%REQ(:PATH)%" +
		"%RESP(:STATUS_CODE)%" +
		"%CLIENT_IP%"
)
