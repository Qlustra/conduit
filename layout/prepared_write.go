package layout

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrPreparedWriteStale reports that content or its binding changed after
// preparation. Prepare a new write from the desired current content.
var ErrPreparedWriteStale = errors.New("prepared write no longer matches cached content or path")

type preparedWrite struct {
	file    File
	data    []byte
	current func() ([]byte, error)
	written func()
}

func newPreparedWrite(file File, data []byte, current func() ([]byte, error), written func()) *preparedWrite {
	return &preparedWrite{file: file, data: bytes.Clone(data), current: current, written: written}
}

func (w *preparedWrite) Path() string { return w.file.Path() }

// Bytes returns an independent copy of the prepared bytes.
func (w *preparedWrite) Bytes() []byte { return bytes.Clone(w.data) }

func (w *preparedWrite) check() error {
	current, err := w.current()
	if err != nil {
		return err
	}
	if !bytes.Equal(current, w.data) {
		return ErrPreparedWriteStale
	}
	return nil
}

func (w *preparedWrite) Write(ctx Context) error {
	if err := w.check(); err != nil {
		return err
	}
	if err := w.file.WriteBytes(w.data, ctx); err != nil {
		return err
	}
	w.written()
	return nil
}

// WriteExclusive uses O_EXCL to preserve any existing destination, including
// one created after preparation. This is a direct write; a write/close failure
// can leave a partial new file and does not mark cached state synced.
func (w *preparedWrite) WriteExclusive(ctx Context) error {
	if ctx.writePolicy() != WriteDirect {
		return fmt.Errorf("exclusive prepared writes require WriteDirect, got write policy %d", ctx.writePolicy())
	}
	if err := w.check(); err != nil {
		return err
	}
	if err := guardPathParents(w.Path(), ctx.pathSafetyPolicy()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(w.Path()), ctx.DirMode); err != nil {
		return err
	}
	out, err := os.OpenFile(w.Path(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, ctx.FileMode)
	if err != nil {
		return err
	}
	n, writeErr := out.Write(w.data)
	if writeErr == nil && n != len(w.data) {
		writeErr = io.ErrShortWrite
	}
	closeErr := out.Close()
	if writeErr != nil || closeErr != nil {
		return errors.Join(writeErr, closeErr)
	}
	w.written()
	return nil
}
