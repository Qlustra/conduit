package conduit

import (
	"github.com/qlustra/conduit/layout"
)

// Context is an alias for layout.Context.
//
// Use it from the root conduit package when you want the convenience facade.
// See layout.Context for the full field and behavior documentation.
type Context = layout.Context

// SyncPolicy is an alias for layout.SyncPolicy.
//
// It controls which in-memory states Sync and SyncDeep may write. See
// layout.SyncPolicy for the full policy model and constant descriptions.
type SyncPolicy = layout.SyncPolicy

// EnsurePolicy is an alias for layout.EnsurePolicy.
//
// It controls which node kinds Ensure and EnsureDeep may materialize. See
// layout.EnsurePolicy for the full policy model and constant descriptions.
type EnsurePolicy = layout.EnsurePolicy

// Reporter is an alias for layout.Reporter.
//
// See layout.Reporter for the reporting contract used during deep traversal.
type Reporter = layout.Reporter

// ValidateOptions is an alias for layout.ValidateOptions.
//
// See layout.ValidateOptions for the validation traversal options.
type ValidateOptions = layout.ValidateOptions

// Report is an alias for layout.Report.
//
// See layout.Report for the full reporting API.
type Report = layout.Report

// Entry is an alias for layout.Entry.
//
// See layout.Entry for the full path-level reporting semantics.
type Entry = layout.Entry

// Operation is an alias for layout.Operation.
//
// See layout.Operation for the operation enum used by reports and result
// interpretation.
type Operation = layout.Operation

// ResultCode is an alias for layout.ResultCode.
//
// See layout.ResultCode for the full result taxonomy.
type ResultCode = layout.ResultCode

// DefaultContext aliases layout.DefaultContext for callers using the root
// conduit facade.
//
// See layout.DefaultContext for the exact default modes and ensure/sync policy.
var DefaultContext = layout.DefaultContext

const (
	EnsureDirs      EnsurePolicy = layout.EnsureDirs
	EnsureFiles     EnsurePolicy = layout.EnsureFiles
	EnsureExecs     EnsurePolicy = layout.EnsureExecs
	EnsureSyncables EnsurePolicy = layout.EnsureSyncables

	EnsureAll      EnsurePolicy = layout.EnsureAll
	EnsureScaffold EnsurePolicy = layout.EnsureScaffold
	EnsureNone     EnsurePolicy = layout.EnsureNone
)

const (
	SyncOnLoaded      SyncPolicy = layout.SyncOnLoaded
	SyncOnSynced      SyncPolicy = layout.SyncOnSynced
	SyncOnDirty       SyncPolicy = layout.SyncOnDirty
	SyncOnDiskUnknown SyncPolicy = layout.SyncOnDiskUnknown
	SyncOnDiskMissing SyncPolicy = layout.SyncOnDiskMissing
	SyncOnDiskPresent SyncPolicy = layout.SyncOnDiskPresent

	SyncRewrite    SyncPolicy = layout.SyncRewrite
	SyncIfDirty    SyncPolicy = layout.SyncIfDirty
	SyncIfUnsynced SyncPolicy = layout.SyncIfUnsynced
	SyncIfMissing  SyncPolicy = layout.SyncIfMissing
)

const (
	OpEnsure   Operation = layout.OpEnsure
	OpLoad     Operation = layout.OpLoad
	OpDiscover Operation = layout.OpDiscover
	OpScan     Operation = layout.OpScan
	OpSync     Operation = layout.OpSync
	OpValidate Operation = layout.OpValidate
)

const (
	EnsureEnsured       ResultCode = layout.EnsureEnsured
	EnsureSkippedPolicy ResultCode = layout.EnsureSkippedPolicy
	EnsureFailed        ResultCode = layout.EnsureFailed

	LoadLoaded        ResultCode = layout.LoadLoaded
	LoadMissing       ResultCode = layout.LoadMissing
	LoadTraversed     ResultCode = layout.LoadTraversed
	LoadNotApplicable ResultCode = layout.LoadNotApplicable
	LoadFailed        ResultCode = layout.LoadFailed

	DiscoverPresent       ResultCode = layout.DiscoverPresent
	DiscoverMissing       ResultCode = layout.DiscoverMissing
	DiscoverTraversed     ResultCode = layout.DiscoverTraversed
	DiscoverNotApplicable ResultCode = layout.DiscoverNotApplicable
	DiscoverFailed        ResultCode = layout.DiscoverFailed

	ScanPresent       ResultCode = layout.ScanPresent
	ScanMissing       ResultCode = layout.ScanMissing
	ScanTraversed     ResultCode = layout.ScanTraversed
	ScanNotApplicable ResultCode = layout.ScanNotApplicable
	ScanFailed        ResultCode = layout.ScanFailed

	SyncWritten          ResultCode = layout.SyncWritten
	SyncTraversed        ResultCode = layout.SyncTraversed
	SyncNotApplicable    ResultCode = layout.SyncNotApplicable
	SyncSkippedNoContent ResultCode = layout.SyncSkippedNoContent
	SyncSkippedPolicy    ResultCode = layout.SyncSkippedPolicy
	SyncFailed           ResultCode = layout.SyncFailed

	ValidateOK            ResultCode = layout.ValidateOK
	ValidateTraversed     ResultCode = layout.ValidateTraversed
	ValidateNotApplicable ResultCode = layout.ValidateNotApplicable
	ValidateFailed        ResultCode = layout.ValidateFailed
)

// Ops

// Compose aliases layout.Compose for callers that prefer the root conduit
// facade.
//
// See layout.Compose for full behavior details.
var Compose = layout.Compose

// EnsureDeep aliases layout.EnsureDeep for callers that prefer the root
// conduit facade.
//
// See layout.EnsureDeep for full behavior details.
var EnsureDeep = layout.EnsureDeep

// LoadDeep aliases layout.LoadDeep for callers that prefer the root conduit
// facade.
//
// See layout.LoadDeep for full behavior details.
var LoadDeep = layout.LoadDeep

// DiscoverDeep aliases layout.DiscoverDeep for callers that prefer the root
// conduit facade.
//
// See layout.DiscoverDeep for full behavior details.
var DiscoverDeep = layout.DiscoverDeep

// SyncDeep aliases layout.SyncDeep for callers that prefer the root conduit
// facade.
//
// See layout.SyncDeep for full behavior details.
var SyncDeep = layout.SyncDeep

// ScanDeep aliases layout.ScanDeep for callers that prefer the root conduit
// facade.
//
// See layout.ScanDeep for full behavior details.
var ScanDeep = layout.ScanDeep

// DefaultDeep aliases layout.DefaultDeep for callers that prefer the root
// conduit facade.
//
// See layout.DefaultDeep for full behavior details.
var DefaultDeep = layout.DefaultDeep

// RenderDeep aliases layout.RenderDeep for callers that prefer the root
// conduit facade.
//
// See layout.RenderDeep for full behavior details.
var RenderDeep = layout.RenderDeep

// ValidateDeep aliases layout.ValidateDeep for callers that prefer the root
// conduit facade.
//
// See layout.ValidateDeep for full behavior details.
var ValidateDeep = layout.ValidateDeep
