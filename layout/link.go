package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

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

type FileLink struct {
	Link
}

type DirLink struct {
	Link
}

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

func (l Link) Path() string {
	return l.path
}

func (l Link) Base() string {
	return filepath.Base(l.path)
}

func (l Link) Ext() string {
	_, ext := splitBaseExt(l.Base())
	return ext
}

func (l Link) Stem() string {
	stem, _ := splitBaseExt(l.Base())
	return stem
}

func (l Link) ComposedBaseDir() (Dir, bool) {
	if !l.composedBase {
		return Dir{}, false
	}
	return newDirWithCompose(l.composeBase, l.composeBase, true), true
}

func (l Link) DeclaredPath() (string, bool) {
	if !l.hasDeclared {
		return "", false
	}
	return l.declaredPath, true
}

func (l Link) JoinDeclaredPath(parts ...string) (string, bool) {
	declared, ok := l.DeclaredPath()
	if !ok {
		return "", false
	}
	return joinDeclaredPath(declared, parts...), true
}

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

func (l Link) RelTo(base Pather) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base path must not be nil")
	}
	return filepath.Rel(base.Path(), l.Path())
}

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

func (l Link) RelToPath(base string) (string, error) {
	return filepath.Rel(filepath.Clean(base), l.Path())
}

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

func (l Link) Exists() bool {
	ok, _ := l.isSymlink()
	return ok
}

func (l Link) Target() (string, bool) {
	if l.target == nil {
		return "", false
	}
	return *l.target, true
}

func (l *Link) MustTarget() string {
	if l.target == nil {
		panic("link target is not loaded")
	}
	return *l.target
}

func (l *Link) SetTarget(target string) {
	l.target = &target
	l.memory = MemoryDirty
}

func (l *Link) SetDefaultTarget(target string) bool {
	if l.target != nil {
		return false
	}
	l.SetTarget(target)
	return true
}

func (l Link) HasTarget() bool {
	return l.target != nil
}

func (l Link) HasContent() bool {
	return l.HasTarget()
}

func (l *Link) ClearTarget() {
	l.target = nil
	l.memory = MemoryUnknown
}

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

func (l Link) DiskState() DiskState {
	return l.disk
}

func (l Link) MemoryState() MemoryState {
	return l.memory
}

func (l Link) HasKnownDiskState() bool {
	return l.disk != DiskUnknown
}

func (l Link) WasObservedOnDisk() bool {
	return l.disk == DiskPresent
}

func (l Link) HasBeenLoaded() bool {
	return l.memory == MemoryLoaded || l.memory == MemorySynced || l.memory == MemoryDirty
}

func (l Link) IsDirty() bool {
	return l.memory == MemoryDirty
}

func (l FileLink) TargetFile() (File, bool) {
	targetPath, ok := l.ResolvedTargetPath()
	if !ok {
		return File{}, false
	}
	return NewFile(targetPath), true
}

func (l *FileLink) MustTargetFile() File {
	target, ok := l.TargetFile()
	if !ok {
		panic("link target is not loaded")
	}
	return target
}

func (l DirLink) TargetDir() (Dir, bool) {
	targetPath, ok := l.ResolvedTargetPath()
	if !ok {
		return Dir{}, false
	}
	return NewDir(targetPath), true
}

func (l *DirLink) MustTargetDir() Dir {
	target, ok := l.TargetDir()
	if !ok {
		panic("link target is not loaded")
	}
	return target
}

// Compose

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

func (l *Link) Unload() {
	l.target = nil
	l.memory = MemoryUnknown
}

// Discover

func (l *Link) Discover() (DiskState, error) {
	return l.Scan()
}

// Sync

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
