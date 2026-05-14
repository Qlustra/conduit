package layout

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// LinkSlotItem is the built-in link node family supported by LinkSlot.
type LinkSlotItem interface {
	Link | FileLink | DirLink
}

// LinkSlotEntry is one cached key-item pair returned by LinkSlot.Entries.
type LinkSlotEntry[T LinkSlotItem] struct {
	Name string
	Item T
}

// LinkSlot models repeated direct-child symlink entries under one directory.
//
// Each key maps directly to a symlink path below the slot root. LinkSlot caches
// composed items in memory and discovers or loads them only when explicitly
// asked.
type LinkSlot[T LinkSlotItem] struct {
	root  Dir
	mu    sync.RWMutex
	items map[string]T
}

// NewLinkSlot returns a link slot rooted at root.
func NewLinkSlot[T LinkSlotItem](root Dir) LinkSlot[T] {
	return LinkSlot[T]{root: root}
}

// Path returns the slot root directory path.
func (s *LinkSlot[T]) Path() string {
	return s.root.Path()
}

func (s *LinkSlot[T]) ComposedBaseDir() (Dir, bool) {
	return s.root.ComposedBaseDir()
}

func (s *LinkSlot[T]) DeclaredPath() (string, bool) {
	return s.root.DeclaredPath()
}

func (s *LinkSlot[T]) JoinDeclaredPath(parts ...string) (string, bool) {
	return s.root.JoinDeclaredPath(parts...)
}

func (s *LinkSlot[T]) ComposedRelativePath() (string, bool) {
	return s.root.ComposedRelativePath()
}

func (s *LinkSlot[T]) JoinComposedPath(parts ...string) (string, bool) {
	return s.root.JoinComposedPath(parts...)
}

// Exists reports whether the slot root currently exists on disk.
func (s *LinkSlot[T]) Exists() bool {
	return s.root.Exists()
}

// Root returns the slot root directory handle.
func (s *LinkSlot[T]) Root() Dir {
	return s.root
}

// Len returns the number of cached items.
func (s *LinkSlot[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}

// Has reports whether a symlink entry with name currently exists on disk.
func (s *LinkSlot[T]) Has(name string) bool {
	ok, err := isSymlinkPath(s.root.File(name).Path())
	return err == nil && ok
}

// Get returns the cached item for name without composing a missing one.
func (s *LinkSlot[T]) Get(name string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[name]
	return item, ok
}

// Put inserts or replaces a cached item without touching disk.
func (s *LinkSlot[T]) Put(name string, item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}
	s.items[name] = item
}

// Remove evicts a cached item without touching disk.
func (s *LinkSlot[T]) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
}

