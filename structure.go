package conduit

import (
	"github.com/qlustra/conduit/internal/composer"
	"github.com/qlustra/conduit/internal/managed"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

// Compile prepares a tree's layout declaration for managed composition.
func Compile[T any]() (*Structure[T], error) {
	structure := &Structure[T]{}

	layoutCharter, layoutCharterErr := sail.CompileLegend[legend.Layout]()
	if layoutCharterErr != nil {
		return structure, layoutCharterErr
	}

	layoutChart, layoutChartErr := sail.Populate[T](layoutCharter)
	if layoutChartErr != nil {
		return structure, layoutChartErr
	}
	if err := composer.ValidateChart(layoutChart); err != nil {
		return structure, err
	}

	structure.layoutCharter = layoutCharter
	structure.layoutChart = layoutChart
	structure.layoutComposer = composer.New(layoutCharter)

	return structure, nil
}

type Structure[T any] struct {
	layoutComposer *composer.Engine
	layoutCharter  *sail.Charter[legend.Layout]
	layoutChart    *legend.Layout
}

// Compose constructs a fresh layout at root without touching the filesystem.
func (s *Structure[T]) Compose(root string) (*Layout[T], error) {
	return managed.Compose[T](s.layoutCharter, s.layoutChart, s.layoutComposer, root)
}
