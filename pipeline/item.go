package pipeline

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/qlustra/conduit/layout"
)

// Blob is an opaque byte subject with pipeline metadata.
type Blob struct {
	Key  string
	Name string
	Path string
	Data []byte
}

// Item is one typed unit flowing through a task.
type Item[T any] struct {
	Key   string
	Name  string
	Path  string
	File  layout.File
	Dir   layout.Dir
	Value T
	Data  []byte
}

type subjectKind uint8

const (
	subjectUnknown subjectKind = iota
	subjectSingle
	subjectMulti
)

func cloneItems[T any](items []Item[T]) []Item[T] {
	cloned := make([]Item[T], len(items))
	copy(cloned, items)
	for i := range cloned {
		cloned[i].Data = cloneBytes(cloned[i].Data)
	}
	return cloned
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

func itemFromSlotEntry[T any](slot *layout.Slot[T], entry layout.SlotEntry[T]) Item[T] {
	return itemFromSlotEntryName(slot, entry.Name, entry.Item)
}

func itemFromSlotEntryName[T any](slot *layout.Slot[T], name string, value T) Item[T] {
	dir := slot.Root().Dir(name)
	path := itemPathFromDir(dir, name)
	return Item[T]{Key: name, Name: dir.Base(), Path: path, Dir: dir, Value: value}
}

func itemFromFileSlotEntry[T any](slot *layout.FileSlot[T], entry layout.FileSlotEntry[T]) Item[T] {
	return itemFromFileSlotEntryName(slot, entry.Name, entry.Item)
}

func itemFromFileSlotEntryName[T any](slot *layout.FileSlot[T], name string, value T) Item[T] {
	file := slot.Root().File(name)
	path := itemPathFromFile(file, name)
	return Item[T]{Key: name, Name: file.Base(), Path: path, File: file, Value: value}
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
