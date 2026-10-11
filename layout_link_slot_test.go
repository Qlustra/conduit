package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkSlotComposeMemberCachesComposedItems(t *testing.T) {
	type tree struct {
		Members LinkSlot[*Link] `layout:"links" path:"context"`
	}
	_, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	first, err := slot.ComposeMember("README")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}
	second, err := slot.ComposeMember("README")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}

	if first != second {
		t.Fatalf("ComposeMember() returned different cached items for the same key")
	}
	if got := first.Path(); got != filepath.Join(slot.Path(), "README") {
		t.Fatalf("Path() = %q", got)
	}
	if keys := slot.Keys(); len(keys) != 1 || keys[0] != "README" {
		t.Fatalf("Keys() = %v, want [README]", keys)
	}
}

func TestLinkSlotDeleteRemovesSymlinkAndCache(t *testing.T) {
	type tree struct {
		Members LinkSlot[*Link] `layout:"links" path:"context"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	item, err := slot.ComposeMember("README")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}
	item.SetTarget("docs/README.md")
	slot.Put("README", item)

	if _, err := manager.Sync(DefaultContext); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if err := slot.Delete("README", DefaultContext); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := os.Lstat(item.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Lstat(item path) err = %v, want not-exist", err)
	}
	if _, ok := slot.Get("README"); ok {
		t.Fatalf("Get(\"README\") ok = true after Delete(), want false")
	}
}

func TestLinkSlotHasAndRequireUseSymlinkSemantics(t *testing.T) {
	type tree struct {
		Members LinkSlot[*Link] `layout:"links" path:"context"`
	}
	_, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	if slot.Has("missing") {
		t.Fatal("Has(missing) = true, want false")
	}
	if _, err := slot.Require("missing"); err == nil {
		t.Fatal("Require() error = nil, want error for missing symlink")
	}

	if err := os.MkdirAll(slot.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(slot root) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(slot.Path(), "plain.txt"), []byte("plain"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if slot.Has("plain.txt") {
		t.Fatal("Has(plain.txt) = true, want false for regular file")
	}
	if _, err := slot.Require("plain.txt"); err == nil {
		t.Fatal("Require(plain.txt) error = nil, want non-symlink error")
	}

	if err := os.Symlink("docs/README.md", filepath.Join(slot.Path(), "README")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}
	if !slot.Has("README") {
		t.Fatal("Has(README) = false, want true")
	}

	item, err := slot.Require("README")
	if err != nil {
		t.Fatalf("Require(README) error = %v", err)
	}
	if got := item.Path(); got != filepath.Join(slot.Path(), "README") {
		t.Fatalf("Path() = %q", got)
	}
}

func TestLoadLayoutLoadsLinkSlotEntriesIncludingDirectoryTargets(t *testing.T) {
	type root struct {
		Context LinkSlot[*Link] `layout:"links" path:"context"`
	}

	base := t.TempDir()
	manager, layout := composeTree[root](t, base)

	if err := os.MkdirAll(filepath.Join(base, "context"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(context) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "shared", "assets"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(shared assets) error = %v", err)
	}
	if err := os.Symlink("../shared/assets", filepath.Join(base, "context", "assets")); err != nil {
		t.Fatalf("os.Symlink(assets) error = %v", err)
	}
	if err := os.Symlink("docs/README.md", filepath.Join(base, "context", "README")); err != nil {
		t.Fatalf("os.Symlink(README) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "context", "plain.txt"), []byte("plain"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(plain) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "context", "nested"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(nested) error = %v", err)
	}

	if _, err := manager.Load(DefaultContext); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if keys := layout.Context.Keys(); len(keys) != 2 || keys[0] != "README" || keys[1] != "assets" {
		t.Fatalf("Context.Keys() = %v, want [README assets]", keys)
	}

	readme, ok := layout.Context.Get("README")
	if !ok {
		t.Fatalf("Context.Get(\"README\") = false, want true")
	}
	if got := readme.MustTarget(); got != "docs/README.md" {
		t.Fatalf("readme.MustTarget() = %q, want %q", got, "docs/README.md")
	}
	if got := readme.MemoryState(); got != MemoryLoaded {
		t.Fatalf("readme.MemoryState() = %v, want %v", got, MemoryLoaded)
	}

	assets, ok := layout.Context.Get("assets")
	if !ok {
		t.Fatalf("Context.Get(\"assets\") = false, want true")
	}
	if got := assets.MustTarget(); got != "../shared/assets" {
		t.Fatalf("assets.MustTarget() = %q, want %q", got, "../shared/assets")
	}
	if _, ok := layout.Context.Get("plain.txt"); ok {
		t.Fatalf("Context.Get(\"plain.txt\") = true, want false")
	}
	if _, ok := layout.Context.Get("nested"); ok {
		t.Fatalf("Context.Get(\"nested\") = true, want false")
	}
}

func TestDiscoverLayoutDiscoversLinkSlotEntriesWithoutReplacingMemory(t *testing.T) {
	type root struct {
		Context LinkSlot[*Link] `layout:"links" path:"context"`
	}

	base := t.TempDir()
	manager, layout := composeTree[root](t, base)

	cached, err := layout.Context.ComposeMember("README")
	if err != nil {
		t.Fatalf("Context.ComposeMember() error = %v", err)
	}
	cached.SetTarget("cached/README.md")
	layout.Context.Put("README", cached)

	if err := os.MkdirAll(filepath.Join(base, "context"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(context) error = %v", err)
	}
	if err := os.Symlink("docs/README.md", filepath.Join(base, "context", "README")); err != nil {
		t.Fatalf("os.Symlink(README) error = %v", err)
	}
	if err := os.Symlink("../shared/assets", filepath.Join(base, "context", "assets")); err != nil {
		t.Fatalf("os.Symlink(assets) error = %v", err)
	}

	if _, err := manager.Discover(DefaultContext); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	cached, ok := layout.Context.Get("README")
	if !ok {
		t.Fatalf("Context.Get(\"README\") = false, want true")
	}
	if got := cached.MustTarget(); got != "cached/README.md" {
		t.Fatalf("cached.MustTarget() = %q, want cached value preserved", got)
	}
	if got := cached.MemoryState(); got != MemoryDirty {
		t.Fatalf("cached.MemoryState() = %v, want %v", got, MemoryDirty)
	}
	if got := cached.DiskState(); got != DiskPresent {
		t.Fatalf("cached.DiskState() = %v, want %v", got, DiskPresent)
	}

	discovered, ok := layout.Context.Get("assets")
	if !ok {
		t.Fatalf("Context.Get(\"assets\") = false, want true")
	}
	if discovered.HasTarget() {
		t.Fatal("discovered.HasTarget() = true, want false")
	}
	if got := discovered.MemoryState(); got != MemoryUnknown {
		t.Fatalf("discovered.MemoryState() = %v, want %v", got, MemoryUnknown)
	}
	if got := discovered.DiskState(); got != DiskPresent {
		t.Fatalf("discovered.DiskState() = %v, want %v", got, DiskPresent)
	}
}

func TestLinkSlotSupportsTypedLinkVariants(t *testing.T) {
	type tree struct {
		Members LinkSlot[*FileLink] `layout:"links" path:"context"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	files := &model.Members

	readme, err := files.ComposeMember("README")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}
	readme.SetTarget("docs/README.md")
	files.Put("README", readme)

	if _, err := manager.Sync(DefaultContext); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	got, ok := files.MustComposeMember("README").TargetFile()
	if !ok {
		t.Fatal("TargetFile() ok = false, want true")
	}
	if got.Path() != filepath.Join(files.Path(), "docs", "README.md") {
		t.Fatalf("TargetFile().Path() = %q", got.Path())
	}
}
