package util

// Here we listed all known redis group/category which can be used to help user
// organize its code in such a way

const (
	RedisCommandBitmap = iota
	RedisCommandGeneric
	RedisCommandGeo
	RedisCommandHash
	RedisCommandHyperLogLog
	RedisCommandList
	RedisCommandPubSub
	RedisCommandScript
	RedisCommandSet
	RedisCommandSortedSet
	RedisCommandStream
	RedisCommandString
	RedisCommandTransaction

	RedisCommandUnknown
)

type CommandInfo struct {
}

type commandInfoMap map[string]CommandInfo

type commandInfoMapStruct struct {
	infoMap commandInfoMap
	tag     int
}

var commandInfoMapList []commandInfoMapStruct

var redisCommandBitmap = commandInfoMap{
	"BITCOUNT":    CommandInfo{},
	"BITFIELD":    CommandInfo{},
	"BITFIELD_RO": CommandInfo{},
	"BITOP":       CommandInfo{},
	"BITOPS":      CommandInfo{},
	"GETBIT":      CommandInfo{},
	"SETBIT":      CommandInfo{},
}

var redisCommandGeneric = commandInfoMap{
	"COPY":        CommandInfo{},
	"DEL":         CommandInfo{},
	"DUMP":        CommandInfo{},
	"EXISTS":      CommandInfo{},
	"EXPIRE":      CommandInfo{},
	"EXPIREAT":    CommandInfo{},
	"EXPIRETIME":  CommandInfo{},
	"KEYS":        CommandInfo{},
	"MIGRATE":     CommandInfo{},
	"MOVE":        CommandInfo{},
	"OBJECT":      CommandInfo{},
	"PERSISTENT":  CommandInfo{},
	"PEXPIRE":     CommandInfo{},
	"PEXPIREAT":   CommandInfo{},
	"PEXPIRETIME": CommandInfo{},
	"PTTL":        CommandInfo{},
	"RANDOMKEY":   CommandInfo{},
	"RENAME":      CommandInfo{},
	"RENAMENX":    CommandInfo{},
	"RESTORE":     CommandInfo{},
	"SCAN":        CommandInfo{},
	"SORT":        CommandInfo{},
	"SORT_RO":     CommandInfo{},
	"TOUCH":       CommandInfo{},
	"TTL":         CommandInfo{},
	"TYPE":        CommandInfo{},
	"UNLINK":      CommandInfo{},
	"WAIT":        CommandInfo{},
}

var redisCommandGeo = commandInfoMap{
	"GEOADD":               CommandInfo{},
	"GEODIST":              CommandInfo{},
	"GEOHASH":              CommandInfo{},
	"GEOPOS":               CommandInfo{},
	"GEORADIUS":            CommandInfo{},
	"GEORADIUS_RO":         CommandInfo{},
	"GEORADIUSBYMEMBER":    CommandInfo{},
	"GEORADIUSBYMEMBER_RO": CommandInfo{},
	"GEOSERACH":            CommandInfo{},
	"GEOSEARCHSTORE":       CommandInfo{},
}

var redisCommandHash = commandInfoMap{

	"HDEL":         CommandInfo{},
	"HEXISTS":      CommandInfo{},
	"HGET":         CommandInfo{},
	"HGETALL":      CommandInfo{},
	"HINCRBY":      CommandInfo{},
	"HINCRBYFLOAT": CommandInfo{},
	"HKEYS":        CommandInfo{},
	"HLEN":         CommandInfo{},
	"HMGET":        CommandInfo{},
	"HMSET":        CommandInfo{},
	"HRANDFIELD":   CommandInfo{},
	"HSCAN":        CommandInfo{},
	"HSET":         CommandInfo{},
	"HSETNX":       CommandInfo{},
	"HSTRLEN":      CommandInfo{},
	"HVALS":        CommandInfo{},
}

var redisCommandHyperLogLog = commandInfoMap{
	"PFADD":      CommandInfo{},
	"PFCOUNT":    CommandInfo{},
	"PFDEBUG":    CommandInfo{},
	"PFMERGE":    CommandInfo{},
	"PFSELFTEST": CommandInfo{},
}

var redisCommandList = commandInfoMap{
	"BLMOVE":     CommandInfo{},
	"BLMPOP":     CommandInfo{},
	"BLPOP":      CommandInfo{},
	"BRPOP":      CommandInfo{},
	"BRPOPFLUSH": CommandInfo{},
	"LINDEX":     CommandInfo{},
	"LINSERT":    CommandInfo{},
	"LLEN":       CommandInfo{},
	"LMOVE":      CommandInfo{},
	"LMPOP":      CommandInfo{},
	"LPOP":       CommandInfo{},
	"LPOS":       CommandInfo{},
	"LPUSH":      CommandInfo{},
	"LPUSHX":     CommandInfo{},
	"LRANGE":     CommandInfo{},
	"LREM":       CommandInfo{},
	"LSET":       CommandInfo{},
	"LTRIM":      CommandInfo{},
	"RPOP":       CommandInfo{},
	"RPOPLPUSH":  CommandInfo{},
	"RPUSH":      CommandInfo{},
	"RPUSHX":     CommandInfo{},
}

