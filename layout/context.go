package layout

import "os"

// SyncPolicy is a bitmask that controls which in-memory states Sync and
// SyncDeep may write.
//
// Memory-state bits select whether loaded, synced, or dirty values are
// eligible to be written. Disk-state bits optionally further restrict writes
// based on the last observed disk state.
type SyncPolicy uint8

const (
	// SyncOnLoaded allows sync for content that was loaded from disk and has not
	// changed since loading.
	SyncOnLoaded SyncPolicy = 1 << iota

	// SyncOnSynced allows sync for content that was already written by Conduit.
	SyncOnSynced

	// SyncOnDirty allows sync for content that was set or modified in memory.
	SyncOnDirty

	// SyncOnDiskUnknown allows sync when disk state has not been observed.
	SyncOnDiskUnknown

	// SyncOnDiskMissing allows sync only when disk was last observed missing.
	SyncOnDiskMissing

	// SyncOnDiskPresent allows sync only when disk was last observed present.
	SyncOnDiskPresent
)

const (
	// SyncRewrite allows loaded, synced, and dirty content to be written.
	SyncRewrite SyncPolicy = SyncOnLoaded | SyncOnSynced | SyncOnDirty

	// SyncIfDirty allows only dirty content to be written.
	SyncIfDirty SyncPolicy = SyncOnDirty

	// SyncIfUnsynced allows loaded and dirty content to be written, but skips
	// content that is already marked synced.
	SyncIfUnsynced SyncPolicy = SyncOnLoaded | SyncOnDirty

	// SyncIfMissing allows sync only when disk was last observed missing.
	//
	// Because it does not set any memory-state bits directly, the memory side
	// still defaults to SyncRewrite semantics.
	SyncIfMissing SyncPolicy = SyncOnDiskMissing
)

const (
	syncPolicyMemoryMask = SyncOnLoaded | SyncOnSynced | SyncOnDirty
	syncPolicyDiskMask   = SyncOnDiskUnknown | SyncOnDiskMissing | SyncOnDiskPresent
)

// Context carries per-operation filesystem modes, sync policy, and optional
// reporting hooks.
type Context struct {
	// DirMode is used when creating directories.
	DirMode os.FileMode

	// FileMode is used when creating regular files.
	FileMode os.FileMode

	// ExecMode is used when creating Exec files. When zero, Exec falls back to
	// FileMode and adds execute bits automatically.
	ExecMode os.FileMode

	// SyncPolicy controls which cached values Sync and SyncDeep may write.
	SyncPolicy SyncPolicy

	// Reporter, when non-nil, receives path-level results during deep
	// traversal.
	Reporter Reporter
}

// DefaultContext is the default mode and sync policy set used throughout the
// library examples.
//
// It creates directories with mode 0o755, regular files with 0o644,
// executables with 0o755, and uses SyncRewrite behavior.
var DefaultContext = Context{
	DirMode:    0o755,
	FileMode:   0o644,
	ExecMode:   0o755,
	SyncPolicy: SyncRewrite,
}

func (ctx Context) syncPolicy() SyncPolicy {
	return ctx.SyncPolicy
}

func (p SyncPolicy) normalizedMemory() SyncPolicy {
	p &= syncPolicyMemoryMask
	if p == 0 {
		return SyncRewrite
	}
	return p
}

func (p SyncPolicy) allowsMemory(state MemoryState) bool {
	switch state {
	case MemoryLoaded:
		return p&SyncOnLoaded != 0
	case MemorySynced:
		return p&SyncOnSynced != 0
	case MemoryDirty:
		return p&SyncOnDirty != 0
	default:
		return false
	}
}

func (p SyncPolicy) allowsDisk(state DiskState) bool {
	p &= syncPolicyDiskMask
	if p == 0 {
		return true
	}

	switch state {
	case DiskUnknown:
		return p&SyncOnDiskUnknown != 0
	case DiskMissing:
		return p&SyncOnDiskMissing != 0
	case DiskPresent:
		return p&SyncOnDiskPresent != 0
	default:
		return false
	}
}

func (p SyncPolicy) allows(memory MemoryState, disk DiskState) bool {
	return p.normalizedMemory().allowsMemory(memory) && p.allowsDisk(disk)
}
