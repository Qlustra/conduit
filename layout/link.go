package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

// Link models a symlink entry with explicit disk and memory state.
//
// Link manages only the symlink node at its own path. It does not create,
// validate, or synchronize the target payload beyond resolving and observing
// the target path when asked.
type Link struct {
	path         string
	composeBase  string
	composedBase bool
	declaredPath string
	hasDeclared  bool

	target *string
	disk   DiskState
	memory MemoryState
}

// FileLink is a Link that exposes the resolved target as a File handle.
type FileLink struct {
	Link
}

// DirLink is a Link that exposes the resolved target as a Dir handle.
type DirLink struct {
	Link
}

// NewLink returns a standalone symlink handle for path.
func NewLink(path string) Link {
	return newLinkWithCompose(path, "", false)
}

func newLinkWithCompose(path string, composeBase string, composed bool) Link {
	link := Link{
		path:   filepath.Clean(path),
		disk:   DiskUnknown,
		memory: MemoryUnknown,
	}
	if composed {
		link.composeBase = filepath.Clean(composeBase)
		link.composedBase = true
	}
	return link
}

// Path returns the symlink path itself.
func (l Link) Path() string {
	return l.path
}

// Base returns the final path element of the link path.
func (l Link) Base() string {
	return filepath.Base(l.path)
}

// Ext returns the final extension of the link path.
func (l Link) Ext() string {
	_, ext := splitBaseExt(l.Base())
	return ext
}

// Stem returns the final path element of the link path without its final
// extension.
func (l Link) Stem() string {
	stem, _ := splitBaseExt(l.Base())
	return stem
}

// ComposedBaseDir returns the root directory that anchored composition, when
// the handle belongs to a composed tree.
func (l Link) ComposedBaseDir() (Dir, bool) {
	if !l.composedBase {
		return Dir{}, false
	}
	return newDirWithCompose(l.composeBase, l.composeBase, true), true
}

// DeclaredPath returns the node's own layout tag fragment when the handle was
// attached through Compose.
func (l Link) DeclaredPath() (string, bool) {
	if !l.hasDeclared {
		return "", false
	}
	return l.declaredPath, true
}

// JoinDeclaredPath joins parts onto the node's declared layout fragment.
func (l Link) JoinDeclaredPath(parts ...string) (string, bool) {
	declared, ok := l.DeclaredPath()
	if !ok {
		return "", false
	}
	return joinDeclaredPath(declared, parts...), true
}

// ComposedRelativePath returns the path relative to the tree's compose base.
func (l Link) ComposedRelativePath() (string, bool) {
	if !l.composedBase {
		return "", false
	}
	rel, err := filepath.Rel(l.composeBase, l.path)
	if err != nil {
		return "", false
	}
	return rel, true
}

// JoinComposedPath joins parts onto the compose-base-relative path.
func (l Link) JoinComposedPath(parts ...string) (string, bool) {
	rel, ok := l.ComposedRelativePath()
	if !ok {
		return "", false
	}
	if len(parts) == 0 {
		return rel, true
	}
	return filepath.Join(append([]string{rel}, parts...)...), true
}

// RelTo returns the path relative to base.
func (l Link) RelTo(base Pather) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base path must not be nil")
	}
	return filepath.Rel(base.Path(), l.Path())
}

