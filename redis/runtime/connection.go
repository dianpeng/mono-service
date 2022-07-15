package runtime

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"github.com/tidwall/redcon"
)

const (
	ConnTypeId = "redis.conn"
)

var (
	methodProtoConnRemoteAddr = pl.MustNewFuncProto("redis.conn.remoteAddr", "%0")
	methodProtoConnClose      = pl.MustNewFuncProto("redis.conn.close", "%0")

	methodProtoConnWriteError  = pl.MustNewFuncProto("redis.conn.writeError", "%s")
	methodProtoConnWriteString = pl.MustNewFuncProto("redis.conn.writeString", "%s")

	methodProtoConnWriteInt    = pl.MustNewFuncProto("redis.conn.writeInt", "%d")
	methodProtoConnWriteInt64  = pl.MustNewFuncProto("redis.conn.writeInt64", "%d")
	methodProtoConnWriteUInt64 = pl.MustNewFuncProto("redis.conn.writeUint64", "%d")

	methodProtoConnWriteList = pl.MustNewFuncProto("redis.conn.writeList", "%l")
	methodProtoConnWriteNull = pl.MustNewFuncProto("redis.conn.writeNull", "%0")
)

type conn struct {
	c redcon.Conn
}

func ValIsConn(c pl.Val) bool {
	return c.Id() == ConnTypeId
}

func (c *conn) Conn() redcon.Conn {
	return c.c
}

func (c *conn) Index(
	_ pl.Val,
) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("%s index: unsupported operation", c.Id())
}

func (c *conn) IndexSet(
	_ pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("%s index set: unsupported operation", c.Id())
}

func (c *conn) Dot(
	_ string,
) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("%s index: unsupported operation", c.Id())
}

func (c *conn) DotSet(
	_ string,
	_ pl.Val,
) error {
	return fmt.Errorf("%s dot set: unsupported operation", c.Id())
}

func (c *conn) ToString() (string, error) {
	return c.Id(), nil
}

func (c *conn) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"type":       c.Id(),
			"remoteAddr": c.Conn().RemoteAddr(),
		},
	)
}

func (c *conn) Info() string {
	return c.Id()
}

func (c *conn) ToNative() interface{} {
	return c
}

func (c *conn) Id() string {
	return ConnTypeId
}

func (c *conn) IsThreadSafe() bool {
	return false
}

func (c *conn) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("%s: does not support iterator", c.Id())
}

func (c *conn) Method(name string, arg []pl.Val) (pl.Val, error) {
	switch name {
	case "remoteAddr":
		if _, err := methodProtoConnRemoteAddr.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValStr(c.Conn().RemoteAddr()), nil

	case "close":
		if _, err := methodProtoConnClose.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		err := c.c.Close()
		return pl.NewValNull(), err

	case "writeError":
		if _, err := methodProtoConnWriteError.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		c.c.WriteError(arg[0].String())
		return pl.NewValNull(), nil

	case "writeString":
		if _, err := methodProtoConnWriteString.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		c.c.WriteString(arg[0].String())
		return pl.NewValNull(), nil

	case "writeInt":
		if _, err := methodProtoConnWriteInt.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		c.c.WriteInt(int(arg[0].Int()))
		return pl.NewValNull(), nil

	case "writeInt64":
		if _, err := methodProtoConnWriteInt.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		c.c.WriteInt64(arg[0].Int())
		return pl.NewValNull(), nil

	case "writeUint64":
		if _, err := methodProtoConnWriteInt.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		v := arg[0].Int()
		if v < 0 {
			return pl.NewValNull(), fmt.Errorf("%s method writeUint64: value is negative", c.Id())
		}
		c.c.WriteUint64(uint64(v))
		return pl.NewValNull(), nil

	case "writeNull":
		if _, err := methodProtoConnWriteNull.Check(arg); err != nil {
			return pl.NewValNull(), nil
		}
		c.c.WriteNull()
		return pl.NewValNull(), nil

	case "writeList":
		if _, err := methodProtoConnWriteList.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		list := arg[0].List()
		l := list.Length()

		c.c.WriteArray(l)
		for i := 0; i < l; i++ {
			v := list.At(i)
			str, err := v.ToString()
			if err != nil {
				return pl.NewValNull(), fmt.Errorf("%s method writeList: %dth element is not string", c.Id(), i)
			} else {
				c.c.WriteBulkString(str)
			}
		}

		return pl.NewValNull(), nil

	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("%s method %s: unknown method", c.Id(), name)
}

func newConnection(c redcon.Conn) *conn {
	return &conn{
		c: c,
	}
}

func NewConnectionVal(c redcon.Conn) pl.Val {
	return pl.NewValUsr(newConnection(c))
}
