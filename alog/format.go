package alog

import (
	"fmt"
	"strconv"
	"strings"
)

// the following format of access log is mostly based on Envoy's format string which is
// sort of ubiniques inside of the cloud native world.
// The following code is largely based on the envoy's C++ Code with some extension to
// be used in our cases.

const (
	cmdOnly          = 1
	cmdParamRequired = 2
	cmdParamOptional = 3
)

// We simply just compile the full format into a simple bytecode for formatting

const (
	fmtText   = 0
	fmtLookUp = 1
)

const (
	fStartTime = iota

	// size
	fReqHeaderBytes
	fBytesReceived
	fResponseHeadersBytes
	fResponseTrailersBytes

	// time
	fDuration
	fBytesSent
	fRequestDuration
	fResponseDuration

	fConnectionTerminationDetails

	// meta information
	fConnectionId
	fVirtualHost
	fRouterInfo

	// http transaction information
	fReq
	fResp
	fURI
	fTrailer
	fResponseCode
	fResponseCodeDetail
	fHost

	// framework information
	fRequestMiddleware
	fResponseMiddleware
	fApplicationMiddleware

	// other information
	fServiceName
	fClientIp
	fProtocol
	fScheme
)

type bytecode struct {
	op     int
	param  interface{}
	length int // if < 0 means not in used
}

// for different types of formatter we need to extract different types of information out
const (
	HFNotUsed = iota

	// well known field
	HFMethod
	HFStatusCode
	HFProtocolVersion
	HFTLSCipherSuite
	HFTLSVersion
	HFPath
	HFHost
	HFAuthority
	HFUserAgent
	HFCookie

	HFUser
)

const (
	// URL related
	UFNotUsed = iota

	UFScheme
	UFUsername
	UFPassword
	UFHost
	UFPath
	UFRawPath
	UFRawQuery
	UFFragment

	UFUser
)

const (
	MFNotUsed = iota

	MFLastRequest
	MFLastResponse
	MFFirstRequest
	MFFirstResponse

	MFCount
	MFRequestName
	MFResponseName
	MFApplicationName

	MFUser
)

var cmdMap = map[string]int{
	"START_TIME":                     fStartTime,
	"REQ_HEADER_BYTES":               fReqHeaderBytes,
	"BYTES_RECEIVED":                 fBytesReceived,
	"RESPONSE_HEADERS_BYTES":         fResponseHeadersBytes,
	"RESPONSE_TRAILERS_BYTES":        fResponseTrailersBytes,
	"DURATION":                       fDuration,
	"BYTES_SENT":                     fBytesSent,
	"REQUEST_DURATION":               fRequestDuration,
	"RESPONSE_DURATION":              fResponseDuration,
	"CONNECTION_TERMINATION_DETAILS": fConnectionTerminationDetails,
	"CONNECTION_ID":                  fConnectionId,
	"REQ":                            fReq,
	"RESP":                           fResp,
	"URI":                            fURI,
	"TRAILER":                        fTrailer,
	"RESPONSE_CODE":                  fResponseCode,
	"RESPONSE_CODE_DETAIL":           fResponseCodeDetail,
	"HOST":                           fHost,
	"REQUEST_MIDDLEWARE":             fRequestMiddleware,
	"RESPONSE_MIDDLEWARE":            fResponseMiddleware,
	"APPLICATION_MIDDLEWARE":         fApplicationMiddleware,
	"SERVICE_NAME":                   fServiceName,
	"CLIENT_IP":                      fClientIp,
	"PROTOCOL":                       fProtocol,
	"SCHEME":                         fScheme,
}

