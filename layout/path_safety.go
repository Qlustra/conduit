package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type expectedNodeKind uint8

const (
	expectFile expectedNodeKind = iota + 1
	expectDir
	expectLink
)

func guardPathMutation(path string, policy PathSafetyPolicy, kind expectedNodeKind) error {
	if err := guardPathParents(path, policy); err != nil {
		return err
	}
	return guardNodeKind(path, kind)
}

func guardPathParents(path string, policy PathSafetyPolicy) error {
	if policy == PathSafetyFollowSymlinks {
		return nil
	}

	current := filepath.Clean(filepath.Dir(path))
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("path parent %s is a symlink", current)
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func guardNodeKind(path string, kind expectedNodeKind) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	switch kind {
	case expectFile:
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %s is a symlink, not a file", path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("path %s is not a file", path)
		}
	case expectDir:
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %s is a symlink, not a directory", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %s is not a directory", path)
		}
	case expectLink:
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("path %s is not a symlink", path)
		}
	default:
		return fmt.Errorf("unsupported node-kind guard %d", kind)
	}

	return nil
}
