package layout

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	_ ContentState     = (*testMapFile)(nil)
	_ StateSnapshotter = (*testMapFile)(nil)
	_ WritePreparer    = (*testMapFile)(nil)
)

func TestPreparedWriteDoesNotMutateBeforeSuccessfulWrite(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "new", "state.json"))
	f.Set(map[string]string{"name": "prepared"})
	prepared, err := f.PrepareWrite()
	if err != nil {
		t.Fatal(err)
	}
	if f.DiskState() != DiskUnknown || f.MemoryState() != MemoryDirty {
		t.Fatal("preparation changed state")
	}
	if _, err := os.Stat(filepath.Dir(f.Path())); !os.IsNotExist(err) {
		t.Fatalf("preparation created parent: %v", err)
	}
	data := prepared.Bytes()
	data[0] = 'x'
	if prepared.Bytes()[0] == 'x' {
		t.Fatal("Bytes exposed prepared buffer")
	}
	if err := prepared.Write(DefaultContext); err != nil {
		t.Fatal(err)
	}
	if f.DiskState() != DiskPresent || f.MemoryState() != MemorySynced {
		t.Fatal("successful write did not update state")
	}
	got, err := os.ReadFile(f.Path())
	if err != nil || string(got) != `{"name":"prepared"}` {
		t.Fatalf("written bytes = %q, error = %v", got, err)
	}
}

func TestPreparedWriteRejectsStaleContentAndBinding(t *testing.T) {
	for _, mutation := range []string{"set", "map", "clear", "recompose"} {
		t.Run(mutation, func(t *testing.T) {
			var f testMapFile
			path := filepath.Join(t.TempDir(), "state.json")
			f.ComposePath(path)
			f.Set(map[string]string{"name": "before"})
			prepared, err := f.PrepareWrite()
			if err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "set":
				f.Set(map[string]string{"name": "after"})
			case "map":
				f.MustGet()["name"] = "after"
			case "clear":
				f.Clear()
			case "recompose":
				f.ComposePath(path)
				f.Set(map[string]string{"name": "before"})
			}
			if err := prepared.Write(DefaultContext); !errors.Is(err, ErrPreparedWriteStale) {
				t.Fatalf("Write error = %v, want stale preparation", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("stale write touched disk: %v", err)
			}
		})
	}
}

func TestPreparedWriteFailurePreservesState(t *testing.T) {
	var f testMapFile
	f.ComposePath(t.TempDir()) // A directory cannot be replaced by a regular file.
	f.Set(map[string]string{"a": "b"})
	prepared, err := f.PrepareWrite()
	if err != nil {
		t.Fatal(err)
	}
	if err := prepared.Write(DefaultContext); err == nil {
		t.Fatal("Write succeeded over directory")
	}
	if f.DiskState() != DiskUnknown || f.MemoryState() != MemoryDirty {
		t.Fatal("failed write changed state")
	}
}

func TestPreparedWriteExclusivePreservesExistingContent(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	f.Set(map[string]string{"source": "memory"})
	prepared, err := f.PrepareWrite()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.Path(), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := prepared.WriteExclusive(DefaultContext); !os.IsExist(err) {
		t.Fatalf("WriteExclusive error = %v, want existing-entry error", err)
	}
	data, err := os.ReadFile(f.Path())
	if err != nil || string(data) != "existing" {
		t.Fatalf("existing content changed: %q, %v", data, err)
	}
	if f.DiskState() != DiskUnknown || f.MemoryState() != MemoryDirty {
		t.Fatal("exclusive collision changed state")
	}
}

func TestPreparedWriteExclusiveSuccessAndUnsupportedPolicy(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "new", "state.json"))
	f.Set(map[string]string{"a": "b"})
	prepared, err := f.PrepareWrite()
	if err != nil {
		t.Fatal(err)
	}
	ctx := DefaultContext
	ctx.WritePolicy = WriteAtomicReplace
	if err := prepared.WriteExclusive(ctx); err == nil {
		t.Fatal("exclusive write accepted atomic replacement policy")
	}
	if _, err := os.Stat(filepath.Dir(f.Path())); !os.IsNotExist(err) {
		t.Fatalf("unsupported policy created parent: %v", err)
	}
	if err := prepared.WriteExclusive(DefaultContext); err != nil {
		t.Fatal(err)
	}
	if f.DiskState() != DiskPresent || f.MemoryState() != MemorySynced {
		t.Fatal("exclusive success did not update state")
	}
}

