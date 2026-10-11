package layout

import (
	"fmt"
	"iter"
	"os"
	"sort"
	"sync"

	"github.com/qlustra/conduit/internal/contracts"
)

// FileSlotEntry is one cached key-item pair returned by FileSlot.Entries.
type FileSlotEntry[T any] struct {
	Name string
	Item T
}

// FileSlot models repeated direct-child files under one directory.
//
// Each key maps directly to a file path below the slot root. FileSlot caches
// composed items in memory. Managed Layout operations discover and load members.
type FileSlot[T any] struct {
	root     Dir
	mu       sync.RWMutex
	items    map[string]T
	composer contracts.MemberComposer
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
	if err := validateSlotItemName("file slot item", name); err != nil {
		return false
	}
	info, err := os.Lstat(s.root.File(name).Path())
	return err == nil && info.Mode().IsRegular()
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
func (s *FileSlot[T]) Delete(name string, ctx Context) error {
	if err := validateSlotItemName("file slot item", name); err != nil {
		return err
	}
	if err := s.root.File(name).DeleteIfExists(ctx); err != nil {
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

// ChildKind identifies the physical kind of a slot member.
func (s *FileSlot[T]) ChildKind() string { return "file" }

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

// ComposeMember returns the cached item for name, composing and caching it on demand when
// needed.
//
// ComposeMember does not ensure the file exists on disk.
// Uncached members require a member composer installed during managed composition.
func (s *FileSlot[T]) ComposeMember(name string) (T, error) {
	if err := validateSlotItemName("file slot item", name); err != nil {
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
func (s *FileSlot[T]) MustComposeMember(name string) T {
	v, err := s.ComposeMember(name)
	if err != nil {
		panic(err)
	}
	return v
}

// Require returns the named item only if its file already exists on disk.
func (s *FileSlot[T]) Require(name string) (T, error) {
	if err := validateSlotItemName("file slot item", name); err != nil {
		var zero T
		return zero, err
	}
	child := s.root.File(name)
	info, err := os.Lstat(child.Path())
	if err != nil {
		var zero T
		return zero, fmt.Errorf("file slot item %q not found under %s: %w", name, s.Path(), err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		var zero T
		return zero, fmt.Errorf("file slot item %q not found under %s: path is a symlink", name, s.Path())
	}
	if info.IsDir() {
		var zero T
		return zero, fmt.Errorf("file slot item %q not found under %s: path is a directory", name, s.Path())
	}

	return s.ComposeMember(name)
}

// Compose

// ComposePath binds the slot root and clears its cache and member composer.
func (s *FileSlot[T]) ComposePath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root = NewDir(path)
	s.items = make(map[string]T)
	s.composer = nil
}

func (s *FileSlot[T]) SetComposeBase(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root.SetComposeBase(path)
}

func (s *FileSlot[T]) SetDeclaredPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.root.SetDeclaredPath(path)
}

// SetMemberComposer configures composition of uncached members.
func (s *FileSlot[T]) SetMemberComposer(composer contracts.MemberComposer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.composer = composer
}

// Ensure

// Ensure creates the slot root directory.
func (s *FileSlot[T]) Ensure(ctx Context) error {
	return s.root.Ensure(ctx)
}
