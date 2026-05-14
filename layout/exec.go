package layout

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var (
	errExecPathEmpty        = errors.New("exec path must not be empty")
	errExecNilContext       = errors.New("context must not be nil")
	errExecInterpreterEmpty = errors.New("interpreter must contain a command")
)

// Exec is a File with executable creation and process-launch helpers.
type Exec struct {
	File
}

// NewExec returns a standalone executable file handle for path.
func NewExec(path string) Exec {
	return Exec{File: NewFile(path)}
}

// ComposePath binds the executable handle to path.
func (e *Exec) ComposePath(path string) {
	e.File = NewFile(path)
}

// Ensure creates the file and ensures executable permissions.
func (e Exec) Ensure(ctx Context) error {
	return e.EnsureExecutable(ctx)
}

// EnsureExecutable creates the file if needed and applies executable mode.
//
// When ctx.ExecMode is zero, FileMode is used with execute bits added.
func (e Exec) EnsureExecutable(ctx Context) error {
	ctx.FileMode = e.executableMode(ctx)
	if err := e.File.Ensure(ctx); err != nil {
		return err
	}
	return os.Chmod(e.Path(), ctx.FileMode)
}

// IsExecutable reports whether Path currently points to an executable regular
// file.
func (e Exec) IsExecutable() bool {
	return e.File.IsExecutable()
}

// RunOptions configures process execution for Exec helpers.
type RunOptions struct {
	// Dir sets the process working directory.
	Dir string

	// Args supplies argv after the executable path.
	Args []string

	// Env appends environment variables to the current process environment.
	Env []string

	// Stdin connects process standard input.
	Stdin io.Reader

	// Stdout connects process standard output.
	Stdout io.Writer

	// Stderr connects process standard error.
	Stderr io.Writer

	// Interpreter, when set, runs the managed file as an argument to the named
	// interpreter command.
	Interpreter []string
}

// Command builds an exec.Cmd for the managed file.
//
// Invalid configuration is reflected in cmd.Err rather than being returned
// separately.
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

// Run executes the managed file using opts.
func (e Exec) Run(ctx context.Context, opts RunOptions) error {
	return e.Command(ctx, opts).Run()
}

// Output executes the managed file and captures standard output.
//
// It returns an error if opts.Stdout or opts.Stderr is already set.
func (e Exec) Output(ctx context.Context, opts RunOptions) ([]byte, error) {
	cmd := e.Command(ctx, opts)
	if opts.Stdout != nil || opts.Stderr != nil {
		return nil, errors.New("Output cannot be used with Stdout or Stderr set")
	}
	return cmd.Output()
}

// CombinedOutput executes the managed file and captures combined standard
// output and standard error.
//
// It returns an error if opts.Stdout or opts.Stderr is already set.
func (e Exec) CombinedOutput(ctx context.Context, opts RunOptions) ([]byte, error) {
	cmd := e.Command(ctx, opts)
	if opts.Stdout != nil || opts.Stderr != nil {
		return nil, errors.New("CombinedOutput cannot be used with Stdout or Stderr set")
	}
	return cmd.CombinedOutput()
}

// Ensure

// EnsureDeep ensures the executable node during deep traversal.
func (e Exec) EnsureDeep(ctx Context) (ResultCode, error) {
	err := e.Ensure(ctx)
	result := EnsureEnsured
	if err != nil {
		result = EnsureFailed
	}
	return result, err
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

// Validate

// Validate reports an error when Path exists but is not an executable regular
// file.
func (e Exec) Validate() error {
	if e.Path() == "" {
		return nil
	}

	info, err := os.Stat(e.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("path %s is not an executable regular file", e.Path())
	}

	return nil
}
