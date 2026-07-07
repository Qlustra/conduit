package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// Byte

// TransformFunc is the byte transform callback used by byte tasks.
type TransformFunc = layout.TransformFunc

// SortContentFunc orders two byte items.
type SortContentFunc func(a Item[Blob], b Item[Blob]) bool

// PickContentFunc reports whether an item should be selected by Pick.
type PickContentFunc func(item Item[Blob]) bool

// SelectContentFunc selects one item from the available items.
type SelectContentFunc func(items []Item[Blob]) Item[Blob]

// SplitContentFunc expands one byte item into zero or more byte items.
type SplitContentFunc func(ctx context.Context, lctx layout.Context, split ByteSplitter, item Item[Blob]) error

// FilterContentFunc controls whether a byte item is retained.
type FilterContentFunc func(ctx context.Context, lctx layout.Context, filter ByteFilter, item Item[Blob]) (bool, error)

// MapContentFunc maps one byte item to its destination file.
type MapContentFunc func(ctx context.Context, lctx layout.Context, item Item[Blob]) (layout.File, error)

// Typed

// ProcessTypedFunc updates one typed item while preserving its type.
type ProcessTypedFunc[I any] func(ctx context.Context, lctx layout.Context, item Item[I]) (I, error)

// FilterTypedFunc controls whether a typed item is retained.
type FilterTypedFunc[I any] func(ctx context.Context, lctx layout.Context, item Item[I]) (bool, error)

// SortTypedFunc orders two typed items.
type SortTypedFunc[I any] func(a Item[I], b Item[I]) bool

// SplitTypedFunc expands one typed item into zero or more same-typed items.
type SplitTypedFunc[I any] func(ctx context.Context, lctx layout.Context, splitter TypedSplitter[I], item Item[I]) error
