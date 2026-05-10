package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"
)

type SlotEntry[T any] struct {
	Name string
	Item T
}

type Slot[T any] struct {
	root  Dir
	mu    sync.RWMutex
	items map[string]T
}

func NewSlot[T any](root Dir) Slot[T] {
	return Slot[T]{root: root}
}

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

func (s *Slot[T]) Exists() bool {
	return s.root.Exists()
}

func (s *Slot[T]) Root() Dir {
	return s.root
}

func (s *Slot[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}

func (s *Slot[T]) Has(name string) bool {
	_, err := os.Stat(s.root.Dir(name).Path())
	return err == nil
}

func (s *Slot[T]) Get(name string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[name]
	return item, ok
}

func (s *Slot[T]) Put(name string, item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]T)
	}
	s.items[name] = item
}

func (s *Slot[T]) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, name)
}

func (s *Slot[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[string]T)
}

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

func (s *Slot[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, entry := range s.Entries() {
			if !yield(entry.Name, entry.Item) {
				return
			}
		}
	}
}

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

func (s *Slot[T]) At(name string) (T, error) {
	s.mu.RLock()
	if item, ok := s.items[name]; ok {
		s.mu.RUnlock()
		return item, nil
	}
	s.mu.RUnlock()

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

	if existing, ok := s.items[name]; ok {
		return existing, nil
	}

	s.items[name] = item
	return item, nil
}

func (s *Slot[T]) MustAt(name string) T {
	v, err := s.At(name)
	if err != nil {
		panic(err)
	}
	return v
}

func (s *Slot[T]) Add(name string, ctx Context) (T, error) {
	childRoot := s.root.Dir(name)

	if err := childRoot.Ensure(ctx); err != nil {
		var zero T
		return zero, err
	}

	item, err := ComposeAs[T](childRoot)
	if err != nil {
		var zero T
		return zero, err
	}

	if err := EnsureDeep(item, ctx); err != nil {
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

func (s *Slot[T]) Require(name string) (T, error) {
	child := s.root.Dir(name)
	if _, err := os.Stat(child.Path()); err != nil {
		var zero T
		return zero, fmt.Errorf("slot item %q not found under %s: %w", name, s.Path(), err)
	}

	return s.At(name)
}

// Compose

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

func (s *Slot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}

func (s *Slot[T]) EnsureDeep(ctx Context) error {
	return s.ensureDeep(ctx)
}

func (s *Slot[T]) ensureDeep(ctx Context) error {
	if err := s.root.Ensure(ctx); err != nil {
		return err
	}

	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := s.root.Dir(name).Ensure(ctx); err != nil {
			return fmt.Errorf("slot item root %q: %w", name, err)
		}
		if err := EnsureDeep(item, ctx); err != nil {
			return fmt.Errorf("slot item %q: %w", name, err)
		}
	}

	return nil
}

func (s *Slot[T]) ensureDeepReport(ctx Context) error {
	return reportEnsure(ctx, s.Path(), func() error {
		return s.ensureDeep(ctx)
	})
}

// Load

func (s *Slot[T]) LoadDeep(ctx Context) error {
	return s.loadDeep(ctx)
}

func (s *Slot[T]) loadDeep(ctx Context) error {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return fmt.Errorf("compose slot item %q: %w", name, err)
		}

		if err := LoadDeep(item, ctx); err != nil {
			return fmt.Errorf("load slot item %q: %w", name, err)
		}
	}

	return nil
}

func (s *Slot[T]) loadDeepReport(ctx Context) error {
	return reportLoad(ctx, s.Path(), func() (ResultCode, error) {
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

			if err := LoadDeep(item, ctx); err != nil {
				return LoadFailed, fmt.Errorf("load slot item %q: %w", name, err)
			}
		}

		return LoadTraversed, nil
	})
}

// Scan

func (s *Slot[T]) ScanDeep(ctx Context) error {
	return s.scanDeep(ctx)
}

func (s *Slot[T]) scanDeep(ctx Context) error {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := ScanDeep(item, ctx); err != nil {
			return fmt.Errorf("scan slot item %q: %w", name, err)
		}
	}

	return nil
}

func (s *Slot[T]) scanDeepReport(ctx Context) error {
	return reportScan(ctx, s.Path(), func() (ResultCode, error) {
		s.mu.RLock()
		items := make(map[string]T, len(s.items))
		for name, item := range s.items {
			items[name] = item
		}
		s.mu.RUnlock()

		for name, item := range items {
			if err := ScanDeep(item, ctx); err != nil {
				return ScanFailed, fmt.Errorf("scan slot item %q: %w", name, err)
			}
		}

		return ScanTraversed, nil
	})
}

// Discover

func (s *Slot[T]) DiscoverDeep(ctx Context) error {
	return s.discoverDeep(ctx)
}

func (s *Slot[T]) discoverDeep(ctx Context) error {
	entries, err := os.ReadDir(s.root.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		item, err := s.At(name)
		if err != nil {
			return fmt.Errorf("compose slot item %q: %w", name, err)
		}

		if err := DiscoverDeep(item, ctx); err != nil {
			return fmt.Errorf("discover slot item %q: %w", name, err)
		}
	}

	return nil
}

func (s *Slot[T]) discoverDeepReport(ctx Context) error {
	return reportDiscover(ctx, s.Path(), func() (ResultCode, error) {
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

			if err := DiscoverDeep(item, ctx); err != nil {
				return DiscoverFailed, fmt.Errorf("discover slot item %q: %w", name, err)
			}
		}

		return DiscoverTraversed, nil
	})
}

// Sync

func (s *Slot[T]) SyncDeep(ctx Context) error {
	return s.syncDeep(ctx)
}

func (s *Slot[T]) syncDeep(ctx Context) error {
	s.mu.RLock()
	items := make(map[string]T, len(s.items))
	for name, item := range s.items {
		items[name] = item
	}
	s.mu.RUnlock()

	for name, item := range items {
		if err := EnsureDeep(item, ctx); err != nil {
			return fmt.Errorf("ensure slot item %q: %w", name, err)
		}
		if err := SyncDeep(item, ctx); err != nil {
			return fmt.Errorf("sync slot item %q: %w", name, err)
		}
	}

	return nil
}

func (s *Slot[T]) syncDeepReport(ctx Context) error {
	return reportSync(ctx, s.Path(), func() (ResultCode, error) {
		s.mu.RLock()
		items := make(map[string]T, len(s.items))
		for name, item := range s.items {
			items[name] = item
		}
		s.mu.RUnlock()

		for name, item := range items {
			if err := EnsureDeep(item, ctx); err != nil {
				return SyncFailed, fmt.Errorf("ensure slot item %q: %w", name, err)
			}
			if err := SyncDeep(item, ctx); err != nil {
				return SyncFailed, fmt.Errorf("sync slot item %q: %w", name, err)
			}
		}

		return SyncTraversed, nil
	})
}

// Render

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
