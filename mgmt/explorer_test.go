package mgmt_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/qlustra/conduit/mgmt"
)

func TestExplorerSelectionTraversalAndPolicyOverride(t *testing.T) {
	root := t.TempDir()
	writeExploreFiles(t, root, map[string]string{
		"a.md":                    "root note",
		"b.txt":                   "text",
		"notes/a.md":              "nested note",
		"notes/deep/hidden.md":    "too deep",
		"private/confidential.md": "pruned",
	})
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
		Select:   func(entry mgmt.ExploreEntry) bool { return strings.HasSuffix(entry.RelativePath, ".md") },
		Prune:    func(entry mgmt.ExploreEntry) bool { return entry.RelativePath == "private" },
		MaxDepth: 2,
	}})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{})
	if err != nil || !inventory.Complete || len(inventory.Diagnostics) != 0 {
		t.Fatalf("scan = %+v, %v", inventory, err)
	}
	assertExplorePaths(t, inventory.Entries, "a.md", "notes/a.md")
	if inventory.Root != root || explorer.Root().Path() != root {
		t.Fatalf("root = %q / %q, want %q", inventory.Root, explorer.Root().Path(), root)
	}
	if inventory.Entries[1].Depth != 2 || inventory.Entries[1].Size != int64(len("nested note")) {
		t.Fatalf("unexpected metadata: %+v", inventory.Entries[1])
	}
	if inventory.Entries[0].ModTime.IsZero() || !inventory.Entries[0].IsRegular() {
		t.Fatalf("missing file metadata: %+v", inventory.Entries[0])
	}

	// A policy override replaces (rather than ambiguously merging) defaults.
	override, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{
		Selection: &mgmt.ExploreSelection{MaxDepth: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertExplorePaths(t, override.Entries, "a.md", "b.txt", "notes", "private")

	// Exclusion does not implicitly prune: notes itself was not selected above,
	// but its selected child was still found. An explicit prune stops descent.
	pruned, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{
		Selection: &mgmt.ExploreSelection{Prune: func(entry mgmt.ExploreEntry) bool { return entry.Depth == 0 }},
	})
	if err != nil || !pruned.Complete || len(pruned.Entries) != 0 {
		t.Fatalf("root-pruned scan = %+v, %v", pruned, err)
	}
}

func TestExplorerScanIsMetadataOnlyAndGatherReadsOnlySelectedFiles(t *testing.T) {
	root := t.TempDir()
	writeExploreFiles(t, root, map[string]string{"a.txt": "alpha", "excluded.txt": "secret", "nested/b.txt": "beta"})
	var selected []string
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
		Select: func(entry mgmt.ExploreEntry) bool {
			selected = append(selected, entry.RelativePath)
			if entry.RelativePath == "excluded.txt" {
				// Removing an excluded file before gathering proves it is not read.
				if err := os.Remove(entry.Path); err != nil {
					t.Fatal(err)
				}
				return false
			}
			return true
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	gathered, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{})
	if err != nil || !gathered.Complete {
		t.Fatalf("gather = %+v, %v", gathered, err)
	}
	assertExplorePaths(t, gathered.Entries, "a.txt", "nested", "nested/b.txt")
	if len(gathered.Files) != 2 || string(gathered.Files[0].Content) != "alpha" || string(gathered.Files[1].Content) != "beta" {
		t.Fatalf("gathered files = %+v", gathered.Files)
	}
	if len(selected) != 4 {
		t.Fatalf("selector saw %v", selected)
	}

	// Scan still succeeds when a selected file vanishes immediately after its
	// metadata is inspected: it never opens file contents.
	explorer, err = mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
		Select: func(entry mgmt.ExploreEntry) bool {
			if entry.IsRegular() {
				if err := os.Remove(entry.Path); err != nil {
					t.Fatal(err)
				}
			}
			return true
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{})
	if err != nil || !inventory.Complete {
		t.Fatalf("metadata-only scan = %+v, %v", inventory, err)
	}
	assertExplorePaths(t, inventory.Entries, "a.txt", "nested", "nested/b.txt")
}

