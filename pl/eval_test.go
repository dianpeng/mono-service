package pl

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// a simple testing frameworking which is designed for testing the basic
// expression of the module. It assumes user will write output => as return
// action
func test(code string) (Val, bool) {
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

	_, err = eval.Eval("test", module)
	if err != nil {
		fmt.Printf(":eval\n%s", err.Error())
		return NewValNull(), false
	}
	return *ret, true
}

func testInt64(code string, expect int64) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.Type != ValInt {
		fmt.Printf(":return invalid type %s\n", val.Id())
		return false
	}

	if val.Int() != expect {
		fmt.Printf(":return invalid value %d\n", val.Int())
		return false
	}

	return true
}

func testInt(code string, expect int) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.Type != ValInt {
		fmt.Printf(":return invalid type %s\n", val.Id())
		return false
	}

	if val.Int() != int64(expect) {
		fmt.Printf(":return invalid value %d\n", val.Int())
		return false
	}

	return true
}

func testReal(code string, expect float64) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.Type != ValReal {
		fmt.Printf(":return invalid type %s\n", val.Id())
		return false
	}

	if val.Real() != expect {
		fmt.Printf(":return invalid value %f\n", val.Real())
		return false
	}

	return true
}

func testBool(code string, expect bool) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.Type != ValBool {
		fmt.Printf(":return invalid type %s\n", val.Id())
		return false
	}

	if val.Bool() != expect {
		fmt.Printf(":return invalid value %t\n", val.Bool())
		return false
	}

	return true
}

func testCond(code string, expect bool) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.ToBoolean() != expect {
		fmt.Printf(":return invalid cond %t\n", val.Bool())
		return false
	}

	return true
}

func testNull(code string) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.Type != ValNull {
		fmt.Printf(":return invalid type %s\n", val.Id())
		return false
	}

	return true
}

func testString(code string, expect string) bool {
	val, ok := test(code)
	if !ok {
		return false
	}

	if val.Type != ValStr {
		fmt.Printf(":return invalid type %s\n", val.Id())
		return false
	}

	if val.String() != expect {
		fmt.Printf(":return invalid value %s\n", val.String())
		return false
	}

	return true
}

type actionOutput map[string]Val

func (a *actionOutput) intAt(idx string, val int) bool {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return false
	}

	if x.Type != ValInt {
		fmt.Printf("actionOutput(%s) not int", idx)
		return false
	}

	if int(x.Int()) != val {
		fmt.Printf("actionOutput(%s) unexpected: %d = %d", idx, x.Int(), val)
		return false
	}

	return true
}

func (a *actionOutput) realAt(idx string, val float64) bool {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return false
	}

	if x.Type != ValReal {
		fmt.Printf("actionOutput(%s) not real", idx)
		return false
	}

	if x.Real() != val {
		fmt.Printf("actionOutput(%s) unexpected: %f = %f", idx, x.Real(), val)
		return false
	}

	return true
}

func (a *actionOutput) boolAt(idx string, val bool) bool {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return false
	}

	if x.Type != ValBool {
		fmt.Printf("actionOutput(%s) not bool", idx)
		return false
	}

	if x.Bool() != val {
		fmt.Printf("actionOutput(%s) unexpected(bool)", idx)
		return false
	}

	return true
}

func (a *actionOutput) strAt(idx string, val string) bool {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return false
	}

	if x.Type != ValStr {
		fmt.Printf("actionOutput(%s) not string", idx)
		return false
	}

	if x.String() != val {
		fmt.Printf("actionOutput(%s) unexpected: %s = %s\n", idx, x.String(), val)
		return false
	}

	return true
}

func (a *actionOutput) nullAt(idx string) bool {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return false
	}

	if x.Type != ValNull {
		fmt.Printf("actionOutput(%s) not null", idx)
		return false
	}
	return true
}

func (a *actionOutput) listAt(idx string) *List {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return nil
	}

	if x.Type != ValList {
		fmt.Printf("actionOutput(%s) not list", idx)
		return nil
	}
	return x.List()
}

func (a *actionOutput) mapAt(idx string) *Map {
	x, ok := (*a)[idx]
	if !ok {
		fmt.Printf("actionOutput(%s) not found", idx)
		return nil
	}

	if x.Type != ValMap {
		fmt.Printf("actionOutput(%s) not map", idx)
		return nil
	}
	return x.Map()
}

