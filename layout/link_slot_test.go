package layout

import (
	"path/filepath"
	"testing"
)

func TestLinkSlotRejectsNamesOutsideDirectChildren(t *testing.T) {
	var slot LinkSlot[Link]
	slot.ComposePath(filepath.Join(t.TempDir(), "context"))

	invalid := []string{"", ".", "..", "nested/README", "../README", `/tmp/README`}
	for _, name := range invalid {
		if slot.Has(name) {
			t.Fatalf("Has(%q) = true, want false", name)
		}
		if _, err := slot.ComposeMember(name); err == nil {
			t.Fatalf("ComposeMember(%q) error = nil, want non-nil", name)
		}
		if _, err := slot.Require(name); err == nil {
			t.Fatalf("Require(%q) error = nil, want non-nil", name)
		}
		if err := slot.Delete(name, DefaultContext); err == nil {
			t.Fatalf("Delete(%q) error = nil, want non-nil", name)
		}
	}
}
