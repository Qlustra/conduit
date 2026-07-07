package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// ByteFilter is the operation-scoped helper passed to FilterContentFunc callbacks.
type ByteFilter interface {
	// Read returns the current item bytes, materializing file-backed data when
	// necessary.
	Read() ([]byte, error)
}

type byteFilter struct {
	ctx  context.Context
	lctx layout.Context
	item *Item[Blob]
}

func (f byteFilter) Read() ([]byte, error) { return materializeByteItem(f.ctx, f.lctx, f.item) }
