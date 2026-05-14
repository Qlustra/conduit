package layout

import "os"

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

// EnsurePolicy is a bitmask that controls which node kinds Ensure and
// EnsureDeep may materialize.
//
// The zero value preserves the historical default behavior and is treated as
// EnsureAll. Use EnsureNone when an explicit no-op ensure pass is desired.
type EnsurePolicy uint8

const (
	// EnsureDirs allows directory materialization.
	EnsureDirs EnsurePolicy = 1 << iota

	// EnsureFiles allows raw File materialization.
	EnsureFiles

	// EnsureExecs allows Exec materialization.
	EnsureExecs

	// EnsureSyncables allows stateful Syncer-backed nodes such as Format-backed
	// typed files to materialize their own backing files during ensure.
	EnsureSyncables

	ensurePolicyNoneSentinel
)

const (
	// EnsureAll preserves the library's historical ensure behavior.
	EnsureAll EnsurePolicy = EnsureDirs | EnsureFiles | EnsureExecs | EnsureSyncables

	// EnsureScaffold materializes only raw filesystem scaffolding and skips
	// stateful syncable nodes.
	EnsureScaffold EnsurePolicy = EnsureDirs | EnsureFiles | EnsureExecs

	// EnsureNone disables ensure materialization explicitly.
	EnsureNone EnsurePolicy = ensurePolicyNoneSentinel
)

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

// Context carries per-operation filesystem modes, ensure and sync policy, and
// optional reporting hooks.
type Context struct {
	// DirMode is used when creating directories.
	DirMode os.FileMode

	// FileMode is used when creating regular files.
	FileMode os.FileMode

	// ExecMode is used when creating Exec files. When zero, Exec falls back to
	// FileMode and adds execute bits automatically.
	ExecMode os.FileMode

	// EnsurePolicy controls which node kinds Ensure and EnsureDeep may
	// materialize. The zero value behaves like EnsureAll.
	EnsurePolicy EnsurePolicy

	// SyncPolicy controls which cached values Sync and SyncDeep may write.
	SyncPolicy SyncPolicy

	// PathSafetyPolicy controls whether mutating filesystem operations reject
	// symlink parents during path resolution.
	PathSafetyPolicy PathSafetyPolicy

	// Reporter, when non-nil, receives path-level results during deep
	// traversal.
	Reporter Reporter
}

// DefaultContext is the default mode and sync policy set used throughout the
// library examples.
//
// It creates directories with mode 0o755, regular files with 0o644,
// executables with 0o755, uses EnsureAll behavior, and uses SyncRewrite
// behavior.
var DefaultContext = Context{
	DirMode:          0o755,
	FileMode:         0o644,
	ExecMode:         0o755,
	EnsurePolicy:     EnsureAll,
	SyncPolicy:       SyncRewrite,
	PathSafetyPolicy: PathSafetyRejectSymlinkParents,
}

func (ctx Context) ensurePolicy() EnsurePolicy {
	return ctx.EnsurePolicy.normalized()
}

func (ctx Context) withEnsurePolicy(policy EnsurePolicy) Context {
	ctx.EnsurePolicy = policy
	return ctx
}

func (p EnsurePolicy) normalized() EnsurePolicy {
	if p&EnsureNone != 0 {
		return 0
	}
	p &= EnsureAll
	if p == 0 {
		return EnsureAll
	}
	return p
}

// Allow returns p with bits enabled.
//
// Bits outside the public EnsureAll mask are ignored. Allow clears the
// explicit EnsureNone sentinel so callers can build a policy up from
// EnsureNone.
func (p EnsurePolicy) Allow(bits EnsurePolicy) EnsurePolicy {
	if p&EnsureNone != 0 {
		p = 0
	}
	return p | (bits & EnsureAll)
}

// Deny returns p with bits disabled.
//
// Bits outside the public EnsureAll mask are ignored. When the resulting mask
// would be empty, Deny returns EnsureNone so the explicit no-op intent is
// preserved instead of falling back to default EnsureAll normalization.
func (p EnsurePolicy) Deny(bits EnsurePolicy) EnsurePolicy {
	if p&EnsureNone != 0 {
		return p
	}

	p &^= bits & EnsureAll
	if p == 0 {
		return EnsureNone
	}
	return p
}

// Has reports whether all requested bits are enabled on p.
//
// Bits outside the public EnsureAll mask are ignored.
func (p EnsurePolicy) Has(bits EnsurePolicy) bool {
	p = p.normalized()
	bits &= EnsureAll
	return bits != 0 && p&bits == bits
}

func (p EnsurePolicy) allowsDir() bool {
	return p.normalized()&EnsureDirs != 0
}

func (p EnsurePolicy) allowsFile() bool {
	return p.normalized()&EnsureFiles != 0
}

func (p EnsurePolicy) allowsExec() bool {
	return p.normalized()&EnsureExecs != 0
}

func (p EnsurePolicy) allowsSyncable() bool {
	return p.normalized()&EnsureSyncables != 0
}

func (ctx Context) syncPolicy() SyncPolicy {
	return ctx.SyncPolicy
}

func (ctx Context) pathSafetyPolicy() PathSafetyPolicy {
	return ctx.PathSafetyPolicy
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
