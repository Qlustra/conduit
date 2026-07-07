package pipeline

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/qlustra/conduit/layout"
)

type itemType uint8

const (
	itemTypeUnknown itemType = iota
	itemTypeFile
	itemTypeSlotEntry
)

// Item is one unit flowing through a task.
type Item[T any] struct {
	// Key is the logical identity used by duplicate handling and results.
	Key string
	// Name is the display or basename-style name for the item.
	Name string
	// Path is the relative path used by directory sinks and results.
	Path string
	// File is the backing file for file-backed items.
	File layout.File
	// Dir is the backing directory for directory-backed typed items.
	Dir layout.Dir
	// Value is the typed payload flowing through typed tasks.
	Value T
	// Data is the in-memory byte payload for byte tasks.
	Data []byte
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	copyData := make([]byte, len(data))
	copy(copyData, data)
	return copyData
}

func itemFromFile(file layout.File) Item[Blob] {
	name := file.Base()
	path := itemPathFromFile(file, name)
	blob := Blob{Key: name, Name: name, Path: path}
	return Item[Blob]{Key: name, Name: name, Path: path, File: file, Value: blob}
}

func itemFromBlob(blob Blob) Item[Blob] {
	key, name, path := normalizeBlobMetadata(blob)
	blob.Key = key
	blob.Name = name
	blob.Path = path
	return Item[Blob]{Key: key, Name: name, Path: path, Value: blob, Data: cloneBytes(blob.Data)}
}

func normalizeBlobMetadata(blob Blob) (key string, name string, path string) {
	key = blob.Key
	name = blob.Name
	path = blob.Path
	if key == "" {
		key = name
	}
	if key == "" {
		key = path
	}
	if name == "" && path != "" {
		name = filepath.Base(path)
	}
	if name == "" {
		name = key
	}
	if path == "" {
		path = name
	}
	if key == "" {
		key = name
	}
	return key, name, path
}

func normalizeBlobItem(item Item[Blob]) Item[Blob] {
	data := item.Data
	if data == nil && item.Value.Data != nil {
		data = item.Value.Data
	}
	data = cloneBytes(data)

	key := item.Key
	name := item.Name
	path := item.Path
	if key == "" {
		key = item.Value.Key
	}
	if name == "" {
		name = item.Value.Name
	}
	if path == "" {
		path = item.Value.Path
	}
	if hasFile(item.File) {
		fallbackName := item.File.Base()
		if name == "" {
			name = fallbackName
		}
		if path == "" {
			path = itemPathFromFile(item.File, fallbackName)
		}
	}
	key, name, path = normalizeBlobMetadata(Blob{Key: key, Name: name, Path: path})

	blob := item.Value
	blob.Key = key
	blob.Name = name
	blob.Path = path
	blob.Data = cloneBytes(data)

	item.Key = key
	item.Name = name
	item.Path = path
	item.Data = data
	item.Value = blob
	return item
}

func itemPathFromFile(file layout.File, fallback string) string {
	if rel, ok := file.ComposedRelativePath(); ok {
		return rel
	}
	return fallback
}

func itemFromSlot[T any](slot *layout.Slot[T]) Item[*layout.Slot[T]] {
	dir := slot.Root()
	name := dir.Base()
	path := itemPathFromDir(dir, name)
	return Item[*layout.Slot[T]]{Key: name, Name: name, Path: path, Dir: dir, Value: slot}
}

func itemFromFileSlot[T any](slot *layout.FileSlot[T]) Item[*layout.FileSlot[T]] {
	dir := slot.Root()
	name := dir.Base()
	path := itemPathFromDir(dir, name)
	return Item[*layout.FileSlot[T]]{Key: name, Name: name, Path: path, Dir: dir, Value: slot}
}

func itemPathFromDir(dir layout.Dir, fallback string) string {
	if rel, ok := dir.ComposedRelativePath(); ok {
		return rel
	}
	return fallback
}

func materializeByteItem(ctx context.Context, lctx layout.Context, item *Item[Blob]) ([]byte, error) {
	if item == nil {
		return nil, fmt.Errorf("item must not be nil")
	}
	if item.Data != nil {
		return cloneBytes(item.Data), nil
	}
	if !hasFile(item.File) {
		return nil, fmt.Errorf("item %q has no file or data", item.Name)
	}
	_ = ctx
	handle, err := item.File.OpenRead(lctx, layout.OpenExisting)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(handle)
	closeErr := handle.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	item.Data = cloneBytes(data)
	return data, nil
}

func hasFile(file layout.File) bool {
	path := filepath.Clean(file.Path())
	return path != "" && path != "."
}

func hasDir(dir layout.Dir) bool {
	path := filepath.Clean(dir.Path())
	return path != "" && path != "."
}
