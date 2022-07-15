package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/redis/util"
	"github.com/tidwall/redcon"
)

const (
	CommandTypeId = "redis.command"
)

type command struct {
	args [][]byte
	name string
}

func ValIsCommand(v pl.Val) bool {
	return v.Id() == CommandTypeId
}

func (c *command) Argument() [][]byte {
	return c.args
}

func (c *command) Name() string {
	return c.name
}

func (c *command) ArgumentSize() int {
	return len(c.args)
}

func (c *command) At(i int) []byte {
	return c.args[i]
}

func (c *command) StringList() []string {
	x := make([]string, 0, len(c.args))
	for i := 0; i < len(c.args); i++ {
		x = append(x, string(c.args[i]))
	}
	return x
}

func (c *command) property(prop string) (pl.Val, bool) {
	switch prop {
	case "length":
		return pl.NewValInt(c.ArgumentSize()), true
	case "command":
		return pl.NewValStr(c.Name()), true
	case "category":
		return pl.NewValStr(
			util.CommandCategoryName(c.Name()),
		), true
	default:
		break
	}
	return pl.NewValNull(), false
}

func (c *command) Index(
	key pl.Val,
) (pl.Val, error) {
	if key.IsInt() {
		idx, err := key.ToIndex()
		if err != nil {
			return pl.NewValNull(), fmt.Errorf("%s index: invalid index %s", c.Id(), err.Error())
		}
		if idx >= c.ArgumentSize() {
			return pl.NewValNull(), fmt.Errorf("%s index: index out of range", c.Id())
		}
		return pl.NewValStr(
			string(c.args[idx]),
		), nil
	}

	if key.IsString() {
		if v, ok := c.property(key.String()); ok {
			return v, nil
		}
	}

	return pl.NewValNull(), fmt.Errorf("%s index: invalid index", c.Id())
}

func (c *command) IndexSet(
	_ pl.Val,
	_ pl.Val,
) error {
	return fmt.Errorf("%s index set: unsupported operation", c.Id())
}

func (c *command) Dot(
	name string,
) (pl.Val, error) {
	if v, ok := c.property(name); ok {
		return v, nil
	}
	return pl.NewValNull(), fmt.Errorf("%s dot: unsupported operation", c.Id())
}

func (c *command) DotSet(
	_ string,
	_ pl.Val,
) error {
	return fmt.Errorf("%s dot set: unsupported operation", c.Id())
}

func (c *command) ToString() (string, error) {
	return strings.Join(
		c.StringList(),
		",",
	), nil
}

func (c *command) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(c.StringList())
}

var (
	methodProtoCommandAsString = pl.MustNewFuncProto("redis.command.asString", "%d")
	methodProtoCommandAsInt    = pl.MustNewFuncProto("redis.command.asInt", "%d")
	methodProtoCommandAsReal   = pl.MustNewFuncProto("redis.command.asReal", "%d")
	methodProtoCommandAsBool   = pl.MustNewFuncProto("redis.command.asBool", "%d")

	methodProtoCommandIsBitmap      = pl.MustNewFuncProto("redis.command.isBitmap", "%0")
	methodProtoCommandIsGeneric     = pl.MustNewFuncProto("redis.command.isGeneric", "%0")
	methodProtoCommandIsGeo         = pl.MustNewFuncProto("redis.command.isGeo", "%0")
	methodProtoCommandIsHash        = pl.MustNewFuncProto("redis.command.isHash", "%0")
	methodProtoCommandIsHyperLogLog = pl.MustNewFuncProto("redis.command.isHyperLogLog", "%0")
	methodProtoCommandIsList        = pl.MustNewFuncProto("redis.command.isList", "%0")
	methodProtoCommandIsPubSub      = pl.MustNewFuncProto("redis.command.isPubSub", "%0")
	methodProtoCommandIsScript      = pl.MustNewFuncProto("redis.command.isScript", "%0")
	methodProtoCommandIsSet         = pl.MustNewFuncProto("redis.command.isSet", "%0")
	methodProtoCommandIsSortedSet   = pl.MustNewFuncProto("redis.command.isSortedSet", "%0")
	methodProtoCommandIsString      = pl.MustNewFuncProto("redis.command.isString", "%0")
	methodProtoCommandIsStream      = pl.MustNewFuncProto("redis.command.isStream", "%0")
	methodProtoCommandIsTransaction = pl.MustNewFuncProto("redis.command.isTransaction", "%0")
	methodProtoCommandIsUnknown     = pl.MustNewFuncProto("redis.command.isUnknown", "%0")
)

func (c *command) toindex(name string, a pl.Val) (int, error) {
	v, err := a.ToIndex()
	if err != nil {
		return -1, fmt.Errorf("%s method %s: invalid index %s", c.Id(), name, err.Error())
	}
	if v >= c.ArgumentSize() {
		return -1, fmt.Errorf("%s method %s: index out of range", c.Id(), name)
	}
	return v, nil
}

