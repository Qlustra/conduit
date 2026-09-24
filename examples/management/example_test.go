package management_test

import (
	"context"
	"fmt"
	"os"
	"strings"

	notesapp "github.com/qlustra/conduit/examples/management"
	"github.com/qlustra/conduit/mgmt"
)

func Example() {
	root, err := os.MkdirTemp("", "conduit-notes-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	ctx := context.Background()

	// NewNotes composes and binds; Bootstrap creates the missing structure.
	notes, err := notesapp.NewNotes(root, mgmt.SpaceOptions{})
	if err != nil {
		panic(err)
	}
	if _, err = notes.Bootstrap(ctx, mgmt.BootstrapPolicy{}); err != nil {
		panic(err)
	}

	// Domain input initializes both the typed manifest and the seeded README.
	created, err := notes.Notebooks.Create(ctx, "conduit", notesapp.NewNotebook{
		Title: "Conduit research",
	}, mgmt.CreatePolicy{})
	if err != nil {
		panic(err)
	}
	notebook := created.Value

	// Update reads, edits, and persists the manifest through the shared runtime.
	if _, err = notebook.Manifest.Update(ctx, func(value *notesapp.Manifest) error {
		value.Title = "Conduit design notes"
		return nil
	}, mgmt.UpdatePolicy{}); err != nil {
		panic(err)
	}

	manifest, err := notebook.Manifest.Read(ctx, mgmt.ReadPolicy{})
	if err != nil {
		panic(err)
	}
	readme, err := notebook.Readme.Read(ctx, mgmt.ReadPolicy{})
	if err != nil {
		panic(err)
	}
	members, err := notes.Notebooks.List(ctx, mgmt.ListPolicy{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Manifest:", manifest.Title)
	fmt.Println("Seeded README:", strings.SplitN(readme, "\n", 2)[0])
	fmt.Println("Notebooks:", len(members.Entries), members.Entries[0].Key)
	// Output:
	// Manifest: Conduit design notes
	// Seeded README: # Conduit research
	// Notebooks: 1 conduit
}
