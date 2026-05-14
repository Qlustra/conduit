package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"
)

// FileSlotEntry is one cached key-item pair returned by FileSlot.Entries.
type FileSlotEntry[T any] struct {
	Name string
	Item T
}

// FileSlot models repeated direct-child files under one directory.
//
// Each key maps directly to a file path below the slot root. FileSlot caches
// composed items in memory and discovers or loads them only when explicitly
// asked.
type FileSlot[T any] struct {
	root  Dir
	mu    sync.RWMutex
	items map[string]T
}

// NewFileSlot returns a file slot rooted at root.
func NewFileSlot[T any](root Dir) FileSlot[T] {
	return FileSlot[T]{root: root}
}

// Path returns the slot root directory path.
func (s *FileSlot[T]) Path() string {
	return s.root.Path()
}

func (s *FileSlot[T]) ComposedBaseDir() (Dir, bool) {
	return s.root.ComposedBaseDir()
}

func (s *FileSlot[T]) DeclaredPath() (string, bool) {
	return s.root.DeclaredPath()
}

func (s *FileSlot[T]) JoinDeclaredPath(parts ...string) (string, bool) {
	return s.root.JoinDeclaredPath(parts...)
}

func (s *FileSlot[T]) ComposedRelativePath() (string, bool) {
	return s.root.ComposedRelativePath()
}

func (s *FileSlot[T]) JoinComposedPath(parts ...string) (string, bool) {
	return s.root.JoinComposedPath(parts...)
}

// Exists reports whether the slot root currently exists on disk.
func (s *FileSlot[T]) Exists() bool {
	return s.root.Exists()
}

// Root returns the slot root directory handle.
func (s *FileSlot[T]) Root() Dir {
	return s.root
}

// Len returns the number of cached items.
func (s *FileSlot[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}

// Has reports whether a regular file with name currently exists on disk.
func (s *FileSlot[T]) Has(name string) bool {
	info, err := os.Stat(s.root.File(name).Path())
	return err == nil && !info.IsDir()
}

// Get returns the cached item for name without composing a missing one.
func (s *FileSlot[T]) Get(name string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[name]
	return item, ok
}

// Put inserts or replaces a cached item without touching disk.
func (s *FileSlot[T]) Put(name string, item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}
	s.items[name] = item
}

// Remove evicts a cached item without touching disk.
func (s *FileSlot[T]) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
}

// Delete removes the child file from disk and evicts the cached item.
func (s *FileSlot[T]) Delete(name string) error {
	if err := s.root.File(name).DeleteIfExists(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
	return nil
}

// Clear drops all cached items without touching disk.
func (s *FileSlot[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[string]T)
}

// Entries returns a sorted snapshot of cached entries.
//
// Entries is cache-based only; it does not discover from disk.
func (s *FileSlot[T]) Entries() []FileSlotEntry[T] {
	s.mu.RLock()
	entries := make([]FileSlotEntry[T], 0, len(s.items))
	for name, item := range s.items {
		entries = append(entries, FileSlotEntry[T]{
			Name: name,
			Item: item,
		})
	}
	s.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}

// All iterates cached entries in sorted key order.
//
// All is cache-based only; it does not discover from disk.
func (s *FileSlot[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, entry := range s.Entries() {
			if !yield(entry.Name, entry.Item) {
				return
			}
		}
	}
}

// Keys returns the cached item names in sorted order.
func (s *FileSlot[T]) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.items))
	for k := range s.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// At returns the cached item for name, composing and caching it on demand when
// needed.
//
// At does not ensure the file exists on disk.
func (s *FileSlot[T]) At(name string) (T, error) {
	s.mu.RLock()
	if item, ok := s.items[name]; ok {
		s.mu.RUnlock()
		return item, nil
	}
	s.mu.RUnlock()

	composeBase := s.root.Path()
	if base, ok := s.root.ComposedBaseDir(); ok {
		composeBase = base.Path()
	}

	item, err := composePathAs[T](s.root.File(name).Path(), composeBase)
	if err != nil {
		var zero T
		return zero, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}

	if existing, ok := s.items[name]; ok {
		return existing, nil
	}

	s.items[name] = item
	return item, nil
}

// MustAt returns At(name) or panics on composition error.
func (s *FileSlot[T]) MustAt(name string) T {
	v, err := s.At(name)
	if err != nil {
		panic(err)
	}
	return v
}

// Add ensures the slot root exists on disk, composes the item, ensures its
// declared structure, and caches it.
func (s *FileSlot[T]) Add(name string, ctx Context) (T, error) {
	if err := s.root.Ensure(ctx); err != nil {
		var zero T
		return zero, err
	}

	s.mu.RLock()
	if existing, ok := s.items[name]; ok {
		s.mu.RUnlock()
		if _, err := EnsureDeep(existing, ctx); err != nil {
			var zero T
			return zero, err
		}
		return existing, nil
	}
	s.mu.RUnlock()

	item, err := s.At(name)
	if err != nil {
		var zero T
		return zero, err
	}

	if _, err := EnsureDeep(item, ctx); err != nil {
		var zero T
		return zero, err
	}

	return item, nil
}

