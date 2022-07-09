package hpl

import (
  "fmt"
  "github.com/dianpeng/mono-service/pl"
  "github.com/dianpeng/mono-service/alog"
  "strings"
)

type accesslog struct {
  l *alog.Log
}

func newAccessLog(
  l *alog.Log,
) *accesslog {
  return &accesslog {
    l : l,
  }
}

func ValIsAccessLog(
  v pl.Val,
) bool {
  return v.Id() == ".access_log"
}

func (l *accesslog) Index(key pl.Val) (pl.Val, error) {
  if key.IsString() {
    switch key.String() {
    case "format":
      return pl.NewValStr(l.l.Format.Raw), nil
    default:
      break
    }

    return pl.NewValNull(),
          fmt.Errorf("access_log index: unknown field %s", key.String())
  } else if key.IsInt() {
    idx = key.Int()
    if idx < 0 || idx >= len(l.l.Appendix) {
      return pl.NewValNull(), fmt.Errorf("access_log index: index out of range")
    }
    return pl.NewValStr(l.l.Appendix[idx]), nil
  } else {
    return pl.NewValNull(), fmt.Errorf("access_log index: invalid key type")
  }
}

func (l *accesslog) Dot(key string) (pl.Val, error) {
  return l.Index(pl.NewValStr(key))
}

func (l *accesslog) IndexSet(key pl.Val, val pl.Val) error {
}