func TestFormatSnapshotRestoresIndependentContentAndMetadata(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	f.Set(map[string]string{"name": "original"})
	if err := f.Save(DefaultContext); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Load(); err != nil {
		t.Fatal(err)
	}
	restore, err := f.SnapshotState()
	if err != nil {
		t.Fatal(err)
	}
	if f.MemoryState() != MemoryLoaded || f.DiskState() != DiskPresent {
		t.Fatal("snapshot changed state")
	}
	f.MustGet()["name"] = "changed through alias"
	f.Clear()
	f.disk = DiskUnknown
	restore()
	if got := f.MustGet()["name"]; got != "original" {
		t.Fatalf("restored value = %q", got)
	}
	if f.MemoryState() != MemoryLoaded || f.DiskState() != DiskPresent {
		t.Fatal("snapshot did not restore metadata")
	}
	data, err := os.ReadFile(f.Path())
	if err != nil || string(data) != `{"name":"original"}` {
		t.Fatalf("snapshot touched disk: %q, %v", data, err)
	}
}

type aliasBytesCodec struct{}

func (aliasBytesCodec) Marshal(value []byte) ([]byte, error)  { return value, nil }
func (aliasBytesCodec) Unmarshal(data []byte) ([]byte, error) { return data, nil }

func TestFormatSnapshotClonesCodecBuffer(t *testing.T) {
	var f Format[[]byte, aliasBytesCodec]
	f.Set([]byte("original"))
	restore, err := f.SnapshotState()
	if err != nil {
		t.Fatal(err)
	}
	f.MustGet()[0] = 'x'
	restore()
	if string(f.MustGet()) != "original" {
		t.Fatal("snapshot retained alias to original content")
	}
}

type failingPreparationCodec struct{}

func (failingPreparationCodec) Marshal(string) ([]byte, error) {
	return nil, errors.New("cannot encode")
}
func (failingPreparationCodec) Unmarshal([]byte) (string, error) { return "", nil }

func TestFormatPreparationErrorsPreserveState(t *testing.T) {
	var f Format[string, failingPreparationCodec]
	f.ComposePath(filepath.Join(t.TempDir(), "missing", "state"))
	if _, err := f.PrepareWrite(); err == nil {
		t.Fatal("PrepareWrite accepted absent cached content")
	}
	f.Set("not encodable")
	if _, err := f.PrepareWrite(); err == nil {
		t.Fatal("PrepareWrite lost encoding error")
	}
	if _, err := f.SnapshotState(); err == nil {
		t.Fatal("SnapshotState lost encoding error")
	}
	if f.MemoryState() != MemoryDirty || f.DiskState() != DiskUnknown || f.MustGet() != "not encodable" {
		t.Fatal("failed preparation changed state")
	}
	if _, err := os.Stat(filepath.Dir(f.Path())); !os.IsNotExist(err) {
		t.Fatalf("failed preparation touched disk: %v", err)
	}
}

func TestFormatValueCodecsLeaveCacheUntouched(t *testing.T) {
	var f testMapFile
	f.Set(map[string]string{"cached": "value"})
	data, err := f.EncodeValue(map[string]string{"candidate": "new"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.DecodeValue(data)
	if err != nil || !reflect.DeepEqual(got, map[string]string{"candidate": "new"}) {
		t.Fatalf("decoded = %v, error = %v", got, err)
	}
	if !reflect.DeepEqual(f.MustGet(), map[string]string{"cached": "value"}) || f.MemoryState() != MemoryDirty || f.DiskState() != DiskUnknown {
		t.Fatal("codec adapter changed cache")
	}
}

type preservedContent struct {
	Stored  string `json:"stored"`
	Runtime *int   `json:"-"`
}
type preservedContentCodec struct{}

func (preservedContentCodec) Marshal(value preservedContent) ([]byte, error) {
	return json.Marshal(value)
}
func (preservedContentCodec) Unmarshal(data []byte) (preservedContent, error) {
	var value preservedContent
	err := json.Unmarshal(data, &value)
	return value, err
}

func TestFormatPreserveStateRestoresExactCacheAfterTemporaryLoad(t *testing.T) {
	var f Format[preservedContent, preservedContentCodec]
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	if err := os.WriteFile(f.Path(), []byte(`{"stored":"disk"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runtime := 42
	f.Set(preservedContent{Stored: "memory", Runtime: &runtime})
	original := f.content
	restore := f.PreserveState()
	if f.content != original || f.disk != DiskUnknown || f.memory != MemoryDirty {
		t.Fatal("preservation changed cache or metadata")
	}
	if _, err := f.Load(); err != nil {
		t.Fatal(err)
	}
	if f.MustGet().Stored != "disk" || f.MustGet().Runtime != nil || f.memory != MemoryLoaded {
		t.Fatal("temporary load did not replace content")
	}
	restore()
	if f.content != original || f.MustGet().Stored != "memory" || f.MustGet().Runtime != &runtime {
		t.Fatal("preservation did not restore exact original content, including codec-ignored field")
	}
	if f.disk != DiskUnknown || f.memory != MemoryDirty {
		t.Fatal("preservation did not restore original metadata")
	}
	data, err := os.ReadFile(f.Path())
	if err != nil || string(data) != `{"stored":"disk"}` {
		t.Fatalf("preservation changed disk: %q, %v", data, err)
	}
}
