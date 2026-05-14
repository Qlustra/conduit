package layout

import "reflect"

// Node

// Node is the minimal filesystem-handle contract shared by concrete node
// types.
type Node interface {
	Path() string
	Exists() bool
}

// Pather is the minimal contract for values that expose a filesystem path.
type Pather interface {
	Path() string
}

// Compose

// Composable is implemented by values that can be bound to a concrete path
// during composition.
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

// DeepEnsurer is implemented by values that handle EnsureDeep traversal
// themselves.
type DeepEnsurer interface {
	EnsureDeep(ctx Context) (ResultCode, error)
}

// Load

// Loadable is implemented by stateful values that can load content from disk
// into memory.
type Loadable interface {
	Load() (bool, error)
	HasContent() bool
	Unload()
}

var (
	loaderType     = reflect.TypeOf((*Loadable)(nil)).Elem()
	deepLoaderType = reflect.TypeOf((*DeepLoader)(nil)).Elem()
)

// DeepLoader is implemented by values that handle LoadDeep traversal
// themselves.
type DeepLoader interface {
	LoadDeep(ctx Context) (ResultCode, error)
}

// Discover

var (
	discovererType     = reflect.TypeOf((*Discoverable)(nil)).Elem()
	deepDiscovererType = reflect.TypeOf((*DeepDiscoverer)(nil)).Elem()
)

// Discoverable is implemented by values that can observe disk state without
// loading content into memory.
type Discoverable interface {
	Discover() (DiskState, error)
}

// DeepDiscoverer is implemented by values that handle DiscoverDeep traversal
// themselves.
type DeepDiscoverer interface {
	DiscoverDeep(ctx Context) (ResultCode, error)
}

// Sync

var (
	syncerType     = reflect.TypeOf((*Syncer)(nil)).Elem()
	deepSyncerType = reflect.TypeOf((*DeepSyncer)(nil)).Elem()
)

// Syncer is implemented by stateful values that can write eligible cached
// content back to disk.
type Syncer interface {
	Sync(ctx Context) (ResultCode, error)
}

// DeepSyncer is implemented by values that handle SyncDeep traversal
// themselves.
type DeepSyncer interface {
	SyncDeep(ctx Context) (ResultCode, error)
}

// Scan

var (
	scannerType     = reflect.TypeOf((*Scannable)(nil)).Elem()
	deepScannerType = reflect.TypeOf((*DeepScanner)(nil)).Elem()
)

// DiskState records what a stateful node currently knows about the
// corresponding filesystem entry.
type DiskState uint8

// MemoryState records what a stateful node currently knows about its in-memory
// value.
type MemoryState uint8

const (
	// DiskUnknown means disk state has not been observed, or known state was
	// discarded during composition.
	DiskUnknown DiskState = iota

	// DiskMissing means the entry was checked and was absent on disk.
	DiskMissing

	// DiskPresent means the entry was checked and was present on disk.
	DiskPresent
)

const (
	// MemoryUnknown means no meaningful in-memory content is currently loaded.
	MemoryUnknown MemoryState = iota

	// MemoryLoaded means in-memory content reflects what was loaded from disk.
	MemoryLoaded

	// MemorySynced means in-memory content was written to disk by Conduit.
	MemorySynced

	// MemoryDirty means in-memory content was set or changed after load or sync.
	MemoryDirty
)

// Scannable is implemented by values that can refresh disk-state metadata
// without changing their cached in-memory content.
type Scannable interface {
	Scan() (DiskState, error)
}

// DeepScanner is implemented by values that handle ScanDeep traversal
// themselves.
type DeepScanner interface {
	ScanDeep(ctx Context) (ResultCode, error)
}

// Render

var (
	renderableType   = reflect.TypeOf((*Renderable)(nil)).Elem()
	templatableType  = reflect.TypeOf((*Templatable)(nil)).Elem()
	deepRendererType = reflect.TypeOf((*DeepRenderer)(nil)).Elem()
)

// Renderable is implemented by values that render derived text through custom
// logic.
type Renderable interface {
	Render() (string, error)
	SetRendered(string)
}

// Templatable is implemented by values that render derived text from a
// template string.
type Templatable interface {
	Template() string
	RenderTemplate(string) (string, error)
	SetRendered(string)
}

// DeepRenderer is implemented by values that handle RenderDeep traversal
// themselves.
type DeepRenderer interface {
	RenderDeep() error
}

// Default

var (
	defaulterType     = reflect.TypeOf((*Defaulter)(nil)).Elem()
	deepDefaulterType = reflect.TypeOf((*DeepDefaulter)(nil)).Elem()
)