func TestExplorerSymlinksAreNeverTraversedOrGathered(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	writeExploreFiles(t, root, map[string]string{"a.txt": "local"})
	writeExploreFiles(t, outside, map[string]string{"outside.txt": "outside"})
	for _, link := range []struct{ name, target string }{
		{"directory-link", outside},
		{"file-link", filepath.Join(outside, "outside.txt")},
		{"dangling-link", filepath.Join(outside, "absent")},
	} {
		if err := os.Symlink(link.target, filepath.Join(root, link.name)); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}
	}
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gathered, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{})
	if err != nil || !gathered.Complete {
		t.Fatalf("gather = %+v, %v", gathered, err)
	}
	assertExplorePaths(t, gathered.Entries, "a.txt", "dangling-link", "directory-link", "file-link")
	if len(gathered.Files) != 1 || string(gathered.Files[0].Content) != "local" {
		t.Fatalf("read through a link: %+v", gathered.Files)
	}
	if gathered.Entries[1].Mode&fs.ModeSymlink == 0 {
		t.Fatalf("link metadata was dereferenced: %+v", gathered.Entries[1])
	}

	rejected, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{Symlinks: mgmt.ExploreRejectSymlinks})
	if !errors.Is(err, mgmt.ErrExploreSymlink) || rejected.Complete || len(rejected.Diagnostics) != 3 {
		t.Fatalf("reject scan = %+v, %v", rejected, err)
	}
	assertExplorePaths(t, rejected.Entries, "a.txt", "dangling-link", "directory-link", "file-link")

	// The observation root must itself be a directory, not a link to one.
	linkedRoot, err := mgmt.NewExplorer(filepath.Join(root, "directory-link"), mgmt.ExplorerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := linkedRoot.Scan(context.Background(), mgmt.ScanPolicy{})
	if !errors.Is(err, mgmt.ErrExploreSymlink) || len(inventory.Entries) != 0 {
		t.Fatalf("linked root scan = %+v, %v", inventory, err)
	}
}

func TestExplorerGatherLimitsAndFailFast(t *testing.T) {
	root := t.TempDir()
	writeExploreFiles(t, root, map[string]string{
		"a-large": "12345", "b-small": "12", "c-too-much": "123", "d-fits": "3", "e-empty": "", "f-full": "1",
	})
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gathered, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{MaxFileBytes: 4, MaxTotalBytes: 3})
	if !errors.Is(err, mgmt.ErrExploreLimit) || gathered.Complete || len(gathered.Diagnostics) != 3 {
		t.Fatalf("bounded gather = %+v, %v", gathered, err)
	}
	assertExplorePaths(t, gathered.Entries, "a-large", "b-small", "c-too-much", "d-fits", "e-empty", "f-full")
	if len(gathered.Files) != 3 || gathered.Files[0].Entry.RelativePath != "b-small" || gathered.Files[1].Entry.RelativePath != "d-fits" || gathered.Files[2].Entry.RelativePath != "e-empty" {
		t.Fatalf("bounded content = %+v", gathered.Files)
	}
	if string(gathered.Files[0].Content)+string(gathered.Files[1].Content)+string(gathered.Files[2].Content) != "123" {
		t.Fatalf("unexpected bounded bytes: %+v", gathered.Files)
	}
	if gathered.Diagnostics[0].Operation != "gather" || gathered.Diagnostics[0].RelativePath != "a-large" {
		t.Fatalf("missing failure location: %+v", gathered.Diagnostics[0])
	}

	stopped, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{
		ScanPolicy: mgmt.ScanPolicy{Diagnostics: mgmt.ExploreFailFast}, MaxFileBytes: 4,
	})
	if !errors.Is(err, mgmt.ErrExploreLimit) || len(stopped.Diagnostics) != 1 || len(stopped.Files) != 0 {
		t.Fatalf("fail-fast gather = %+v, %v", stopped, err)
	}
	assertExplorePaths(t, stopped.Entries, "a-large")
}

