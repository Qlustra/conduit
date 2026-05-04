package layout

import (
	"path/filepath"
	"testing"
)

func TestDirPathHelpers(t *testing.T) {
	dir := NewDir(filepath.Join("workspace", "services", "api.v2"))

	if got := dir.Base(); got != "api.v2" {
		t.Fatalf("Base() = %q, want %q", got, "api.v2")
	}
	if got := dir.Stem(); got != "api" {
		t.Fatalf("Stem() = %q, want %q", got, "api")
	}
}

func TestFilePathHelpers(t *testing.T) {
	file := NewFile(filepath.Join("workspace", "configs", "archive.tar.gz"))

	if got := file.Base(); got != "archive.tar.gz" {
		t.Fatalf("Base() = %q, want %q", got, "archive.tar.gz")
	}
	if got := file.Ext(); got != ".gz" {
		t.Fatalf("Ext() = %q, want %q", got, ".gz")
	}
	if got := file.Stem(); got != "archive.tar" {
		t.Fatalf("Stem() = %q, want %q", got, "archive.tar")
	}
}

func TestPathHelpersPreserveDotfilesAndExtensionlessNames(t *testing.T) {
	file := NewFile(filepath.Join("workspace", ".env"))
	dir := NewDir(".config")

	if got := file.Ext(); got != "" {
		t.Fatalf("Ext() for dotfile = %q, want empty", got)
	}
	if got := file.Stem(); got != ".env" {
		t.Fatalf("Stem() for dotfile = %q, want %q", got, ".env")
	}
	if got := dir.Stem(); got != ".config" {
		t.Fatalf("Stem() for dot-dir = %q, want %q", got, ".config")
	}
}

func TestComposedPathsTrackComposeBaseAcrossTree(t *testing.T) {
	type service struct {
		Root   Dir  `layout:"."`
		Config File `layout:"config.yaml"`
		Logs   Dir  `layout:"logs"`
	}

	type workspace struct {
		Root     Dir            `layout:"."`
		Services Slot[*service] `layout:"services"`
	}

	var ws workspace
	if err := Compose("workspace", &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	base, ok := ws.Root.ComposedBaseDir()
	if !ok {
		t.Fatal("Root.ComposedBaseDir() ok = false, want true")
	}
	if got := base.Path(); got != "workspace" {
		t.Fatalf("Root.ComposedBaseDir().Path() = %q, want %q", got, "workspace")
	}

	if got, ok := ws.Root.ComposedRelativePath(); !ok || got != "." {
		t.Fatalf("Root.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, ".")
	}
	if got, ok := ws.Root.JoinComposedPath("bin", "tool"); !ok || got != filepath.Join("bin", "tool") {
		t.Fatalf("Root.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("bin", "tool"))
	}

	svc := ws.Services.MustAt("api")

	if got, ok := ws.Services.ComposedRelativePath(); !ok || got != "services" {
		t.Fatalf("Services.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, "services")
	}
	if got, ok := ws.Services.JoinComposedPath("api"); !ok || got != filepath.Join("services", "api") {
		t.Fatalf("Services.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api"))
	}

	if got, ok := svc.Root.ComposedRelativePath(); !ok || got != filepath.Join("services", "api") {
		t.Fatalf("service.Root.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api"))
	}
	if got, ok := svc.Config.ComposedRelativePath(); !ok || got != filepath.Join("services", "api", "config.yaml") {
		t.Fatalf("service.Config.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api", "config.yaml"))
	}

	logs := svc.Root.Dir("logs")
	if got, ok := logs.ComposedRelativePath(); !ok || got != filepath.Join("services", "api", "logs") {
		t.Fatalf("derived logs.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api", "logs"))
	}
	if got, ok := svc.Config.JoinComposedPath("bak"); !ok || got != filepath.Join("services", "api", "config.yaml", "bak") {
		t.Fatalf("Config.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api", "config.yaml", "bak"))
	}
}

func TestComposedPathsAreUnavailableWithoutComposition(t *testing.T) {
	dir := NewDir("workspace")
	file := NewFile(filepath.Join("workspace", "config.yaml"))
	slot := NewSlot[*struct{}](dir)

	if _, ok := dir.ComposedBaseDir(); ok {
		t.Fatal("Dir.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := dir.ComposedRelativePath(); ok {
		t.Fatal("Dir.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := dir.JoinComposedPath("child"); ok {
		t.Fatal("Dir.JoinComposedPath() ok = true, want false")
	}

	if _, ok := file.ComposedBaseDir(); ok {
		t.Fatal("File.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := file.ComposedRelativePath(); ok {
		t.Fatal("File.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := file.JoinComposedPath("child"); ok {
		t.Fatal("File.JoinComposedPath() ok = true, want false")
	}

	if _, ok := slot.ComposedBaseDir(); ok {
		t.Fatal("Slot.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := slot.ComposedRelativePath(); ok {
		t.Fatal("Slot.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := slot.JoinComposedPath("child"); ok {
		t.Fatal("Slot.JoinComposedPath() ok = true, want false")
	}
}