var redisCommandPubSub = commandInfoMap{
	"PSUBSCRIBE":   CommandInfo{},
	"PUBLISH":      CommandInfo{},
	"PUBSUB":       CommandInfo{},
	"PUNSUBSCRIBE": CommandInfo{},
	"SPUBLISH":     CommandInfo{},
	"SSUBSCRIBE":   CommandInfo{},
	"SUBSCRIBE":    CommandInfo{},
	"SUNSUBSCRIBE": CommandInfo{},
	"UNSUBSCRIBE":  CommandInfo{},
}

var redisCommandScript = commandInfoMap{
	"EVAL":       CommandInfo{},
	"EVAL_RO":    CommandInfo{},
	"EVALSHA":    CommandInfo{},
	"EVALSHA_RO": CommandInfo{},
	"FCALL":      CommandInfo{},
	"FCALL_RO":   CommandInfo{},
	"FUNCTION":   CommandInfo{},
	"SCRIPT":     CommandInfo{},
}

var redisCommandSet = commandInfoMap{
	"SADD":        CommandInfo{},
	"SCARD":       CommandInfo{},
	"SDIFF":       CommandInfo{},
	"SDIFFSTORE":  CommandInfo{},
	"SINTER":      CommandInfo{},
	"SINTERCARD":  CommandInfo{},
	"SINTERSTORE": CommandInfo{},
	"SISMEMBER":   CommandInfo{},
	"SMEMEBERS":   CommandInfo{},
	"SMISMEMBER":  CommandInfo{},
	"SMOVE":       CommandInfo{},
	"SPOP":        CommandInfo{},
	"SRANDMEMBER": CommandInfo{},
	"SREM":        CommandInfo{},
	"SSCAN":       CommandInfo{},
	"SUNION":      CommandInfo{},
	"SUNIONSTORE": CommandInfo{},
}

var redisCommandSortedSet = commandInfoMap{
	"BZMPOP":           CommandInfo{},
	"BZPOPMAX":         CommandInfo{},
	"BZPOPMIN":         CommandInfo{},
	"ZADD":             CommandInfo{},
	"ZCARD":            CommandInfo{},
	"ZCOUNT":           CommandInfo{},
	"ZDIFF":            CommandInfo{},
	"ZDIFFSTORE":       CommandInfo{},
	"ZINCRBY":          CommandInfo{},
	"ZINTER":           CommandInfo{},
	"ZINTERCARD":       CommandInfo{},
	"ZINTERSTORE":      CommandInfo{},
	"ZLEXCOUNT":        CommandInfo{},
	"ZMPOP":            CommandInfo{},
	"ZMSCORE":          CommandInfo{},
	"ZPOPMAX":          CommandInfo{},
	"ZPOPMIN":          CommandInfo{},
	"ZRANDMEMBER":      CommandInfo{},
	"ZRANGE":           CommandInfo{},
	"ZRANGEBYLEX":      CommandInfo{},
	"ZRANGEBYSCORE":    CommandInfo{},
	"ZRANGESTORE":      CommandInfo{},
	"ZRANK":            CommandInfo{},
	"ZREM":             CommandInfo{},
	"ZREMRANGEBYLEX":   CommandInfo{},
	"ZREMRANGEBYRANK":  CommandInfo{},
	"ZREMRANGEBYSCORE": CommandInfo{},
	"ZREVRANGE":        CommandInfo{},
	"ZREVRANGEBYLEX":   CommandInfo{},
	"ZREVRANGEBYSCORE": CommandInfo{},
	"ZREVRANK":         CommandInfo{},
	"ZSCAN":            CommandInfo{},
	"ZSCORE":           CommandInfo{},
	"ZUNION":           CommandInfo{},
	"ZUNIONSTORE":      CommandInfo{},
}

var redisCommandStream = commandInfoMap{
	"XACK":       CommandInfo{},
	"XADD":       CommandInfo{},
	"XAUTOCLAIM": CommandInfo{},
	"XCLAIM":     CommandInfo{},
	"XDEL":       CommandInfo{},
	"XGROUP":     CommandInfo{},
	"XINFO":      CommandInfo{},
	"XLEN":       CommandInfo{},
	"XPENDING":   CommandInfo{},
	"XRANGE":     CommandInfo{},
	"XREAD":      CommandInfo{},
	"XREADGROUP": CommandInfo{},
	"XREVRANGE":  CommandInfo{},
	"XSETID":     CommandInfo{},
	"XTRIM":      CommandInfo{},
}

