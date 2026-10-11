package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"

	"github.com/qlustra/conduit/internal/contracts"
)

// LinkSlotItem is the built-in link node family supported by LinkSlot,
// as values or pointers. Pointer members retain their cached state when used.
type LinkSlotItem interface {
	Link | *Link | FileLink | *FileLink | DirLink | *DirLink
}

// LinkSlotEntry is one cached key-item pair returned by LinkSlot.Entries.
type LinkSlotEntry[T LinkSlotItem] struct {
	Name string
	Item T
}

// LinkSlot models repeated direct-child symlink entries under one directory.
//
// Each key maps directly to a symlink path below the slot root. LinkSlot caches
// composed items in memory. Managed Layout operations discover and load members.
type LinkSlot[T LinkSlotItem] struct {
	root     Dir
	mu       sync.RWMutex
	items    map[string]T
	composer contracts.MemberComposer
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
	if err := validateSlotItemName("link slot item", name); err != nil {
		return false
	}
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
func (s *LinkSlot[T]) Delete(name string, ctx Context) error {
	if err := validateSlotItemName("link slot item", name); err != nil {
		return err
	}
	link := NewLink(s.root.File(name).Path())
	if err := link.Delete(ctx); err != nil {
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

// ChildKind identifies the physical kind of a slot member.
func (s *LinkSlot[T]) ChildKind() string { return "link" }

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

// ComposeMember returns the cached item for name, composing and caching it on demand when
// needed.
//
// ComposeMember does not ensure the symlink exists on disk.
// Uncached members require a member composer installed during managed composition.
func (s *LinkSlot[T]) ComposeMember(name string) (T, error) {
	if err := validateSlotItemName("link slot item", name); err != nil {
		var zero T
		return zero, err
	}
	s.mu.RLock()
	if item, ok := s.items[name]; ok {
		s.mu.RUnlock()
		return item, nil
	}
	root, composer := s.root, s.composer
	s.mu.RUnlock()

	item, err := composeSlotMember[T](composer, root, root.File(name).Path())
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

// MustComposeMember returns ComposeMember(name) or panics on composition error.
func (s *LinkSlot[T]) MustComposeMember(name string) T {
	v, err := s.ComposeMember(name)
	if err != nil {
		panic(err)
	}
	return v
}

// Require returns the named item only if its symlink entry already exists on
// disk.
func (s *LinkSlot[T]) Require(name string) (T, error) {
	if err := validateSlotItemName("link slot item", name); err != nil {
		var zero T
		return zero, err
	}
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

	return s.ComposeMember(name)
}

// ComposePath binds the slot root and clears its cache and member composer.
func (s *LinkSlot[T]) ComposePath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root = NewDir(path)
	s.items = make(map[string]T)
	s.composer = nil
}

func (s *LinkSlot[T]) SetComposeBase(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root.SetComposeBase(path)
}

func (s *LinkSlot[T]) SetDeclaredPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root.SetDeclaredPath(path)
}

// SetMemberComposer configures composition of uncached members.
func (s *LinkSlot[T]) SetMemberComposer(composer contracts.MemberComposer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.composer = composer
}

// Ensure creates the slot root directory.
func (s *LinkSlot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
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