// JoinRelTo joins parts onto the path relative to base.
func (l Link) JoinRelTo(base Pather, parts ...string) (string, error) {
	rel, err := l.RelTo(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

// RelToPath returns the path relative to base.
func (l Link) RelToPath(base string) (string, error) {
	return filepath.Rel(filepath.Clean(base), l.Path())
}

// JoinRelToPath joins parts onto the path relative to base.
func (l Link) JoinRelToPath(base string, parts ...string) (string, error) {
	rel, err := l.RelToPath(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

// Exists reports whether a symlink exists at Path.
//
// Dangling symlinks still count as existing.
func (l Link) Exists() bool {
	ok, _ := l.isSymlink()
	return ok
}

// Target returns the cached raw symlink target string, if any.
func (l Link) Target() (string, bool) {
	if l.target == nil {
		return "", false
	}
	return *l.target, true
}

// MustTarget returns the cached target string or panics when it is absent.
func (l *Link) MustTarget() string {
	if l.target == nil {
		panic("link target is not loaded")
	}
	return *l.target
}

// SetTarget stores a raw symlink target string in memory and marks the link
// dirty.
func (l *Link) SetTarget(target string) {
	l.target = &target
	l.memory = MemoryDirty
}

// SetDefaultTarget stores target only when no cached target is present.
//
// It returns whether the default was applied.
func (l *Link) SetDefaultTarget(target string) bool {
	if l.target != nil {
		return false
	}
	l.SetTarget(target)
	return true
}

// HasTarget reports whether a target string is currently cached.
func (l Link) HasTarget() bool {
	return l.target != nil
}

// HasContent reports whether a target string is currently cached.
func (l Link) HasContent() bool {
	return l.HasTarget()
}

// ClearTarget removes the cached target string and resets memory state.
func (l *Link) ClearTarget() {
	l.target = nil
	l.memory = MemoryUnknown
}

// ResolvedTargetPath resolves the cached target against the link's parent
// directory when it is relative.
func (l Link) ResolvedTargetPath() (string, bool) {
	target, ok := l.Target()
	if !ok {
		return "", false
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target), true
	}
	return filepath.Clean(filepath.Join(filepath.Dir(l.path), target)), true
}

// TargetExists reports whether the resolved target currently exists.
func (l Link) TargetExists() (bool, error) {
	targetPath, ok := l.ResolvedTargetPath()
	if !ok {
		return false, nil
	}
	_, err := os.Stat(targetPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsDangling reports whether the cached target currently resolves to a missing
// filesystem entry.
func (l Link) IsDangling() (bool, error) {
	if !l.HasTarget() {
		return false, nil
	}
	exists, err := l.TargetExists()
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// Validate reports an error when Path exists but is not a symlink.
func (l Link) Validate() error {
	if l.Path() == "" {
		return nil
	}

	info, err := os.Lstat(l.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("path %s is not a symlink", l.Path())
	}

	return nil
}

// Delete removes the symlink when it exists, clears cached target state, and
// marks disk state missing.
//
// Delete fails if Path exists but is not a symlink.
func (l *Link) Delete() error {
	info, err := os.Lstat(l.Path())
	if err != nil {
		if os.IsNotExist(err) {
			l.ClearTarget()
			l.disk = DiskMissing
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("path %s is not a symlink", l.Path())
	}
	if err := os.Remove(l.Path()); err != nil {
		return err
	}
	l.ClearTarget()
	l.disk = DiskMissing
	return nil
}

// DiskState returns the last known disk-state metadata.
func (l Link) DiskState() DiskState {
	return l.disk
}

// MemoryState returns the last known memory-state metadata.
func (l Link) MemoryState() MemoryState {
	return l.memory
}

// HasKnownDiskState reports whether disk state is something other than
// DiskUnknown.
func (l Link) HasKnownDiskState() bool {
	return l.disk != DiskUnknown
}

// WasObservedOnDisk reports whether the last known disk state is DiskPresent.
func (l Link) WasObservedOnDisk() bool {
	return l.disk == DiskPresent
}

// HasBeenLoaded reports whether memory state has progressed beyond
// MemoryUnknown.
func (l Link) HasBeenLoaded() bool {
	return l.memory == MemoryLoaded || l.memory == MemorySynced || l.memory == MemoryDirty
}

// IsDirty reports whether the cached target has changed since load or sync.
func (l Link) IsDirty() bool {
	return l.memory == MemoryDirty
}

// TargetFile returns the resolved link target as a File handle.
func (l FileLink) TargetFile() (File, bool) {
	targetPath, ok := l.ResolvedTargetPath()
	if !ok {
		return File{}, false
	}
	return NewFile(targetPath), true
}

// MustTargetFile returns the resolved link target as a File handle or panics
// when no target is cached.
func (l *FileLink) MustTargetFile() File {
	target, ok := l.TargetFile()
	if !ok {
		panic("link target is not loaded")
	}
	return target
}

// Validate reports an error when Path exists but is not a symlink, or when a
// cached resolved target exists and is not a file.
func (l FileLink) Validate() error {
	if err := l.Link.Validate(); err != nil {
		return err
	}

	target, ok := l.TargetFile()
	if !ok {
		return nil
	}

	info, err := os.Stat(target.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("resolved link target %s is not a file", target.Path())
	}

	return nil
}

// TargetDir returns the resolved link target as a Dir handle.
func (l DirLink) TargetDir() (Dir, bool) {
	targetPath, ok := l.ResolvedTargetPath()
	if !ok {
		return Dir{}, false
	}
	return NewDir(targetPath), true
}

// MustTargetDir returns the resolved link target as a Dir handle or panics
// when no target is cached.
func (l *DirLink) MustTargetDir() Dir {
	target, ok := l.TargetDir()
	if !ok {
		panic("link target is not loaded")
	}
	return target
}

// Validate reports an error when Path exists but is not a symlink, or when a
// cached resolved target exists and is not a directory.
func (l DirLink) Validate() error {
	if err := l.Link.Validate(); err != nil {
		return err
	}

	target, ok := l.TargetDir()
	if !ok {
		return nil
	}

	info, err := os.Stat(target.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("resolved link target %s is not a directory", target.Path())
	}

	return nil
}

// Compose

// ComposePath binds the link to path and resets cached target and state.
func (l *Link) ComposePath(path string) {
	l.path = filepath.Clean(path)
	l.composeBase = ""
	l.composedBase = false
	l.declaredPath = ""
	l.hasDeclared = false
	l.target = nil
	l.disk = DiskUnknown
	l.memory = MemoryUnknown
}

func (l *Link) setComposeBase(path string) {
	l.composeBase = filepath.Clean(path)
	l.composedBase = true
}

func (l *Link) setDeclaredPath(path string) {
	l.declaredPath = path
	l.hasDeclared = true
}

// Load

// Load reads the raw symlink target from disk into memory.
//
// Load succeeds for dangling symlinks because the raw target string is still
// readable through os.Readlink.
func (l *Link) Load() (bool, error) {
	target, state, err := l.readTarget()
	if err != nil {
		l.target = nil
		l.disk = state
		l.memory = MemoryUnknown
		return false, err
	}
	if state == DiskMissing {
		l.target = nil
		l.disk = DiskMissing
		l.memory = MemoryUnknown
		return false, nil
	}
	l.target = &target
	l.disk = DiskPresent
	l.memory = MemoryLoaded
	return true, nil
}

// Unload clears the cached target and resets memory state.
//
// It preserves the current disk-state metadata.
func (l *Link) Unload() {
	l.target = nil
	l.memory = MemoryUnknown
}

// Discover

// Discover refreshes disk-state metadata without replacing the cached target.
//
// For Link, Discover has the same local effect as Scan.
func (l *Link) Discover() (DiskState, error) {
	return l.Scan()
}

// Sync

// Sync creates or rewrites the symlink from the cached target when policy
// allows the current state.
//
// Sync ensures the parent directory for the link itself but does not create or
// validate the link target payload.
func (l *Link) Sync(ctx Context) (ResultCode, error) {
	if l.target == nil {
		return SyncSkippedNoContent, nil
	}
	if !ctx.syncPolicy().allows(l.memory, l.disk) {
		return SyncSkippedPolicy, nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), ctx.DirMode); err != nil {
		return SyncFailed, err
	}
	if err := l.replace(*l.target); err != nil {
		return SyncFailed, err
	}
	l.disk = DiskPresent
	l.memory = MemorySynced
	return SyncWritten, nil
}

// Scan

// Scan refreshes disk-state metadata without replacing the cached target.
//
// Scan fails if Path exists but is not a symlink.
func (l *Link) Scan() (DiskState, error) {
	_, state, err := l.readTarget()
	if err != nil {
		l.disk = state
		return state, err
	}
	l.disk = state
	return state, nil
}

func (l Link) isSymlink() (bool, error) {
	info, err := os.Lstat(l.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

func (l *Link) readTarget() (string, DiskState, error) {
	info, err := os.Lstat(l.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return "", DiskMissing, nil
		}
		return "", DiskUnknown, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "", DiskUnknown, fmt.Errorf("path %s is not a symlink", l.Path())
	}
	target, err := os.Readlink(l.Path())
	if err != nil {
		return "", DiskUnknown, err
	}
	return target, DiskPresent, nil
}

func (l Link) replace(target string) error {
	info, err := os.Lstat(l.Path())
	if err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("path %s is not a symlink", l.Path())
		}
		if err := os.Remove(l.Path()); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.Symlink(target, l.Path())
}
