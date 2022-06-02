package alog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dianpeng/mono-service/util"
	hrouter "github.com/julienschmidt/httprouter"
)

// Session log format
const (
	// general
	fmtServiceName = iota
	fmtClientIp
	fmtHostname
	fmtReqUUID

	// http request
	fmtReqMethod
	fmtReqVersion
	fmtReqScheme
	fmtReqURI
	fmtReqURIQuery
	fmtReqURIPath
	fmtReqReqPath

	// special known header
	fmtReqHeaderHost
	fmtReqHeaderUA
	fmtReqHeaderVia
	fmtReqHeaderCL
	fmtReqHeaderTE

	// detail information about requests
	fmtReqSummary

	// response
	fmtRespStatus
	fmtRespHeaderCL
	fmtRespHeaderTE

	fmtLiteral
)

type fmtBytecode struct {
	op   int
	text string
}

type SessionLogFormat struct {
	Raw string
	bc  []fmtBytecode
}

type HttpResponseSummary interface {
	Status() int
	Header() http.Header
}

type VHostInformation interface {
	VHostName() string
	VHostEndpoint() string
}

type SessionInfo interface {
	ServiceName() string
	SessionPhase() string
	ErrorDescription() string
}

type SessionLog struct {
	Format *SessionLogFormat

	HttpRequest         *http.Request
	RouterParams        hrouter.Params
	HttpResponseSummary HttpResponseSummary

	VHost   VHostInformation
	Session SessionInfo

	// customized information appending after inside of the session log
	Appendix []string
}

func getFormatKey(op int) string {
	switch op {
	case fmtServiceName:
		return "ServiceName"
	case fmtClientIp:
		return "ClientIp"
	case fmtHostname:
		return "Hostname"
	case fmtReqUUID:
		return "ReqUUID"
	case fmtReqMethod:
		return "ReqMethod"
	case fmtReqVersion:
		return "ReqVersion"
	case fmtReqScheme:
		return "ReqScheme"
	case fmtReqURI:
		return "ReqURI"
	case fmtReqURIQuery:
		return "ReqURIQuery"
	case fmtReqURIPath:
		return "ReqURIPath"
	case fmtReqReqPath:
		return "ReqReqPath"
	case fmtReqHeaderHost:
		return "ReqHeaderHost"
	case fmtReqHeaderUA:
		return "ReqHeaderUA"
	case fmtReqHeaderVia:
		return "ReqHeaderVia"
	case fmtReqHeaderCL:
		return "ReqHeaderCL"
	case fmtReqHeaderTE:
		return "ReqHeaderTE"
	case fmtReqSummary:
		return "ReqSummary"
	case fmtRespStatus:
		return "RespStatus"
	case fmtRespHeaderCL:
		return "RespHeaderCL"
	case fmtRespHeaderTE:
		return "RespHeaderTE"
	default:
		return "Literal"
	}
}

func parseFormatKey(x string) int {
	switch x {
	case "ServiceName", "service_name":
		return fmtServiceName
	case "ClientIp", "client_ip":
		return fmtClientIp
	case "Hostname", "hostname":
		return fmtHostname
	case "ReqUUID", "req_uuid":
		return fmtReqUUID
	case "ReqMethod", "req_method":
		return fmtReqMethod
	case "ReqVersion", "req_version":
		return fmtReqVersion
	case "ReqScheme", "req_scheme":
		return fmtReqScheme
	case "ReqURI", "req_uri":
		return fmtReqURI
	case "ReqURIQuery", "req_uri_query":
		return fmtReqURIQuery
	case "ReqURIPath", "req_uri_path":
		return fmtReqURIPath
	case "ReqReqPath", "req_req_path":
		return fmtReqReqPath
	case "ReqHeaderHost", "req_header_host":
		return fmtReqHeaderHost
	case "ReqHeadrUA", "req_header_ua":
		return fmtReqHeaderUA
	case "ReqHeaderVia", "req_header_via":
		return fmtReqHeaderVia
	case "ReqHeaderCL", "req_header_cl":
		return fmtReqHeaderCL
	case "ReqHeaderTE", "req_header_te":
		return fmtReqHeaderTE

	case "ReqSummary", "req_summary":
		return fmtReqSummary
	case "RespStatus", "resp_status":
		return fmtRespStatus
	case "RespHeaderCL", "resp_header_cl":
		return fmtRespHeaderCL
	case "RespHeaderTE", "resp_header_te":
		return fmtRespHeaderTE

	default:
		return fmtLiteral
	}
}

