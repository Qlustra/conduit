package path

import (
	"fmt"
	"os"
	"path/filepath"
)

type expectedNodeKind uint8

const (
	ExpectFile expectedNodeKind = iota + 1
	ExpectDir
	ExpectLink
)

// PathSafetyPolicy controls how mutating operations treat symlinks encountered
// while resolving destination paths.
type PathSafetyPolicy uint8

const (
	// PathSafetyRejectSymlinkParents rejects existing symlink parents during
	// mutating path resolution.
	PathSafetyRejectSymlinkParents PathSafetyPolicy = iota

	// PathSafetyFollowSymlinks preserves the library's historical path-following
	// behavior.
	PathSafetyFollowSymlinks
)

func GuardPathMutation(path string, policy PathSafetyPolicy, kind expectedNodeKind) error {
	if err := GuardPathParents(path, policy); err != nil {
		return err
	}
	return GuardNodeKind(path, kind)
}

func GuardPathParents(path string, policy PathSafetyPolicy) error {
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

func GuardNodeKind(path string, kind expectedNodeKind) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	switch kind {
	case ExpectFile:
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %s is a symlink, not a file", path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("path %s is not a file", path)
		}
	case ExpectDir:
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %s is a symlink, not a directory", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %s is not a directory", path)
		}
	case ExpectLink:
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("path %s is not a symlink", path)
		}
	default:
		return fmt.Errorf("unsupported node-kind guard %d", kind)
	}

	return nil
}
