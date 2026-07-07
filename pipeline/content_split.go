package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// ByteSplitter is the operation-scoped helper passed to SplitContentFunc callbacks.
type ByteSplitter interface {
	// Read returns the current item bytes, materializing file-backed data when
	// necessary.
	Read() ([]byte, error)

	// Emit appends item to the split output.
	Emit(item Item[Blob])

	// EmitBytes appends an in-memory byte blob named name.
	EmitBytes(name string, data []byte)

	// EmitString appends an in-memory string blob named name.
	EmitString(name string, data string)

	// EmitFile appends a file-backed byte item.
	EmitFile(file layout.File)

	// EmitBlob appends blob as an in-memory byte item.
	EmitBlob(blob Blob)
}

type byteSplitter struct {
	ctx   context.Context
	lctx  layout.Context
	item  *Item[Blob]
	items []Item[Blob]
}

func (s *byteSplitter) Read() ([]byte, error) {
	return materializeByteItem(s.ctx, s.lctx, s.item)
}

func (s *byteSplitter) Emit(item Item[Blob]) { s.items = append(s.items, item) }
func (s *byteSplitter) EmitBytes(name string, data []byte) {
	s.EmitBlob(Blob{Name: name, Data: data})
}
func (s *byteSplitter) EmitString(name string, data string) { s.EmitBytes(name, []byte(data)) }
func (s *byteSplitter) EmitFile(file layout.File)           { s.Emit(itemFromFile(file)) }
func (s *byteSplitter) EmitBlob(blob Blob)                  { s.Emit(itemFromBlob(blob)) }
