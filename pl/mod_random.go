package pl

import (
	"bytes"
	"math/rand"
)

var randomstringlettertable = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_")

func initModRandom() {
	addrefMF(
		"rand",
		"real",
		"",
		"%0",
		rand.Float64,
	)

	addrefMF(
		"rand",
		"int63",
		"",
		"%0",
		rand.Int63,
	)

	addrefMF(
		"rand",
		"str",
		"",
		"%d",
		func(n int) string {
			sz := len(randomstringlettertable)
			b := new(bytes.Buffer)
			for i := 0; i < n; i++ {
				idx := rand.Intn(sz)
				b.WriteRune(randomstringlettertable[idx])
			}
			return b.String()
		},
	)
}
