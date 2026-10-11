package conduit

import (
	"github.com/qlustra/conduit/internal/composer"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

type Structure[T any] struct {
	layoutComposer *composer.Engine
	layoutCharter  *sail.Charter[legend.Layout]
	layoutChart    *legend.Layout
}

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
