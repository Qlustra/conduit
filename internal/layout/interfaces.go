package layout

import (
	"os"
	"reflect"
)

// Node

type Node interface {
	Path() string
	Exists() bool
}

// Compose

type Composable interface {
	ComposePath(string)
}

var composableEntryType = reflect.TypeOf((*Composable)(nil)).Elem()

// Ensure

var (
	dirType         = reflect.TypeOf(Dir{})
	fileType        = reflect.TypeOf(File{})
	deepEnsurerType = reflect.TypeOf((*DeepEnsurer)(nil)).Elem()
)

type DeepEnsurer interface {
	EnsureDeep(dirMode, fileMode os.FileMode) error
}

// Load

type Loadable interface {
	Load() (bool, error)
	HasContent() bool
	Unload()
}

var (
	loaderType     = reflect.TypeOf((*Loadable)(nil)).Elem()
	deepLoaderType = reflect.TypeOf((*DeepLoader)(nil)).Elem()
)

type DeepLoader interface {
	LoadDeep() error
}

// Sync

var (
	syncerType     = reflect.TypeOf((*Syncer)(nil)).Elem()
	deepSyncerType = reflect.TypeOf((*DeepSyncer)(nil)).Elem()
)

type Syncer interface {
	Sync() error
}

type DeepSyncer interface {
	SyncDeep() error
}

// Scan

var (
	scannerType     = reflect.TypeOf((*Scannable)(nil)).Elem()
	deepScannerType = reflect.TypeOf((*DeepScanner)(nil)).Elem()
)

type DiskState uint8
type MemoryState uint8

const (
	// DiskUnknown
	// We have not inspected disk state, or we explicitly discarded knowledge.
	DiskUnknown DiskState = iota
	// DiskMissing
	// We checked disk and content was absent.
	DiskMissing
	// DiskPresent
	// We checked disk and content exists, but have not loaded it into memory.
	DiskPresent
)

const (
	// MemoryUnknown
	// In-memory content is not yet correlated with disk state (no operation has been performed yet).
	MemoryUnknown MemoryState = iota
	// MemoryLoaded
	// In-memory content reflects what was loaded from disk.
	MemoryLoaded
	// MemorySynced
	// In-memory content was written to disk by us.
	MemorySynced
	// MemoryDirty
	// In-memory content was mutated after it was loaded from or synced to disk.
	MemoryDirty
)

type Scannable interface {
	Scan() (DiskState, error)
}

type DeepScanner interface {
	ScanDeep() error
}
