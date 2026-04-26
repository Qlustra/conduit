package conduit

import (
	"github.com/qlustra/conduit/internal/formats"
	"github.com/qlustra/conduit/internal/layout"
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

// Layout

type File = layout.File
type Dir = layout.Dir
type Exec = layout.Exec
type Slot[T any] = layout.Slot[T]
type Codec[T any] = layout.Codec[T]
type Format[T any] = layout.Format[T, Codec[T]]
type TextTemplate[C any] = layout.TextTemplate[C]
type RunOptions = layout.RunOptions
type Defaulter = layout.Defaulter
type Renderable = layout.Renderable
type Templatable = layout.Templatable

var Compose = layout.Compose
var EnsureDeep = layout.EnsureDeep
var LoadDeep = layout.LoadDeep
var DiscoverDeep = layout.DiscoverDeep
var SyncDeep = layout.SyncDeep
var ScanDeep = layout.ScanDeep
var DefaultDeep = layout.DefaultDeep
var RenderDeep = layout.RenderDeep

// Formats

type JSONFile[T any] = formats.JSONFile[T]
type YAMLFile[T any] = formats.YAMLFile[T]
type TOMLFile[T any] = formats.TOMLFile[T]