func TestExplorerCollectsReadFailuresAndRetainsIndependentFiles(t *testing.T) {
	for _, failFast := range []bool{false, true} {
		t.Run(map[bool]string{false: "collect", true: "fail-fast"}[failFast], func(t *testing.T) {
			root := t.TempDir()
			writeExploreFiles(t, root, map[string]string{"a-missing": "gone", "b-ok": "kept"})
			explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
				Select: func(entry mgmt.ExploreEntry) bool {
					if entry.RelativePath == "a-missing" {
						if err := os.Remove(entry.Path); err != nil {
							t.Fatal(err)
						}
					}
					return true
				},
			}})
			if err != nil {
				t.Fatal(err)
			}
			policy := mgmt.GatherPolicy{}
			if failFast {
				policy.Diagnostics = mgmt.ExploreFailFast
			}
			gathered, err := explorer.Gather(context.Background(), policy)
			if !errors.Is(err, fs.ErrNotExist) || gathered.Complete || len(gathered.Diagnostics) != 1 {
				t.Fatalf("read failure = %+v, %v", gathered, err)
			}
			if failFast {
				assertExplorePaths(t, gathered.Entries, "a-missing")
				if len(gathered.Files) != 0 {
					t.Fatalf("continued after first error: %+v", gathered.Files)
				}
			} else {
				assertExplorePaths(t, gathered.Entries, "a-missing", "b-ok")
				if len(gathered.Files) != 1 || string(gathered.Files[0].Content) != "kept" {
					t.Fatalf("lost independent result: %+v", gathered.Files)
				}
			}
		})
	}
}

func TestExplorerGatherRechecksSelectedContent(t *testing.T) {
	t.Run("file grows past limit", func(t *testing.T) {
		root := t.TempDir()
		writeExploreFiles(t, root, map[string]string{"growing": "1"})
		explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
			Select: func(entry mgmt.ExploreEntry) bool {
				if err := os.WriteFile(entry.Path, []byte("123456"), 0600); err != nil {
					t.Fatal(err)
				}
				return true
			},
		}})
		if err != nil {
			t.Fatal(err)
		}
		gathered, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{MaxFileBytes: 2})
		if !errors.Is(err, mgmt.ErrExploreLimit) || len(gathered.Files) != 0 || len(gathered.Diagnostics) != 1 {
			t.Fatalf("growing file = %+v, %v", gathered, err)
		}
	})
	t.Run("file becomes symlink", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		writeExploreFiles(t, root, map[string]string{"changing": "initial"})
		writeExploreFiles(t, outside, map[string]string{"target": "outside"})
		// Establish support before executing the selector, which must stay free
		// of Skip calls halfway through traversal.
		probe := filepath.Join(outside, "probe")
		if err := os.Symlink(filepath.Join(outside, "target"), probe); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}
		explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
			Select: func(entry mgmt.ExploreEntry) bool {
				if err := os.Remove(entry.Path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "target"), entry.Path); err != nil {
					t.Fatal(err)
				}
				return true
			},
		}})
		if err != nil {
			t.Fatal(err)
		}
		gathered, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{})
		if err == nil || len(gathered.Files) != 0 || len(gathered.Diagnostics) != 1 {
			t.Fatalf("followed changed symlink: %+v, %v", gathered, err)
		}
	})
}

func TestExplorerContinuesAfterDirectoryReadFailure(t *testing.T) {
	root := t.TempDir()
	writeExploreFiles(t, root, map[string]string{"a-branch/hidden": "lost", "b-independent": "visible"})
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
		Select: func(entry mgmt.ExploreEntry) bool {
			if entry.RelativePath == "a-branch" {
				if err := os.RemoveAll(entry.Path); err != nil {
					t.Fatal(err)
				}
			}
			return true
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{})
	if !errors.Is(err, fs.ErrNotExist) || inventory.Complete || len(inventory.Diagnostics) != 1 {
		t.Fatalf("failed branch = %+v, %v", inventory, err)
	}
	assertExplorePaths(t, inventory.Entries, "a-branch", "b-independent")
	if inventory.Diagnostics[0].RelativePath != "a-branch" {
		t.Fatalf("diagnostic path = %+v", inventory.Diagnostics[0])
	}
}

