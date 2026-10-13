package layout

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qlustra/conduit/internal/contracts"
)

func composeSlotMember[T any](composer contracts.MemberComposer, root Dir, path string) (T, error) {
	if composer == nil {
		var zero T
		return zero, fmt.Errorf("compose slot member at %s: no member composer configured", path)
	}

	composeBase := root.Path()
	if base, ok := root.ComposedBaseDir(); ok {
		composeBase = base.Path()
	}

	storage := new(T)
	if err := composer.ComposeMember(storage, path, composeBase); err != nil {
		var zero T
		return zero, err
	}
	return *storage, nil
}

func validateSlotItemName(kind string, name string) error {
	if name == "" {
		return fmt.Errorf("invalid %s name %q: name must not be empty", kind, name)
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("invalid %s name %q: name must not be absolute", kind, name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid %s name %q: name must identify a direct child", kind, name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid %s name %q: name must identify a single direct child", kind, name)
	}
	if clean := filepath.Clean(name); clean != name {
		return fmt.Errorf("invalid %s name %q: name must remain unchanged after path cleaning", kind, name)
	}
	return nil
}
