package pl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type ausertype struct {
	tname string
}

func (a *ausertype) Id(_ interface{}) string {
	return a.tname
}

func newUsr(n string) Val {
	x := &ausertype{
		tname: n,
	}
	return NewValUVal(
		x,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		x.Id,
		nil,
		nil,
		nil,
	)
}

func TestParse0(t *testing.T) {
	assert := assert.New(t)

	// empty argument
	{
		a, err := NewFuncProto("a", "")
		assert.True(err == nil)
		assert.True(a.noarg)
	}
	{
		a, err := NewFuncProto("a", "%0")
		assert.True(err == nil)
		assert.True(a.noarg)
		assert.True(a.Name == "a")
		assert.True(a.Descriptor == "%0")
	}

	// always pass
	{
		a, err := NewFuncProto("a", "%-")
		assert.True(err == nil)
		assert.True(a.alwayspass)
	}

	// basic types
	testSingle := func(a *FuncProto, t int, n string) {
		assert.True(len(a.d) == 1)
		assert.False(a.d[0].varlen)
		assert.True(len(a.d[0].d) == 1)
		assert.True(len(a.d[0].d[0].or) == 1)
		assert.True(a.d[0].d[0].or[0].opcode == opc(t))
		assert.True(a.d[0].d[0].or[0].tname == n)
	}

	{
		a, err := NewFuncProto("a", "%d")
		assert.True(err == nil)
		testSingle(a, PInt, "")
	}

	{
		a, err := NewFuncProto("a", "%u")
		assert.True(err == nil)
		testSingle(a, PUInt, "")
	}

	{
		a, err := NewFuncProto("a", "%f")
		assert.True(err == nil)
		testSingle(a, PReal, "")
	}

	{
		a, err := NewFuncProto("a", "%F")
		assert.True(err == nil)
		testSingle(a, PUReal, "")
	}

	{
		a, err := NewFuncProto("a", "%s")
		assert.True(err == nil)
		testSingle(a, PString, "")
	}

	{
		a, err := NewFuncProto("a", "%S")
		assert.True(err == nil)
		testSingle(a, PNEString, "")
	}

	{
		a, err := NewFuncProto("a", "%b")
		assert.True(err == nil)
		testSingle(a, PBool, "")
	}

	{
		a, err := NewFuncProto("a", "%Y")
		assert.True(err == nil)
		testSingle(a, PTrue, "")
	}

	{
		a, err := NewFuncProto("a", "%N")
		assert.True(err == nil)
		testSingle(a, PFalse, "")
	}

	{
		a, err := NewFuncProto("a", "%n")
		assert.True(err == nil)
		testSingle(a, PNull, "")
	}

	{
		a, err := NewFuncProto("a", "%m")
		assert.True(err == nil)
		testSingle(a, PMap, "")
	}

	{
		a, err := NewFuncProto("a", "%l")
		assert.True(err == nil)
		testSingle(a, PList, "")
	}

	{
		a, err := NewFuncProto("a", "%p")
		assert.True(err == nil)
		testSingle(a, PPair, "")
	}

	{
		a, err := NewFuncProto("a", "%r")
		assert.True(err == nil)
		testSingle(a, PRegexp, "")
	}

	{
		a, err := NewFuncProto("a", "%a")
		assert.True(err == nil)
		testSingle(a, PAny, "")
	}

	{
		a, err := NewFuncProto("a", "%U")
		assert.True(err == nil)
		testSingle(a, PUsr, "")
	}
}

func TestParse1(t *testing.T) {
	assert := assert.New(t)

	testSingle := func(a *FuncProto, t int, n string, varlen bool) {
		assert.True(len(a.d) == 1)
		assert.True(a.d[0].varlen == varlen)
		assert.True(len(a.d[0].d) == 1)
		assert.True(len(a.d[0].d[0].or) == 1)
		assert.True(a.d[0].d[0].or[0].opcode == opc(t))
		assert.True(a.d[0].d[0].or[0].tname == n)
	}

	// parsing dangling decorators
	// %p*, %U[a.name]

	{
		a, err := NewFuncProto("a", "%U[]")
		assert.True(err == nil)
		testSingle(a, PUsr, "", false)
	}

	{
		a, err := NewFuncProto("a", "%U['']")
		assert.True(err == nil)
		testSingle(a, PUsr, "", false)
	}

	{
		a, err := NewFuncProto("a", "%U['a.bc.d\\\\\\'']")
		assert.True(err == nil)
		testSingle(a, PUsr, "a.bc.d\\'", false)
	}

	{
		a, err := NewFuncProto("a", "%d*")
		assert.True(err == nil)
		testSingle(a, PInt, "", true)
	}

	{
		a, err := NewFuncProto("a", "%U*")
		assert.True(err == nil)
		testSingle(a, PUsr, "", true)
	}

	{
		a, err := NewFuncProto("a", "%U[]*")
		assert.True(err == nil)
		testSingle(a, PUsr, "", true)
	}

	{
		a, err := NewFuncProto("a", "%U[\"\"]*")
		assert.True(err == nil)
		testSingle(a, PUsr, "", true)
	}

	{
		a, err := NewFuncProto("a", "%U[\"aaa\"]*")
		assert.True(err == nil)
		testSingle(a, PUsr, "aaa", true)
	}

}