var paramMap = map[int]int{
	fStartTime: cmdParamOptional,

	fReqHeaderBytes:   cmdOnly,
	fBytesReceived:    cmdOnly,
	fRequestDuration:  cmdOnly,
	fResponseDuration: cmdOnly,

	fConnectionTerminationDetails: cmdOnly,

	fConnectionId: cmdOnly,
	fVirtualHost:  cmdOnly,

	fReq:                cmdParamRequired,
	fResp:               cmdParamRequired,
	fURI:                cmdParamRequired,
	fTrailer:            cmdParamRequired,
	fResponseCode:       cmdOnly,
	fResponseCodeDetail: cmdOnly,
	fHost:               cmdOnly,

	fRequestMiddleware:     cmdParamRequired,
	fResponseMiddleware:    cmdParamRequired,
	fApplicationMiddleware: cmdParamRequired,

	fServiceName: cmdOnly,
	fClientIp:    cmdOnly,
	fProtocol:    cmdOnly,
	fScheme:      cmdOnly,
}

var wellknownHttp = map[string]int{
	":METHOD":           HFMethod,
	":STATUS_CODE":      HFStatusCode,
	":PROTOCOL_VERSION": HFProtocolVersion,
	":TLS_CIPHER_SUITE": HFTLSCipherSuite,
	":TLS_VERSION":      HFTLSVersion,
	":PATH":             HFPath,
	":Host":             HFHost,
	":AUTHORITY":        HFAuthority,
	":USER_AGENT":       HFUserAgent,
	":COOKIE":           HFCookie,
}

var wellknownURI = map[string]int{
	// URI related
	":SCHEME":    UFScheme,
	":USERNAME":  UFUsername,
	":PASSWORD":  UFPassword,
	":HOST":      UFHost,
	":PATH":      UFPath,
	":RAW_PATH":  UFRawPath,
	":RAW_QUERY": UFRawQuery,
	":FRAGMENT":  UFFragment,
}

var wellknownMiddleware = map[string]int{
	":LAST_REQUEST":  MFLastRequest,
	":LAST_RESPONSE": MFLastResponse,

	":FIRST_REQUEST":  MFFirstRequest,
	":FIRST_RESPONSE": MFFirstResponse,

	":COUNT":            MFCount,
	":REQUEST_NAME":     MFRequestName,
	":RESPONSE_NAME":    MFResponseName,
	":APPLICATION_NAME": MFApplicationName,
}

type FormatField struct {
	t     int    // type of http component
	cname string // if http component is reqCustomize, then contains the name
}

func notUsedHttpField() FormatField {
	return FormatField{
		t: 0,
	}
}

type FormatParam struct {
	field   FormatField
	orField FormatField
}

type program []bytecode

type formatParser struct {
	prog program
}

func (p *formatParser) parse(f string) error {
	start := 0

	for start < len(f) {
		format := f[start:]

		// finding the %info% field inside of it. Notes %% will be used as escape for us
		percentBeg := strings.Index(
			format,
			"%",
		)

		if percentBeg == -1 {
			p.prog = append(p.prog, bytecode{
				op:    fmtText,
				param: format,
			})
			break
		}

		percentEnd := strings.Index(
			format[percentBeg+1:],
			"%",
		)

		if percentEnd == -1 {
			return fmt.Errorf("invalid %% pair, %% must be closed with another %%")
		}

		percentEnd += percentBeg + 1

		field := format[percentBeg+1 : percentEnd]
		name := field
		var param string
		length := -1

		// now trying to parse the field
		if field == "" {
			p.prog = append(p.prog, bytecode{
				op:    fmtText,
				param: "%",
			})
		} else {

			// now try to parse internal field, which is something like A(params):z
			lpar := strings.Index(
				field,
				"(",
			)

			colon := 0

			if lpar != -1 {
				rpar := strings.Index(
					field,
					")",
				)
				param = field[lpar+1 : rpar]
				colon = rpar + 1
				name = field[:lpar]
			}

			colonIndex := strings.Index(field[colon:], ":")
			if colonIndex != -1 {
				l, err := strconv.Atoi(field[colonIndex+1:])
				if err != nil {
					return fmt.Errorf("length is invalid")
				}
				length = l
			}

			cmd, ok := cmdMap[name]
			if !ok {
				return fmt.Errorf("access log format field %s is unknown", name)
			}

			opt, ok := paramMap[cmd]
			if !ok {
				panic(fmt.Sprintf("BUG: %s does not have parameter flags", name))
			}

			switch opt {
			case cmdOnly:
				if param != "" {
					return fmt.Errorf("cmd: %s does not require parameter", name)
				}
				break

			case cmdParamRequired:
				if param == "" {
					return fmt.Errorf("cmd: %s does not require parameter", name)
				}
				break

			default:
				break
			}

			if err := p.parseCommand(cmd, param, length); err != nil {
				return err
			}
		}

		start += percentEnd + 1
	}

	return nil
}