// Require returns the named item only if its file already exists on disk.
func (s *FileSlot[T]) Require(name string) (T, error) {
	child := s.root.File(name)
	info, err := os.Stat(child.Path())
	if err != nil {
		var zero T
		return zero, fmt.Errorf("file slot item %q not found under %s: %w", name, s.Path(), err)
	}
	if info.IsDir() {
		var zero T
		return zero, fmt.Errorf("file slot item %q not found under %s: path is a directory", name, s.Path())
	}

	return s.At(name)
}

// Compose

// ComposePath binds the slot root and clears the cache.
func (s *FileSlot[T]) ComposePath(path string) {
	s.root = NewDir(path)
	s.items = make(map[string]T)
}

func (s *FileSlot[T]) setComposeBase(path string) {
	s.root.setComposeBase(path)
}

func (s *FileSlot[T]) setDeclaredPath(path string) {
	s.root.setDeclaredPath(path)
}

// Ensure

// Ensure creates the slot root directory.
func (s *FileSlot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}

// EnsureDeep ensures the slot root and all currently cached items.
//
// It does not invent uncached entries.
func (s *FileSlot[T]) EnsureDeep(ctx Context) (ResultCode, error) {
	if err := s.root.Ensure(ctx); err != nil {
		return EnsureFailed, err
	}

	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := EnsureDeep(item, ctx); err != nil {
			return EnsureFailed, fmt.Errorf("file slot item %q: %w", name, err)
		}
	}

	return EnsureEnsured, nil
}

// Load

// LoadDeep discovers direct child files from disk, composes them, and loads
// them recursively.
func (s *FileSlot[T]) LoadDeep(ctx Context) (ResultCode, error) {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return LoadMissing, nil
		}
		return LoadFailed, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return LoadFailed, fmt.Errorf("compose file slot item %q: %w", name, err)
		}

		if _, err := LoadDeep(item, ctx); err != nil {
			return LoadFailed, fmt.Errorf("load file slot item %q: %w", name, err)
		}
	}

	return LoadTraversed, nil
}

// Scan

// ScanDeep scans only currently cached items.
//
// It does not discover new file slot entries from disk.
func (s *FileSlot[T]) ScanDeep(ctx Context) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := ScanDeep(item, ctx); err != nil {
			return ScanFailed, fmt.Errorf("scan file slot item %q: %w", name, err)
		}
	}

	return ScanTraversed, nil
}

// Discover

// DiscoverDeep discovers direct child files from disk and scans them
// recursively without loading typed content.
func (s *FileSlot[T]) DiscoverDeep(ctx Context) (ResultCode, error) {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return DiscoverMissing, nil
		}
		return DiscoverFailed, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return DiscoverFailed, fmt.Errorf("compose file slot item %q: %w", name, err)
		}

		if _, err := DiscoverDeep(item, ctx); err != nil {
			return DiscoverFailed, fmt.Errorf("discover file slot item %q: %w", name, err)
		}
	}

	return DiscoverTraversed, nil
}

// Sync

// SyncDeep ensures and syncs only currently cached items.
//
// The preparation ensure phase respects ctx.EnsurePolicy.
//
// It does not invent uncached entries.
func (s *FileSlot[T]) SyncDeep(ctx Context) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := EnsureDeep(item, ctx); err != nil {
			return SyncFailed, fmt.Errorf("ensure file slot item %q: %w", name, err)
		}
		if _, err := SyncDeep(item, ctx); err != nil {
			return SyncFailed, fmt.Errorf("sync file slot item %q: %w", name, err)
		}
	}

	return SyncTraversed, nil
}

// Render

// RenderDeep renders only currently cached items.
func (s *FileSlot[T]) RenderDeep() error {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := RenderDeep(item); err != nil {
			return fmt.Errorf("render file slot item %q: %w", name, err)
		}
	}

	return nil
}

// Default

// DefaultDeep applies defaults only to currently cached items.
func (s *FileSlot[T]) DefaultDeep() error {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := DefaultDeep(item); err != nil {
			return fmt.Errorf("default file slot item %q: %w", name, err)
		}
	}

	return nil
}

// Validate

// ValidateDeep validates only currently cached items.
func (s *FileSlot[T]) ValidateDeep(opts ValidateOptions) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := ValidateDeep(item, opts); err != nil {
			return ValidateFailed, fmt.Errorf("validate file slot item %q: %w", name, err)
		}
	}

	return ValidateTraversed, nil
}
