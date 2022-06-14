package pl

import (
	"time"
)

func init() {
	addrefMF(
		"time",
		"unix",
		"",
		"%0",
		func() int64 {
			return time.Now().Unix()
		},
	)

	addrefMF(
		"time",
		"unix_milli",
		"",
		"%0",
		func() int64 {
			return time.Now().UnixMilli()
		},
	)

	addrefMF(
		"time",
		"unix_micro",
		"",
		"%0",
		func() int64 {
			return time.Now().UnixMicro()
		},
	)

	addrefMF(
		"time",
		"unix_nano",
		"",
		"%0",
		func() int64 {
			return time.Now().UnixNano()
		},
	)

	addrefMF(
		"time",
		"now_format",
		"",
		"%s",
		func(format string) string {
			return time.Now().Format(format)
		},
	)

	addrefMF(
		"time",
		"http_date",
		"",
		"%0",
		func() string {
			return time.Now().Format(time.RFC3339)
		},
	)

	addrefMF(
		"time",
		"http_datenano",
		"",
		"%0",
		func() string {
			return time.Now().Format(time.RFC3339Nano)
		},
	)
}