func TestTryExpr(t *testing.T) {
	assert := assert.New(t)
	assert.True(testString(
		`
	  test{
	    let a = try foo() else "hello world";
	    output => a;
	  }
	  `, "hello world"))

	assert.True(testString(
		`
	  test {
	    let a = try foo() else {
        try bar() else {
          try q() else {
            "hello world";
          };
        };
      };

	    output => a;
	  }
	  `, "hello world"))

	assert.True(testString(
		`
	  test{
	    let a = try foo() else let reason reason;
	    output => a;
	  }
	  `, "foo unknown var"))

	assert.True(testString(
		`
	  test{
      let reason = null;
	    let a = try foo() else let r r;
	    output => a;
	  }
	  `, "foo unknown var"))

	assert.True(testString(
		`
	  fn f0() {
	    return f1();
	  }
	  fn f1() {
	    return f2();
	  }
	  fn f2() {
	    return f3();
	  }
	  fn f3() {
	    return try foo() else "hello world";
	  }
	  test{
	    output => f0();
	  }
	  `, "hello world"))

	assert.True(testString(
		`
  fn f0() {
    return f1();
  }
  fn f1() {
    return f2();
  }
  fn f2() {
    return f3();
  }
  fn f3() {
    return foo();
  }
  test{
    output => try f0() else "hello world";
  }
  `, "hello world"))

	assert.True(testString(
		`
  session {
    xxx = try uuvv() else "hello world";
  }
  test{
    output => xxx;
  }
  `, "hello world"))
}

func TestTryStatement(t *testing.T) {
	assert := assert.New(t)

	// basic
	{
		assert.True(testString(
			`
    test{
      try {
        foo();
      } else {
      }

      output => "hello world";
    }
    `, "hello world"))

		assert.True(testString(
			`
    test{
      try {
        foo();
      } else {
        output => "hello world";
      }
    }
    `, "hello world"))

		assert.True(testString(
			`
    test{
      try {
        foo();
        output => "bar";
      } else {
        output => "hello world";
      }
    }
    `, "hello world"))

		assert.True(testString(
			`
    test{
      try {
        xxx = 100;
        output => "bar";
      } else {
        output => "hello world";
      }
    }
    `, "hello world"))
	}

	// capture the error value
	{
		assert.True(testString(
			`
    test{
      try {
        foo();
      } else let bar {
      }

      output => "hello world";
    }
    `, "hello world"))

		assert.True(testString(
			`
    test{
      try {
        foo();
      } else let bar {
        print(bar);
        output => "hello world";
      }
    }
    `, "hello world"))

		assert.True(testString(
			`
    test{
      try {
        foo();
        output => "bar";
      } else let bar {
        print(bar);
        output => "hello world";
      }
    }
    `, "hello world"))

		assert.True(testString(
			`
    test{
      try {
        xxx = 100;
        output => "bar";
      } else let bar {
        print(bar);
        output => "hello world";
      }
    }
    `, "hello world"))
	}
}

func TestMCall(t *testing.T) {
	assert := assert.New(t)
	{
		assert.True(testString(
			`
    test{
      let a = {};
      a:set("a", 10);
      a:set("b", 20);
      output => (a.a + a.b):to_string();
    }
    `, "30"))
	}
}

func TestVCall(t *testing.T) {
	assert := assert.New(t)

	{
		assert.True(testString(
			`
    test{
      let obj = {
        a : fn() {
          return "hello";
        },
        b : fn() {
          return "world";
        }
      };
      output => obj.a() + " " + obj.b();
    }
    `, "hello world"))
	}

	{
		assert.True(testString(
			`
      test{
        output => (fn() { return 10:to_string(); })();
      }
    `, "10"))
	}
	{
		assert.False(testString(
			`
    test{
      output => 10();
    }
    `, ""))
		assert.False(testString(
			`
    test{
      output => ""();
    }
    `, ""))
		assert.False(testString(
			`
    test{
      output => {}();
    }
    `, ""))
		assert.False(testString(
			`
    test{
      output => []();
    }
    `, ""))
	}
}

