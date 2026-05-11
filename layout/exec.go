package layout

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
)

var (
	errExecPathEmpty        = errors.New("exec path must not be empty")
	errExecNilContext       = errors.New("context must not be nil")
	errExecInterpreterEmpty = errors.New("interpreter must contain a command")
)

type Exec struct {
	File
}

func NewExec(path string) Exec {
	return Exec{File: NewFile(path)}
}

func (e *Exec) ComposePath(path string) {
	e.File = NewFile(path)
}

func (e Exec) Ensure(ctx Context) error {
	return e.EnsureExecutable(ctx)
}

func (e Exec) EnsureExecutable(ctx Context) error {
	ctx.FileMode = e.executableMode(ctx)
	if err := e.File.Ensure(ctx); err != nil {
		return err
	}
	return os.Chmod(e.Path(), ctx.FileMode)
}

func (e Exec) IsExecutable() bool {
	info, err := os.Stat(e.Path())
	if err != nil {
		return false
	}
	return info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

type RunOptions struct {
	Dir         string
	Args        []string
	Env         []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interpreter []string
}

func (e Exec) Command(ctx context.Context, opts RunOptions) *exec.Cmd {
	cmd, err := e.command(ctx, opts)
	if err != nil {
		return &exec.Cmd{Path: e.Path(), Err: err}
	}
	return cmd
}

func (e Exec) command(ctx context.Context, opts RunOptions) (*exec.Cmd, error) {
	if ctx == nil {
		return nil, errExecNilContext
	}
	if e.Path() == "" {
		return nil, errExecPathEmpty
	}

	var cmd *exec.Cmd

	if len(opts.Interpreter) > 0 {
		if opts.Interpreter[0] == "" {
			return nil, errExecInterpreterEmpty
		}
		argv := make([]string, 0, len(opts.Interpreter)+1+len(opts.Args))
		argv = append(argv, opts.Interpreter...)
		argv = append(argv, e.Path())
		argv = append(argv, opts.Args...)
		cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, e.Path(), opts.Args...)
	}

	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	return cmd, nil
}

func (e Exec) Run(ctx context.Context, opts RunOptions) error {
	return e.Command(ctx, opts).Run()
}

func (e Exec) Output(ctx context.Context, opts RunOptions) ([]byte, error) {
	cmd := e.Command(ctx, opts)
	if opts.Stdout != nil || opts.Stderr != nil {
		return nil, errors.New("Output cannot be used with Stdout or Stderr set")
	}
	return cmd.Output()
}

func (e Exec) CombinedOutput(ctx context.Context, opts RunOptions) ([]byte, error) {
	cmd := e.Command(ctx, opts)
	if opts.Stdout != nil || opts.Stderr != nil {
		return nil, errors.New("CombinedOutput cannot be used with Stdout or Stderr set")
	}
	return cmd.CombinedOutput()
}

// Ensure

func (e Exec) EnsureDeep(ctx Context) error {
	return e.Ensure(ctx)
}

func (e Exec) executableMode(ctx Context) os.FileMode {
	mode := ctx.ExecMode
	if mode == 0 {
		mode = ctx.FileMode
	}
	if mode&0o111 == 0 {
		mode |= 0o111
	}
	return mode
}

// Report

func (e Exec) ensureDeepReport(ctx Context) error {
	return reportEnsure(ctx, e.Path(), func() error {
		return e.Ensure(ctx)
	})
}

func (e Exec) loadReport(ctx Context) error {
	return reportLoad(ctx, e.Path(), func() (ResultCode, error) {
		return LoadNotApplicable, nil
	})
}

func (e Exec) discoverReport(ctx Context) error {
	return reportDiscover(ctx, e.Path(), func() (ResultCode, error) {
		return DiscoverNotApplicable, nil
	})
}

func (e Exec) scanReport(ctx Context) error {
	return reportScan(ctx, e.Path(), func() (ResultCode, error) {
		return ScanNotApplicable, nil
	})
}

func (e Exec) syncReport(ctx Context) error {
	return reportSync(ctx, e.Path(), func() (ResultCode, error) {
		return SyncNotApplicable, nil
	})
}
