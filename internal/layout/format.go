package layout

import (
	"fmt"
	"os"
)

type Format[T any, C Codec[T]] struct {
	File
	content *T
	disk    DiskState
	memory  MemoryState
}

// Ops

func (f Format[T, C]) Get() (T, bool) {
	if f.content == nil {
		var zero T
		return zero, false
	}
	return *f.content, true
}

func (f *Format[T, C]) MustGet() T {
	if f.content == nil {
		panic("file content is not loaded")
	}
	return *f.content
}

func (f *Format[T, C]) Set(value T) {
	f.content = &value
	f.memory = MemoryDirty
}

func (f *Format[T, C]) Clear() {
	f.content = nil
	f.memory = MemoryUnknown
}

func (f *Format[T, C]) Delete() error {
	if err := f.File.DeleteIfExists(); err != nil {
		return err
	}
	f.Clear()
	f.disk = DiskMissing
	return nil
}

// Codec

type Codec[T any] interface {
	Marshal(T) ([]byte, error)
	Unmarshal([]byte) (T, error)
}

func (f Format[T, C]) codec() C {
	var c C
	return c
}

func (f Format[T, C]) Write(value T, dirMode os.FileMode, fileMode os.FileMode) error {
	data, err := f.codec().Marshal(value)
	if err != nil {
		return err
	}
	return f.File.WriteBytes(data, dirMode, fileMode)
}

func (f Format[T, C]) Read() (T, error) {
	data, err := f.File.ReadBytes()
	if err != nil {
		var zero T
		return zero, err
	}
	return f.codec().Unmarshal(data)
}

func (f Format[T, C]) ReadIfExists() (T, bool, error) {
	data, ok, err := f.File.ReadBytesIfExists()
	if err != nil || !ok {
		var zero T
		return zero, ok, err
	}

	value, err := f.codec().Unmarshal(data)
	return value, true, err
}

func (f *Format[T, C]) LoadOrInit(defaultValue T) error {
	loaded, err := f.Load()
	if err != nil {
		return err
	}
	if !loaded {
		f.Set(defaultValue)
	}
	return nil
}

func (f *Format[T, C]) Save(dirMode, fileMode os.FileMode) error {
	if f.content == nil {
		return fmt.Errorf("file content is not loaded")
	}
	if err := f.Write(*f.content, dirMode, fileMode); err != nil {
		return err
	}
	f.disk = DiskPresent
	f.memory = MemorySynced
	return nil
}

// Compose

func (f *Format[T, C]) ComposePath(path string) {
	f.File = NewFile(path)
	f.content = nil
	f.disk = DiskUnknown
	f.memory = MemoryUnknown
}

// Load

func (f *Format[T, C]) Load() (bool, error) {
	value, exists, err := f.ReadIfExists()
	if err != nil {
		return false, err
	}
	if !exists {
		f.content = nil
		f.disk = DiskMissing
		f.memory = MemoryUnknown
		return false, nil
	}
	f.content = &value
	f.disk = DiskPresent
	f.memory = MemoryLoaded
	return true, nil
}

func (f Format[T, C]) HasContent() bool {
	return f.content != nil
}

func (f *Format[T, C]) Unload() {
	f.content = nil
	f.memory = MemoryUnknown
}

// Sync

func (f *Format[T, C]) Sync() error {
	if f.content == nil {
		return nil
	}
	return f.Save(0o755, 0o644)
}

// States

func (f Format[T, C]) DiskState() DiskState {
	return f.disk
}

func (f Format[T, C]) MemoryState() MemoryState {
	return f.memory
}

func (f Format[T, C]) HasKnownDiskState() bool {
	return f.disk != DiskUnknown
}

func (f Format[T, C]) WasObservedOnDisk() bool {
	return f.disk == DiskPresent
}

func (f Format[T, C]) HasBeenLoaded() bool {
	return f.memory == MemoryLoaded || f.memory == MemorySynced || f.memory == MemoryDirty
}

func (f Format[T, C]) IsDirty() bool {
	return f.memory == MemoryDirty
}

// Scan

func (f *Format[T, C]) Scan() (DiskState, error) {
	_, err := os.Stat(f.Path())
	if err == nil {
		f.disk = DiskPresent
		return f.disk, nil
	}
	if os.IsNotExist(err) {
		f.disk = DiskMissing
		return f.disk, nil
	}
	return f.disk, err
}
