package managed

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"

	"github.com/qlustra/conduit/internal/contracts"
	legend "github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

type composedMembers struct {
	children subject.Children
	err      func() error
}

func isMemberSource(at subject.Encounter) bool {
	switch at.Marker().(type) {
	case *legend.DirectoryMembers, *legend.DocumentMembers, *legend.LinkMembers:
		return true
	case *legend.Layout:
		// Entered members already participate through their containing source.
		return at.Depth() > 0
	default:
		return false
	}
}

// composeMembersFromFilesystem reads the collection directory and supplies
// matching members composed lazily during traversal.
func composeMembersFromFilesystem(node layout.CollectionCarrier) (composedMembers, error) {
	composer, ok := node.(contracts.MembersComposer)
	if !ok {
		return composedMembers{}, fmt.Errorf("member source at %q must support member composition", node.Path())
	}
	kind := node.ChildKind()
	switch kind {
	case "dir", "file", "link":
	default:
		return composedMembers{}, fmt.Errorf("member source at %q has unsupported child kind %q", node.Path(), kind)
	}
	entries, err := os.ReadDir(node.Path())
	if err != nil {
		return composedMembers{}, err
	}
	children, compositionErr := composer.ComposeMembers(PhysicalMemberNames(node.Path(), kind, entries))
	if compositionErr == nil {
		return composedMembers{}, fmt.Errorf("member source at %q must supply a composition error callback", node.Path())
	}
	return composedMembers{children: children, err: compositionErr}, nil
}

// Classify each entry only when traversal asks for its next member. This keeps
// disk checks, member composition, and the member's operation in legacy order.
func PhysicalMemberNames(path, kind string, entries []os.DirEntry) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, entry := range entries {
			if kind == "dir" {
				if entry.IsDir() && !yield(entry.Name(), nil) {
					return
				}
				continue
			}
			info, err := os.Lstat(filepath.Join(path, entry.Name()))
			if err != nil {
				if kind == "link" && os.IsNotExist(err) {
					continue
				}
				yield(entry.Name(), err)
				return
			}
			matches := info.Mode().IsRegular()
			if kind == "link" {
				matches = info.Mode()&os.ModeSymlink != 0
			}
			if matches && !yield(entry.Name(), nil) {
				return
			}
		}
	}
}
