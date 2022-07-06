package pl

import (
	"github.com/dianpeng/mono-service/util"
	"github.com/google/uuid"
	"math/rand"
)

func init() {
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
		util.RandomString,
	)

	addrefMF(
		"rand",
		"uuid",
		"",
		"%0",
		uuid.NewString,
	)
}
