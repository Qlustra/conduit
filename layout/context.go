package layout

import "os"

type SyncPolicy uint8

const (
	SyncOnLoaded SyncPolicy = 1 << iota
	SyncOnSynced
	SyncOnDirty
	SyncOnDiskUnknown
	SyncOnDiskMissing
	SyncOnDiskPresent
)

const (
	SyncRewrite    SyncPolicy = SyncOnLoaded | SyncOnSynced | SyncOnDirty
	SyncIfDirty    SyncPolicy = SyncOnDirty
	SyncIfUnsynced SyncPolicy = SyncOnLoaded | SyncOnDirty
	SyncIfMissing  SyncPolicy = SyncOnDiskMissing
)

const (
	syncPolicyMemoryMask = SyncOnLoaded | SyncOnSynced | SyncOnDirty
	syncPolicyDiskMask   = SyncOnDiskUnknown | SyncOnDiskMissing | SyncOnDiskPresent
)

type Context struct {
	DirMode    os.FileMode
	FileMode   os.FileMode
	ExecMode   os.FileMode
	SyncPolicy SyncPolicy
	Reporter   Reporter
}

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
