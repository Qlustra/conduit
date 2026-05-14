package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"
)

// SlotEntry is one cached key-item pair returned by Slot.Entries.
type SlotEntry[T any] struct {
	Name string
	Item T
}

// Slot models repeated child layouts under one directory.
//
// Each key maps to one child root below the slot path. Slot caches composed
// items in memory and discovers or loads them only when explicitly asked.
type Slot[T any] struct {
	root  Dir
	mu    sync.RWMutex
	items map[string]T
}

// NewSlot returns a slot rooted at root.
func NewSlot[T any](root Dir) Slot[T] {
	return Slot[T]{root: root}
}

// Path returns the slot root directory path.
func (s *Slot[T]) Path() string {
	return s.root.Path()
}

func (s *Slot[T]) ComposedBaseDir() (Dir, bool) {
	return s.root.ComposedBaseDir()
}

func (s *Slot[T]) DeclaredPath() (string, bool) {
	return s.root.DeclaredPath()
}

func (s *Slot[T]) JoinDeclaredPath(parts ...string) (string, bool) {
	return s.root.JoinDeclaredPath(parts...)
}

func (s *Slot[T]) ComposedRelativePath() (string, bool) {
	return s.root.ComposedRelativePath()
}

func (s *Slot[T]) JoinComposedPath(parts ...string) (string, bool) {
	return s.root.JoinComposedPath(parts...)
}

// Exists reports whether the slot root currently exists on disk.
func (s *Slot[T]) Exists() bool {
	return s.root.Exists()
}

// Root returns the slot root directory handle.
func (s *Slot[T]) Root() Dir {
	return s.root
}

// Len returns the number of cached items.
func (s *Slot[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}

// Has reports whether a child directory with name currently exists on disk.
func (s *Slot[T]) Has(name string) bool {
	if err := validateSlotItemName("slot item", name); err != nil {
		return false
	}
	_, err := os.Stat(s.root.Dir(name).Path())
	return err == nil
}

// Get returns the cached item for name without composing a missing one.
func (s *Slot[T]) Get(name string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[name]
	return item, ok
}

// Put inserts or replaces a cached item without touching disk.
func (s *Slot[T]) Put(name string, item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}
	s.items[name] = item
}

// Remove evicts a cached item without touching disk.
func (s *Slot[T]) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
}