func TestUpvalue(t *testing.T) {
	assert := assert.New(t)

	{
		assert.True(testString(
			`
    fn foo() {
      let a = "hello world";
      return fn() {
        return a;
      };
    }
    test{

      output => foo()();
    }
    `, "hello world"))
	}

	// multiple collapsing
	{
		assert.True(testString(
			`
    fn foo() {
      let a = "hello world";
      return fn() {
        return fn() {
          return fn() {
            return a;
          };
        };
      };
    }
    test{

      output => foo()()()();
    }
    `, "hello world"))
	}

	{
		assert.True(testString(
			`
    fn foo(flag) {
      let a = "hello world";
      let b = "Hello World";

      return fn() {
        return fn() {
          if flag {
            return fn() {
              return a;
            };
          } else {
            return b;
          }
        };
      };
    }

    test{
      let lhs = foo(true)()()();
      let rhs = foo(false)()();
      output => lhs + rhs;
    }
    `, "hello worldHello World"))
	}
}

func TestEval1(t *testing.T) {
	assert := assert.New(t)

	// basic action
	{
		output := make(actionOutput)

		eval := NewEvaluatorWithContextCallback(
			func(_ *Evaluator, vname string) (Val, error) {
				if vname == "a" {
					return NewValStr("Hello World"), nil
				}
				if vname == "abs" {
					return NewValNativeFunction(
						"abs",
						func(args []Val) (Val, error) {
							a0 := args[0]
							must(a0.Type == ValInt, "must be int")
							return NewValInt64(-a0.Int()), nil
						},
					), nil
				}
				return NewValNull(), fmt.Errorf("%s unknown var", vname)
			},
			nil,
			func(_ *Evaluator, aname string, aval Val) error {
				output[aname] = aval
				return nil
			})

		module, err := CompileModule(
			`
mm => {
  act_int => 10;
  act_real => 10.0;
  act_true => true;
  act_false => false;
  act_null => null;
  act_list_empty => [];
  act_map_empty => {};
  act_func => abs(-1);
  act_var => a;

  act_map1 => {'a': a};
  act_list1 => [1, true, 3.0, a, false, null, []];
}`, nil)

		if err != nil {
			fmt.Printf(":module %s", err.Error())
		}
		assert.True(err == nil)

		_, err = eval.Eval("mm", module)
		if err != nil {
			fmt.Printf(":eval %s", err.Error())
		}
		assert.True(err == nil)
		assert.Equal(len(output), 11)

		// checking all the action
		assert.True(output.intAt("act_int", 10))
		assert.True(output.realAt("act_real", 10.0))
		assert.True(output.boolAt("act_true", true))
		assert.True(output.boolAt("act_false", false))
		assert.True(output.strAt("act_var", "Hello World"))
		assert.True(output.intAt("act_func", 1))
		{
			l := output.listAt("act_list_empty")
			assert.True(l != nil)
			assert.True(len(l.Data) == 0)
		}
		{
			l := output.mapAt("act_map_empty")
			assert.True(l != nil)
			assert.True(l.Length() == 0)
		}

		{
			l := output.listAt("act_list1")
			assert.True(l != nil)
			assert.Equal(l.Length(), 7, "length")
			{
				e := l.Data[0]
				assert.True(e.Type == ValInt)
				assert.Equal(e.Int(), int64(1), "int")
			}
			{
				e := l.Data[1]
				assert.True(e.Type == ValBool)
				assert.Equal(e.Bool(), true, "bool")
			}
			{
				e := l.Data[2]
				assert.True(e.Type == ValReal)
				assert.Equal(e.Real(), 3.0, "real")
			}
			{
				e := l.Data[3]
				assert.True(e.Type == ValStr)
				assert.Equal(e.String(), "Hello World", "str")
			}
			{
				e := l.Data[5]
				assert.True(e.Type == ValNull)
			}
			{
				e := l.Data[6]
				assert.True(e.Type == ValList)
				assert.True(e.List() != nil)
				assert.True(len(e.List().Data) == 0)
			}
		}
		{
			m := output.mapAt("act_map1")
			assert.Equal(m.Length(), 1)
			v, ok := m.Get("a")
			assert.True(ok)
			assert.Equal(v.Type, ValStr, "map.type")
			assert.Equal(v.String(), "Hello World")
		}
	}

	// list/map comprehension
	{
		output := make(actionOutput)

		eval := NewEvaluatorWithContextCallback(
			func(_ *Evaluator, vname string) (Val, error) {
				if vname == "a" {
					return NewValStr("Hello World"), nil
				}
				if vname == "abs" {
					return NewValNativeFunction(
						"abs",
						func(args []Val) (Val, error) {
							a0 := args[0]
							must(a0.Type == ValInt, "must be int")
							return NewValInt64(-a0.Int()), nil
						},
					), nil
				}
				return NewValNull(), fmt.Errorf("%s unknown var", vname)

			},
			nil,
			func(_ *Evaluator, aname string, aval Val) error {
				output[aname] = aval
				return nil
			})

		module, err := CompileModule(
			`
mm {
  empty_list => [];
  list1 => [1];
  list2 => [1, true];
  list3 => [1, a, [1]];

  empty_map => {};
  map1 => {
    'a' : {}
  };
  map2 => {
    'a' : 1,
    'b' : true
  };
  map3 => {
    'a' : {
      'b' : {
        'c' : 1
      }
    }
  };
}`, nil)

		if err != nil {
			fmt.Printf(":module %s", err.Error())
		}
		assert.True(err == nil)

		_, err = eval.Eval("mm", module)
		if err != nil {
			fmt.Printf(":eval %s", err.Error())
		}
		assert.True(err == nil)

		{
			l := output.listAt("empty_list")
			assert.True(l != nil)
			assert.True(l.Length() == 0)
		}
		{
			l := output.listAt("list1")
			assert.True(l != nil)
			assert.True(l.Length() == 1)
			assert.True(l.Data[0].Type == ValInt)
			assert.True(l.Data[0].Int() == 1)
		}
		{
			l := output.listAt("list2")
			assert.True(l != nil)
			assert.True(len(l.Data) == 2)
			assert.True(l.Data[0].Type == ValInt)
			assert.True(l.Data[0].Int() == 1)
			assert.True(l.Data[1].Type == ValBool)
			assert.True(l.Data[1].Bool())
		}
		{
			l := output.listAt("list3")
			assert.True(l != nil)
			assert.True(len(l.Data) == 3)
			assert.True(l.Data[0].Type == ValInt)
			assert.True(l.Data[0].Int() == 1)

			assert.True(l.Data[1].Type == ValStr)
			assert.True(l.Data[1].String() == "Hello World")

			assert.True(l.Data[2].Type == ValList)
			assert.True(l.Data[2].List() != nil)
			assert.True(l.Data[2].List().Data[0].Type == ValInt)
			assert.True(l.Data[2].List().Data[0].Int() == 1)
		}

		{
			l := output.mapAt("empty_map")
			assert.True(l != nil)
			assert.True(l.Length() == 0)
		}
		{
			l := output.mapAt("map1")
			assert.True(l != nil)
			assert.True(l.Length() == 1)

			v0, ok := l.Get("a")
			assert.True(ok)
			assert.True(v0.Type == ValMap)
		}
		{
			l := output.mapAt("map2")
			assert.True(l != nil)
			assert.True(l.Length() == 2)

			v1, ok := l.Get("b")
			assert.True(ok)
			assert.True(v1.Type == ValBool)
			assert.True(v1.Bool())
		}

		{
			l := output.mapAt("map3")
			assert.True(l != nil)
			assert.True(l.Length() == 1)

			v0, ok := l.Get("a")
			assert.True(ok)
			assert.True(v0.Type == ValMap)

			v00, ok := v0.Map().Get("b")
			assert.True(ok)
			assert.True(v00.Type == ValMap)

			v000, ok := v00.Map().Get("c")
			assert.True(ok)
			assert.True(v000.Type == ValInt)
			assert.True(v000.Int() == 1)
		}
	}
}

