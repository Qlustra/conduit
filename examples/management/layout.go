// Package management demonstrates a generated control surface for a notes Space.
// Layouts describe storage; topology declarations describe the named operations.
package management

import (
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

// Manifest is the editable metadata of one notebook.
type Manifest struct {
	Title string `json:"title"`
}

// NotebookReadme seeds a human-maintained document from the initial manifest.
// Later changes to the manifest do not regenerate this seeded document.
type NotebookReadme struct {
	layout.TextTemplate[Manifest]
}

func (*NotebookReadme) Template() string {
	return "# {{.Title}}\n\nKeep your research notes here.\n"
}

type NotebookLayout struct {
	Root     layout.Dir                 `layout:"."`
	Manifest formats.JSONFile[Manifest] `layout:"notebook.json" manage:"required"`
	Readme   NotebookReadme             `layout:"README.md" manage:"role=seeded"`
}

type NotesLayout struct {
	Root      layout.Dir                   `layout:"."`
	Notebooks layout.Slot[*NotebookLayout] `layout:"notebooks"`
}