func (p *formatParser) parseSize(
	flag int,
	param string,
	length int,
) error {
	p.prog = append(p.prog, bytecode{
		op:     flag,
		length: length,
	})
	return nil
}

func (p *formatParser) parseToggle(
	flag int,
	_ string,
	length int,
) error {
	p.prog = append(p.prog, bytecode{
		op:     flag,
		length: length,
	})
	return nil
}

func (p *formatParser) asField(
	table map[string]int,
	cflag int,
	field string,
	output *FormatField,
) {
	t, ok := table[field]
	if !ok {
		output.t = cflag
		output.cname = field
	} else {
		output.t = t
	}
}

func (p *formatParser) parseField(
	table map[string]int,
	flag int,
	cflag int,
	field string,
	length int,
) error {
	quest := strings.Index(field, "?")

	var par FormatParam

	if quest == -1 {
		p.asField(
			table,
			cflag,
			field,
			&par.field,
		)
	} else {
		first := field[:quest]
		second := field[quest+1:]

		p.asField(
			table,
			cflag,
			first,
			&par.field,
		)

		p.asField(
			table,
			cflag,
			second,
			&par.orField,
		)
	}

	p.prog = append(p.prog, bytecode{
		op:     flag,
		param:  par,
		length: length,
	})

	return nil
}

func (p *formatParser) parseHttpField(
	flag int,
	param string,
	length int,
) error {
	return p.parseField(
		wellknownHttp,
		flag,
		HFUser,
		param,
		length,
	)
}

func (p *formatParser) parseURIField(
	flag int,
	param string,
	length int,
) error {
	return p.parseField(
		wellknownURI,
		flag,
		UFUser,
		param,
		length,
	)
}

func (p *formatParser) parseMiddleware(
	flag int,
	param string,
	length int,
) error {
	return p.parseField(
		wellknownMiddleware,
		flag,
		MFUser,
		param,
		length,
	)
}

func (p *formatParser) parseStartTime(
	format string,
	length int,
) error {

	p.prog = append(p.prog, bytecode{
		op:     fStartTime,
		param:  format,
		length: length,
	})
	return nil
}

func (p *formatParser) parseCommand(flag int, param string, length int) error {
	switch flag {
	case fStartTime:
		// specialized since it needs to parse the format
		return p.parseStartTime(
			param,
			length,
		)

	case fReqHeaderBytes,
		fBytesReceived,
		fResponseHeadersBytes,
		fResponseTrailersBytes,
		fDuration,
		fBytesSent,
		fRequestDuration:
		return p.parseSize(flag, param, length)

	case fConnectionTerminationDetails,
		fConnectionId,
		fVirtualHost:

		return p.parseToggle(flag, param, length)

	case fRouterInfo, fReq, fResp, fTrailer:
		return p.parseHttpField(flag, param, length)

	case fURI:
		return p.parseURIField(fURI, param, length)

	case fResponseCode, fResponseCodeDetail, fHost:
		return p.parseToggle(flag, param, length)

	case fRequestMiddleware, fResponseMiddleware, fApplicationMiddleware:
		return p.parseMiddleware(flag, param, length)

	case fServiceName, fClientIp, fProtocol, fScheme:
		return p.parseToggle(flag, param, length)

	default:
		return nil
	}
}
