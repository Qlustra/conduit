package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkComposeAndTypedTargets(t *testing.T) {
	type project struct {
		Root   Dir      `layout:"dir" path:"."`
		Readme FileLink `layout:"content" path:"README.link"`
		Assets DirLink  `layout:"content" path:"assets.link"`
	}

	base := t.TempDir()

	_, layout := composeTree[project](t, base)

	layout.Readme.SetTarget("docs/README.md")
	layout.Assets.SetTarget("../shared/assets")

	if got := layout.Readme.Path(); got != filepath.Join(base, "README.link") {
		t.Fatalf("Readme.Path() = %q", got)
	}
	if got, ok := layout.Readme.DeclaredPath(); !ok || got != "README.link" {
		t.Fatalf("Readme.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, "README.link")
	}
	if got, ok := layout.Assets.ComposedRelativePath(); !ok || got != "assets.link" {
		t.Fatalf("Assets.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, "assets.link")
	}

	readmeTarget, ok := layout.Readme.TargetFile()
	if !ok {
		t.Fatal("Readme.TargetFile() ok = false, want true")
	}
	if got := readmeTarget.Path(); got != filepath.Join(base, "docs", "README.md") {
		t.Fatalf("Readme.TargetFile().Path() = %q, want %q", got, filepath.Join(base, "docs", "README.md"))
	}

	assetsTarget, ok := layout.Assets.TargetDir()
	if !ok {
		t.Fatal("Assets.TargetDir() ok = false, want true")
	}
	if got := assetsTarget.Path(); got != filepath.Join(filepath.Dir(base), "shared", "assets") {
		t.Fatalf("Assets.TargetDir().Path() = %q, want %q", got, filepath.Join(filepath.Dir(base), "shared", "assets"))
	}
}

func TestEnsureLayoutDoesNotMaterializeLinks(t *testing.T) {
	type project struct {
		Root   Dir  `layout:"dir" path:"."`
		Config File `layout:"file" path:"config.yaml"`
		Link   Link `layout:"content" path:"config.link"`
	}

	base := t.TempDir()

	manager, layout := composeTree[project](t, base)

	if _, err := manager.Ensure(DefaultContext); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	if _, err := os.Stat(layout.Config.Path()); err != nil {
		t.Fatalf("os.Stat(config) error = %v", err)
	}
	if _, err := os.Lstat(layout.Link.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Lstat(link) error = %v, want not-exist", err)
	}
}
