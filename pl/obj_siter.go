package pl

import (
	"fmt"
)

// script iterator
type scriptIter struct {
	prog    *program
	upvalue []Val

	current Val   // yield value
	err     error // pending error
	next    bool  // whether we have any value

	eval *Evaluator // current evaluator
	pc   int        // program counter where to resume from

	stack []Val // local stack for the scriptIter, the evaluator will delegate
	// all the stack operation by using script's own stack instead
	// its shared stack

	frame funcframe
}

// used internally by the evaluator when generate yield
func (s *scriptIter) onYield(value Val) error {
	s.current = value
	s.next = true
	return nil
}

func (s *scriptIter) onReturn(_ Val) error {
	s.next = false
	return nil
}

func (s *scriptIter) resume() {
	s.err = nil
	s.next = false

	pc, err := s.eval.resumeSIter(
		s,
	)
	s.pc = pc
	s.err = err
}

func (s *scriptIter) SetUp(e *Evaluator, args []Val) error {
	s.eval = e
	s.err = nil
	s.next = false

	pc, err := e.runSIter(
		s,
		args,
	)

	s.pc = pc
	s.err = err
	return err
}

func (s *scriptIter) Has() bool {
	return s.next
}

func (s *scriptIter) Next() (bool, error) {
	s.resume()
	return s.next, s.err
}

func (s *scriptIter) Deref() (Val, Val, error) {
	if !s.Has() {
		return NewValNull(), NewValNull(), fmt.Errorf("iterator out of bound")
	}
	if s.err != nil {
		return NewValNull(), NewValNull(), s.err
	}
	if !s.current.IsPair() {
		return NewValNull(), NewValNull(), fmt.Errorf("invalid value from yield, must be pair")
	}
	return s.current.Pair().First, s.current.Pair().Second, nil
}

func newScriptIter(prog *program) *scriptIter {
	return &scriptIter{
		prog: prog,
	}
}