func getEntry(log *SessionLog, bc *fmtBytecode) string {
	opcode := bc.op
	switch opcode {
	case fmtServiceName:
		return log.Session.ServiceName()
	case fmtClientIp:
		return log.HttpRequest.RemoteAddr
	case fmtHostname:
		return util.GetHostname()
	case fmtReqUUID:
		return ""

	case fmtReqMethod:
		return log.HttpRequest.Method
	case fmtReqVersion:
		return log.HttpRequest.Proto
	case fmtReqScheme:
		if log.HttpRequest.TLS != nil {
			return "https"
		} else {
			return "http"
		}
	case fmtReqURI:
		return log.HttpRequest.URL.String()
	case fmtReqURIQuery:
		return log.HttpRequest.URL.RawQuery
	case fmtReqURIPath:
		return log.HttpRequest.URL.Path
	case fmtReqReqPath:
		return log.HttpRequest.RequestURI
	case fmtReqHeaderHost:
		return log.HttpRequest.Host
	case fmtReqHeaderUA:
		return log.HttpRequest.Header.Get("User-Agent")
	case fmtReqHeaderVia:
		return log.HttpRequest.Header.Get("Via")
	case fmtReqHeaderCL:
		return fmt.Sprintf("%d", log.HttpRequest.ContentLength)
	case fmtReqHeaderTE:
		b := new(bytes.Buffer)
		for _, x := range log.HttpRequest.TransferEncoding {
			b.WriteString(x)
			b.WriteString(", ")
		}
		return b.String()

	case fmtReqSummary:
		b := new(bytes.Buffer)
		b.WriteString(fmt.Sprintf("[Method=%s]", log.HttpRequest.Method))
		b.WriteString(fmt.Sprintf("[Path=%s]", log.HttpRequest.RequestURI))
		b.WriteString(fmt.Sprintf("[Host=%s]", log.HttpRequest.Host))
		for key, value := range log.HttpRequest.Header {
			b.WriteString(fmt.Sprintf("[%s=%s]", key, value))
		}
		return b.String()

	case fmtRespStatus:
		return fmt.Sprintf("%d", log.HttpResponseSummary.Status())
	case fmtRespHeaderCL:
		return log.HttpResponseSummary.Header().Get("Content-Length")
	case fmtRespHeaderTE:
		return log.HttpResponseSummary.Header().Get("Transfer-Encoding")

	default:
		return bc.text
	}
}

func (f *SessionLogFormat) parse() error {
	start := 0
	size := len(f.Raw)

	for start < size {
		curSlice := f.Raw[size:]
		loc := strings.Index(curSlice, "{{")
		if loc == -1 {
			f.bc = append(f.bc, fmtBytecode{
				op:   fmtLiteral,
				text: f.Raw[size:],
			})
			break
		} else {
			if loc != 0 {
				f.bc = append(f.bc, fmtBytecode{
					op:   fmtLiteral,
					text: curSlice[size:loc],
				})
			}

			endLoc := strings.Index(curSlice, "}}")
			if endLoc == -1 {
				return fmt.Errorf("invalid placeholder")
			}

			key := strings.Trim(curSlice[loc+2:endLoc], " ")
			opcode := parseFormatKey(key)
			if opcode == fmtLiteral {
				return fmt.Errorf("unknown format key")
			}
			f.bc = append(f.bc, fmtBytecode{
				op: opcode,
			})

			size += endLoc + 2
		}
	}
	return nil
}

func (f *SessionLogFormat) toText(log *SessionLog, d string, scheme bool) string {
	b := new(bytes.Buffer)
	for _, entry := range f.bc {
		if scheme {
			b.WriteString(getFormatKey(entry.op))
			b.WriteRune(':')
		}
		b.WriteString(getEntry(log, &entry))
		b.WriteString(d)
	}
	for _, a := range log.Appendix {
		b.WriteString(a)
		b.WriteString(d)
	}
	return b.String()
}

func (f *SessionLogFormat) toJSON(log *SessionLog) string {
	obj := make(map[string]interface{})
	for _, entry := range f.bc {
		obj[getFormatKey(entry.op)] = getEntry(log, &entry)
	}
	obj["$appendix"] = log.Appendix

	x, _ := json.Marshal(&obj)
	return string(x)
}

func NewSessionLogFormat(xx string) (*SessionLogFormat, error) {
	f := &SessionLogFormat{
		Raw: xx,
	}
	if err := f.parse(); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *SessionLog) ToText(d string, scheme bool) string {
	return s.Format.toText(s, d, scheme)
}

func (s *SessionLog) ToJSON() string {
	return s.Format.toJSON(s)
}
