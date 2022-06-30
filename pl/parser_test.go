package pl

// Here the testing of parser just verify its binary result, ie whether it passes
// or failed. W.R.T the actual correctness, it is verified by the evaluation
// part for simplicity.

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func dumpProg(x *Module) {
	fmt.Printf("%s\n", x.Dump())
}

func verifyP(code string) {
	p := newParser(code, nil)
	prog, err := p.parse()
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}
	dumpProg(prog)
}

func Test(t *testing.T) {
	verifyP("f{v=if 0{;;")
	verifyP("```0\n\x8c\x8c\x8c000\n0```")
}

func TestExpr(t *testing.T) {
	assert := assert.New(t)
	{
		p := newParser(
			`
p {
  let _1 = 1 + 2;
  let _2 = 1 * 2;
  let _3 = 1 + 2 * 3;
  let _4 = 1 + 2 * 3 / 4;
  let _5 = 1 ** 2;
  let _6 = true == false;
  let _7 = true != false;
  let _8 = _7 > 1;
  let _9 = _8 >= 2;
  let _10 = _9 < 3;
  let _11 = _10 <= 20;
}
`, nil)
		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}

	{
		p := newParser(
			`
pppp {
  let a = true && false;
  let b = true || 10;
  let c = (a && true) || 10;
}
`, nil)
		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
		dumpProg(prog)
		assert.True(err == nil)
	}

	// ternary
	{
		p := newParser(
			`
pppp {
  let a = true if 1 else 2;
  let b = true if (true if 10 else 20) else 30;
  let c = if true {
    100;
  } else {
    200;
  };

  let d = if 1 > 100 && a {
    a + b + c;
  } elif false {
    a * b + c;
  } elif true {
    a + b * c;
  } else {
    a ** b + c;
  };
}
`, nil)
		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
		dumpProg(prog)
		assert.True(err == nil)
	}

}

func TestParser1(t *testing.T) {
	assert := assert.New(t)
	{
		p := newParser(`


module1 {
  act_int => 10;
  act_real => 20.0;
  act_null => null;
  act_bool = true;
  act_list = [1, 10, true, null];
  act_map = {
    'a' : 100,
    'b' : 200
  };
  act_func = xx(10, 20);
  act_var = $.a.b[2].c;
}

module2 {}

`, nil)

		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}

	{
		p := newParser(`
module_call{
  s1 = $;
  s2 = $.a;
  s3 = $.a[1];
  s4 = $.a[2].a;
  s5 = $[1][2][3];
}
`, nil)

		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}

	{
		p := newParser(`
module_call{
  s1 = a5;
  s2 = a5[1];
  s3 = a5.a[1];
  s4 = a5.a[2].a;
  s5 = a5[1][2][3];
}
`, nil)

		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}

	// call expression nesting
	{
		p := newParser(`
module_call{
  call0 = c();
  call1 = c1(10);
  call2 = c2(10, [], {}, 'aaaa', "Asdasdsa");
  call3 = c3(c4(), c5.a[1].c.d, $);
  call4 = c4(c10(c11(), 10)).a[1].cc;
}
`, nil)

		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}
}

func TestParserStringInterpolation(t *testing.T) {
	assert := assert.New(t)
	{
		p := newParser(`
module1 {
  str1 = '';
  str2 = 'abcd';
  str3 = 'aabc{{$}}accc';
  str4 = 'abbccdd{{$.a.b.c}}accc';
  str5 = 'aabbcccd{{eval().a[1]}}';
  str6 = '{{eval(a, 10)}}';
  str7 = '{{eval(a, 10)}}xxxx';
}
`, nil)

		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}

	{
		p := newParser(`
module1{
  str1 = '{';
  str2 = '{{{';
  str3 = '{{{{';
}
`, nil)

		prog, err := p.parse()
		if err != nil {
			fmt.Printf("%s", err.Error())
			dumpProg(prog)
		}
		assert.True(err == nil)
	}
}
