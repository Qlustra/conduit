package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qlustra/conduit/layout"
)

type destinationMode uint8

const (
	destinationFlatten destinationMode = iota + 1

	destinationPreserveStructure
)

type destination struct {
	root layout.Dir
	mode destinationMode
}

func newDestination(root layout.Dir, mode destinationMode) destination {
	return destination{root: root, mode: mode}
}

func (dest destination) resolveFile(item Item[Blob]) (layout.File, error) {
	rel, err := itemOutputPath(item)
	if err != nil {
		return layout.File{}, err
	}
	if err := validateRelativeOutputPath(rel); err != nil {
		return layout.File{}, err
	}

	switch dest.mode {
	case destinationFlatten:
		base := filepath.Base(rel)
		if err := validateRelativeOutputPath(base); err != nil {
			return layout.File{}, err
		}
		return dest.root.File(base), nil
	case destinationPreserveStructure:
		return dest.root.File(rel), nil
	default:
		return layout.File{}, fmt.Errorf("unsupported destination mode %d", dest.mode)
	}
}

func itemOutputPath(item Item[Blob]) (string, error) {
	rel := item.Path
	if rel == "" {
		rel = item.Name
	}
	if rel == "" {
		rel = item.Key
	}
	if rel == "" {
		return "", fmt.Errorf("item has no output path")
	}
	return filepath.Clean(rel), nil
}

func validateRelativeOutputPath(rel string) error {
	if rel == "" {
		return fmt.Errorf("output path must not be empty")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("output path %q must be relative", rel)
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." {
		return fmt.Errorf("output path %q is not valid", rel)
	}
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("output path %q must not escape destination", rel)
	}
	return nil
}
