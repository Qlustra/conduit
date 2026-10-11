package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestExecEnsureLayoutUsesExecModeForExistingFiles(t *testing.T) {
	type root struct {
		Script Exec `layout:"exec" path:"bin/script.sh"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	if err := os.MkdirAll(filepath.Dir(layout.Script.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(layout.Script.Path(), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	ctx := Context{DirMode: 0o755, FileMode: 0o644, ExecMode: 0o751}
	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	info, err := os.Stat(layout.Script.Path())
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o751 {
		t.Fatalf("mode = %v, want %v", got, os.FileMode(0o751))
	}
}

func TestLayoutEnsureExecPolicyIsIndependentOfRawFiles(t *testing.T) {
	type tree struct {
		Script Exec `layout:"exec" path:"bin/script.sh"`
		Raw    File `layout:"file" path:"raw.txt"`
	}
	for _, test := range []struct {
		name        string
		policy      EnsurePolicy
		script, raw bool
	}{
		{"execs", EnsureExecs, true, false},
		{"files", EnsureFiles, false, true},
		{"none", EnsureNone, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager, model := composeTree[tree](t, t.TempDir())
			var report Report
			ctx := DefaultContext
			ctx.EnsurePolicy = test.policy
			ctx.ExecMode = 0o750
			ctx.Reporter = &report
			if _, err := manager.Ensure(ctx); err != nil {
				t.Fatal(err)
			}
			for _, node := range []struct {
				path string
				want bool
			}{{model.Script.Path(), test.script}, {model.Raw.Path(), test.raw}} {
				info, err := os.Stat(node.path)
				if node.want && err != nil {
					t.Fatalf("expected %q: %v", node.path, err)
				}
				if !node.want && !os.IsNotExist(err) {
					t.Fatalf("unexpected materialization %q: %v", node.path, err)
				}
				if node.want && node.path == model.Script.Path() && info.Mode().Perm() != 0o750 {
					t.Fatalf("script mode = %v", info.Mode().Perm())
				}
			}
			scriptCode, rawCode := EnsureSkippedPolicy, EnsureSkippedPolicy
			if test.script {
				scriptCode = EnsureEnsured
			}
			if test.raw {
				rawCode = EnsureEnsured
			}
			assertEntries(t, report.Entries(), []Entry{{Op: OpEnsure, Path: model.Script.Path(), Result: scriptCode}, {Op: OpEnsure, Path: model.Raw.Path(), Result: rawCode}})
		})
	}
}

func TestLayoutReportsRawExecAsNotApplicable(t *testing.T) {
	type tree struct {
		Script Exec `layout:"exec" path:"bin/script.sh"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	for _, test := range []struct {
		name       string
		operation  func(Context) (ResultCode, error)
		op         Operation
		rootResult ResultCode
		leafResult ResultCode
	}{
		{"scan", manager.Scan, OpScan, ScanTraversed, ScanNotApplicable},
		{"load", manager.Load, OpLoad, LoadTraversed, LoadNotApplicable},
		{"discover", manager.Discover, OpDiscover, DiscoverTraversed, DiscoverNotApplicable},
		{"sync", manager.Sync, OpSync, SyncTraversed, SyncNotApplicable},
	} {
		t.Run(test.name, func(t *testing.T) {
			var report Report
			ctx := DefaultContext
			ctx.Reporter = &report
			code, err := test.operation(ctx)
			if err != nil || code != test.rootResult {
				t.Fatalf("%s() = %v, %v; want %v, nil", test.name, code, err, test.rootResult)
			}
			assertEntries(t, report.Entries(), []Entry{
				{Op: test.op, Path: model.Script.Path(), Result: test.leafResult},
			})
		})
	}
}
