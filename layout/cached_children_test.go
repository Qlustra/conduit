package layout

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	_ CachedChildren = (*Slot[*Dir])(nil)
	_ CachedChildren = (*FileSlot[*File])(nil)
	_ CachedChildren = (*LinkSlot[Link])(nil)
)

func TestCachedChildrenRetainPointerIdentityWithoutDiscovery(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}
	root := NewDir(t.TempDir())
	if err := os.Mkdir(filepath.Join(root.Path(), "uncached"), 0o755); err != nil {
		t.Fatal(err)
	}
	slot := NewSlot[*item](root)
	zeta, err := slot.BindChild("zeta")
	if err != nil {
		t.Fatal(err)
	}
	alpha, err := slot.BindChild("alpha")
	if err != nil {
		t.Fatal(err)
	}
	children := slot.CachedChildren()
	if len(children) != 2 || children[0].Name != "alpha" || children[1].Name != "zeta" {
		t.Fatalf("children = %#v", children)
	}
	if children[0].Value != alpha || children[1].Value != zeta {
		t.Fatal("cached traversal lost pointer identity")
	}
	if children[0].Path != filepath.Join(root.Path(), "alpha") || slot.ChildKind() != "dir" {
		t.Fatal("incorrect physical child metadata")
	}
	children[0].Value.(*item).Config.Set(map[string]string{"changed": "through traversal"})
	cached, _ := slot.Get("alpha")
	if cached.Config.MustGet()["changed"] != "through traversal" {
		t.Fatal("traversal changed a copy instead of cached content")
	}
	if _, err := os.Stat(children[0].Path); !os.IsNotExist(err) {
		t.Fatalf("binding created directory: %v", err)
	}
	if _, err := slot.BindChild("../outside"); err == nil || slot.Len() != 2 {
		t.Fatal("invalid physical child name was accepted")
	}
	// Using the returned snapshot does not retain a collection lock.
	slot.Remove("alpha")
	if children[0].Value != alpha {
		t.Fatal("eviction modified the captured snapshot")
	}
}

func TestFileSlotCachedChildrenRetainContentReferences(t *testing.T) {
	root := NewDir(t.TempDir())
	if err := os.WriteFile(filepath.Join(root.Path(), "uncached.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	slot := NewFileSlot[*testMapFile](root)
	bound, err := slot.BindChild("state.json")
	if err != nil {
		t.Fatal(err)
	}
	children := slot.CachedChildren()
	if len(children) != 1 || children[0].Name != "state.json" || children[0].Value != bound || children[0].Path != filepath.Join(root.Path(), "state.json") || slot.ChildKind() != "file" {
		t.Fatalf("children = %#v", children)
	}
	children[0].Value.(*testMapFile).Set(map[string]string{"a": "b"})
	cached, _ := slot.Get("state.json")
	if !cached.HasContent() {
		t.Fatal("traversal did not mutate cached file")
	}
	if _, err := os.Stat(children[0].Path); !os.IsNotExist(err) {
		t.Fatalf("binding created file: %v", err)
	}
}

func TestCachedChildrenHonestlyExposeValueItems(t *testing.T) {
	slot := NewSlot[Dir](NewDir(t.TempDir()))
	if _, err := slot.BindChild("entry"); err != nil {
		t.Fatal(err)
	}
	child := slot.CachedChildren()[0]
	value, ok := child.Value.(Dir)
	if !ok {
		t.Fatalf("value slot exposed artificial mutable reference: %T", child.Value)
	}
	value.ComposePath("changed-copy")
	cached, _ := slot.Get("entry")
	if cached.Path() != child.Path {
		t.Fatal("changing returned value changed cached slot")
	}
	links := NewLinkSlot[Link](NewDir(t.TempDir()))
	if _, err := links.BindChild("link"); err != nil {
		t.Fatal(err)
	}
	linkChild := links.CachedChildren()[0]
	if _, ok := linkChild.Value.(Link); !ok || links.ChildKind() != "link" || linkChild.Path != filepath.Join(links.Path(), "link") {
		t.Fatalf("link child = %#v", linkChild)
	}
}