var redisCommandString = commandInfoMap{
	"APPEND":      CommandInfo{},
	"DECR":        CommandInfo{},
	"DECRBY":      CommandInfo{},
	"GET":         CommandInfo{},
	"GETDEL":      CommandInfo{},
	"GETEX":       CommandInfo{},
	"GETRANGE":    CommandInfo{},
	"GETSET":      CommandInfo{},
	"INCR":        CommandInfo{},
	"INCRBY":      CommandInfo{},
	"INCRBYFLOAT": CommandInfo{},
	"LCS":         CommandInfo{},
	"MGET":        CommandInfo{},
	"MSET":        CommandInfo{},
	"MSETNX":      CommandInfo{},
	"PSETEX":      CommandInfo{},
	"SET":         CommandInfo{},
	"SETEX":       CommandInfo{},
	"SETNX":       CommandInfo{},
	"SETRANGE":    CommandInfo{},
	"STRLEN":      CommandInfo{},
	"SUBSTR":      CommandInfo{},
}

var redisCommandTransaction = commandInfoMap{
	"DISCARD": CommandInfo{},
	"EXEC":    CommandInfo{},
	"MULTI":   CommandInfo{},
	"UNWATCH": CommandInfo{},
	"WATCH":   CommandInfo{},
}

func init() {
	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandBitmap,
			tag:     RedisCommandBitmap,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandGeneric,
			tag:     RedisCommandGeneric,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandGeo,
			tag:     RedisCommandGeo,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandHash,
			tag:     RedisCommandHash,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandHyperLogLog,
			tag:     RedisCommandHyperLogLog,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandList,
			tag:     RedisCommandList,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandPubSub,
			tag:     RedisCommandPubSub,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandScript,
			tag:     RedisCommandScript,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandSet,
			tag:     RedisCommandSet,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandSortedSet,
			tag:     RedisCommandSortedSet,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandString,
			tag:     RedisCommandString,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandStream,
			tag:     RedisCommandStream,
		},
	)

	commandInfoMapList = append(
		commandInfoMapList,
		commandInfoMapStruct{
			infoMap: redisCommandTransaction,
			tag:     RedisCommandTransaction,
		},
	)
}

func CommandCategory(name string) int {
	for _, x := range commandInfoMapList {
		_, has := x.infoMap[name]
		if has {
			return x.tag
		}
	}
	return RedisCommandUnknown
}

func CommandCategoryName(name string) string {
	return RedisCommandTypeName(CommandCategory(name))
}

func CommandIsBitmap(name string) bool {
	return CommandCategory(name) == RedisCommandBitmap
}

func CommandIsGeneric(name string) bool {
	return CommandCategory(name) == RedisCommandGeneric
}

func CommandIsGeo(name string) bool {
	return CommandCategory(name) == RedisCommandGeo
}

func CommandIsHash(name string) bool {
	return CommandCategory(name) == RedisCommandHash
}

func CommandIsHyperLogLog(name string) bool {
	return CommandCategory(name) == RedisCommandHyperLogLog
}

func CommandIsList(name string) bool {
	return CommandCategory(name) == RedisCommandList
}

func CommandIsPubSub(name string) bool {
	return CommandCategory(name) == RedisCommandPubSub
}

func CommandIsScript(name string) bool {
	return CommandCategory(name) == RedisCommandScript
}

func CommandIsSet(name string) bool {
	return CommandCategory(name) == RedisCommandSet
}

func CommandIsSortedSet(name string) bool {
	return CommandCategory(name) == RedisCommandSortedSet
}

func CommandIsString(name string) bool {
	return CommandCategory(name) == RedisCommandString
}

func CommandIsStream(name string) bool {
	return CommandCategory(name) == RedisCommandStream
}

func CommandIsTransaction(name string) bool {
	return CommandCategory(name) == RedisCommandTransaction
}

func CommandIsUnknown(name string) bool {
	return CommandCategory(name) == RedisCommandTransaction
}

func RedisCommandTypeName(t int) string {
	switch t {
	case RedisCommandBitmap:
		return "bitmap"
	case RedisCommandGeneric:
		return "generic"
	case RedisCommandGeo:
		return "geo"
	case RedisCommandHash:
		return "hash"
	case RedisCommandHyperLogLog:
		return "hyper_log_log"
	case RedisCommandList:
		return "list"
	case RedisCommandPubSub:
		return "pubsub"
	case RedisCommandScript:
		return "script"
	case RedisCommandSet:
		return "set"
	case RedisCommandSortedSet:
		return "sorted_set"
	case RedisCommandStream:
		return "stream"
	case RedisCommandString:
		return "string"
	case RedisCommandTransaction:
		return "transaction"
	default:
		return "unknown"
	}
}
