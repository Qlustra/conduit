package layout

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReportRenderTreeEmpty(t *testing.T) {
	var report Report

	if got := report.RenderTree(); got != "" {
		t.Fatalf("RenderTree() = %q, want empty string", got)
	}
}

func TestReportRenderTreeNestedSiblings(t *testing.T) {
	var report Report
	report.Record(Entry{
		Op:     OpEnsure,
		Path:   filepath.Join("workspace"),
		Result: EnsureEnsured,
	})
	report.Record(Entry{
		Op:     OpLoad,
		Path:   filepath.Join("workspace", "services"),
		Result: LoadTraversed,
	})
	report.Record(Entry{
		Op:     OpEnsure,
		Path:   filepath.Join("workspace", "services", "api"),
		Result: EnsureEnsured,
	})
	report.Record(Entry{
		Op:     OpLoad,
		Path:   filepath.Join("workspace", "services", "api", "config.json"),
		Result: LoadLoaded,
	})
	report.Record(Entry{
		Op:     OpLoad,
		Path:   filepath.Join("workspace", "services", "worker", "config.json"),
		Result: LoadMissing,
	})

	want := strings.Join([]string{
		"`- workspace [ensure:ensured]",
		"   `- services [load:traversed]",
		"      |- api [ensure:ensured]",
		"      |  `- config.json [load:loaded]",
		"      `- worker",
		"         `- config.json [load:missing]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}

func TestReportRenderTreeMultipleEntriesSamePath(t *testing.T) {
	var report Report
	path := filepath.Join("workspace", "config.json")

	report.Record(Entry{
		Op:     OpLoad,
		Path:   path,
		Result: LoadLoaded,
	})
	report.Record(Entry{
		Op:     OpSync,
		Path:   path,
		Result: SyncWritten,
	})

	want := strings.Join([]string{
		"`- workspace",
		"   `- config.json [load:loaded] [sync:written]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}

func TestReportRenderTreeAbsolutePaths(t *testing.T) {
	var report Report
	firstPath := filepath.Join(string(filepath.Separator), "srv", "workspace", "config.json")
	secondPath := filepath.Join(string(filepath.Separator), "var", "log", "app.log")

	report.Record(Entry{
		Op:     OpScan,
		Path:   firstPath,
		Result: ScanPresent,
	})
	report.Record(Entry{
		Op:     OpScan,
		Path:   secondPath,
		Result: ScanMissing,
	})

	want := strings.Join([]string{
		"`- /",
		"   |- srv",
		"   |  `- workspace",
		"   |     `- config.json [scan:present]",
		"   `- var",
		"      `- log",
		"         `- app.log [scan:missing]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}

func TestReportRenderTreeTrimsSharedAbsolutePrefix(t *testing.T) {
	var report Report
	base := filepath.Join(string(filepath.Separator), "tmp", "workspace", "project", "run")

	report.Record(Entry{
		Op:     OpSync,
		Path:   filepath.Join(base, "codebase"),
		Result: SyncNotApplicable,
	})
	report.Record(Entry{
		Op:     OpSync,
		Path:   filepath.Join(base, "codebase", ".gitignore"),
		Result: SyncWritten,
	})
	report.Record(Entry{
		Op:     OpSync,
		Path:   filepath.Join(base, "codebase", "source", "apps", "go"),
		Result: SyncTraversed,
	})

	want := strings.Join([]string{
		"`- run",
		"   `- codebase [sync:not_applicable]",
		"      |- .gitignore [sync:written]",
		"      `- source",
		"         `- apps",
		"            `- go [sync:traversed]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}
