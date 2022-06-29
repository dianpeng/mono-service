package hpl

import (
	"crypto/tls"
	"fmt"
	"github.com/dianpeng/mono-service/pl"
)

// TLS connection state object wrapper
type tlsConnState struct {
	state *tls.ConnectionState
}

func (c *tlsConnState) VersionString() string {
	switch c.state.Version {
	case tls.VersionTLS10:
		return "tls_10"
	case tls.VersionTLS11:
		return "tls_11"
	case tls.VersionTLS12:
		return "tls_12"
	case tls.VersionTLS13:
		return "tls_13"
	case tls.VersionSSL30:
		return "ssl_30"
	default:
		return "unknown"
	}
}

func (c *tlsConnState) CipherSuiteString() string {
	return tls.CipherSuiteName(c.state.CipherSuite)
}

func (c *tlsConnState) ConnectionState() *tls.ConnectionState {
	return c.state
}

func (c *tlsConnState) Index(key pl.Val) (pl.Val, error) {
	if !key.IsString() {
		return pl.NewValNull(),
			fmt.Errorf("invalid index type, http.tlsconnstate must use string index")
	}

	return c.Dot(key.String())
}

func (c *tlsConnState) Dot(key string) (pl.Val, error) {
	switch key {
	case "serverName":
		return pl.NewValStr(c.state.ServerName), nil
	case "version":
		return pl.NewValStr(c.VersionString()), nil
	case "cipherSuit":
		return pl.NewValStr(c.CipherSuiteString()), nil
	case "negotiatedProtocol":
		return pl.NewValStr(c.state.NegotiatedProtocol), nil

	default:
		return pl.NewValNull(), nil
	}
}

func (c *tlsConnState) IndexSet(_ pl.Val, _ pl.Val) error {
	return fmt.Errorf("%s does not support index set", c.Id())
}

func (c *tlsConnState) DotSet(_ string, _ pl.Val) error {
	return fmt.Errorf("%s does not support dot set", c.Id())
}

func (c *tlsConnState) ToString() (string, error) {
	return c.Info(), nil
}

func (c *tlsConnState) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"version":            c.VersionString(),
			"cipherSuit":         c.CipherSuiteString(),
			"negotiatedProtocol": c.state.NegotiatedProtocol,
			"serverName":         c.state.ServerName,
		},
	)
}

func (c *tlsConnState) Method(name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("%d's method %s is unknown", c.Id(), name)
}

func (c *tlsConnState) Info() string {
	return c.Id()
}

func (c *tlsConnState) ToNative() interface{} {
	return c.state
}

func (c *tlsConnState) Id() string {
	return TLSConnStateTypeId
}

func (c *tlsConnState) IsImmutable() bool {
	return true
}

func (c *tlsConnState) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("%s does not support iterator", c.Id())
}

func NewTLSConnStateVal(
	state *tls.ConnectionState,
) pl.Val {
	if state == nil {
		return pl.NewValNull()
	} else {
		return pl.NewValUsr(
			&tlsConnState{state: state},
		)
	}
}