// Delete removes the child directory tree from disk and evicts the cached
// item.
func (s *Slot[T]) Delete(name string) error {
	if err := validateSlotItemName("slot item", name); err != nil {
		return err
	}
	if err := s.root.Dir(name).DeleteIfExists(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
	return nil
}

// Clear drops all cached items without touching disk.
func (s *Slot[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[string]T)
}

// Entries returns a sorted snapshot of cached entries.
//
// Entries is cache-based only; it does not discover from disk.
func (s *Slot[T]) Entries() []SlotEntry[T] {
	s.mu.RLock()
	entries := make([]SlotEntry[T], 0, len(s.items))
	for name, item := range s.items {
		entries = append(entries, SlotEntry[T]{
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
func (s *Slot[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, entry := range s.Entries() {
			if !yield(entry.Name, entry.Item) {
				return
			}
		}
	}
}

// Keys returns the cached item names in sorted order.
func (s *Slot[T]) Keys() []string {
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
// At does not ensure the item exists on disk.
func (s *Slot[T]) At(name string) (T, error) {
	if err := validateSlotItemName("slot item", name); err != nil {
		var zero T
		return zero, err
	}
	s.mu.RLock()
	// Fast path under RLock. On a miss, compose outside the mutex and
	// re-check under Lock so concurrent callers converge on one cached item.
	if item, ok := s.items[name]; ok {
		s.mu.RUnlock()
		return item, nil
	}
	s.mu.RUnlock()

	// Compose outside the lock, then check again under the write lock.
	item, err := ComposeAs[T](s.root.Dir(name))
	if err != nil {
		var zero T
		return zero, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}

	// Another goroutine may have populated s.items[name] while we were composing.
	if existing, ok := s.items[name]; ok {
		return existing, nil
	}

	s.items[name] = item
	return item, nil
}

// MustAt returns At(name) or panics on composition error.
func (s *Slot[T]) MustAt(name string) T {
	v, err := s.At(name)
	if err != nil {
		panic(err)
	}
	return v
}

// Add ensures the child root exists on disk, composes the item, ensures its
// declared structure, and caches it.
func (s *Slot[T]) Add(name string, ctx Context) (T, error) {
	if err := validateSlotItemName("slot item", name); err != nil {
		var zero T
		return zero, err
	}
	childRoot := s.root.Dir(name)

	if err := childRoot.Ensure(ctx); err != nil {
		var zero T
		return zero, err
	}

	s.mu.RLock()
	if existing, ok := s.items[name]; ok {
		s.mu.RUnlock()
		return existing, nil
	}
	s.mu.RUnlock()

	item, err := ComposeAs[T](childRoot)
	if err != nil {
		var zero T
		return zero, err
	}

	if _, err := EnsureDeep(item, ctx); err != nil {
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

// Require returns the named item only if its child directory already exists on
// disk.
func (s *Slot[T]) Require(name string) (T, error) {
	if err := validateSlotItemName("slot item", name); err != nil {
		var zero T
		return zero, err
	}
	child := s.root.Dir(name)
	if _, err := os.Stat(child.Path()); err != nil {
		var zero T
		return zero, fmt.Errorf("slot item %q not found under %s: %w", name, s.Path(), err)
	}

	return s.At(name)
}

// Compose

// ComposePath binds the slot root and clears the cache.
func (s *Slot[T]) ComposePath(path string) {
	s.root = NewDir(path)
	s.items = make(map[string]T)
}

func (s *Slot[T]) setComposeBase(path string) {
	s.root.setComposeBase(path)
}

func (s *Slot[T]) setDeclaredPath(path string) {
	s.root.setDeclaredPath(path)
}

// Ensure

// Ensure creates the slot root directory.
func (s *Slot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}

// EnsureDeep ensures the slot root and all currently cached items.
//
// It does not invent uncached entries.
func (s *Slot[T]) EnsureDeep(ctx Context) (ResultCode, error) {
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
		if err := s.root.Dir(name).Ensure(ctx); err != nil {
			return EnsureFailed, fmt.Errorf("slot item root %q: %w", name, err)
		}
		if _, err := EnsureDeep(item, ctx); err != nil {
			return EnsureFailed, fmt.Errorf("slot item %q: %w", name, err)
		}
	}

	return EnsureEnsured, nil
}

// Load

// LoadDeep discovers child directories from disk, composes them, and loads
// them recursively.
func (s *Slot[T]) LoadDeep(ctx Context) (ResultCode, error) {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return LoadMissing, nil
		}
		return LoadFailed, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return LoadFailed, fmt.Errorf("compose slot item %q: %w", name, err)
		}

		if _, err := LoadDeep(item, ctx); err != nil {
			return LoadFailed, fmt.Errorf("load slot item %q: %w", name, err)
		}
	}

	return LoadTraversed, nil
}

// Scan

// ScanDeep scans only currently cached items.
//
// It does not discover new slot entries from disk.
func (s *Slot[T]) ScanDeep(ctx Context) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := ScanDeep(item, ctx); err != nil {
			return ScanFailed, fmt.Errorf("scan slot item %q: %w", name, err)
		}
	}

	return ScanTraversed, nil
}

// Discover

// DiscoverDeep discovers child directories from disk and scans them
// recursively without loading typed content.
func (s *Slot[T]) DiscoverDeep(ctx Context) (ResultCode, error) {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return DiscoverMissing, nil
		}
		return DiscoverFailed, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return DiscoverFailed, fmt.Errorf("compose slot item %q: %w", name, err)
		}

		if _, err := DiscoverDeep(item, ctx); err != nil {
			return DiscoverFailed, fmt.Errorf("discover slot item %q: %w", name, err)
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
func (s *Slot[T]) SyncDeep(ctx Context) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := EnsureDeep(item, ctx); err != nil {
			return SyncFailed, fmt.Errorf("ensure slot item %q: %w", name, err)
		}
		if _, err := SyncDeep(item, ctx); err != nil {
			return SyncFailed, fmt.Errorf("sync slot item %q: %w", name, err)
		}
	}

	return SyncTraversed, nil
}

// Render

// RenderDeep renders only currently cached items.
func (s *Slot[T]) RenderDeep() error {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := RenderDeep(item); err != nil {
			return fmt.Errorf("render slot item %q: %w", name, err)
		}
	}

	return nil
}

// Default

// DefaultDeep applies defaults only to currently cached items.
func (s *Slot[T]) DefaultDeep() error {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := DefaultDeep(item); err != nil {
			return fmt.Errorf("default slot item %q: %w", name, err)
		}
	}

	return nil
}

// Validate

// ValidateDeep validates only currently cached items.
func (s *Slot[T]) ValidateDeep(opts ValidateOptions) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := ValidateDeep(item, opts); err != nil {
			return ValidateFailed, fmt.Errorf("validate slot item %q: %w", name, err)
		}
	}

	return ValidateTraversed, nil
}
