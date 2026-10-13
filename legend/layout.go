package legend

import "github.com/qlustra/conduit/internal/layoutdecl"

// Layout records declared participation. Layout roles come from layout tags;
// local paths come from path tags. Untagged fields do not participate.
type Layout = layoutdecl.Layout

// Participants preserves declaration order across filesystem roles. Groups are
// static nesting; collection sources are opaque live leaves.
type Participants = layoutdecl.Participants

// Directory requires the native Dir API, including on authored embeddings.
type Directory = layoutdecl.Directory

// Content records required, detect, seeded, and derived declaration facts.
// Conduit owns their operational meaning and native carrier behavior.
type Content = layoutdecl.Content

// RawFile requires the native File API. Empty permits authored initialization.
type RawFile = layoutdecl.RawFile

// Executable records a managed executable with its native Exec API.
type Executable = layoutdecl.Executable

// Group nests annotated structs; its local path supplies their relative base.
type Group = layoutdecl.Group

// DirectoryMembers records a directory-member source and container annotations.
type DirectoryMembers = layoutdecl.DirectoryMembers

// DocumentMembers records a file-member source, with separate container and
// member content annotations. Conduit owns their propagation rules.
type DocumentMembers = layoutdecl.DocumentMembers

// LinkMembers records a symlink-member source and container annotations.
type LinkMembers = layoutdecl.LinkMembers
