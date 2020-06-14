package main

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

var ErrClosedPipe = errors.New("bufpipe: read/write on closed pipe")

type pipe struct {
	cond *sync.Cond
	buf  *bytes.Buffer
	rErr error
	wErr error
}

type PipeReader struct {
	*pipe
}

type PipeWriter struct {
	*pipe
}

func newPipe() (*PipeReader, *PipeWriter) {
	p := &pipe{
		buf:  new(bytes.Buffer),
		cond: sync.NewCond(new(sync.Mutex)),
	}
	return &PipeReader{
			pipe: p,
		}, &PipeWriter{
			pipe: p,
		}
}

func (r *PipeReader) Read(data []byte) (int, error) {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()

RETRY:
	n, err := r.buf.Read(data)
	// If not closed and no read, wait for writing.
	if err == io.EOF && r.rErr == nil && n == 0 {
		r.cond.Wait()
		goto RETRY
	}
	if err == io.EOF {
		return n, r.rErr
	}
	return n, err
}

func (r *PipeReader) Close() error {
	return r.CloseWithError(nil)
}

func (r *PipeReader) CloseWithError(err error) error {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()

	if err == nil {
		err = ErrClosedPipe
	}
	r.wErr = err
	return nil
}

func (w *PipeWriter) Write(data []byte) (int, error) {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	if w.wErr != nil {
		return 0, w.wErr
	}

	n, err := w.buf.Write(data)
	w.cond.Signal()
	return n, err
}

func (w *PipeWriter) Close() error {
	return w.CloseWithError(nil)
}

func (w *PipeWriter) CloseWithError(err error) error {
	w.cond.L.Lock()
	defer w.cond.L.Unlock()

	if err == nil {
		err = io.EOF
	}
	w.rErr = err
	return nil
}
