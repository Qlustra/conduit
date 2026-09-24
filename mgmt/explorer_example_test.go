package mgmt_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qlustra/conduit/mgmt"
)

func ExampleExplorer_Gather() {
	root, err := os.MkdirTemp("", "conduit-notes-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	if err := os.WriteFile(filepath.Join(root, "design.md"), []byte("control surface"), 0600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(root, "debug.log"), []byte("not a note"), 0600); err != nil {
		panic(err)
	}

	// A chosen tree needs neither a layout declaration nor a project marker.
	notes, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{
		Selection: mgmt.ExploreSelection{
			Select: func(entry mgmt.ExploreEntry) bool {
				return entry.IsRegular() && strings.HasSuffix(entry.RelativePath, ".md")
			},
			Prune: func(entry mgmt.ExploreEntry) bool {
				return filepath.Base(entry.Path) == ".git"
			},
		},
	})
	if err != nil {
		panic(err)
	}
	gathered, err := notes.Gather(context.Background(), mgmt.GatherPolicy{
		MaxFileBytes:  1 << 20,
		MaxTotalBytes: 8 << 20,
	})
	if err != nil {
		// Both the partial files and path-specific diagnostics remain available.
		panic(err)
	}
	for _, file := range gathered.Files {
		fmt.Printf("%s: %s\n", file.Entry.RelativePath, file.Content)
	}
	// Output:
	// design.md: control surface
}
