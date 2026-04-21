package conduit

import (
	"github.com/qlustra/conduit/internal/formats"
	"github.com/qlustra/conduit/internal/layout"
)

// Layout

type File = layout.File
type Dir = layout.Dir
type Slot[T any] = layout.Slot[T]
type Codec[T any] = layout.Codec[T]
type Format[T any] = layout.Format[T, Codec[T]]

var Compose = layout.Compose
var EnsureDeep = layout.EnsureDeep
var LoadDeep = layout.LoadDeep
var SyncDeep = layout.SyncDeep
var ScanDeep = layout.ScanDeep

// Formats

type JSONFile[T any] = formats.JSONFile[T]
type YAMLFile[T any] = formats.YAMLFile[T]
type TOMLFile[T any] = formats.TOMLFile[T]
