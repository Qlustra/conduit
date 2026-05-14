package layout

import (
	"fmt"
	"os"
)

// Format is a codec-backed typed file with explicit disk and memory state.
//
// Format combines a file path with cached typed content and tracks the two
// independent axes that matter in Conduit:
//   - what is currently known about the file on disk
//   - what is currently known about the cached in-memory value
//
// Concrete wrappers such as formats.JSONFile, formats.YAMLFile, and
// formats.TOMLFile embed Format with a fixed codec.
type Format[T any, C Codec[T]] struct {
	File
	content *T
	disk    DiskState
	memory  MemoryState
}

// Ops

// Get returns the cached typed value, if any.
func (f Format[T, C]) Get() (T, bool) {
	if f.content == nil {
		var zero T
		return zero, false
	}
	return *f.content, true
}

// MustGet returns the cached typed value or panics when no value is loaded.
func (f *Format[T, C]) MustGet() T {
	if f.content == nil {
		panic("file content is not loaded")
	}
	return *f.content
}

// Set replaces the cached typed value and marks memory state dirty.
func (f *Format[T, C]) Set(value T) {
	f.content = &value
	f.memory = MemoryDirty
}

// SetDefault stores value only when no cached content is present.
//
// It returns whether the default was applied.
func (f *Format[T, C]) SetDefault(value T) bool {
	if f.content != nil {
		return false
	}
	f.Set(value)
	return true
}

// Clear removes cached content and resets memory state to MemoryUnknown.
//
// It preserves the current disk-state metadata.
func (f *Format[T, C]) Clear() {
	f.content = nil
	f.memory = MemoryUnknown
}

// Delete removes the file from disk when it exists, clears cached content, and
// marks disk state missing.
func (f *Format[T, C]) Delete() error {
	if err := f.File.DeleteIfExists(); err != nil {
		return err
	}
	f.Clear()
	f.disk = DiskMissing
	return nil
}

// Codec

// Codec converts between typed values and raw file bytes for Format.
type Codec[T any] interface {
	Marshal(T) ([]byte, error)
	Unmarshal([]byte) (T, error)
}

func (f Format[T, C]) codec() C {
	var c C
	return c
}

// Write marshals value and writes it directly to disk.
//
// Unlike Save and Sync, Write does not update the cached content or tracked
// disk and memory state.
func (f Format[T, C]) Write(value T, ctx Context) error {
	data, err := f.codec().Marshal(value)
	if err != nil {
		return err
	}
	return f.File.WriteBytes(data, ctx.DirMode, ctx.FileMode)
}

// Read reads and unmarshals the file from disk without changing cached state.
func (f Format[T, C]) Read() (T, error) {
	data, err := f.File.ReadBytes()
	if err != nil {
		var zero T
		return zero, err
	}
	return f.codec().Unmarshal(data)
}

// ReadIfExists reads and unmarshals the file when it exists.
//
// It returns ok == false for a missing file and does not change cached state.
func (f Format[T, C]) ReadIfExists() (T, bool, error) {
	data, ok, err := f.File.ReadBytesIfExists()
	if err != nil || !ok {
		var zero T
		return zero, ok, err
	}

	value, err := f.codec().Unmarshal(data)
	return value, true, err
}

// LoadOrInit loads existing content or installs defaultValue in memory when
// the file is missing.
//
// When the file is missing, the default is cached and memory state becomes
// MemoryDirty. Nothing is written until Save, Sync, or SyncDeep is called.
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

// Save writes the currently cached value to disk.
//
// Save fails when no cached content is present. On success it marks disk state
// present and memory state synced.
func (f *Format[T, C]) Save(ctx Context) error {
	if f.content == nil {
		return fmt.Errorf("file content is not loaded")
	}
	return f.saveLoaded(ctx)
}

func (f *Format[T, C]) saveLoaded(ctx Context) error {
	if err := f.Write(*f.content, ctx); err != nil {
		return err
	}
	f.disk = DiskPresent
	f.memory = MemorySynced
	return nil
}

// EnsureDeep materializes the backing file for a syncable typed file when the
// current ensure policy allows syncable nodes.
func (f Format[T, C]) EnsureDeep(ctx Context) (ResultCode, error) {
	if !ctx.ensurePolicy().allowsSyncable() {
		return EnsureSkippedPolicy, nil
	}

	fileCtx := ctx.withEnsurePolicy(EnsureFiles)
	err := f.File.Ensure(fileCtx)
	result := EnsureEnsured
	if err != nil {
		result = EnsureFailed
	}
	return result, err
}

// Compose

// ComposePath binds the format to path and resets its cached content and state.
func (f *Format[T, C]) ComposePath(path string) {
	f.File = NewFile(path)
	f.content = nil
	f.disk = DiskUnknown
	f.memory = MemoryUnknown
}

// Load

// Load reads the file from disk into the cached typed value.
//
// It returns whether the file existed. When the file is missing, cached
// content is cleared, disk state becomes DiskMissing, and memory state becomes
// MemoryUnknown.
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

// HasContent reports whether a typed value is currently cached in memory.
func (f Format[T, C]) HasContent() bool {
	return f.content != nil
}

// Unload clears cached content and resets memory state to MemoryUnknown.
//
// It preserves the current disk-state metadata.
func (f *Format[T, C]) Unload() {
	f.content = nil
	f.memory = MemoryUnknown
}

// Discover

// Discover refreshes disk-state metadata without replacing cached content.
//
// For Format, Discover has the same local effect as Scan.
func (f *Format[T, C]) Discover() (DiskState, error) {
	return f.Scan()
}

// Sync

// Sync writes the cached value when content is present and ctx.SyncPolicy
// allows the current memory and disk state.
//
// Sync returns a skip result instead of an error when no content is cached or
// when policy excludes the current state.
func (f *Format[T, C]) Sync(ctx Context) (ResultCode, error) {
	if f.content == nil {
		return SyncSkippedNoContent, nil
	}
	if !ctx.syncPolicy().allows(f.memory, f.disk) {
		return SyncSkippedPolicy, nil
	}
	if err := f.saveLoaded(ctx); err != nil {
		return SyncFailed, err
	}
	return SyncWritten, nil
}

// States

// DiskState returns the last known disk-state metadata.
func (f Format[T, C]) DiskState() DiskState {
	return f.disk
}

// MemoryState returns the last known memory-state metadata.
func (f Format[T, C]) MemoryState() MemoryState {
	return f.memory
}

// HasKnownDiskState reports whether disk state is something other than
// DiskUnknown.
func (f Format[T, C]) HasKnownDiskState() bool {
	return f.disk != DiskUnknown
}

// WasObservedOnDisk reports whether the last known disk state is DiskPresent.
func (f Format[T, C]) WasObservedOnDisk() bool {
	return f.disk == DiskPresent
}

// HasBeenLoaded reports whether memory state has progressed beyond
// MemoryUnknown.
func (f Format[T, C]) HasBeenLoaded() bool {
	return f.memory == MemoryLoaded || f.memory == MemorySynced || f.memory == MemoryDirty
}

// IsDirty reports whether the cached value was set or changed in memory since
// the last load or sync.
func (f Format[T, C]) IsDirty() bool {
	return f.memory == MemoryDirty
}

// Scan

// Scan refreshes disk-state metadata without replacing cached content.
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
