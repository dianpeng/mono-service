package vhost

import (
	"github.com/dianpeng/mono-service/alog"
	"net/http"
	"time"
)

type logProvider struct {
	s       *serviceHandler
	startTs time.Time
	hreq    *http.Request
	hresp   responseWriterWrapper

	duration            int64
	requestDuration     int64
	responseDuration    int64
	applicationDuration int64
}

// implementation of various log provider related functions -------------------
func (l *logProvider) FormatStartTime(
	fmt string,
) string {
	return l.startTs.Format(fmt)
}

func (l *logProvider) ReqHeaderBytes() (int64, bool) {
	return 0, false
}

func (l *logProvider) BytesReceived() (int64, bool) {
	return 0, false
}

func (l *logProvider) ResponseHeadersBytes() (int64, bool) {
	return 0, false
}

func (l *logProvider) ResponseTrailersBytes() (int64, bool) {
	return 0, false
}

func (l *logProvider) BytesSent() (int64, bool) {
	return 0, false
}

func (l *logProvider) Duration() (int64, bool) {
	return l.duration, true
}

func (l *logProvider) RequestDuration() (int64, bool) {
	return l.requestDuration, true
}

func (l *logProvider) ResponseDuration() (int64, bool) {
	return l.responseDuration, true
}

func (l *logProvider) ConnectionTerminationDetails() (string, bool) {
	return "", false
}

func (l *logProvider) ConnectionId() (string, bool) {
	return "", false
}

func (l *logProvider) VirtualHost() (string, bool) {
	return "", false
}

func (l *logProvider) RouterInfo(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) Req(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) Resp(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) URI(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) Trailer(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) ResponseCode() (int64, bool) {
	return 0, false
}

func (l *logProvider) ResponseCodeDetail() (string, bool) {
	return "", false
}

func (l *logProvider) RequestMiddleware(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) ResponseMiddleware(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) ApplicationMiddleware(_ alog.FormatParam) (string, bool) {
	return "", false
}

func (l *logProvider) Host() (string, bool) {
	return "", false
}

func (l *logProvider) ServiceName() (string, bool) {
	return "", false
}

func (l *logProvider) ClientIp() (string, bool) {
	return "", false
}

func (l *logProvider) Protocol() (string, bool) {
	return "", false
}

func (l *logProvider) Scheme() (string, bool) {
	return "", false
}