func TestParseGroup(t *testing.T) {
	assert := assert.New(t)
	{
		a, err := NewFuncProto("a", "{%d}{%f%s}")
		assert.True(err == nil)
		assert.True(len(a.d) == 2)
		{
			d0 := a.d[0]
			assert.False(d0.varlen)
			assert.True(len(d0.d) == 1)
			assert.True(len(d0.d[0].or) == 1)
			assert.True(d0.d[0].or[0].opcode == PInt)
		}

		{
			d1 := a.d[1]
			assert.False(d1.varlen)
			assert.True(len(d1.d) == 2)

			assert.True(len(d1.d[0].or) == 1)
			assert.True(d1.d[0].or[0].opcode == PReal)

			assert.True(len(d1.d[1].or) == 1)
			assert.True(d1.d[1].or[0].opcode == PString)
		}
	}
}

func TestParseOr(t *testing.T) {
	assert := assert.New(t)
	{
		a, err := NewFuncProto("a", "(%d|%U['aa']|%f)")
		assert.True(err == nil)
		assert.True(len(a.d) == 1)
		assert.False(a.d[0].varlen)
		assert.True(len(a.d[0].d) == 1)
		or := a.d[0].d[0].or

		assert.True(len(or) == 3)
		assert.True(or[0].opcode == PInt)

		assert.True(or[1].opcode == PUsr)
		assert.True(or[1].tname == "aa")

		assert.True(or[2].opcode == PReal)
	}

	{
		_, err := NewFuncProto("a", "(%d|%U['aa']|%f*)")
		assert.False(err == nil)
	}
}

func TestCheck0(t *testing.T) {
	assert := assert.New(t)
	{
		a, err := NewFuncProto("a", "")
		assert.True(err == nil)
		alen, err := a.Check(nil)
		assert.True(err == nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "%0")
		assert.True(err == nil)
		alen, err := a.Check(nil)
		assert.True(err == nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "%-")
		assert.True(err == nil)
		alen, err := a.Check(nil)
		assert.True(err == nil)
		assert.True(alen == 0)

		alen, err = a.Check([]Val{NewValInt(-1)})
		assert.True(err == nil)
		assert.True(alen == 1)
	}

	{
		a, err := NewFuncProto("a", "%d")
		assert.True(err == nil)
		alen, err := a.Check([]Val{NewValInt(1)})
		assert.True(err == nil)
		assert.True(alen == 1)

		alen, err = a.Check([]Val{NewValReal(1.0)})
		assert.True(err != nil)
		assert.True(alen == 1)

		alen, err = a.Check([]Val{})
		assert.True(err != nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "%d%s")
		assert.True(err == nil)
		alen, err := a.Check([]Val{NewValInt(1), NewValStr("")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValReal(1.0), NewValStr("")})
		assert.True(err != nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{})
		assert.True(err != nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "(%d|%f)%s")
		assert.True(err == nil)
		alen, err := a.Check([]Val{NewValInt(1), NewValStr("")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValReal(1.0), NewValStr("")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValNull(), NewValStr("")})
		assert.True(err != nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{})
		assert.True(err != nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "{(%d|%f|%n)%s}{%b}")
		assert.True(err == nil)
		alen, err := a.Check([]Val{NewValInt(1), NewValStr("")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValReal(1.0), NewValStr("")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValNull(), NewValStr("")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValBool(true)})
		assert.True(err == nil)
		assert.True(alen == 1)

		alen, err = a.Check([]Val{})
		assert.True(err != nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "%d*")
		assert.True(err == nil)
		alen, err := a.Check([]Val{NewValInt(1)})
		assert.True(err == nil)
		assert.True(alen == 1)

		alen, err = a.Check([]Val{NewValInt(1), NewValInt(2)})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{NewValInt(1), NewValInt(2), NewValInt(3)})
		assert.True(err == nil)
		assert.True(alen == 3)

		alen, err = a.Check([]Val{NewValReal(1.0)})
		assert.True(err != nil)
		assert.True(alen == 1)

		alen, err = a.Check([]Val{})
		assert.True(err != nil)
		assert.True(alen == 0)
	}

	{
		a, err := NewFuncProto("a", "%U['atype']*")
		assert.True(err == nil)

		alen, err := a.Check([]Val{newUsr("atype")})
		assert.True(err == nil)
		assert.True(alen == 1)

		alen, err = a.Check([]Val{newUsr("atype"), newUsr("atype")})
		assert.True(err == nil)
		assert.True(alen == 2)

		alen, err = a.Check([]Val{newUsr("atype"), newUsr("atype"), newUsr("atype")})
		assert.True(err == nil)
		assert.True(alen == 3)

		alen, err = a.Check([]Val{newUsr("btype")})
		assert.True(err != nil)
		assert.True(alen == 1)
	}
}
