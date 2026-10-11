package composer

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolvePath(base, composeBase, declaredPath string) (string, error) {
	if filepath.IsAbs(declaredPath) {
		return "", fmt.Errorf("path %q must be relative to the compose root", declaredPath)
	}

	path := filepath.Join(base, declaredPath)
	relative, err := filepath.Rel(composeBase, path)
	if err != nil {
		return "", fmt.Errorf("path %q relative to compose root %q: %w", declaredPath, composeBase, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q resolves outside compose root %q", declaredPath, composeBase)
	}
	return path, nil
}
