package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"

	"github.com/qlustra/conduit/internal/contracts"
)

// SlotEntry is one cached key-item pair returned by Slot.Entries.
type SlotEntry[T any] struct {
	Name string
	Item T
}

// Slot models repeated child layouts under one directory.
//
// Each key maps to one child root below the slot path. Slot caches composed
// items in memory. Managed Layout operations discover and load members.
type Slot[T any] struct {
	root     Dir
	mu       sync.RWMutex
	items    map[string]T
	composer contracts.MemberComposer
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
func (s *Slot[T]) Delete(name string, ctx Context) error {
	if err := validateSlotItemName("slot item", name); err != nil {
		return err
	}
	if err := s.root.Dir(name).DeleteIfExists(ctx); err != nil {
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

// ChildKind identifies the physical kind of a slot member.
func (s *Slot[T]) ChildKind() string { return "dir" }

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

// ComposeMember returns the cached item for name, composing and caching it on demand when
// needed.
//
// ComposeMember does not ensure the item exists on disk.
// Uncached members require a member composer installed during managed composition.
func (s *Slot[T]) ComposeMember(name string) (T, error) {
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
	root, composer := s.root, s.composer
	s.mu.RUnlock()

	// Compose outside the lock, then check again under the write lock.
	item, err := composeSlotMember[T](composer, root, root.Dir(name).Path())
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

// MustComposeMember returns ComposeMember(name) or panics on composition error.
func (s *Slot[T]) MustComposeMember(name string) T {
	v, err := s.ComposeMember(name)
	if err != nil {
		panic(err)
	}
	return v
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

	return s.ComposeMember(name)
}

// Compose

// ComposePath binds the slot root and clears its cache and member composer.
func (s *Slot[T]) ComposePath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root = NewDir(path)
	s.items = make(map[string]T)
	s.composer = nil
}

func (s *Slot[T]) SetComposeBase(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root.SetComposeBase(path)
}

func (s *Slot[T]) SetDeclaredPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root.SetDeclaredPath(path)
}

// SetMemberComposer configures composition of uncached members.
func (s *Slot[T]) SetMemberComposer(composer contracts.MemberComposer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.composer = composer
}

// Ensure

// Ensure creates the slot root directory.
func (s *Slot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}