func TestStrInterpo(t *testing.T) {
	assert := assert.New(t)
	{
		output := make(actionOutput)

		eval := NewEvaluatorWithContextCallback(
			func(_ *Evaluator, vname string) (Val, error) {
				if vname == "a" {
					return NewValStr("Hello World"), nil
				}
				if vname == "abs" {
					return NewValNativeFunction(
						"abs",
						func(args []Val) (Val, error) {
							a0 := args[0]
							must(a0.Type == ValInt, "must be int")
							return NewValInt64(-a0.Int()), nil
						},
					), nil
				}
				return NewValNull(), fmt.Errorf("%s unknown var", vname)
			},
			nil,

			func(_ *Evaluator, aname string, aval Val) error {
				output[aname] = aval
				return nil
			})

		module, err := CompileModule(
			`
// whatever module
"mm" => {
  let var1 = 10;
  let var2 = 'xxxx';

  v1 => "aa{{var1}}bb";
  v2 => "aa{{var2}},{{a}},{{abs(-100)}}";
}`, nil)

		if err != nil {
			fmt.Printf(":module %s", err.Error())
		}
		assert.True(err == nil)

		_, err = eval.Eval("mm", module)
		if err != nil {
			fmt.Printf(":eval %s", err.Error())
		}
		assert.True(output.strAt("v1", "aa10bb"))
		assert.True(output.strAt("v2", "aaxxxx,Hello World,100"))
	}
}

