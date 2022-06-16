package pl

import (
	"fmt"
)

type Pair struct {
	First  Val
	Second Val
}

func (p *Pair) ToNative() interface{} {
	return [2]interface{}{
		p.First.ToNative(),
		p.Second.ToNative(),
	}
}

func (p *Pair) Index(idx Val) (Val, error) {
	i, err := idx.ToIndex()
	if err != nil {
		return NewValNull(), err
	}
	if i == 0 {
		return p.First, nil
	}
	if i == 1 {
		return p.Second, nil
	}

	return NewValNull(), fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")
}

func (p *Pair) IndexSet(idx, val Val) error {
	i, err := idx.ToIndex()
	if err != nil {
		return err
	}
	if i == 0 {
		p.First = val
		return nil
	}
	if i == 1 {
		p.Second = val
		return nil
	}

	return fmt.Errorf("invalid index, 0 or 1 is allowed on Pair")
}

func (p *Pair) Dot(i string) (Val, error) {
	if i == "first" {
		return p.First, nil
	}
	if i == "second" {
		return p.Second, nil
	}

	return NewValNull(), fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")
}

func (p *Pair) DotSet(i string, val Val) error {
	if i == "first" {
		p.First = val
		return nil
	}
	if i == "second" {
		p.Second = val
		return nil
	}

	return fmt.Errorf("invalid field name, 'first'/'second' is allowed on Pair")
}

func (p *Pair) Info() string {
	return fmt.Sprintf("[pair: %s=>%s]", p.First.Info(), p.Second.Info())
}
