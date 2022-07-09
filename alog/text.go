package alog

import (
	"bytes"
	"fmt"
)

type totext struct{}

func (t *totext) fmtInt(
	vv func() (int64, bool),
	ep string,
) string {
	v, ok := vv()
	if ok {
		return fmt.Sprintf("%d", v)
	} else {
		return ep
	}
}

func (t *totext) fmtStr(
	vv func() (string, bool),
	ep string,
) string {
	v, ok := vv()
	if ok {
		return v
	} else {
		return ep
	}
}

func (t *totext) fmtField(
	vv func(FormatParam) (string, bool),
	param FormatParam,
	ep string,
) string {
	v, ok := vv(param)
	if ok {
		return v
	} else {
		return ep
	}
}

func (t *totext) toText(
	prog program,
	provider Provider,
	ep string,
	dl string,
) *bytes.Buffer {
	buf := new(bytes.Buffer)

	for _, bc := range prog {
		switch bc.op {
		case fStartTime:
			f := bc.param.(string)
			buf.WriteString(
				provider.FormatStartTime(
					f,
				),
			)
			break

		case fReqHeaderBytes:
			buf.WriteString(
				t.fmtInt(
					provider.ReqHeaderBytes,
					ep,
				),
			)
			break

		case fBytesReceived:
			buf.WriteString(
				t.fmtInt(
					provider.BytesReceived,
					ep,
				),
			)
			break

		case fResponseHeadersBytes:
			buf.WriteString(
				t.fmtInt(
					provider.ResponseHeadersBytes,
					ep,
				),
			)
			break

		case fResponseTrailersBytes:
			buf.WriteString(
				t.fmtInt(
					provider.ResponseTrailersBytes,
					ep,
				),
			)
			break

		case fDuration:
			buf.WriteString(
				t.fmtInt(
					provider.Duration,
					ep,
				),
			)
			break

		case fBytesSent:
			buf.WriteString(
				t.fmtInt(
					provider.BytesSent,
					ep,
				),
			)
			break

		case fRequestDuration:
			buf.WriteString(
				t.fmtInt(
					provider.RequestDuration,
					ep,
				),
			)
			break

		case fResponseDuration:
			buf.WriteString(
				t.fmtInt(
					provider.ResponseDuration,
					ep,
				),
			)
			break

		case fConnectionTerminationDetails:
			buf.WriteString(
				t.fmtStr(
					provider.ConnectionTerminationDetails,
					ep,
				),
			)
			break

		case fConnectionId:
			buf.WriteString(
				t.fmtStr(
					provider.ConnectionId,
					ep,
				),
			)
			break

		case fVirtualHost:
			buf.WriteString(
				t.fmtStr(
					provider.VirtualHost,
					ep,
				),
			)
			break

		case fRouterInfo:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.RouterInfo,
					par,
					ep,
				),
			)
			break

		case fReq:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.Req,
					par,
					ep,
				),
			)
			break

		case fResp:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.Resp,
					par,
					ep,
				),
			)
			break

		case fURI:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.URI,
					par,
					ep,
				),
			)
			break

		case fTrailer:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.Trailer,
					par,
					ep,
				),
			)
			break

		case fResponseCode:
			buf.WriteString(
				t.fmtInt(
					provider.ResponseCode,
					ep,
				),
			)
			break

		case fResponseCodeDetail:
			buf.WriteString(
				t.fmtStr(
					provider.ResponseCodeDetail,
					ep,
				),
			)
			break

		case fHost:
			buf.WriteString(
				t.fmtStr(
					provider.Host,
					ep,
				),
			)
			break

		case fRequestMiddleware:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.RequestMiddleware,
					par,
					ep,
				),
			)
			break

		case fResponseMiddleware:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.ResponseMiddleware,
					par,
					ep,
				),
			)
			break

		case fApplicationMiddleware:
			par := bc.param.(FormatParam)
			buf.WriteString(
				t.fmtField(
					provider.ApplicationMiddleware,
					par,
					ep,
				),
			)
			break

		case fServiceName:
			buf.WriteString(
				t.fmtStr(
					provider.ServiceName,
					ep,
				),
			)
			break

		case fClientIp:
			buf.WriteString(
				t.fmtStr(
					provider.ClientIp,
					ep,
				),
			)
			break

		case fProtocol:
			buf.WriteString(
				t.fmtStr(
					provider.Protocol,
					ep,
				),
			)
			break

		case fScheme:
			buf.WriteString(
				t.fmtStr(
					provider.Scheme,
					ep,
				),
			)
			break

		default:
			panic("should not reach here")
			break
		}

		buf.WriteString(dl)
	}

	return buf
}

func toText(
	prog program,
	provider Provider,
	ep string,
	dl string,
) *bytes.Buffer {
	tt := totext{}
	return tt.toText(prog, provider, ep, dl)
}
