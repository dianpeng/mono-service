package hpl

import (
	"encoding/json"
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"net/url"
	"strings"
)

type Url struct {
	url *url.URL
}

func ValIsUrl(v pl.Val) bool {
	return v.Id() == ".url"
}

func (h *Url) URL() *url.URL {
	return h.url
}

func (h *Url) Index(key pl.Val) (pl.Val, error) {
	if key.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, URL name must be string")
	}

	switch key.String() {
	case "scheme":
		return pl.NewValStr(h.url.Scheme), nil
	case "host":
		return pl.NewValStr(h.url.Host), nil
	case "hostname":
		return pl.NewValStr(h.url.Hostname()), nil
	case "port":
		return pl.NewValStr(h.url.Port()), nil
	case "path":
		return pl.NewValStr(h.url.Path), nil
	case "query":
		return pl.NewValStr(h.url.RawQuery), nil
	case "url":
		return pl.NewValStr(h.url.String()), nil
	case "userInfo":
		return pl.NewValStr(h.url.User.String()), nil
	case "requestURI":
		return pl.NewValStr(h.url.RequestURI()), nil
	case "href":
		return pl.NewValStr(h.url.String()), nil
	default:
		return pl.NewValNull(), fmt.Errorf("unknown component %s in URL", key.String())
	}
}

func (h *Url) IndexSet(key pl.Val, val pl.Val) error {
	if key.Type == pl.ValStr {
		return h.DotSet(key.String(), val)
	} else {
		return fmt.Errorf(".url index set type must be string")
	}
}

func (h *Url) Dot(name string) (pl.Val, error) {
	return h.Index(pl.NewValStr(name))
}

func (h *Url) SetUserInfo(userInfo string) {
	// if userInfo contains : then we treat it as a spearation
	pos := strings.LastIndex(userInfo, ":")
	if pos == -1 {
		// just a username no password
		h.url.User = url.User(userInfo)
	} else {
		h.url.User = url.UserPassword(
			userInfo[:pos],
			userInfo[pos+1:],
		)
	}
}

func (h *Url) DotSet(key string, val pl.Val) error {
	str, err := val.ToString()
	if err != nil {
		return fmt.Errorf(".url component set, value cannot convert to string: %s", err.Error())
	}

	switch key {
	case "scheme":
		h.url.Scheme = str
		break
	case "user":
		h.SetUserInfo(str)
		break
	case "host":
		h.url.Host = str
		break
	case "path":
		h.url.Path = str
		break
	case "query":
		h.url.RawQuery = str
		break
	case "hash":
		h.url.Fragment = str
		break
	default:
		return fmt.Errorf(".url component set, unknown field %s", key)
	}
	return nil
}

func (h *Url) ToString() (string, error) {
	return h.url.String(), nil
}

func (h *Url) ToJSON() (string, error) {
	blob, err := json.Marshal(h.url)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

var (
	methodProtoUrlIsAbs = pl.MustNewFuncProto(".url.isAbs", "%0")
)

func (h *Url) Method(name string, args []pl.Val) (pl.Val, error) {
	switch name {
	case "isAbs":
		if _, err := methodProtoUrlIsAbs.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(h.url.IsAbs()), nil

	default:
		break
	}
	return pl.NewValNull(), fmt.Errorf("method: .url:%s is unknown", name)
}

func (h *Url) Info() string {
	return fmt.Sprintf(
		".url[scheme=%s;user=%s;host=%s;path=%s;query=%s;frag=%s]",
		h.url.Scheme,
		h.url.User.String(),
		h.url.Host,
		h.url.Path,
		h.url.RawQuery,
		h.url.Fragment,
	)
}

func (h *Url) Id() string {
	return UrlTypeId
}

func (h *Url) ToNative() interface{} {
	return h.url
}

func (h *Url) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf(".url does not support iterator")
}

func NewUrlVal(url *url.URL) pl.Val {
	x := &Url{
		url: url,
	}
	return pl.NewValUsr(x)
}