func TestLocal(t *testing.T) {
	assert := assert.New(t)
	{
		output := make(actionOutput)

		eval := NewEvaluatorWithContextCallback(
			func(_ *Evaluator, vname string) (Val, error) {
				if vname == "a" {
					return NewValStr("Hello World"), nil
				}
				if vname == "abs" {
					return NewValNativeFunction(
						"abs",
						func(args []Val) (Val, error) {
							a0 := args[0]
							must(a0.Type == ValInt, "must be int")
							return NewValInt64(-a0.Int()), nil
						},
					), nil
				}
				return NewValNull(), fmt.Errorf("%s unknown var", vname)
			},
			nil,
			func(_ *Evaluator, aname string, aval Val) error {
				output[aname] = aval
				return nil
			})

		module, err := CompileModule(
			`
// whatever module
"mm" => {
  let var1 = 10;
  let var2 = true;
  let var3 = false;
  let var4 = [1, 2, {'a': [10, 20]}, 3];
  let var5 = a;
  let var6 = abs(-100);
  v1 => var1;
  v2 => var2;
  v3 => var3;
  v4 => var4[2].a[1];
  v5 => var5;
  v6 => var6;
}`, nil)

		if err != nil {
			fmt.Printf(":module %s", err.Error())
		}
		assert.True(err == nil)

		_, err = eval.Eval("mm", module)
		if err != nil {
			fmt.Printf(":eval %s", err.Error())
		}
		assert.True(err == nil)
		assert.True(output.intAt("v1", 10))
		assert.True(output.boolAt("v2", true))
		assert.True(output.boolAt("v3", false))
		assert.True(output.intAt("v4", 20))
		assert.True(output.intAt("v6", 100))
		assert.True(output.strAt("v5", "Hello World"))
	}
}

func TestICall(t *testing.T) {
	assert := assert.New(t)
	now := time.Now().Unix()

	assert.True(testInt64(
		`
test{
  output => time::unix();
}
`, now))
}

// arithmetic operation testing
func TestArith1(t *testing.T) {
	assert := assert.New(t)

	assert.True(testInt(
		`
test{
  output => 200 - 1 + 1;
}
`, 200))

	// (1) integer
	assert.True(testInt(
		`
test{
  output => 1 + 2;
}
`, 3))

	assert.True(testInt(
		`
test{
  output => 1 + 2 * 3;
}
`, 1+2*3))

	assert.True(testInt(
		`
test{
  output => 2 + 2 / 2;
}
`, 3))

	assert.True(testInt(
		`
test{
  output => 2 ** 3 + 1;
}
`, 9))

	assert.True(testInt(
		`
test{
  output => 2 * (1+2);
}
`, 6))

	assert.True(testInt(
		`
test{
  output => 1 - 2;
}
`, -1))

	assert.True(testInt(
		`
test{
  output => 1 - 2 * 3;
}
`, 1-2*3))

	assert.True(testInt(
		`
test{
  output => 2 - 2 / 2;
}
`, 1))

	assert.True(testInt(
		`
test{
  output => 2 ** 3 - 1;
}
`, 7))

	assert.True(testInt(
		`
test{
  output => 2 * (1-2);
}
`, -2))

	// (2) real
	assert.True(testReal(
		`
test{
  output => 1.0 + 2.0;
}
`, 3.0))

	assert.True(testReal(
		`
test{
  output => 1.0 + 2.0 * 3.0;
}
`, 7.0))

	assert.True(testReal(
		`
test{
  output => 2.0 + 2.0 / 2.0;
}
`, 3.0))

	assert.True(testReal(
		`
test{
  output => 2.0 ** 3.0 + 1.0;
}
`, 9.0))

	assert.True(testReal(
		`
test{
  output => 2.0 * (1.0+2.0);
}
`, 6.0))

	assert.True(testReal(
		`
test{
  output => 1.0 - 2.0;
}
`, -1.0))

	assert.True(testReal(
		`
test{
  output => 1.0 - 2.0 * 3.0;
}
`, 1.0-2.0*3.0))

	assert.True(testReal(
		`
test{
  output => 2.0 - 2.0 / 2.0;
}
`, 1.0))

	assert.True(testReal(
		`
test{
  output => 2.0 ** 3.0 - 1.0;
}
`, 7.0))

	assert.True(testReal(
		`
test{
  output => 2.0 * (1.0-2.0);
}
`, -2.0))

	// (3) num
	assert.True(testReal(
		`
test{
  output => 1.0 + 2;
}
`, 3.0))

	assert.True(testReal(
		`
test{
  output => 1 + 2.0 * 3.0;
}
`, 7.0))

	assert.True(testReal(
		`
test{
  output => 2 + 2.0 / 2.0;
}
`, 3.0))

	assert.True(testReal(
		`
test{
  output => 2.0 ** 3 + 1;
}
`, 9.0))

	assert.True(testReal(
		`
test{
  output => 2.0 * (1.0+2);
}
`, 6.0))

	assert.True(testReal(
		`
test{
  output => 1.0 - 2;
}
`, -1.0))

	assert.True(testReal(
		`
test{
  output => 1.0 - 2.0 * 3;
}
`, 1.0-2.0*3.0))

	assert.True(testReal(
		`
test{
  output => 2.0 - 2.0 / 2;
}
`, 1.0))

	assert.True(testReal(
		`
test{
  output => 2.0 ** 3.0 - 1;
}
`, 7.0))

	assert.True(testReal(
		`
test{
  output => 2.0 * (1.0-2);
}
`, -2.0))

	// (3) string concatenation
	assert.True(testString(
		`
test {
  output => 'a' + 'b';
}
`, "ab"))

	assert.True(testString(
		`
test {
  output => '' + 'b';
}
`, "b"))

	assert.True(testString(
		`
test {
  output => 'a' + '';
}
`, "a"))

	assert.True(testString(
		`
test {
  output => 'a' + 1;
}
`, "a1"))

	assert.True(testString(
		`
test {
  output => 'a' + true;
}
`, "atrue"))

}

