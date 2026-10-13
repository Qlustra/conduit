// Package managed holds the live layout runtime shared by Conduit's factories
// and control carriers.
package managed

import (
	"fmt"

	"github.com/qlustra/conduit/internal/composer"
	legend "github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/sail"
)

// Bind retains a composed tree exactly as supplied, including its content and
// member caches. It does not compose paths or allocate tree storage.
func Bind[T any](charter *sail.Charter[legend.Layout], chart *legend.Layout, tree *T) (*Layout[T], error) {
	vessel, err := sail.Materialize(charter, chart, tree)
	if err != nil {
		return nil, err
	}
	return &Layout[T]{layoutVessel: vessel}, nil
}

func Compose[T any](charter *sail.Charter[legend.Layout], chart *legend.Layout, engine *composer.Engine, root string) (*Layout[T], error) {
	if charter == nil || chart == nil || engine == nil {
		return nil, fmt.Errorf("compose: a compiled Structure is required")
	}
	composition, err := composer.Plan[T](root, chart, engine)
	if err != nil {
		return nil, err
	}
	return Bind(charter, chart, composition.Compose())
}

type Layout[T any] struct {
	layoutVessel *sail.Vessel[T, legend.Layout]
}

func (l *Layout[T]) Tree() *T {
	if l == nil || l.layoutVessel == nil {
		return nil
	}
	return l.layoutVessel.Subject
}