// Defaulter is implemented by values that can install in-memory defaults
// without consulting disk.
type Defaulter interface {
	Default() error
}

// DeepDefaulter is implemented by values that handle DefaultDeep traversal
// themselves.
type DeepDefaulter interface {
	DefaultDeep() error
}

// Report

// Reporter receives path-level outcomes recorded during deep traversal.
type Reporter interface {
	Record(Entry)
}

// Validate

var (
	validatorType     = reflect.TypeOf((*Validator)(nil)).Elem()
	deepValidatorType = reflect.TypeOf((*DeepValidator)(nil)).Elem()
)

// ValidateOptions carries optional reporting hooks for ValidateDeep.
type ValidateOptions struct {
	// Reporter, when non-nil, receives path-level results during validation
	// traversal.
	Reporter Reporter
}

// Validator is implemented by values that can validate their current state
// without mutating disk or memory.
type Validator interface {
	Validate() error
}

// DeepValidator is implemented by values that handle ValidateDeep traversal
// themselves.
type DeepValidator interface {
	ValidateDeep(opts ValidateOptions) (ResultCode, error)
}

// Enums

// Operation identifies which deep traversal operation produced a report entry
// or root result.
type Operation uint8

const (
	// OpEnsure identifies EnsureDeep.
	OpEnsure Operation = iota + 1

	// OpLoad identifies LoadDeep.
	OpLoad

	// OpDiscover identifies DiscoverDeep.
	OpDiscover

	// OpScan identifies ScanDeep.
	OpScan

	// OpSync identifies SyncDeep.
	OpSync

	// OpValidate identifies ValidateDeep.
	OpValidate
)

// String returns the lowercase operation name used in reports.
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
	case OpValidate:
		return "validate"
	default:
		return "unknown"
	}
}

// ResultCode is an operation-specific outcome code returned by deep
// traversal operations and recorded in reports.
//
// Interpret a ResultCode together with the Operation that produced it.
type ResultCode uint8

const (
	// EnsureEnsured reports that the visited node was ensured successfully.
	EnsureEnsured ResultCode = iota + 1

	// EnsureSkippedPolicy reports that ensure was skipped by the current ensure
	// policy.
	EnsureSkippedPolicy

	// EnsureFailed reports that ensure failed.
	EnsureFailed
)

const (
	// LoadLoaded reports that content was loaded from disk.
	LoadLoaded ResultCode = iota + 16

	// LoadMissing reports that the target was observed missing on disk.
	LoadMissing

	// LoadTraversed reports that traversal continued through a container node.
	LoadTraversed

	// LoadNotApplicable reports that load does not apply to the visited node.
	LoadNotApplicable

	// LoadFailed reports that load failed.
	LoadFailed
)

const (
	// DiscoverPresent reports that the target was observed present on disk.
	DiscoverPresent ResultCode = iota + 32

	// DiscoverMissing reports that the target was observed missing on disk.
	DiscoverMissing

	// DiscoverTraversed reports that traversal continued through a container
	// node.
	DiscoverTraversed

	// DiscoverNotApplicable reports that discover does not apply to the visited
	// node.
	DiscoverNotApplicable

	// DiscoverFailed reports that discover failed.
	DiscoverFailed
)

const (
	// ScanPresent reports that the target was observed present on disk.
	ScanPresent ResultCode = iota + 48

	// ScanMissing reports that the target was observed missing on disk.
	ScanMissing

	// ScanTraversed reports that traversal continued through a container node.
	ScanTraversed

	// ScanNotApplicable reports that scan does not apply to the visited node.
	ScanNotApplicable

	// ScanFailed reports that scan failed.
	ScanFailed
)

const (
	// SyncWritten reports that content was written to disk.
	SyncWritten ResultCode = iota + 64

	// SyncTraversed reports that traversal continued through a container node.
	SyncTraversed

	// SyncNotApplicable reports that sync does not apply to the visited node.
	SyncNotApplicable

	// SyncSkippedNoContent reports that sync had nothing cached to write.
	SyncSkippedNoContent

	// SyncSkippedPolicy reports that sync was skipped by the current policy.
	SyncSkippedPolicy

	// SyncFailed reports that sync failed.
	SyncFailed
)

const (
	// ValidateOK reports that validation completed successfully for the visited
	// node.
	ValidateOK ResultCode = iota + 80

	// ValidateTraversed reports that validation continued through a container
	// node.
	ValidateTraversed

	// ValidateNotApplicable reports that validation does not apply to the
	// visited node.
	ValidateNotApplicable

	// ValidateFailed reports that validation failed.
	ValidateFailed
)
