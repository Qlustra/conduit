package layout

import "os"

type SyncPolicy uint8

const (
	SyncOnLoaded SyncPolicy = 1 << iota
	SyncOnSynced
	SyncOnDirty
)

const (
	SyncRewrite    SyncPolicy = SyncOnLoaded | SyncOnSynced | SyncOnDirty
	SyncIfDirty    SyncPolicy = SyncOnDirty
	SyncIfUnsynced SyncPolicy = SyncOnLoaded | SyncOnDirty
)

type Context struct {
	DirMode    os.FileMode
	FileMode   os.FileMode
	ExecMode   os.FileMode
	SyncPolicy SyncPolicy
}

var DefaultContext = Context{
	DirMode:    0o755,
	FileMode:   0o644,
	ExecMode:   0o755,
	SyncPolicy: SyncRewrite,
}

func (ctx Context) syncPolicy() SyncPolicy {
	if ctx.SyncPolicy == 0 {
		return SyncRewrite
	}
	return ctx.SyncPolicy
}

func (p SyncPolicy) allows(state MemoryState) bool {
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
