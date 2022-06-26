package main

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
	"os"
	pa "path"
	"path/filepath"
)

type testResult struct {
	total       int
	pass        int
	compileFail int
	constFail   int
	sessionFail int
	execFail    int
}

type testcontext struct {
	theValue pl.Val
}

func (x *testcontext) LoadVar(e *pl.Evaluator, name string) (pl.Val, error) {
	switch name {
	case "test":
		return pl.NewValStr("testing_driver"), nil
	case "version":
		return pl.NewValStr("1.0"), nil
	case "theValue":
		return x.theValue, nil
	case "callback":
		return pl.NewValNativeFunction(
			"callback",
			func(args []pl.Val) (pl.Val, error) {
				if len(args) < 1 || !args[0].IsClosure() {
					fmt.Printf("ID: %s\n", args[0].Id())
					return pl.NewValNull(), fmt.Errorf("invalid type, must be closure")
				}
				return args[0].Closure().Call(
					e,
					args[1:],
				)
			},
		), nil

	case "return_something":
		return pl.NewValNativeFunction(
			"return_something",
			func(args []pl.Val) (pl.Val, error) {
				return pl.NewValStr("Hello"), nil
			},
		), nil

	default:
		return pl.NewValNull(), fmt.Errorf("unknown variable: %s", name)
	}
}

func (x *testcontext) StoreVar(
	_ *pl.Evaluator,
	name string,
	val pl.Val,
) error {
	switch name {
	case "theValue":
		x.theValue = val
		return nil
	default:
		return fmt.Errorf("unknown variable: %s", name)
	}
}

func (x *testcontext) Action(
	_ *pl.Evaluator,
	name string,
	_ pl.Val,
) error {
	return fmt.Errorf("unknown action: %s", name)
}

func runAllTestFile(path string) (testResult, error) {
	fs, err := os.ReadDir(path)
	if err != nil {
		return testResult{}, err
	}

	t := testResult{}

	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) != ".pl" {
			continue
		}

		fpath := pa.Join(path, f.Name())

		fmt.Printf(">> file: %s\n", fpath)

		data, err := os.ReadFile(fpath)
		if err != nil {
			return testResult{}, err
		}
		t.total++

		p, err := pl.CompileModule(string(data))
		if err != nil {
			fmt.Printf(">> Compile: %s\n", err.Error())
			t.compileFail++
		} else {
			ev := pl.NewEvaluatorWithContext(
				&testcontext{},
			)

			if err := ev.EvalGlobal(p); err != nil {
				fmt.Printf(">> EvalGlobal: %s\n", err.Error())
				t.constFail++
			}
			if err := ev.EvalSession(p); err != nil {
				fmt.Printf(">> EvalSession: %s\n", err.Error())
				t.sessionFail++
			} else {
				if err := ev.Eval("test", p); err != nil {
					fmt.Printf(">> Eval: %s\n", err.Error())
					t.execFail++
				} else {
					t.pass++
				}
			}
		}
	}

	return t, nil
}

func main() {
	r, e := runAllTestFile("assets/test/")
	if e != nil {
		fmt.Printf("===============%s========\n", e.Error())
		return
	}

	fmt.Printf("================================\n")
	fmt.Printf("Total> %d\n", r.total)
	fmt.Printf("Pass> %d\n", r.pass)
	fmt.Printf("CompileFail> %d\n", r.compileFail)
	fmt.Printf("ConstFail> %d\n", r.constFail)
	fmt.Printf("SessionFail> %d\n", r.sessionFail)
	fmt.Printf("ExecFail> %d\n", r.execFail)
	fmt.Printf("================================\n")
}
