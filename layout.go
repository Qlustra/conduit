package conduit

import (
	"github.com/qlustra/conduit/internal/composer"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

type Layout[T any] struct {
	layoutCharter *sail.Charter[legend.Layout]
	layoutChart   *legend.Layout
	composition   *composer.Composition[T]
	layoutVessel  *sail.Vessel[T, legend.Layout]
}

// Tree returns the live tree managed by this Layout, or nil before composition.
func (l *Layout[T]) Tree() *T {
	if l == nil || l.layoutVessel == nil {
		return nil
	}
	return l.layoutVessel.Subject
}

// Compose validates composition at root and constructs a Layout with a fresh tree.
// Composition affects memory only and allocates declared pointer paths through Sail.
func (p *Structure[T]) Compose(root string) (*Layout[T], error) {
	layout := &Layout[T]{
		layoutCharter: p.layoutCharter,
		layoutChart:   p.layoutChart,
	}

	composition, compositionErr := composer.Plan[T](root, p.layoutChart, p.layoutComposer)
	if compositionErr != nil {
		return layout, compositionErr
	}

	layout.composition = composition
	tree := composition.Compose()
	vessel, err := sail.Materialize(p.layoutCharter, p.layoutChart, tree)
	if err != nil {
		return layout, err
	}
	layout.layoutVessel = vessel

	return layout, nil
}
