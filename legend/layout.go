package legend

import (
	"github.com/qlustra/conduit/layout"
	saillegend "github.com/qlustra/sail/legend"
)

// Layout describes lasting participation in a Subject declaration.
// The layout tag selects a role, while path supplies its local path fragment.
// Path resolution and node behavior remain owned by Conduit. Unannotated fields
// do not participate. Unknown marked roles or qualifiers are declaration errors.
type Layout struct {
	saillegend.Legend
	Participants Participants `sail:"lane"`
}

func (Layout) Mark() string { return "layout" }

// Participants is an unordered-role band preserving Subject field order.
// Group is static nesting; member sources remain opaque live collection leaves.
type Participants struct {
	saillegend.Lane
	Directories      saillegend.MultiSpot[Directory]        `sail:"spot" multiplicity:"any"`
	Content          saillegend.MultiSpot[Content]          `sail:"spot" multiplicity:"any"`
	Files            saillegend.MultiSpot[RawFile]          `sail:"spot" multiplicity:"any"`
	Executables      saillegend.MultiSpot[Executable]       `sail:"spot" multiplicity:"any"`
	Groups           saillegend.MultiSpot[Group]            `sail:"spot" multiplicity:"any"`
	DirectoryMembers saillegend.MultiSpot[DirectoryMembers] `sail:"spot" multiplicity:"any"`
	DocumentMembers  saillegend.MultiSpot[DocumentMembers]  `sail:"spot" multiplicity:"any"`
	LinkMembers      saillegend.MultiSpot[LinkMembers]      `sail:"spot" multiplicity:"any"`
}

// Directory requires the complete directory API, including on authored
// carriers that embed Dir by value.
type Directory struct {
	saillegend.Landmark
	Path     saillegend.Feature   `sail:"feature" symbol:"path"`
	Required saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect   saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
}

func (Directory) Family() string { return "dir" }
func (Directory) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[layout.DirAPI]()
}

// Content records declaration annotations, not cached values or filesystem
// observations. Conduit owns their operational meaning and combination rules.
type Content struct {
	saillegend.Landmark
	Path     saillegend.Feature   `sail:"feature" symbol:"path"`
	Required saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect   saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
	Seeded   saillegend.Qualifier `sail:"qualifier" symbol:"seeded"`
	Derived  saillegend.Qualifier `sail:"qualifier" symbol:"derived"`
}

func (Content) Family() string { return "content" }
func (Content) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[layout.ContentFamily]()
}

// RawFile requires layout.FileAPI, including
// pointer-receiver methods. Wrappers may embed File and add or override methods
// with compatible signatures. Empty permits authored raw-file initialization;
// it does not define a content value.
type RawFile struct {
	saillegend.Landmark
	Path     saillegend.Feature   `sail:"feature" symbol:"path"`
	Required saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect   saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
	Empty    saillegend.Qualifier `sail:"qualifier" symbol:"empty"`
}

func (RawFile) Family() string { return "file" }
func (RawFile) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[layout.FileAPI]()
}

// Executable declares a managed executable file.
type Executable struct {
	saillegend.Landmark
	Path     saillegend.Feature   `sail:"feature" symbol:"path"`
	Required saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect   saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
}

func (Executable) Family() string { return "exec" }
func (Executable) Accept() saillegend.Acceptance {
	return saillegend.AcceptContract[layout.ExecAPI]()
}

// Group descends into explicitly annotated nested structs. A grouping
// declaration introduces no new root, ownership, or inheritance of annotations.
// Its Path supplies the base for relative paths of its declared children.
type Group struct {
	saillegend.Region
	Path         saillegend.Feature `sail:"feature" symbol:"path"`
	Participants Participants       `sail:"lane"`
}

func (Group) Family() string { return "group" }

// DirectoryMembers records a directory-member source. Required and Detect
// describe the container. Conduit must confirm live ChildKind is "dir" before
// using this role operationally.
type DirectoryMembers struct {
	saillegend.Landmark
	Path     saillegend.Feature   `sail:"feature" symbol:"path"`
	Required saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect   saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
}

func (DirectoryMembers) Family() string { return "directories" }
func (DirectoryMembers) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[layout.CollectionFamily]()
}

// DocumentMembers records a file-member source. Required and Detect describe
// the container; MemberRequired, Seeded, and Derived describe selected member
// content. These are declaration facts only: Conduit owns canonical propagation
// and combination rules, and must confirm live ChildKind is "file".
type DocumentMembers struct {
	saillegend.Landmark
	Path           saillegend.Feature   `sail:"feature" symbol:"path"`
	Required       saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect         saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
	MemberRequired saillegend.Qualifier `sail:"qualifier" symbol:"member-required"`
	Seeded         saillegend.Qualifier `sail:"qualifier" symbol:"seeded"`
	Derived        saillegend.Qualifier `sail:"qualifier" symbol:"derived"`
}

func (DocumentMembers) Family() string { return "documents" }
func (DocumentMembers) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[layout.CollectionFamily]()
}

// LinkMembers records a symlink-member source. Required and Detect describe
// the container. Conduit must confirm live ChildKind is "link" before using
// this role operationally.
type LinkMembers struct {
	saillegend.Landmark
	Path     saillegend.Feature   `sail:"feature" symbol:"path"`
	Required saillegend.Qualifier `sail:"qualifier" symbol:"required"`
	Detect   saillegend.Qualifier `sail:"qualifier" symbol:"detect"`
}

func (LinkMembers) Family() string { return "links" }
func (LinkMembers) Accept() saillegend.Acceptance {
	return saillegend.AcceptContractFamily[layout.CollectionFamily]()
}
