package management

import (
	"context"
	"fmt"
	"strings"

	"github.com/qlustra/conduit/mgmt"
	"github.com/qlustra/conduit/spec"
)

//go:generate go run github.com/qlustra/conduit/cmd/conduit-gen -type NotesManagement -output management_gen.go

// NotesManagement declares the public names above NotesLayout.
// Its generated Notes controller owns a Space; each collection member receives
// a generated Notebook controller over the same cached child layout.
type NotesManagement struct {
	Space     spec.Space[NotesLayout]
	Notebooks spec.Collection[NotebookManagement] `scope:"from=Space;bind=Notebooks" create:"NotebookCreation"`
}

type NotebookManagement struct {
	Scope    spec.Scope[NotebookLayout]
	Manifest spec.Document[Manifest] `scope:"from=Scope;bind=Manifest"`
	Readme   spec.Document[string]   `scope:"from=Scope;bind=Readme" ops:"read"`
}

// NewNotebook is application input, separate from the stored manifest.
type NewNotebook struct {
	Title string
}

// NotebookCreation supplies domain values and context. The runtime prepares,
// renders, validates, and writes the new member before returning its handle.
var NotebookCreation = mgmt.Creation[NotebookLayout, NewNotebook]{
	Initialize: func(_ context.Context, tree *NotebookLayout, input NewNotebook) error {
		tree.Manifest.Set(Manifest{Title: strings.TrimSpace(input.Title)})
		return nil
	},
	Context: func(_ context.Context, tree *NotebookLayout) error {
		tree.Readme.SetContext(tree.Manifest.MustGet())
		return nil
	},
	Validate: func(_ context.Context, tree *NotebookLayout) error {
		if tree.Manifest.MustGet().Title == "" {
			return fmt.Errorf("notebook title is required")
		}
		return nil
	},
}
