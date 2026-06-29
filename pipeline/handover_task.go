package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// HandoverKeyFunc maps an origin item to a target key.
type HandoverKeyFunc[O any] func(ctx context.Context, lctx layout.Context, origin Item[O]) (string, error)

// BridgeFunc populates one target item from one origin item.
type BridgeFunc[O, T any] func(ctx context.Context, lctx layout.Context, origin Item[O], target *Item[T]) error

// ExtractFunc extracts zero or more target items from one origin item.
type ExtractFunc[O, T any] func(ctx context.Context, lctx layout.Context, origin Item[O], emit EntryEmitter[T]) error

// BuildFunc builds one target item from all origin items.
type BuildFunc[O, T any] func(ctx context.Context, lctx layout.Context, origins []Item[O], target *Item[T]) error

// EntryEmitter collects target entries produced by ExtractFunc callbacks.
type EntryEmitter[T any] interface {
	Emit(key string, populate func(target *Item[T]) error)
}
