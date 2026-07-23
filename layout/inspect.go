package layout

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// InspectFunc inspects streamed content from src.
type InspectFunc func(src io.Reader) error

// MatchFunc matches streamed content from src.
type MatchFunc func(src io.Reader) (bool, error)

// TokenOptions configures token-oriented inspection and matching helpers.
type TokenOptions struct {
	// Split selects the scanner tokenization function. Nil uses
	// bufio.ScanLines.
	Split bufio.SplitFunc

	// InitialBuffer optionally provides the scanner's initial token buffer.
	InitialBuffer []byte

	// MaxTokenSize optionally overrides the scanner's maximum token size. Zero
	// uses bufio.Scanner's default limit.
	MaxTokenSize int
}

// InspectTokenFunc inspects one scanned token.
type InspectTokenFunc func(token string) error

// MatchTokenFunc matches one scanned token.
type MatchTokenFunc func(token string) (bool, error)

// InspectReader passes src to inspect without buffering the full content.
func InspectReader(src io.Reader, inspect InspectFunc) error {
	if src == nil {
		return fmt.Errorf("inspect source must not be nil")
	}
	if inspect == nil {
		return fmt.Errorf("inspect must not be nil")
	}

	return inspect(src)
}

// InspectBytes passes byte data to inspect without requiring a File handle.
func InspectBytes(data []byte, inspect InspectFunc) error {
	return InspectReader(bytes.NewReader(data), inspect)
}

// InspectString passes string data to inspect without requiring a File handle.
func InspectString(data string, inspect InspectFunc) error {
	return InspectReader(strings.NewReader(data), inspect)
}

// MatchReader passes src to match without buffering the full content.
func MatchReader(src io.Reader, match MatchFunc) (bool, error) {
	if src == nil {
		return false, fmt.Errorf("match source must not be nil")
	}
	if match == nil {
		return false, fmt.Errorf("match must not be nil")
	}

	return match(src)
}

// MatchBytes passes byte data to match without requiring a File handle.
func MatchBytes(data []byte, match MatchFunc) (bool, error) {
	return MatchReader(bytes.NewReader(data), match)
}

// MatchString passes string data to match without requiring a File handle.
func MatchString(data string, match MatchFunc) (bool, error) {
	return MatchReader(strings.NewReader(data), match)
}

