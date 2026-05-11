package conduit

import (
	"github.com/qlustra/conduit/layout"
)

// Context

type Context = layout.Context
type SyncPolicy = layout.SyncPolicy
type Reporter = layout.Reporter
type Report = layout.Report
type Entry = layout.Entry
type Operation = layout.Operation
type ResultCode = layout.ResultCode

var DefaultContext = layout.DefaultContext

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
)

const (
	EnsureEnsured ResultCode = layout.EnsureEnsured
	EnsureFailed  ResultCode = layout.EnsureFailed

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
)

// Ops

var Compose = layout.Compose
var EnsureDeep = layout.EnsureDeep
var LoadDeep = layout.LoadDeep
var DiscoverDeep = layout.DiscoverDeep
var SyncDeep = layout.SyncDeep
var ScanDeep = layout.ScanDeep
var DefaultDeep = layout.DefaultDeep
var RenderDeep = layout.RenderDeep
