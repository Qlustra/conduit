package managed

import (
	legend "github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// ensureCarrierPolicy classifies the complete native APIs on the outer carrier.
// Stateful content has syncable semantics even when it also supplies FileAPI;
// executables have executable semantics before their inherited file API.
func ensureCarrierPolicy(node any) (layout.EnsurePolicy, bool) {
	switch node.(type) {
	case layout.ContentState:
		return layout.EnsureSyncables, true
	case layout.ExecAPI:
		return layout.EnsureExecs, true
	case layout.DirAPI:
		return layout.EnsureDirs, true
	case layout.FileAPI:
		return layout.EnsureFiles, true
	default:
		return 0, false
	}
}

// isFilesystemLeaf preserves structural group traversal while recognizing raw
// carrier extensions at entered collection roots, where the marker is Layout.
func isFilesystemLeaf(node any, at subject.Encounter) bool {
	switch at.Marker().(type) {
	case *legend.RawFile, *legend.Directory, *legend.Executable:
		return true
	case *legend.Layout:
		if at.Depth() == 0 {
			return false
		}
	default:
		return false
	}
	switch node.(type) {
	case layout.FileAPI, layout.DirAPI:
		return true
	default:
		return false
	}
}
