package pl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// a simple testing frameworking which is designed for testing the basic
// expression of the module. It assumes user will write output => as return
// action
func testEvalReturn(code string) (Val, bool) {
	rr := NewValNull()
	ret := &rr
	eval := NewEvaluatorWithContextCallback(
		func(_ *Evaluator, vname string) (Val, error) {
			if vname == "a_int" {
				return NewValInt(1), nil
			}
			if vname == "a_str" {
				return NewValStr("hello"), nil
			}
			if vname == "a_real" {
				return NewValReal(1.0), nil
			}
			return NewValNull(), fmt.Errorf("%s unknown var", vname)
		},
		nil,
		func(_ *Evaluator, aname string, aval Val) error {
			if aname == "output" {
				*ret = aval
			}
			return nil
		})

	module, err := CompileModule(code, nil)

	// fmt.Printf(":code\n%s", module.Dump())

	if err != nil {
		fmt.Printf(":module \n%s", err.Error())
		return NewValNull(), false
	}

	err = eval.EvalSession(module)
	if err != nil {
		fmt.Printf(":evalSession \n%s", err.Error())
		return NewValNull(), false
	}

	v, err := eval.Eval("test", module)
	if err != nil {
		fmt.Printf(":eval \n%s", err.Error())
		return NewValNull(), false
	}

	return v, true
}

func TestRuleReturn1(t *testing.T) {
	assert := assert.New(t)
	{
		v, ok := testEvalReturn(
			`
test {
  return "Hello World";
}
`)
		assert.True(ok)
		assert.True(v.IsString())
		assert.True(v.String() == "Hello World")
	}

	{
		v, ok := testEvalReturn(
			`
test {
  let v = 10;
  let u = 20;
  let z = 50;
  return u+v+z;
}
`)
		assert.True(ok)
		assert.True(v.IsInt())
		assert.True(v.Int() == 80)
	}
}
