package pl

import "fmt"

type DynamicVariable struct {
	Key   string
	Value Val
}

func must(cond bool, msg string, a ...interface{}) {
	if !cond {
		f := fmt.Sprintf("must: %s", msg)
		panic(fmt.Sprintf(f, a...))
	}
}

func musterr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err.Error()))
	}
}

func unreachable(msg string) { panic(fmt.Sprintf("unreachable: %s", msg)) }

// module function name
func modFuncName(m string, f string) string {
	return fmt.Sprintf("%s::%s", m, f)
}

// a all in one function for simplicity
func EvalExpression(
	expression string,
	loadVarFn EvalLoadVar,
	storeVarFn EvalStoreVar,
	callFn EvalCall,
	actionFn EvalAction,
) (Val, error) {
	p, err := CompilePolicyAsExpression(expression)
	if err != nil {
		return NewValNull(), err
	}

	e := NewEvaluator(
		loadVarFn,
		storeVarFn,
		callFn,
		actionFn,
	)

	return e.EvalExpr(p)
}
