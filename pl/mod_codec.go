package pl

import (
	"encoding/base64"
	"net/url"
)

func init() {
	addrefMF(
		"codec",
		"b64_tostring",
		"",
		"%s",
		func(input string) string {
			return base64.StdEncoding.EncodeToString([]byte(input))
		},
	)
	addrefMF(
		"codec",
		"b64_fromstring",
		"",
		"%s",
		func(input string) (string, error) {
			o, err := base64.StdEncoding.DecodeString(input)
			if err != nil {
				return "", err
			} else {
				return string(o), nil
			}
		},
	)

	addrefMF(
		"code",
		"url_encode",
		"",
		"%s",
		url.QueryEscape,
	)
	addrefMF(
		"code",
		"url_decode",
		"",
		"%s",
		url.QueryUnescape,
	)

	addrefMF(
		"code",
		"path_encode",
		"",
		"%s",
		url.PathEscape,
	)
	addrefMF(
		"code",
		"path_decode",
		"",
		"%s",
		url.PathUnescape,
	)
}
