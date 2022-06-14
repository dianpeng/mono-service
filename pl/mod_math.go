package pl

import (
	"math"
)

func init() {
	addrefMF(
		"math",
		"abs",
		"",
		"%f",
		math.Abs,
	)

	addrefMF(
		"math",
		"is_inf",
		"",
		"%f",
		func(v float64) bool {
			return math.IsInf(v, 0)
		},
	)

	addrefMF(
		"math",
		"is_nan",
		"",
		"%f",
		math.IsNaN,
	)

	addrefMF(
		"math",
		"max",
		"",
		"%f%f",
		math.Max,
	)

	addrefMF(
		"math",
		"min",
		"",
		"%f%f",
		math.Min,
	)

	addrefMF(
		"math",
		"mod",
		"",
		"%f%f",
		math.Mod,
	)

	addrefMF(
		"math",
		"modf",
		"",
		"%f",
		math.Modf,
	)

	addrefMF(
		"math",
		"pow",
		"",
		"%f%f",
		math.Pow,
	)

	addrefMF(
		"math",
		"pow10",
		"",
		"%f",
		math.Pow10,
	)
}
