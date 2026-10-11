package layout

import (
	"path/filepath"
	"testing"
)

func TestSlotsRequireAMemberComposerForMissingMembers(t *testing.T) {
	dir := NewDir(filepath.Join(t.TempDir(), "members"))
	directories := NewSlot[*Dir](dir)
	documents := NewFileSlot[*File](dir)
	links := NewLinkSlot[*Link](dir)
	for _, test := range []struct {
		name    string
		compose func() error
	}{
		{"directory", func() error { _, err := directories.ComposeMember("member"); return err }},
		{"document", func() error { _, err := documents.ComposeMember("member"); return err }},
		{"link", func() error { _, err := links.ComposeMember("member"); return err }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compose(); err == nil {
				t.Fatal("unconfigured slot constructed a missing member")
			}
		})
	}
	// Reading already registered cache entries does not invoke member construction.
	cached := NewFile(filepath.Join(dir.Path(), "cached"))
	documents.Put("cached", &cached)
	if got, err := documents.ComposeMember("cached"); err != nil || got != &cached {
		t.Fatalf("cached ComposeMember() = %p, %v", got, err)
	}
}
