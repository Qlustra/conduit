package layout

import "reflect"

// Node

type Node interface {
	Path() string
	Exists() bool
}

type Pather interface {
	Path() string
}

// Compose

type Composable interface {
	ComposePath(string)
}

var composableEntryType = reflect.TypeOf((*Composable)(nil)).Elem()

type composeBaseAware interface {
	setComposeBase(string)
}

type declaredPathAware interface {
	setDeclaredPath(string)
}

// Ensure

var (
	dirType         = reflect.TypeOf(Dir{})
	fileType        = reflect.TypeOf(File{})
	deepEnsurerType = reflect.TypeOf((*DeepEnsurer)(nil)).Elem()
)

type DeepEnsurer interface {
	EnsureDeep(ctx Context) (ResultCode, error)
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
	LoadDeep(ctx Context) (ResultCode, error)
}

// Discover

var (
	discovererType     = reflect.TypeOf((*Discoverable)(nil)).Elem()
	deepDiscovererType = reflect.TypeOf((*DeepDiscoverer)(nil)).Elem()
)

type Discoverable interface {
	Discover() (DiskState, error)
}

type DeepDiscoverer interface {
	DiscoverDeep(ctx Context) (ResultCode, error)
}

// Sync

var (
	syncerType     = reflect.TypeOf((*Syncer)(nil)).Elem()
	deepSyncerType = reflect.TypeOf((*DeepSyncer)(nil)).Elem()
)

type Syncer interface {
	Sync(ctx Context) (ResultCode, error)
}

type DeepSyncer interface {
	SyncDeep(ctx Context) (ResultCode, error)
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
	ScanDeep(ctx Context) (ResultCode, error)
}

// Render

var (
	renderableType   = reflect.TypeOf((*Renderable)(nil)).Elem()
	templatableType  = reflect.TypeOf((*Templatable)(nil)).Elem()
	deepRendererType = reflect.TypeOf((*DeepRenderer)(nil)).Elem()
)

type Renderable interface {
	Render() (string, error)
	SetRendered(string)
}

type Templatable interface {
	Template() string
	RenderTemplate(string) (string, error)
	SetRendered(string)
}

type DeepRenderer interface {
	RenderDeep() error
}

// Default

var (
	defaulterType     = reflect.TypeOf((*Defaulter)(nil)).Elem()
	deepDefaulterType = reflect.TypeOf((*DeepDefaulter)(nil)).Elem()
)

type Defaulter interface {
	Default() error
}

type DeepDefaulter interface {
	DefaultDeep() error
}

// Report

type Reporter interface {
	Record(Entry)
}

type Operation uint8

const (
	OpEnsure Operation = iota + 1
	OpLoad
	OpDiscover
	OpScan
	OpSync
)

func (op Operation) String() string {
	switch op {
	case OpEnsure:
		return "ensure"
	case OpLoad:
		return "load"
	case OpDiscover:
		return "discover"
	case OpScan:
		return "scan"
	case OpSync:
		return "sync"
	default:
		return "unknown"
	}
}

type ResultCode uint8

const (
	EnsureEnsured ResultCode = iota + 1
	EnsureFailed
)

const (
	LoadLoaded ResultCode = iota + 16
	LoadMissing
	LoadTraversed
	LoadNotApplicable
	LoadFailed
)

const (
	DiscoverPresent ResultCode = iota + 32
	DiscoverMissing
	DiscoverTraversed
	DiscoverNotApplicable
	DiscoverFailed
)

const (
	ScanPresent ResultCode = iota + 48
	ScanMissing
	ScanTraversed
	ScanNotApplicable
	ScanFailed
)

const (
	SyncWritten ResultCode = iota + 64
	SyncTraversed
	SyncNotApplicable
	SyncSkippedNoContent
	SyncSkippedPolicy
	SyncFailed
)