func TestExplorerCancellationRetainsProgress(t *testing.T) {
	root := t.TempDir()
	writeExploreFiles(t, root, map[string]string{"a": "first", "b": "second"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{
		Select: func(entry mgmt.ExploreEntry) bool {
			if entry.RelativePath == "a" {
				cancel()
			}
			return true
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := explorer.Scan(ctx, mgmt.ScanPolicy{})
	if !errors.Is(err, context.Canceled) || inventory.Complete || len(inventory.Diagnostics) != 1 {
		t.Fatalf("canceled scan = %+v, %v", inventory, err)
	}
	assertExplorePaths(t, inventory.Entries, "a")

	gathered, err := explorer.Gather(ctx, mgmt.GatherPolicy{})
	if !errors.Is(err, context.Canceled) || len(gathered.Entries) != 0 || len(gathered.Files) != 0 {
		t.Fatalf("already-canceled gather = %+v, %v", gathered, err)
	}

	// Cancellation from the final callback must not report successful completion.
	lastCtx, lastCancel := context.WithCancel(context.Background())
	defer lastCancel()
	inventory, err = explorer.Scan(lastCtx, mgmt.ScanPolicy{Selection: &mgmt.ExploreSelection{
		Select: func(entry mgmt.ExploreEntry) bool {
			if entry.RelativePath == "b" {
				lastCancel()
			}
			return true
		},
	}})
	if !errors.Is(err, context.Canceled) || inventory.Complete || len(inventory.Diagnostics) != 1 {
		t.Fatalf("final-entry cancellation = %+v, %v", inventory, err)
	}
	assertExplorePaths(t, inventory.Entries, "a", "b")
}

func TestExplorerBindingAndInvalidInputsDoNotWrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{})
	if !errors.Is(err, fs.ErrNotExist) || inventory.Complete || len(inventory.Diagnostics) != 1 {
		t.Fatalf("missing root = %+v, %v", inventory, err)
	}
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("binding or scan created root: %v", err)
	}
	if _, err := mgmt.NewExplorer("", mgmt.ExplorerOptions{}); err == nil {
		t.Fatal("accepted empty root")
	}
	if _, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{Selection: mgmt.ExploreSelection{MaxDepth: -1}}); err == nil {
		t.Fatal("accepted negative default depth")
	}
	for _, policy := range []mgmt.ScanPolicy{
		{Selection: &mgmt.ExploreSelection{MaxDepth: -1}},
		{Diagnostics: mgmt.ExploreDiagnosticPolicy(99)},
		{Symlinks: mgmt.ExploreSymlinkPolicy(99)},
	} {
		if result, err := explorer.Scan(context.Background(), policy); err == nil || len(result.Diagnostics) != 0 {
			t.Fatalf("invalid policy reached filesystem: %+v, %v", result, err)
		}
	}
	if _, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{MaxFileBytes: -1}); err == nil {
		t.Fatal("accepted negative file limit")
	}
	if _, err := explorer.Gather(context.Background(), mgmt.GatherPolicy{MaxTotalBytes: -1}); err == nil {
		t.Fatal("accepted negative total limit")
	}

	fileRoot := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(fileRoot, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	explorer, err = mgmt.NewExplorer(fileRoot, mgmt.ExplorerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := explorer.Scan(context.Background(), mgmt.ScanPolicy{}); err == nil || result.Complete || len(result.Entries) != 0 {
		t.Fatalf("accepted file as root: %+v, %v", result, err)
	}
}

func writeExploreFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func assertExplorePaths(t *testing.T, entries []mgmt.ExploreEntry, want ...string) {
	t.Helper()
	got := make([]string, len(entries))
	for i, entry := range entries {
		got[i] = entry.RelativePath
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventory paths = %v, want %v", got, want)
	}
}
