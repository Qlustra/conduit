package conduit_test

import (
	"strings"
	"testing"

	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

func TestCompileAcceptsMatchingSlotRoles(t *testing.T) {
	type group struct {
		Directories layout.Slot[*struct{}]        `layout:"directories" path:"directories"`
		Documents   layout.FileSlot[*layout.File] `layout:"documents" path:"documents"`
		Links       layout.LinkSlot[*layout.Link] `layout:"links" path:"links"`
	}
	type tree struct {
		Directories layout.Slot[*struct{}]        `layout:"directories" path:"directories"`
		Documents   layout.FileSlot[*layout.File] `layout:"documents" path:"documents"`
		Links       layout.LinkSlot[*layout.Link] `layout:"links" path:"links"`
		Group       *group                        `layout:"group" path:"nested"`
	}
	if _, err := conduit.Compile[tree](); err != nil {
		t.Fatalf("Compile() rejected matching slot roles: %v", err)
	}
}

func TestCompileRejectsMismatchedSlotRoles(t *testing.T) {
	type wrongDirectories struct {
		Members layout.FileSlot[*layout.File] `layout:"directories" path:"members"`
	}
	type wrongDocuments struct {
		Members layout.LinkSlot[*layout.Link] `layout:"documents" path:"members"`
	}
	type wrongLinks struct {
		Members layout.Slot[*struct{}] `layout:"links" path:"members"`
	}
	cases := []struct {
		name    string
		compile func() error
		kind    string
		want    string
	}{
		{"directories", compileRoleError[wrongDirectories], "file", "dir"},
		{"documents", compileRoleError[wrongDocuments], "link", "file"},
		{"links", compileRoleError[wrongLinks], "dir", "link"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assertRoleError(t, test.compile(), "collection at [0]", "has child kind \""+test.kind+"\"", "expected \""+test.want+"\"")
		})
	}
}

func TestCompileRejectsMismatchedSlotRoleInsideGroup(t *testing.T) {
	type group struct {
		Members layout.FileSlot[*layout.File] `layout:"directories" path:"members"`
	}
	type tree struct {
		Root  layout.Dir `layout:"dir" path:"."`
		Group group      `layout:"group" path:"nested"`
	}
	assertRoleError(t, compileRoleError[tree](), "collection at [1 0]", "has child kind \"file\"", "expected \"dir\"")
}

// These custom collections intentionally have no callable live-source hooks.
// Compilation can only use the state-independent ChildKind declaration.
type roleCustomDirectories struct{ layout.Slot[*struct{}] }
type roleCustomDocuments struct{ layout.FileSlot[*layout.File] }
type roleCustomLinks struct{ layout.LinkSlot[*layout.Link] }

func (*roleCustomDirectories) Path() string {
	panic("compilation must not read a live collection path")
}

func (*roleCustomDirectories) Enter() (subject.Children, error) {
	panic("compilation must not enter a live collection")
}

func (*roleCustomDocuments) Path() string {
	panic("compilation must not read a live collection path")
}

func (*roleCustomDocuments) Enter() (subject.Children, error) {
	panic("compilation must not enter a live collection")
}

func (*roleCustomLinks) Path() string {
	panic("compilation must not read a live collection path")
}

func (*roleCustomLinks) Enter() (subject.Children, error) {
	panic("compilation must not enter a live collection")
}

func (*roleCustomDirectories) ChildKind() string { return "dir" }
func (*roleCustomDocuments) ChildKind() string   { return "file" }
func (*roleCustomLinks) ChildKind() string       { return "link" }

var (
	_ layout.CollectionCarrier = (*roleCustomDirectories)(nil)
	_ layout.CollectionCarrier = (*roleCustomDocuments)(nil)
	_ layout.CollectionCarrier = (*roleCustomLinks)(nil)
)

func TestCompileAcceptsCustomSlotKindsWithoutLiveSourceAccess(t *testing.T) {
	type tree struct {
		Directories *roleCustomDirectories `layout:"directories" path:"directories"`
		Documents   roleCustomDocuments    `layout:"documents" path:"documents"`
		Links       roleCustomLinks        `layout:"links" path:"links"`
	}
	if _, err := conduit.Compile[tree](); err != nil {
		t.Fatalf("Compile() rejected compatible custom collections: %v", err)
	}
}

func TestCompileRejectsCustomSlotKindMismatch(t *testing.T) {
	type tree struct {
		Members roleCustomDocuments `layout:"directories" path:"members"`
	}
	assertRoleError(t, compileRoleError[tree](), "collection at [0]", "has child kind \"file\"", "expected \"dir\"")
}

type roleStateDependentSlot struct {
	roleCustomDirectories
	kind *string
}

func (s *roleStateDependentSlot) ChildKind() string {
	if s.kind == nil {
		panic("child kind requires initialized state")
	}
	return *s.kind
}

func TestCompileReportsStateDependentChildKind(t *testing.T) {
	type tree struct {
		Members roleStateDependentSlot `layout:"directories" path:"members"`
	}
	assertRoleError(t, compileRoleError[tree](), "collection at [0] child kind", "child kind requires initialized state")
}

func TestCompileRequiresConcreteChildKindDeclaration(t *testing.T) {
	type tree struct {
		Members layout.CollectionCarrier `layout:"directories" path:"members"`
	}
	if err := compileRoleError[tree](); err == nil {
		t.Fatal("Compile accepted an interface field without a concrete collection declaration")
	}
}

func TestComposeRejectsMismatchedSlotRoleInMemberChart(t *testing.T) {
	type member struct {
		Documents layout.FileSlot[*layout.File] `layout:"links" path:"documents"`
	}
	type tree struct {
		Members layout.Slot[*member] `layout:"directories" path:"members"`
	}
	structure, err := conduit.Compile[tree]()
	if err != nil {
		t.Fatalf("Compile() error = %v; the member chart is prepared during composition", err)
	}
	_, err = structure.Compose(t.TempDir())
	assertRoleError(t, err, "collection at [0] member Chart", "has child kind \"file\"", "expected \"link\"")
}

func compileRoleError[T any]() error {
	_, err := conduit.Compile[T]()
	return err
}

func assertRoleError(t *testing.T, err error, fragments ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected collection role validation error")
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q lacks %q", err, fragment)
		}
	}
}
