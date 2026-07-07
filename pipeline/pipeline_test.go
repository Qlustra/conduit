package pipeline

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlustra/conduit/layout"
)

func TestSlotSourceSnapshotItems(t *testing.T) {
	t.Run("slot", func(t *testing.T) {
		slot := layout.NewSlot[int](layout.NewDir(filepath.Join(t.TempDir(), "services")))
		slot.Put("api", 7)

		items, err := createSlotSourceFromSlot(&slot).snapshotItems()
		if err != nil {
			t.Fatalf("snapshotItems() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("len(items) = %d, want 1", len(items))
		}
		if items[0].Key != "api" || items[0].Value != 7 {
			t.Fatalf("items[0] = %+v, want key api value 7", items[0])
		}
	})

	t.Run("file slot", func(t *testing.T) {
		slot := layout.NewFileSlot[int](layout.NewDir(filepath.Join(t.TempDir(), "configs")))
		slot.Put("app.json", 9)

		items, err := createSlotSourceFromFileSlot(&slot).snapshotItems()
		if err != nil {
			t.Fatalf("snapshotItems() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("len(items) = %d, want 1", len(items))
		}
		if items[0].Key != "app.json" || items[0].Value != 9 {
			t.Fatalf("items[0] = %+v, want key app.json value 9", items[0])
		}
	})
}

func TestContentCollectionTaskPick(t *testing.T) {
	dest := layout.NewFile(filepath.Join(t.TempDir(), "picked.txt"))
	task := TaskFromBlobs("pick",
		Blob{Name: "first.txt", Data: []byte("first")},
		Blob{Name: "second.txt", Data: []byte("second")},
	).Pick(func(item Item[Blob]) bool {
		return item.Name == "second.txt"
	}).WriteToFile(dest)

	result, err := task.Run(context.Background(), DefaultContext)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.To.Items) != 1 || result.To.Items[0].Name != "second.txt" {
		t.Fatalf("result.To.Items = %+v, want one item named second.txt", result.To.Items)
	}

	data, err := os.ReadFile(dest.Path())
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", dest.Path(), err)
	}
	if !bytes.Equal(data, []byte("second")) {
		t.Fatalf("output = %q, want %q", string(data), "second")
	}
}

func TestContentCollectionTaskSelect(t *testing.T) {
	dest := layout.NewFile(filepath.Join(t.TempDir(), "selected.txt"))
	task := TaskFromBlobs("select",
		Blob{Name: "first.txt", Data: []byte("first")},
		Blob{Name: "second.txt", Data: []byte("second")},
	).Select(func(items []Item[Blob]) Item[Blob] {
		return items[0]
	}).WriteToFile(dest)

	result, err := task.Run(context.Background(), DefaultContext)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.To.Items) != 1 || result.To.Items[0].Name != "first.txt" {
		t.Fatalf("result.To.Items = %+v, want one item named first.txt", result.To.Items)
	}

	data, err := os.ReadFile(dest.Path())
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", dest.Path(), err)
	}
	if !bytes.Equal(data, []byte("first")) {
		t.Fatalf("output = %q, want %q", string(data), "first")
	}
}

func TestByteTaskRejectsMultipleSinks(t *testing.T) {
	dir := t.TempDir()
	destA := layout.NewFile(filepath.Join(dir, "a.txt"))
	destB := layout.NewFile(filepath.Join(dir, "b.txt"))
	task := TaskFromBlob("duplicate-sinks", Blob{Name: "input.txt", Data: []byte("payload")}).
		WriteToFile(destA).
		WriteToFile(destB)

	_, err := task.Run(context.Background(), DefaultContext)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "already has a sink") {
		t.Fatalf("Run() error = %v, want already has a sink", err)
	}
	if destA.Exists() || destB.Exists() {
		t.Fatal("sink destination was written despite configuration error")
	}
}

func TestResolveLayoutContextMergesPartialOverrides(t *testing.T) {
	fallback := layout.DefaultContext
	fallback.Reporter = nil

	resolved := resolveLayoutContext(layout.Context{WritePolicy: layout.WriteAtomicReplace}, fallback)
	if resolved.DirMode != fallback.DirMode {
		t.Fatalf("DirMode = %v, want %v", resolved.DirMode, fallback.DirMode)
	}
	if resolved.FileMode != fallback.FileMode {
		t.Fatalf("FileMode = %v, want %v", resolved.FileMode, fallback.FileMode)
	}
	if resolved.WritePolicy != layout.WriteAtomicReplace {
		t.Fatalf("WritePolicy = %v, want %v", resolved.WritePolicy, layout.WriteAtomicReplace)
	}
}
