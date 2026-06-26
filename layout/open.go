package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

// OpenPolicy controls the creation flags added by File open helpers.
type OpenPolicy uint8

const (
	// OpenExisting adds no creation flags. The file must already exist unless the
	// remaining open flags allow otherwise.
	OpenExisting OpenPolicy = iota

	// OpenOrCreate adds os.O_CREATE.
	OpenOrCreate

	// OpenOrCreateExclusive adds os.O_CREATE and os.O_EXCL.
	OpenOrCreateExclusive
)

// OpenRead opens the file read-only.
func (f File) OpenRead(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_RDONLY|flags, ctx.FileMode)
}

// OpenWrite opens the file write-only without truncating or appending.
func (f File) OpenWrite(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_WRONLY|flags, ctx.FileMode)
}

// OpenRewrite opens the file write-only and truncates it.
func (f File) OpenRewrite(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_WRONLY|os.O_TRUNC|flags, ctx.FileMode)
}

// OpenReadWrite opens the file for reading and writing without truncating or
// appending.
func (f File) OpenReadWrite(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_RDWR|flags, ctx.FileMode)
}

// OpenReadRewrite opens the file for reading and writing and truncates it.
func (f File) OpenReadRewrite(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_RDWR|os.O_TRUNC|flags, ctx.FileMode)
}

// OpenAppend opens the file write-only and appends all writes to the end.
func (f File) OpenAppend(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_WRONLY|os.O_APPEND|flags, ctx.FileMode)
}

// OpenReadAppend opens the file for reading and writing and appends all writes
// to the end.
func (f File) OpenReadAppend(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_RDWR|os.O_APPEND|flags, ctx.FileMode)
}

// OpenRewriteAppend opens the file write-only, truncates it, and appends all
// writes to the end of the resulting file.
func (f File) OpenRewriteAppend(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_WRONLY|os.O_APPEND|os.O_TRUNC|flags, ctx.FileMode)
}

// OpenReadRewriteAppend opens the file for reading and writing, truncates it,
// and appends all writes to the end of the resulting file.
func (f File) OpenReadRewriteAppend(ctx Context, op OpenPolicy) (*os.File, error) {
	flags, err := op.openFlags()
	if err != nil {
		return nil, err
	}
	return f.OpenFile(ctx, os.O_RDWR|os.O_APPEND|os.O_TRUNC|flags, ctx.FileMode)
}

// OpenFile opens the file with explicit os.OpenFile flags and mode.
//
// When flag includes os.O_CREATE, OpenFile creates parent directories using
// ctx.DirMode before opening the file. Existing symlink leaves are rejected;
// symlink parents are governed by ctx.PathSafetyPolicy.
func (f File) OpenFile(ctx Context, flag int, perm os.FileMode) (*os.File, error) {
	if err := guardPathMutation(f.Path(), ctx.pathSafetyPolicy(), expectFile); err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		if err := os.MkdirAll(filepath.Dir(f.Path()), ctx.DirMode); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(f.Path(), flag, perm)
}

func (p OpenPolicy) openFlags() (int, error) {
	switch p {
	case OpenExisting:
		return 0, nil
	case OpenOrCreate:
		return os.O_CREATE, nil
	case OpenOrCreateExclusive:
		return os.O_CREATE | os.O_EXCL, nil
	default:
		return 0, fmt.Errorf("unsupported open policy %d", p)
	}
}
