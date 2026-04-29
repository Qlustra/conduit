package conduit

import (
	"github.com/qlustra/conduit/layout"
)

// Context

type Context = layout.Context
type SyncPolicy = layout.SyncPolicy

var DefaultContext = layout.DefaultContext

const (
	SyncOnLoaded SyncPolicy = layout.SyncOnLoaded
	SyncOnSynced SyncPolicy = layout.SyncOnSynced
	SyncOnDirty  SyncPolicy = layout.SyncOnDirty

	SyncRewrite    SyncPolicy = layout.SyncRewrite
	SyncIfDirty    SyncPolicy = layout.SyncIfDirty
	SyncIfUnsynced SyncPolicy = layout.SyncIfUnsynced
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
