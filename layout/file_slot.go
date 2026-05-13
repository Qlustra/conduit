package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"
)

type FileSlotEntry[T any] struct {
	Name string
	Item T
}

type FileSlot[T any] struct {
	root  Dir
	mu    sync.RWMutex
	items map[string]T
}

func NewFileSlot[T any](root Dir) FileSlot[T] {
	return FileSlot[T]{root: root}
}

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

func (s *FileSlot[T]) Exists() bool {
	return s.root.Exists()
}

func (s *FileSlot[T]) Root() Dir {
	return s.root
}

func (s *FileSlot[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}

func (s *FileSlot[T]) Has(name string) bool {
	info, err := os.Stat(s.root.File(name).Path())
	return err == nil && !info.IsDir()
}

func (s *FileSlot[T]) Get(name string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[name]
	return item, ok
}

func (s *FileSlot[T]) Put(name string, item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}
	s.items[name] = item
}

func (s *FileSlot[T]) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
}

func (s *FileSlot[T]) Delete(name string) error {
	if err := s.root.File(name).DeleteIfExists(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
	return nil
}

func (s *FileSlot[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[string]T)
}

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

func (s *FileSlot[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, entry := range s.Entries() {
			if !yield(entry.Name, entry.Item) {
				return
			}
		}
	}
}

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

func (s *FileSlot[T]) MustAt(name string) T {
	v, err := s.At(name)
	if err != nil {
		panic(err)
	}
	return v
}

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

func (s *FileSlot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}

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