func TestComp1(t *testing.T) {
	assert := assert.New(t)

	// (1) int
	assert.True(testBool(
		`
test{
  output => 1 < 2;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1 <= 2;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1 > 2;
}
`, false))

	assert.True(testBool(
		`
test{
  output => 1 >= 2;
}
`, false))

	assert.True(testBool(
		`
test{
  output => 1 != 2;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1 == 2;
}
`, false))

	// (2) real
	assert.True(testBool(
		`
test{
  output => 1.0 < 2.0;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1.0 <= 2.0;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1.0 > 2.0;
}
`, false))

	assert.True(testBool(
		`
test{
  output => 1.0 >= 2.0;
}
`, false))

	assert.True(testBool(
		`
test{
  output => 1.0 != 2.0;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1.0 == 2.0;
}
`, false))

	// (3) upgrade
	assert.True(testBool(
		`
test{
  output => 1 < 2.0;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1.0 <= 2;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1 > 2.0;
}
`, false))

	assert.True(testBool(
		`
test{
  output => 1.0 >= 2;
}
`, false))

	assert.True(testBool(
		`
test{
  output => 1.0 != 2;
}
`, true))

	assert.True(testBool(
		`
test{
  output => 1 == 2.0;
}
`, false))

	// (4) string
	assert.True(testBool(
		`
test{
  output => 'a' < 'b';
}
`, true))

	assert.True(testBool(
		`
test{
  output => 'a' <= 'b';
}
`, true))

	assert.True(testBool(
		`
test{
  output => 'a' > 'b';
}
`, false))

	assert.True(testBool(
		`
test{
  output => 'a' >= 'b';
}
`, false))

	assert.True(testBool(
		`
test{
  output => 'a' != 'b';
}
`, true))

	assert.True(testBool(
		`
test{
  output => 'a' == 'b';
}
`, false))

}

func TestLogic1(t *testing.T) {
	assert := assert.New(t)

	assert.True(testCond(
		`
test{
  output => 11 || not_existed();
}
`, true))
	assert.True(testInt(
		`
test{
  output => 11 || not_existed();
}
`, 11))

	assert.True(testCond(
		`
test{
  output => false && not_existed();
}
`, false))
	assert.True(testBool(
		`
test{
  output => false && not_existed();
}
`, false))

	assert.True(testCond(
		`
test{
  output => false || 0 || "" || "Hello World";
}
`, true))

	assert.True(testString(
		`
test{
  output => false || 0 || "" || "Hello World";
}
`, "Hello World"))

	assert.True(testCond(
		`
test{
  output => true && 1 && "H" && "null";
}
`, true))

	assert.True(testString(
		`
test{
  output => true && 1 && "H" && "";
}
`, ""))

}

