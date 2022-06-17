package pl

import (
	"fmt"
)

type striter struct {
	r   []rune
	cnt int
}

func newStrIter(s string) Iter {
	return &striter{
		r:   []rune(s),
		cnt: 0,
	}
}

func (s *striter) Has() bool {
	return s.cnt < len(s.r)
}

func (s *striter) Next() bool {
	s.cnt++
	return s.Has()
}

func (s *striter) Deref() (Val, Val, error) {
	if s.Has() {
		x := string(s.r[s.cnt])
		return NewValInt(s.cnt), NewValStr(x), nil
	}
	return NewValNull(), NewValNull(), fmt.Errorf("iterator is out of bound")
}
