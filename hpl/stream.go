package hpl

import (
	"io"
	"strings"
)

type eofReadCloser struct{}

func (e *eofReadCloser) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (e *eofReadCloser) Close() error {
	return nil
}

type eofByteReadCloser struct {
	x io.Reader
}

func neweofByteReadCloserFromString(b string) *eofByteReadCloser {
	return &eofByteReadCloser{
		x: strings.NewReader(b),
	}
}

func neweofByteReadCloserFromStream(stream io.Reader) *eofByteReadCloser {
	return &eofByteReadCloser{
		x: stream,
	}
}

func neweofByteReadCloser(b []byte) *eofByteReadCloser {
	return &eofByteReadCloser{
		x: strings.NewReader(string(b)),
	}
}

func (e *eofByteReadCloser) Read(i []byte) (int, error) {
	if e.x == nil {
		return 0, io.EOF
	} else {
		return e.x.Read(i)
	}
}

func (e *eofByteReadCloser) Close() error {
	e.x = nil
	return nil
}

func NewEofReadCloser() io.ReadCloser {
	return &eofReadCloser{}
}

func NewReadCloserFromString(x string) io.ReadCloser {
	return neweofByteReadCloserFromString(x)
}
