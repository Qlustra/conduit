package layout

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExecEnsureAppliesExecutableMode(t *testing.T) {
	execFile := NewExec(filepath.Join(t.TempDir(), "bin", "tool"))

	if err := os.MkdirAll(filepath.Dir(execFile.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(execFile.Path(), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	ctx := Context{DirMode: 0o755, FileMode: 0o644, ExecMode: 0o750}
	if err := execFile.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	info, err := os.Stat(execFile.Path())
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o750 {
		t.Fatalf("mode = %v, want %v", got, os.FileMode(0o750))
	}
	if !execFile.IsExecutable() {
		t.Fatalf("IsExecutable() = false, want true")
	}
}

func TestExecEnsureDeepUsesExecModeForExistingFiles(t *testing.T) {
	type root struct {
		Script Exec `layout:"bin/script.sh"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.Script.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(layout.Script.Path(), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	ctx := Context{DirMode: 0o755, FileMode: 0o644, ExecMode: 0o751}
	if _, err := EnsureDeep(&layout, ctx); err != nil {
		t.Fatalf("EnsureDeep() error = %v", err)
	}

	info, err := os.Stat(layout.Script.Path())
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o751 {
		t.Fatalf("mode = %v, want %v", got, os.FileMode(0o751))
	}
}

func TestExecOutputWithInterpreter(t *testing.T) {
	execFile := NewExec(filepath.Join(t.TempDir(), "script.sh"))

	if err := os.WriteFile(execFile.Path(), []byte("printf '%s|%s' \"$GREETING\" \"$1\""), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	out, err := execFile.Output(context.Background(), RunOptions{
		Args:        []string{"world"},
		Env:         []string{"GREETING=hello"},
		Interpreter: []string{"sh"},
	})
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if got := string(out); got != "hello|world" {
		t.Fatalf("Output() = %q, want %q", got, "hello|world")
	}
}

func TestExecOutputRejectsExplicitWriters(t *testing.T) {
	execFile := NewExec(filepath.Join(t.TempDir(), "script.sh"))

	if _, err := execFile.Output(context.Background(), RunOptions{Stdout: io.Discard}); err == nil {
		t.Fatal("Output() error = nil, want non-nil")
	}
	if _, err := execFile.CombinedOutput(context.Background(), RunOptions{Stderr: io.Discard}); err == nil {
		t.Fatal("CombinedOutput() error = nil, want non-nil")
	}
}

func TestExecRunRejectsInvalidConfiguration(t *testing.T) {
	execFile := NewExec(filepath.Join(t.TempDir(), "script.sh"))

	if err := execFile.Run(nil, RunOptions{}); !errors.Is(err, errExecNilContext) {
		t.Fatalf("Run(nil) error = %v, want %v", err, errExecNilContext)
	}

	err := execFile.Run(context.Background(), RunOptions{Interpreter: []string{""}})
	if !errors.Is(err, errExecInterpreterEmpty) {
		t.Fatalf("Run(empty interpreter) error = %v, want %v", err, errExecInterpreterEmpty)
	}
}
