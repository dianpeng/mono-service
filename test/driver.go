package main

import (
	"fmt"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/http/runtime"
	"github.com/dianpeng/mono-service/pl"
	"net/http"
	"os"
	pa "path"
	"path/filepath"
	"time"
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
	slot     pl.Val
}

func (x *testcontext) OnLoadVar(e *pl.Evaluator, name string) (pl.Val, error) {
	switch name {
	case "test":
		return pl.NewValStr("testing_driver"), nil
	case "version":
		return pl.NewValStr("1.0"), nil
	case "theValue":
		return x.theValue, nil
	case "slot":
		return x.slot, nil
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

func (x *testcontext) OnStoreVar(
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

func (x *testcontext) OnAction(
	_ *pl.Evaluator,
	name string,
	v pl.Val,
) error {
	if name == "slot" {
		x.slot = v
		return nil
	}
	return fmt.Errorf("unknown action: %s", name)
}

func (x *testcontext) GetHttpClient(url string) (hpl.HttpClient, error) {
	return &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}, nil
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

		p, err := pl.CompileModule(string(data), nil)
		if err != nil {
			fmt.Printf(">> Compile: %s\n", err.Error())
			t.compileFail++
		} else {
			hpl := runtime.NewRuntime()
			hpl.SetModule(p)

			ctx := &testcontext{}

			if err := hpl.OnGlobal(ctx); err != nil {
				fmt.Printf(">> EvalGlobal: %s\n", err.Error())
				t.constFail++
				continue
			}

			if err := hpl.OnTestSession(ctx); err != nil {
				fmt.Printf(">> EvalSession: %s\n", err.Error())
				t.sessionFail++
				continue
			}

			if _, err := hpl.OnTest("test", pl.NewValNull()); err != nil {
				fmt.Printf(">> Eval: %s\n", err.Error())
				t.execFail++
			} else {
				t.pass++
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
