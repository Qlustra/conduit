package layout

import (
	"iter"

	"github.com/qlustra/conduit/internal/contracts"
	"github.com/qlustra/sail/subject"
)

// collectionAPI is the shared non-generic part of the built-in collection APIs.
type collectionAPI interface {
	ChildKind() string
	Clear()
	ComposeMembers(names iter.Seq2[string, error]) (subject.Children, func() error)
	ComposePath(path string)
	ComposedBaseDir() (Dir, bool)
	ComposedRelativePath() (string, bool)
	DeclaredPath() (string, bool)
	Delete(name string, ctx Context) error
	Ensure(ctx Context) error
	Enter() (subject.Children, error)
	Exists() bool
	Has(name string) bool
	JoinComposedPath(parts ...string) (string, bool)
	JoinDeclaredPath(parts ...string) (string, bool)
	Keys() []string
	Len() int
	Path() string
	Remove(name string)
	Root() Dir
	SetComposeBase(path string)
	SetDeclaredPath(path string)
	SetMemberComposer(composer contracts.MemberComposer)
}

// SlotAPI is the complete exported API of a directory-member Slot with member type T.
type SlotAPI[T any] interface {
	collectionAPI
	All() iter.Seq2[string, T]
	ComposeMember(name string) (T, error)
	Entries() []SlotEntry[T]
	Get(name string) (T, bool)
	MustComposeMember(name string) T
	Put(name string, item T)
	Require(name string) (T, error)
}

// FileSlotAPI is the complete exported API of a FileSlot with member type T.
type FileSlotAPI[T any] interface {
	collectionAPI
	All() iter.Seq2[string, T]
	ComposeMember(name string) (T, error)
	Entries() []FileSlotEntry[T]
	Get(name string) (T, bool)
	MustComposeMember(name string) T
	Put(name string, item T)
	Require(name string) (T, error)
}

// LinkSlotAPI is the complete exported API of a LinkSlot with member type T.
type LinkSlotAPI[T LinkSlotItem] interface {
	collectionAPI
	All() iter.Seq2[string, T]
	ComposeMember(name string) (T, error)
	Entries() []LinkSlotEntry[T]
	Get(name string) (T, bool)
	MustComposeMember(name string) T
	Put(name string, item T)
	Require(name string) (T, error)
}
