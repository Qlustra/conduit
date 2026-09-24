package mgmt_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/mgmt"
)

type notebookManifest struct {
	Title string `json:"title"`
}

type notebookReadme struct {
	layout.TextTemplate[notebookManifest]
}

func (*notebookReadme) Template() string { return "# {{.Title}}\n" }

type notebookLayout struct {
	Root     layout.Dir                         `layout:"."`
	Manifest formats.JSONFile[notebookManifest] `layout:"notebook.json" manage:"required"`
	Readme   notebookReadme                     `layout:"README.md" manage:"role=seeded"`
}

type researchLayout struct {
	Root      layout.Dir                   `layout:"."`
	Marker    layout.Dir                   `layout:".notes" manage:"detect,required"`
	Notebooks layout.Slot[*notebookLayout] `layout:"notebooks"`
}

// This is the complete manual path: ordinary storage declarations, a project
// controller, and a typed collection backed by the same cached layout values.
func ExampleCreate() {
	root, err := os.MkdirTemp("", "conduit-research-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	ctx := context.Background()

	var tree researchLayout
	if err := conduit.Compose(root, &tree); err != nil {
		panic(err)
	}
	project, err := mgmt.BindProject(tree.Root, &tree, mgmt.ProjectOptions{})
	if err != nil {
		panic(err)
	}
	// Binding retains the existing layout. Bootstrap creates the marker and
	// collection directories; it does not invent collection members.
	if _, err := mgmt.Bootstrap(ctx, project.Scope(), struct{}{},
		mgmt.Initialization[researchLayout, struct{}]{}, mgmt.BootstrapPolicy{}); err != nil {
		panic(err)
	}
	detection, err := project.Detect(ctx, mgmt.DetectionPolicy{})
	if err != nil {
		panic(err)
	}
	fmt.Println("recognized:", detection.Match)

	notebooks, err := mgmt.BindCollection(project, &tree.Notebooks, mgmt.CollectionOptions{})
	if err != nil {
		panic(err)
	}
	creation := mgmt.Creation[notebookLayout, string]{
		Initialize: func(_ context.Context, notebook *notebookLayout, title string) error {
			notebook.Manifest.Set(notebookManifest{Title: title})
			return nil
		},
		Context: func(_ context.Context, notebook *notebookLayout) error {
			manifest, _ := notebook.Manifest.Get()
			notebook.Readme.SetContext(manifest)
			return nil
		},
		Validate: func(_ context.Context, notebook *notebookLayout) error {
			manifest, _ := notebook.Manifest.Get()
			if manifest.Title == "" {
				return fmt.Errorf("notebook title is required")
			}
			return nil
		},
	}
	created, err := mgmt.Create(ctx, notebooks, "conduit", "Conduit research", creation, mgmt.CreatePolicy{})
	if err != nil {
		// created.Outcome remains available if application partially succeeded.
		panic(err)
	}
	fmt.Println("created:", created.Outcome.Applied())

	manifest, err := mgmt.BindDocument(created.Value, &created.Value.Layout().Manifest,
		mgmt.DocumentOptions[notebookManifest]{})
	if err != nil {
		panic(err)
	}
	if _, err := manifest.Update(ctx, func(value *notebookManifest) error {
		value.Title = "Conduit control surface"
		return nil
	}, mgmt.UpdatePolicy{}); err != nil {
		panic(err)
	}
	value, err := manifest.Read(ctx, mgmt.ReadPolicy{})
	if err != nil {
		panic(err)
	}
	fmt.Println("title:", value.Title)

	// A document update changes that document. The user-maintained README was
	// seeded during creation and keeps its original text.
	readme, err := os.ReadFile(created.Value.Layout().Readme.Path())
	if err != nil {
		panic(err)
	}
	fmt.Printf("readme: %s", readme)
	inventory, err := notebooks.List(ctx, mgmt.ListPolicy{Validate: true})
	if err != nil {
		panic(err)
	}
	fmt.Println("notebook:", inventory.Entries[0].Key)
	// Output:
	// recognized: true
	// created: true
	// title: Conduit control surface
	// readme: # Conduit research
	// notebook: conduit
}

func ExampleBindSpace() {
	parent, err := os.MkdirTemp("", "conduit-space-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(parent)
	root := filepath.Join(parent, "research")
	var tree struct {
		Root  layout.Dir `layout:"."`
		Notes layout.Dir `layout:"notes"`
		Logs  layout.Dir `layout:"logs"`
	}
	if err := conduit.Compose(root, &tree); err != nil {
		panic(err)
	}
	// A Space needs a chosen root and a layout, without a detection marker.
	space, err := mgmt.BindSpace(tree.Root, &tree, mgmt.SpaceOptions{})
	if err != nil {
		panic(err)
	}
	if _, err := space.Scope().Scaffold(context.Background(), mgmt.ScaffoldPolicy{}); err != nil {
		panic(err)
	}
	fmt.Println("notes:", tree.Notes.Exists())
	fmt.Println("logs:", tree.Logs.Exists())
	// Output:
	// notes: true
	// logs: true
}
