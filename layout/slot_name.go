package layout

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