func (c *command) Method(name string, arg []pl.Val) (pl.Val, error) {
	switch name {
	case "isBitmap":
		if _, err := methodProtoCommandIsBitmap.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsBitmap(c.Name()),
		), nil

	case "isGeneric":
		if _, err := methodProtoCommandIsGeneric.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsGeneric(c.Name()),
		), nil

	case "isGeo":
		if _, err := methodProtoCommandIsGeo.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsGeo(c.Name()),
		), nil

	case "isHash":
		if _, err := methodProtoCommandIsHash.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsHash(c.Name()),
		), nil

	case "isHyperLogLog":
		if _, err := methodProtoCommandIsHyperLogLog.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsHyperLogLog(c.Name()),
		), nil

	case "isList":
		if _, err := methodProtoCommandIsList.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsList(c.Name()),
		), nil

	case "isPubSub":
		if _, err := methodProtoCommandIsPubSub.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsPubSub(c.Name()),
		), nil

	case "isScript":
		if _, err := methodProtoCommandIsScript.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsScript(c.Name()),
		), nil

	case "isSet":
		if _, err := methodProtoCommandIsSet.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsSet(c.Name()),
		), nil

	case "isSortedSet":
		if _, err := methodProtoCommandIsSortedSet.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsSortedSet(c.Name()),
		), nil

	case "isString":
		if _, err := methodProtoCommandIsString.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsString(c.Name()),
		), nil

	case "isStream":
		if _, err := methodProtoCommandIsStream.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsStream(c.Name()),
		), nil

	case "isTransaction":
		if _, err := methodProtoCommandIsTransaction.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsTransaction(c.Name()),
		), nil

	case "isUnknown":
		if _, err := methodProtoCommandIsUnknown.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValBool(
			util.CommandIsUnknown(c.Name()),
		), nil

	case "asString":
		if _, err := methodProtoCommandAsString.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		idx, err := c.toindex("asString", arg[0])
		if err != nil {
			return pl.NewValNull(), err
		}
		return pl.NewValStr(string(c.args[idx])), nil

	case "asInt":
		if _, err := methodProtoCommandAsInt.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		idx, err := c.toindex("asInt", arg[0])
		if err != nil {
			return pl.NewValNull(), err
		}
		ival, err := strconv.ParseInt(string(c.args[idx]), 10, 64)
		if err != nil {
			return pl.NewValNull(), fmt.Errorf("%s method asInt: cannot convert to int", c.Id())
		}
		return pl.NewValInt64(ival), nil

	case "asReal":
		if _, err := methodProtoCommandAsReal.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		idx, err := c.toindex("asReal", arg[0])
		if err != nil {
			return pl.NewValNull(), err
		}
		rval, err := strconv.ParseFloat(string(c.args[idx]), 64)
		if err != nil {
			return pl.NewValNull(), fmt.Errorf("%s method asInt: cannot convert to real", c.Id())
		}
		return pl.NewValReal(rval), nil

	case "asBool":
		if _, err := methodProtoCommandAsBool.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		idx, err := c.toindex("asBool", arg[0])
		if err != nil {
			return pl.NewValNull(), err
		}

		argval := string(c.args[idx])

		if argval == "true" {
			return pl.NewValBool(true), nil
		} else if argval == "false" {
			return pl.NewValBool(false), nil
		} else {
			return pl.NewValNull(), fmt.Errorf("%s method asBool: cannot convert %s to bool", c.Id(), argval)
		}

	default:
		break
	}

	return pl.NewValNull(), fmt.Errorf("%s method %s: unknown method", c.Id(), name)
}

func (c *command) Info() string {
	return c.Id()
}

func (c *command) ToNative() interface{} {
	return c
}

func (c *command) Id() string {
	return CommandTypeId
}

func (c *command) IsThreadSafe() bool {
	return true
}

type commanditer struct {
	c      *command
	cursor int
}

func (c *commanditer) Has() bool {
	return c.cursor < c.c.ArgumentSize()
}

func (c *commanditer) Next() (bool, error) {
	c.cursor++
	return c.Has(), nil
}

func (c *commanditer) SetUp(_ *pl.Evaluator, _ []pl.Val) error {
	return nil
}

func (c *commanditer) Deref() (pl.Val, pl.Val, error) {
	if !c.Has() {
		return pl.NewValNull(), pl.NewValNull(), fmt.Errorf("iterator out of bound")
	}
	return pl.NewValInt(c.cursor), pl.NewValStr(string(c.c.At(c.cursor))), nil
}

func (c *command) NewIterator() (pl.Iter, error) {
	return &commanditer{
		c:      c,
		cursor: 0,
	}, nil
}

func newCommand(raw *redcon.Command) *command {
	return &command{
		args: raw.Args[1:],
		name: strings.ToUpper(string(raw.Args[0])),
	}
}

func NewCommandVal(raw *redcon.Command) pl.Val {
	return pl.NewValUsr(newCommand(raw))
}