func TestTernary1(t *testing.T) {
	assert := assert.New(t)

	assert.True(testInt(
		`
test{
  output => 1 if true else 2;
}
`, 1))

	assert.True(testInt(
		`
test{
  output => 1 if false else 2;
}
`, 2))

	assert.True(testInt(
		`
test{
  output => 1 if false else (3 if false else (4 if false else 2));
}
`, 2))

}

func TestIf(t *testing.T) {
	assert := assert.New(t)

	assert.True(testInt(
		`
test{
  output => if true { 10; };
}
`, 10))

	assert.True(testInt(
		`
test{
  output => if false { 10; } else { 1; };
}
`, 1))

	assert.True(testInt(
		`
test{
  output => if false { 10; } elif true { 1; };
}
`, 1))

	assert.True(testInt(
		`
test{
  output => if false { 10; } elif false { ""; } elif false { "X"; } else { 1; };
}
`, 1))

	assert.True(testNull(
		`
test{
  output => if false { 10; };
}
`))

	assert.True(testNull(
		`
test{
  output => if false { 10; } elif false { "XX"; };
}
`))

}

func TestAssign1(t *testing.T) {
	assert := assert.New(t)
	assert.True(testString(
		`
session {
  a = 10;
}

test{
  a = "hello world";
  output => a;
}
`, "hello world"))

	assert.True(testInt(
		`
session {
  a = 0;
}

test{
  a += 10;
  a -= 5;
  a += 5;
  a += 3;
  a -= 3;
  a /= 5;
  a *= 5;
  output => a;
}
`, 10))

	assert.True(testInt(
		`
session {
  a = 2;
}

test{
  a **= 4;
  a /= 8;
  a *= 5;
  output => a;
}
`, 10))

	assert.True(testInt(
		`
session {
  a = 3;
}

test{
  a %= 2;
  output => a;
}
`, 1))

	assert.True(testInt(
		`
session {
  a = 3;
}

test{
  a++;
  a--;
  a++;
  a--;
  a++;
  output => a;
}
`, 4))

	assert.True(testString(
		`
test{
  let a = 100;
  a = "hello world";
  output => a;
}
`, "hello world"))

}

func TestAssign2(t *testing.T) {
	assert := assert.New(t)
	assert.True(testString(
		`
session {
  a = ["a", "b"];
}

test{
  a[10] = "hello world";
  output => a[10];
}
`, "hello world"))

	assert.True(testInt(
		`
session {
  a = ["a", "b"];
}

test{
  a[10] = 0;
  a[10] += 10;
  a[10] -= 10;
  a[10] *= 10;
  a[10] += 2;
  a[10] **= 4;
  a[10] /= 8;
  a[10] *= 5;
  output => a[10];
}
`, 10))

	assert.True(testInt(
		`
session {
  a = ["a", "b"];
}

test{
  a[10] = 0;
  a[10]++;
  a[10]--;
  a[10]++;
  a[10] = 8;
  a[10]++;
  output => a[10];
}
`, 9))

	assert.True(testNull(
		`
session {
  a = [];
}

test{
  a[10] = "hello world";
  output => a[7];
}
`))

	assert.True(testString(
		`
session {
  a = {};
}

test{
  a.b = "hello world";
  output => a.b;
}
`, "hello world"))

	assert.True(testNull(
		`
session {
  a = {};
}

test{
  a["xx"] = null;
  output => a.xx;
}
`))

}

func TestAssign3(t *testing.T) {
	assert := assert.New(t)
	rr := NewValNull()
	ret := &rr

	dvar := &Val{}

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
			if vname == "a" {
				return *dvar, nil
			}
			return NewValNull(), fmt.Errorf("%s unknown var", vname)
		},
		func(_ *Evaluator, fname string, v Val) error {
			if fname == "a" {
				*dvar = v
				return nil
			}
			return fmt.Errorf("%s is unknown var", fname)
		},
		func(_ *Evaluator, aname string, aval Val) error {
			if aname == "output" {
				*ret = aval
			}
			return nil
		})

	code :=
		`
test {
  a = "Hello World";
  output => a;
}
`

	module, err := CompileModule(code, nil)

	// fmt.Printf(":code\n%s", module.Dump())

	assert.True(err == nil)

	_, err = eval.Eval("test", module)
	assert.True(err == nil)

	assert.True(ret.Type == ValStr)
	assert.True(ret.String() == "Hello World")
}

