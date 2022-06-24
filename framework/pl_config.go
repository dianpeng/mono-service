package framework

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
)

// Config accesser wrapper
type PLConfig struct {
	context ServiceContext
	args    []pl.Val
}

func NewPLConfig(c ServiceContext, a []pl.Val) PLConfig {
	return PLConfig{
		context: c,
		args:    a,
	}
}

func (p *PLConfig) tryeval(v pl.Val) (pl.Val, error) {
	if v.IsClosure() {
		cls, _ := v.Usr().(pl.Closure)
		return cls.Call(p.context.Hpl().Eval, []pl.Val{})
	}
	return v, nil
}

func (p *PLConfig) Get(
	index int,
	ptr *pl.Val,
) error {
	if len(p.args) <= index {
		return fmt.Errorf("out of range!")
	}
	arg, err := p.tryeval(p.args[index])
	if err != nil {
		return fmt.Errorf("%d'th elements evaluation error: %s", err.Error())
	}
	*ptr = arg
	return nil
}

func (p *PLConfig) TryGet(
	index int,
	ptr *pl.Val,
	def pl.Val,
) {
	if err := p.Get(index, ptr); err != nil {
		*ptr = def
	}
}

func (p *PLConfig) GetStr(
	index int,
	ptr *string,
) error {
	if len(p.args) <= index {
		return fmt.Errorf("out of range!")
	}

	arg, err := p.tryeval(p.args[index])
	if err != nil {
		return fmt.Errorf("%d'th elements evaluation error: %s", err.Error())
	}

	str, err := arg.ToString()
	if err != nil {
		return fmt.Errorf("%d'th elements cannot be converted to string: %s", index, err.Error())
	}

	*ptr = str
	return nil
}

func (p *PLConfig) TryGetStr(
	index int,
	ptr *string,
	def string,
) {
	if err := p.GetStr(index, ptr); err != nil {
		*ptr = def
	}
}

func (p *PLConfig) GetInt(
	index int,
	ptr *int,
) error {
	if len(p.args) <= index {
		return fmt.Errorf("out of range!")
	}

	arg, err := p.tryeval(p.args[index])
	if err != nil {
		return fmt.Errorf("%d'th elements evaluation error: %s", err.Error())
	}

	if arg.IsInt() {
		return fmt.Errorf("%d'th elements is not int", index)
	}

	*ptr = int(arg.Int())
	return nil
}

func (p *PLConfig) TryGetInt(
	index int,
	ptr *int,
	def int,
) {
	if err := p.GetInt(index, ptr); err != nil {
		*ptr = def
	}
}

func (p *PLConfig) GetInt64(
	index int,
	ptr *int64,
) error {
	if len(p.args) <= index {
		return fmt.Errorf("out of range!")
	}

	arg, err := p.tryeval(p.args[index])
	if err != nil {
		return fmt.Errorf("%d'th elements evaluation error: %s", err.Error())
	}

	if arg.IsInt() {
		return fmt.Errorf("%d'th elements is not int", index)
	}

	*ptr = arg.Int()
	return nil
}

func (p *PLConfig) TryGetInt64(
	index int,
	ptr *int64,
	def int64,
) {
	if err := p.GetInt64(index, ptr); err != nil {
		*ptr = def
	}
}

func (p *PLConfig) GetReal(
	index int,
	ptr *float64,
) error {
	if len(p.args) <= index {
		return fmt.Errorf("out of range!")
	}

	arg, err := p.tryeval(p.args[index])
	if err != nil {
		return fmt.Errorf("%d'th elements evaluation error: %s", err.Error())
	}

	if !arg.IsReal() {
		return fmt.Errorf("%d'th elements is not real", index)
	}

	*ptr = arg.Real()
	return nil
}

func (p *PLConfig) TryGetReal(
	index int,
	ptr *float64,
	def float64,
) {
	if err := p.GetReal(index, ptr); err != nil {
		*ptr = def
	}
}

func (p *PLConfig) GetBool(
	index int,
	ptr *bool,
) error {
	if len(p.args) <= index {
		return fmt.Errorf("out of range!")
	}

	arg, err := p.tryeval(p.args[index])
	if err != nil {
		return fmt.Errorf("%d'th elements evaluation error: %s", err.Error())
	}

	if !arg.IsBool() {
		return fmt.Errorf("%d'th elements is not bool", index)
	}

	*ptr = arg.Bool()
	return nil
}

func (p *PLConfig) TryGetBool(
	index int,
	ptr *bool,
	def bool,
) {
	if err := p.GetBool(index, ptr); err != nil {
		*ptr = def
	}
}

func (p *PLConfig) Any(index int) (pl.Val, error) {
	x := p.At(index)
	return p.tryeval(x)
}

func (p *PLConfig) At(index int) pl.Val {
	if len(p.args) <= index {
		panic("out of range!")
	}

	return p.args[index]
}
