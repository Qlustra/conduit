package layout

import (
	"context"
	"os/exec"
)

// ExecAPI is the complete exported API of Exec and its embedded File.
type ExecAPI interface {
	FileAPI
	CombinedOutput(ctx context.Context, opts RunOptions) ([]byte, error)
	Command(ctx context.Context, opts RunOptions) *exec.Cmd
	EnsureExecutable(ctx Context) error
	Output(ctx context.Context, opts RunOptions) ([]byte, error)
	Run(ctx context.Context, opts RunOptions) error
}
