package hpl

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
)

type cookie struct {
	c *http.Cookie
}

func (c *cookie) Cookie() *http.Cookie {
	return c.c
}

func (c *cookie) Index(key pl.Val) (pl.Val, error) {
	if !key.IsString() {
		return pl.NewValNull(), fmt.Errorf("invalid index, cookie's index type must be string")
	}
	return c.Dot(key.String())
}

func (c *cookie) SameSiteString() string {
	switch c.c.SameSite {
	case http.SameSiteDefaultMode:
		return "default"
	case http.SameSiteLaxMode:
		return "lax"
	case http.SameSiteStrictMode:
		return "strict"
	case http.SameSiteNoneMode:
		return "none"
	default:
		return "unknown"
	}
}

func (c *cookie) Dot(key string) (pl.Val, error) {
	switch key {
	case "name":
		return pl.NewValStr(c.c.Name), nil
	case "value":
		return pl.NewValStr(c.c.Value), nil

	case "path":
		return pl.NewValStr(c.c.Path), nil
	case "domain":
		return pl.NewValStr(c.c.Domain), nil
	case "expireString":
		return pl.NewValStr(c.c.RawExpires), nil

	case "maxAge":
		if c.c.MaxAge == 0 {
			return pl.NewValNull(), nil
		} else if c.c.MaxAge < 0 {
			return pl.NewValInt(-1), nil
		} else {
			return pl.NewValInt(c.c.MaxAge), nil
		}

	case "sameSite":
		return pl.NewValStr(c.SameSiteString()), nil

	case "secure":
		return pl.NewValBool(c.c.Secure), nil

	case "httpOnly":
		return pl.NewValBool(c.c.HttpOnly), nil

	case "raw":
		return pl.NewValStr(c.c.Raw), nil

	case "rawList":
		return pl.NewValStrList(c.c.Unparsed), nil

	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("unknown field name %s for %s", key, c.Id())
}

func (c *cookie) IndexSet(key pl.Val, val pl.Val) error {
	if !key.IsString() {
		return fmt.Errorf("invalid index set key type for %s, must be string", c.Id())
	}
	return c.DotSet(key.String(), val)
}

func (c *cookie) DotSet(key string, val pl.Val) error {
	switch key {
	case "name":
		if val.IsString() {
			c.c.Name = val.String()
		} else {
			return fmt.Errorf("%s's field 'name' must set with value of type string", c.Id())
		}
		break

	case "value":
		if val.IsString() {
			c.c.Value = val.String()
		} else {
			return fmt.Errorf("%s's field 'value' must set with value of type string", c.Id())
		}
		break

	case "path":
		if val.IsString() {
			c.c.Path = val.String()
		} else {
			return fmt.Errorf("%s's field 'path' must set with value of type string", c.Id())
		}
		break

	case "domain":
		if val.IsString() {
			c.c.Domain = val.String()
		} else {
			return fmt.Errorf("%s's field 'domain' must set with value of type string", c.Id())
		}
		break

	case "maxAge":
		if val.IsInt() {
			c.c.MaxAge = int(val.Int())
		} else {
			return fmt.Errorf("%s's field 'maxAge' must set with value of type int", c.Id())
		}
		break

	case "secure":
		if val.IsBool() {
			c.c.Secure = val.Bool()
		} else {
			return fmt.Errorf("%s's field 'secure' must set with value of type bool", c.Id())
		}
		break

	case "httpOnly":
		if val.IsBool() {
			c.c.HttpOnly = val.Bool()
		} else {
			return fmt.Errorf("%s's field 'httpOnly' must set with value of type bool", c.Id())
		}
		break

	default:
		return fmt.Errorf("%s's field %s is unknown", c.Id(), key)
	}

	return nil
}

func (c *cookie) ToString() (string, error) {
	return c.Info(), nil
}

func (c *cookie) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"name":      c.c.Name,
			"value":     c.c.Value,
			"path":      c.c.Path,
			"domain":    c.c.Domain,
			"expire":    c.c.Expires,
			"rawExpire": c.c.RawExpires,
			"maxAge":    c.c.MaxAge,
			"secure":    c.c.Secure,
			"httpOnly":  c.c.HttpOnly,
			"sameSite":  c.SameSiteString(),
			"raw":       c.c.Raw,
			"unparsed":  c.c.Unparsed,
		},
	)
}

var (
	cookieMpProto = pl.MustNewFuncProto("http.cookie.isValid", "%0")
)

func (c *cookie) Method(name string, args []pl.Val) (pl.Val, error) {
	switch name {
	case "isValid":
		if _, err := cookieMpProto.Check(args); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(c.c.Valid() == nil), nil
	default:
		break
	}
	return pl.NewValNull(), fmt.Errorf("%s's method %s is unknown", c.Id(), name)
}

func (c *cookie) Info() string {
	return c.Id()
}

func (c *cookie) ToNative() interface{} {
	return c.c
}

func (c *cookie) Id() string {
	return HttpCookieTypeId
}

func (c *cookie) IsThreadSafe() bool {
	return false
}

func (c *cookie) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("%s does not support iterator", c.Id())
}

func NewCookieVal(c *http.Cookie) pl.Val {
	return pl.NewValUsr(
		&cookie{c: c},
	)
}