// InspectTokensReader scans src into tokens and passes them to inspect.
func InspectTokensReader(src io.Reader, opts TokenOptions, inspect InspectTokenFunc) error {
	if src == nil {
		return fmt.Errorf("inspect token source must not be nil")
	}
	if inspect == nil {
		return fmt.Errorf("inspect token callback must not be nil")
	}
	if err := opts.validate(); err != nil {
		return err
	}

	scanner, err := opts.scanner(src)
	if err != nil {
		return err
	}

	for scanner.Scan() {
		if err := inspect(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// InspectTokensBytes scans byte data into tokens and passes them to inspect.
func InspectTokensBytes(data []byte, opts TokenOptions, inspect InspectTokenFunc) error {
	return InspectTokensReader(bytes.NewReader(data), opts, inspect)
}

// InspectTokensString scans string data into tokens and passes them to inspect.
func InspectTokensString(data string, opts TokenOptions, inspect InspectTokenFunc) error {
	return InspectTokensReader(strings.NewReader(data), opts, inspect)
}

// MatchTokensReader scans src into tokens and passes them to match until one
// matches or scanning ends.
func MatchTokensReader(src io.Reader, opts TokenOptions, match MatchTokenFunc) (bool, error) {
	if src == nil {
		return false, fmt.Errorf("match token source must not be nil")
	}
	if match == nil {
		return false, fmt.Errorf("match token callback must not be nil")
	}
	if err := opts.validate(); err != nil {
		return false, err
	}

	scanner, err := opts.scanner(src)
	if err != nil {
		return false, err
	}

	for scanner.Scan() {
		matched, err := match(scanner.Text())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

// MatchTokensBytes scans byte data into tokens and passes them to match.
func MatchTokensBytes(data []byte, opts TokenOptions, match MatchTokenFunc) (bool, error) {
	return MatchTokensReader(bytes.NewReader(data), opts, match)
}

// MatchTokensString scans string data into tokens and passes them to match.
func MatchTokensString(data string, opts TokenOptions, match MatchTokenFunc) (bool, error) {
	return MatchTokensReader(strings.NewReader(data), opts, match)
}

// InspectLinesReader scans src line by line and passes each line to inspect.
func InspectLinesReader(src io.Reader, inspect InspectTokenFunc) error {
	return InspectTokensReader(src, TokenOptions{Split: bufio.ScanLines}, inspect)
}

// InspectLinesBytes scans byte data line by line and passes each line to
// inspect.
func InspectLinesBytes(data []byte, inspect InspectTokenFunc) error {
	return InspectTokensBytes(data, TokenOptions{Split: bufio.ScanLines}, inspect)
}

// InspectLinesString scans string data line by line and passes each line to
// inspect.
func InspectLinesString(data string, inspect InspectTokenFunc) error {
	return InspectTokensString(data, TokenOptions{Split: bufio.ScanLines}, inspect)
}

// MatchLinesReader scans src line by line and passes each line to match.
func MatchLinesReader(src io.Reader, match MatchTokenFunc) (bool, error) {
	return MatchTokensReader(src, TokenOptions{Split: bufio.ScanLines}, match)
}

// MatchLinesBytes scans byte data line by line and passes each line to match.
func MatchLinesBytes(data []byte, match MatchTokenFunc) (bool, error) {
	return MatchTokensBytes(data, TokenOptions{Split: bufio.ScanLines}, match)
}

// MatchLinesString scans string data line by line and passes each line to
// match.
func MatchLinesString(data string, match MatchTokenFunc) (bool, error) {
	return MatchTokensString(data, TokenOptions{Split: bufio.ScanLines}, match)
}

// Inspect opens the file read-only and passes its streamed content to inspect.
func (f File) Inspect(ctx Context, inspect InspectFunc) error {
	if inspect == nil {
		return fmt.Errorf("inspect must not be nil")
	}

	handle, err := f.OpenRead(ctx, OpenExisting)
	if err != nil {
		return err
	}

	inspectErr := inspect(handle)
	closeErr := handle.Close()
	if inspectErr != nil {
		return inspectErr
	}
	return closeErr
}

// Match opens the file read-only and passes its streamed content to match.
func (f File) Match(ctx Context, match MatchFunc) (bool, error) {
	if match == nil {
		return false, fmt.Errorf("match must not be nil")
	}

	handle, err := f.OpenRead(ctx, OpenExisting)
	if err != nil {
		return false, err
	}

	matched, matchErr := match(handle)
	closeErr := handle.Close()
	if matchErr != nil {
		return false, matchErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	return matched, nil
}

// MatchIfExists matches file content when the file exists and returns false
// without error when it is missing.
func (f File) MatchIfExists(ctx Context, match MatchFunc) (bool, error) {
	matched, err := f.Match(ctx, match)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return matched, err
}

// InspectTokens opens the file read-only, scans its content into tokens, and
// passes them to inspect.
func (f File) InspectTokens(ctx Context, opts TokenOptions, inspect InspectTokenFunc) error {
	if inspect == nil {
		return fmt.Errorf("inspect token callback must not be nil")
	}
	if err := opts.validate(); err != nil {
		return err
	}

	return f.Inspect(ctx, func(src io.Reader) error {
		return InspectTokensReader(src, opts, inspect)
	})
}

// MatchTokens opens the file read-only, scans its content into tokens, and
// passes them to match.
func (f File) MatchTokens(ctx Context, opts TokenOptions, match MatchTokenFunc) (bool, error) {
	if match == nil {
		return false, fmt.Errorf("match token callback must not be nil")
	}
	if err := opts.validate(); err != nil {
		return false, err
	}

	return f.Match(ctx, func(src io.Reader) (bool, error) {
		return MatchTokensReader(src, opts, match)
	})
}

// MatchTokensIfExists matches tokenized file content when the file exists and
// returns false without error when it is missing.
func (f File) MatchTokensIfExists(ctx Context, opts TokenOptions, match MatchTokenFunc) (bool, error) {
	matched, err := f.MatchTokens(ctx, opts, match)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return matched, err
}

// InspectLines opens the file read-only, scans its content line by line, and
// passes each line to inspect.
func (f File) InspectLines(ctx Context, inspect InspectTokenFunc) error {
	return f.InspectTokens(ctx, TokenOptions{Split: bufio.ScanLines}, inspect)
}

// MatchLines opens the file read-only, scans its content line by line, and
// passes each line to match.
func (f File) MatchLines(ctx Context, match MatchTokenFunc) (bool, error) {
	return f.MatchTokens(ctx, TokenOptions{Split: bufio.ScanLines}, match)
}

// MatchLinesIfExists matches line-oriented file content when the file exists
// and returns false without error when it is missing.
func (f File) MatchLinesIfExists(ctx Context, match MatchTokenFunc) (bool, error) {
	return f.MatchTokensIfExists(ctx, TokenOptions{Split: bufio.ScanLines}, match)
}

func (opts TokenOptions) scanner(src io.Reader) (*bufio.Scanner, error) {
	scanner := bufio.NewScanner(src)
	if opts.Split != nil {
		scanner.Split(opts.Split)
	}
	if opts.InitialBuffer != nil || opts.MaxTokenSize > 0 {
		max := opts.MaxTokenSize
		if max == 0 {
			max = bufio.MaxScanTokenSize
		}
		scanner.Buffer(opts.InitialBuffer, max)
	}
	return scanner, nil
}

func (opts TokenOptions) validate() error {
	if opts.MaxTokenSize < 0 {
		return fmt.Errorf("max token size must not be negative")
	}
	return nil
}
