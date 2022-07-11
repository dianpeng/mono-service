package hpl

import (
	"fmt"
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/pl"
)

type accesslog struct {
	l        *alog.Log
	appendix pl.Val
}

func newAccessLog(
	l *alog.Log,
) *accesslog {
	return &accesslog{
		l: l,
		appendix: pl.NewValGoStrList(
			&l.Appendix,
		),
	}
}

func ValIsAccessLog(
	v pl.Val,
) bool {
	return v.Id() == ".access_log"
}

func (l *accesslog) Index(key pl.Val) (pl.Val, error) {
	if !key.IsString() {
		return pl.NewValNull(), fmt.Errorf("%s index: key type is invalid", l.Id())
	}

	switch key.String() {
	case "format":
		return pl.NewValStr(l.l.Format.Raw), nil
	case "appendix":
		return l.appendix, nil
	default:
		return pl.NewValNull(),
			fmt.Errorf("%s index: key %s is unknown", l.Id(), key.String())
	}
}

func (l *accesslog) Dot(key string) (pl.Val, error) {
	return l.Index(pl.NewValStr(key))
}

func (l *accesslog) IndexSet(key pl.Val, val pl.Val) error {
	return fmt.Errorf("%s does not support index set", l.Id())
}

func (l *accesslog) DotSet(key string, val pl.Val) error {
	return fmt.Errorf("%s does not support dot set", l.Id())
}

func (l *accesslog) ToString() (string, error) {
	return l.Info(), nil
}

func (l *accesslog) ToJSON() (pl.Val, error) {
	return pl.MarshalVal(
		map[string]interface{}{
			"format":   l.l.Format.Raw,
			"appendix": l.l.Appendix,
		},
	)
}

func (l *accesslog) Method(name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("%d's method %s is unknown", l.Id(), name)
}

func (l *accesslog) Info() string {
	return l.Id()
}

func (l *accesslog) ToNative() interface{} {
	return l.l
}

func (l *accesslog) Id() string {
	return AccessLogTypeId
}

func (l *accesslog) IsImmutable() bool {
	return false
}

func (l *accesslog) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf("%s does not support iterator", l.Id())
}

func NewAccessLogVal(
	l *alog.Log,
) pl.Val {
	return pl.NewValUsr(
		newAccessLog(l),
	)
}