func TestMatch(t *testing.T) {
	assert := assert.New(t)
	assert.True(testNull(
		`
not_matched{
  output => true;
}
`))

	assert.True(testNull(
		`
not_matched => {
  output => true;
}
`))
}

func TestRegex(t *testing.T) {
	assert := assert.New(t)
	assert.True(testBool(
		`
test{
  output => "aa" ~ R"aa";
}
`, true))

	assert.True(testBool(
		`
test{
  output => "aa" ~ R"ab";
}
`, false))

}

func fib(a int) int {
	if a == 0 {
		return 1
	} else if a <= 2 {
		return a
	} else {
		return fib(a-1) + fib(a-2)
	}
}

func TestSCall(t *testing.T) {
	assert := assert.New(t)
	assert.True(testString(
		`
fn hello_world(x, y) {
  let xx = x + y;
  return xx;
}

test{
  output => hello_world("hello", " world");
}
`, "hello world"))

	assert.True(testInt(
		`
fn fib(n) {
  return if n == 0 {
    1;
  } elif n <= 2 {
    n;
  } else {
    fib(n-1) + fib(n-2);
  };
}

test{
  output => fib(5);
}
`, fib(5)))

	assert.False(testInt(
		`
fn err() {
  return a_unown_variable;
}

fn bar() {
  err();
}

fn foo() {
  bar();
}

fn xx(n) {
  foo();
}

test{
  output => xx(1);
}
`, fib(5)))

}

func TestTemplate(t *testing.T) {
	assert := assert.New(t)
	assert.True(testString(
		`
test{
  output => template "go", {
    'a' : 10,
    'b' : 20,
    'c' : "Hello World",
    'd' : true
  }, `+"```\nHello{{.a}}World{{.c}}\n```;}", "Hello10WorldHello World"))

	assert.True(testString(
		`
test{
  output => template "go[]", {
    'a' : 10,
    'b' : 20,
    'c' : "Hello World",
    'd' : true
  }, `+"```\nHello{{.a}}World{{.c}}\n```;}", "Hello10WorldHello World"))

	assert.True(testString(
		`
test{
  output => template "go[a=10, b=20.0, c=true, d='', e=false, f=null]", {
    'a' : 10,
    'b' : 20,
    'c' : "Hello World",
    'd' : true
  }, `+"```\nHello{{.a}}World{{.c}}\n```;}", "Hello10WorldHello World"))
}

func TestTripCountLoopStatement(t *testing.T) {
	assert := assert.New(t)

	assert.True(testInt(`
test{
  let xx = 0;
  for let i = 1; i <= 100; i++ {
    xx += i;
  }
  output => xx;
}
`, func() int {
		sum := 0
		for i := 1; i < 101; i++ {
			sum += i
		}
		return sum
	}()))

	assert.True(testInt(`
test{
  let xx = 0;
  let i = 0;
  for ; i <= 100; i++ {
    xx += i;
  }
  output => xx;
}
`, func() int {
		sum := 0
		for i := 1; i < 101; i++ {
			sum += i
		}
		return sum
	}()))

	assert.True(testInt(`
test{
  let xx = 0;
  let i = 0;
  for ;; i++ {
    if i >= 101 {
      break;
    }
    xx += i;
  }
  output => xx;
}
`, func() int {
		sum := 0
		for i := 1; i < 101; i++ {
			sum += i
		}
		return sum
	}()))

	assert.True(testInt(`
test{
  let xx = 0;
  let i = 0;
  for ;; {
    if i >= 101 {
      break;
    }
    xx += i;
    i++;
  }
  output => xx;
}
`, func() int {
		sum := 0
		for i := 1; i < 101; i++ {
			sum += i
		}
		return sum
	}()))
}

func TestForeverLoopStatement(t *testing.T) {
	assert := assert.New(t)

	assert.True(testInt(`
test{
  let xx = 0;
  let i = 1;
  for {
    if i > 100 {
      break;
    }
    xx += i;
    i++;
  }
  output => xx;
}

`, func() int {
		sum := 0
		for i := 1; i < 101; i++ {
			sum += i
		}
		return sum
	}()))
}

func TestIfStatment(t *testing.T) {
	assert := assert.New(t)
	assert.True(testString(
		`
test {
  let xx = 0;
  if xx == 0 {
    10;
  } else {
    20;
  }
  output => "1000";
}
`, "1000"))

}
