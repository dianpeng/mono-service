package pl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func runWithResult(code string) (Val, bool, *Module) {
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
		fmt.Printf(":module %s", err.Error())
		return NewValNull(), false, nil
	}

	err = eval.EvalSession(module)
	if err != nil {
		fmt.Printf(":evalSession %s", err.Error())
		return NewValNull(), false, module
	}

	_, err = eval.Eval("test", module)
	if err != nil {
		fmt.Printf(":eval %s", err.Error())
		return NewValNull(), false, module
	}
	return *ret, true, module
}

func Test1(t *testing.T) {
	assert := assert.New(t)

	{
		r, ok, module := runWithResult(`

  fn foo(d) {
    let a = "hello world";
    let b = 10;
    let c = 20;

    return fn() {
      return fn() {
        return (d + b + c):to_string() + a;
      };
    };
  }

  test {
    output => foo(10)()();
  }`)

		assert.True(ok)
		assert.True(r.IsString())
		assert.True(r.String() == "40hello world")

		// now checking the module
		assert.Equal(2, module.GetAnonymousFunctionSize(), "anonymous function size")
		assert.Equal(3, len(module.fn), "function size")

		// finding out the first anonymous function which should have 4 upvalues,
		// though it never uses them
		{
			p := module.getFunction("[anonymous_function_1]")
			assert.True(p != nil)
			assert.True(len(p.upvalue) == 4)
			for _, x := range p.upvalue {
				assert.True(x.onStack)
			}
		}

		{
			p := module.getFunction("[anonymous_function_2]")
			assert.True(p != nil)
			assert.True(len(p.upvalue) == 4)
			for _, x := range p.upvalue {
				assert.False(x.onStack)
			}
		}
	}
}