// Delete removes the child symlink from disk and evicts the cached item.
func (s *LinkSlot[T]) Delete(name string) error {
	link := NewLink(s.root.File(name).Path())
	if err := link.Delete(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
	return nil
}

// Clear drops all cached items without touching disk.
func (s *LinkSlot[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[string]T)
}

// Entries returns a sorted snapshot of cached entries.
//
// Entries is cache-based only; it does not discover from disk.
func (s *LinkSlot[T]) Entries() []LinkSlotEntry[T] {
	s.mu.RLock()
	entries := make([]LinkSlotEntry[T], 0, len(s.items))
	for name, item := range s.items {
		entries = append(entries, LinkSlotEntry[T]{
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
func (s *LinkSlot[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, entry := range s.Entries() {
			if !yield(entry.Name, entry.Item) {
				return
			}
		}
	}
}

// Keys returns the cached item names in sorted order.
func (s *LinkSlot[T]) Keys() []string {
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
// At does not ensure the symlink exists on disk.
func (s *LinkSlot[T]) At(name string) (T, error) {
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
func (s *LinkSlot[T]) MustAt(name string) T {
	v, err := s.At(name)
	if err != nil {
		panic(err)
	}
	return v
}

// Add ensures the slot root exists on disk, composes the item, and caches it.
//
// The link entry itself is still materialized by Sync or SyncDeep.
func (s *LinkSlot[T]) Add(name string, ctx Context) (T, error) {
	if err := s.root.Ensure(ctx); err != nil {
		var zero T
		return zero, err
	}

	s.mu.RLock()
	if existing, ok := s.items[name]; ok {
		s.mu.RUnlock()
		return existing, nil
	}
	s.mu.RUnlock()

	return s.At(name)
}

// Require returns the named item only if its symlink entry already exists on
// disk.
func (s *LinkSlot[T]) Require(name string) (T, error) {
	childPath := s.root.File(name).Path()
	ok, err := isSymlinkPath(childPath)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("link slot item %q not found under %s: %w", name, s.Path(), err)
	}
	if !ok {
		var zero T
		return zero, fmt.Errorf("link slot item %q not found under %s: path is not a symlink", name, s.Path())
	}

	return s.At(name)
}

// ComposePath binds the slot root and clears the cache.
func (s *LinkSlot[T]) ComposePath(path string) {
	s.root = NewDir(path)
	s.items = make(map[string]T)
}

func (s *LinkSlot[T]) setComposeBase(path string) {
	s.root.setComposeBase(path)
}

func (s *LinkSlot[T]) setDeclaredPath(path string) {
	s.root.setDeclaredPath(path)
}

// Ensure creates the slot root directory.
func (s *LinkSlot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}

// EnsureDeep ensures the slot root and visits all currently cached items.
//
// It does not invent uncached entries or materialize the link entries
// themselves.
func (s *LinkSlot[T]) EnsureDeep(ctx Context) (ResultCode, error) {
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
		if _, err := EnsureDeep(any(&item), ctx); err != nil {
			return EnsureFailed, fmt.Errorf("link slot item %q: %w", name, err)
		}
	}

	return EnsureEnsured, nil
}

// LoadDeep discovers direct child symlinks from disk, composes them, and loads
// them recursively.
func (s *LinkSlot[T]) LoadDeep(ctx Context) (ResultCode, error) {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return LoadMissing, nil
		}
		return LoadFailed, err
	}

	for _, entry := range entries {
		childPath := filepath.Join(s.root.Path(), entry.Name())
		ok, err := isSymlinkPath(childPath)
		if err != nil || !ok {
			if err != nil {
				return LoadFailed, err
			}
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return LoadFailed, fmt.Errorf("compose link slot item %q: %w", name, err)
		}

		if _, err := LoadDeep(any(&item), ctx); err != nil {
			return LoadFailed, fmt.Errorf("load link slot item %q: %w", name, err)
		}
		s.Put(name, item)
	}

	return LoadTraversed, nil
}

// ScanDeep scans only currently cached items.
//
// It does not discover new link slot entries from disk.
func (s *LinkSlot[T]) ScanDeep(ctx Context) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := ScanDeep(any(&item), ctx); err != nil {
			return ScanFailed, fmt.Errorf("scan link slot item %q: %w", name, err)
		}
		s.Put(name, item)
	}

	return ScanTraversed, nil
}

// DiscoverDeep discovers direct child symlinks from disk and scans them
// recursively without loading target content.
func (s *LinkSlot[T]) DiscoverDeep(ctx Context) (ResultCode, error) {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return DiscoverMissing, nil
		}
		return DiscoverFailed, err
	}

	for _, entry := range entries {
		childPath := filepath.Join(s.root.Path(), entry.Name())
		ok, err := isSymlinkPath(childPath)
		if err != nil || !ok {
			if err != nil {
				return DiscoverFailed, err
			}
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return DiscoverFailed, fmt.Errorf("compose link slot item %q: %w", name, err)
		}

		if _, err := DiscoverDeep(any(&item), ctx); err != nil {
			return DiscoverFailed, fmt.Errorf("discover link slot item %q: %w", name, err)
		}
		s.Put(name, item)
	}

	return DiscoverTraversed, nil
}

// SyncDeep ensures and syncs only currently cached items.
//
// The preparation ensure phase respects ctx.EnsurePolicy.
//
// It does not invent uncached entries.
func (s *LinkSlot[T]) SyncDeep(ctx Context) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := EnsureDeep(any(&item), ctx); err != nil {
			return SyncFailed, fmt.Errorf("ensure link slot item %q: %w", name, err)
		}
		if _, err := SyncDeep(any(&item), ctx); err != nil {
			return SyncFailed, fmt.Errorf("sync link slot item %q: %w", name, err)
		}
		s.Put(name, item)
	}

	return SyncTraversed, nil
}

// ValidateDeep validates only currently cached items.
func (s *LinkSlot[T]) ValidateDeep(opts ValidateOptions) (ResultCode, error) {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if _, err := ValidateDeep(any(&item), opts); err != nil {
			return ValidateFailed, fmt.Errorf("validate link slot item %q: %w", name, err)
		}
	}

	return ValidateTraversed, nil
}

func isSymlinkPath(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
