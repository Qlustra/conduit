package layout

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// TransformFunc transforms data from src into dst.
type TransformFunc func(dst io.Writer, src io.Reader) error

// TransformReader applies transform to src and returns the complete transformed
// output in memory.
//
// The transform receives streaming interfaces, but TransformReader buffers the
// full output before returning. File Transform helpers use this buffered output
// to preserve all-or-nothing destination writes.
func TransformReader(src io.Reader, transform TransformFunc) ([]byte, error) {
	if src == nil {
		return nil, fmt.Errorf("transform source must not be nil")
	}
	if transform == nil {
		return nil, fmt.Errorf("transform must not be nil")
	}

	var out bytes.Buffer
	if err := transform(&out, src); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// TransformBytes applies transform to data and returns the complete transformed
// output in memory.
func TransformBytes(data []byte, transform TransformFunc) ([]byte, error) {
	return TransformReader(bytes.NewReader(data), transform)
}

// TransformString applies transform to data and returns the complete
// transformed string in memory.
func TransformString(data string, transform TransformFunc) (string, error) {
	out, err := TransformReader(strings.NewReader(data), transform)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// TransformReader applies transform to src and rewrites the file only after the
// transform succeeds.
func (f File) TransformReader(ctx Context, src io.Reader, transform TransformFunc) error {
	out, err := TransformReader(src, transform)
	if err != nil {
		return err
	}
	return f.WriteBytes(out, ctx)
}

// TransformBytes applies transform to data and rewrites the file only after the
// transform succeeds.
func (f File) TransformBytes(ctx Context, data []byte, transform TransformFunc) error {
	out, err := TransformBytes(data, transform)
	if err != nil {
		return err
	}
	return f.WriteBytes(out, ctx)
}

// TransformString applies transform to data and rewrites the file only after
// the transform succeeds.
func (f File) TransformString(ctx Context, data string, transform TransformFunc) error {
	out, err := TransformString(data, transform)
	if err != nil {
		return err
	}
	return f.WriteBytes([]byte(out), ctx)
}

// TransformFile reads src, applies transform, and rewrites the file only after
// the transform succeeds.
func (f File) TransformFile(ctx Context, src File, transform TransformFunc) error {
	data, err := readFileForProcessing(ctx, src)
	if err != nil {
		return err
	}
	return f.TransformBytes(ctx, data, transform)
}

// Transform reads the file, applies transform, and rewrites the same file only
// after the transform succeeds.
func (f File) Transform(ctx Context, transform TransformFunc) error {
	return f.TransformFile(ctx, f, transform)
}

func readFileForProcessing(ctx Context, src File) ([]byte, error) {
	handle, err := src.OpenRead(ctx, OpenExisting)
	if err != nil {
		return nil, err
	}

	data, readErr := io.ReadAll(handle)
	closeErr := handle.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return data, nil
}
