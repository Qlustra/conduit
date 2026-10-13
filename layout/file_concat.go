package layout

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ConcatOptions configures buffered concat helpers.
type ConcatOptions struct {
	// Header is written once before any entries.
	Header []byte

	// Footer is written once after all entries and any final separator.
	Footer []byte

	// Separator is written between entries. When FinalSeparator is true, it is
	// also written after the final entry when at least one entry exists.
	Separator []byte

	// FinalSeparator controls whether Separator is written after the final entry.
	FinalSeparator bool

	// EntryPrefix is written before each entry payload.
	EntryPrefix []byte

	// EntrySuffix is written after each entry payload.
	EntrySuffix []byte
}

// ConcatReaders reads srcs in order and returns the complete concatenated
// output in memory.
func ConcatReaders(opts ConcatOptions, srcs ...io.Reader) ([]byte, error) {
	var out bytes.Buffer
	if _, err := out.Write(opts.Header); err != nil {
		return nil, err
	}

	for i, src := range srcs {
		if src == nil {
			return nil, fmt.Errorf("concat source %d must not be nil", i)
		}
		if i > 0 {
			if _, err := out.Write(opts.Separator); err != nil {
				return nil, err
			}
		}
		if _, err := out.Write(opts.EntryPrefix); err != nil {
			return nil, err
		}
		if _, err := io.Copy(&out, src); err != nil {
			return nil, err
		}
		if _, err := out.Write(opts.EntrySuffix); err != nil {
			return nil, err
		}
	}

	if opts.FinalSeparator && len(srcs) > 0 {
		if _, err := out.Write(opts.Separator); err != nil {
			return nil, err
		}
	}
	if _, err := out.Write(opts.Footer); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// ConcatBytes returns the complete concatenated output in memory.
func ConcatBytes(opts ConcatOptions, srcs ...[]byte) []byte {
	readers := make([]io.Reader, 0, len(srcs))
	for _, src := range srcs {
		readers = append(readers, bytes.NewReader(src))
	}
	out, _ := ConcatReaders(opts, readers...)
	return out
}

// ConcatStrings returns the complete concatenated string in memory.
func ConcatStrings(opts ConcatOptions, srcs ...string) string {
	readers := make([]io.Reader, 0, len(srcs))
	for _, src := range srcs {
		readers = append(readers, strings.NewReader(src))
	}
	out, _ := ConcatReaders(opts, readers...)
	return string(out)
}

// ConcatReaders reads srcs in order and rewrites the file only after all input
// has been concatenated successfully.
func (f File) ConcatReaders(ctx Context, opts ConcatOptions, srcs ...io.Reader) error {
	out, err := ConcatReaders(opts, srcs...)
	if err != nil {
		return err
	}
	return f.WriteBytes(out, ctx)
}

// ConcatBytes rewrites the file with the complete concatenated output.
func (f File) ConcatBytes(ctx Context, opts ConcatOptions, srcs ...[]byte) error {
	return f.WriteBytes(ConcatBytes(opts, srcs...), ctx)
}

// ConcatStrings rewrites the file with the complete concatenated output.
func (f File) ConcatStrings(ctx Context, opts ConcatOptions, srcs ...string) error {
	return f.WriteBytes([]byte(ConcatStrings(opts, srcs...)), ctx)
}

// ConcatFiles reads srcs in order and rewrites the file only after all input
// files have been read and concatenated successfully.
func (f File) ConcatFiles(ctx Context, opts ConcatOptions, srcs ...File) error {
	chunks := make([][]byte, 0, len(srcs))
	for _, src := range srcs {
		data, err := readFileForProcessing(ctx, src)
		if err != nil {
			return err
		}
		chunks = append(chunks, data)
	}

	return f.ConcatBytes(ctx, opts, chunks...)
}
