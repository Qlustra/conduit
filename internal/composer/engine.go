package composer

import (
	"fmt"
	"path/filepath"

	"github.com/qlustra/conduit/internal/contracts"
	legend "github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail"
)

// Engine shares a compiled layout Charter across root and member composition.
type Engine struct {
	charter *sail.Charter[legend.Layout]
}

func New(charter *sail.Charter[legend.Layout]) *Engine {
	return &Engine{charter: charter}
}

func (e *Engine) planMember(target any, chart *legend.Layout, path, composeBase string) (*plan, error) {
	if e == nil || chart == nil {
		return nil, fmt.Errorf("compose: member composition requires an engine and Chart")
	}
	path, err := resolvePath(filepath.Clean(path), filepath.Clean(composeBase), "")
	if err != nil {
		return nil, err
	}
	composeBase = filepath.Clean(composeBase)
	if err := sail.Allocate(target, chart); err != nil {
		return nil, fmt.Errorf("compose: allocating member root: %w", err)
	}

	p := &plan{engine: e, allocations: []participantMarker{chart}}
	_, present, err := sail.Bind[contracts.Composable](target, chart)
	if err != nil {
		return nil, fmt.Errorf("compose: binding member root: %w", err)
	}
	if present {
		assignment := step{marker: chart, path: path, composeBase: composeBase}
		source, present, err := sail.Bind[layout.CollectionCarrier](target, chart)
		if err != nil {
			return nil, fmt.Errorf("compose: binding member source: %w", err)
		}
		if present {
			assignment.members, err = e.prepareMembers(target, chart, source)
			if err != nil {
				return nil, err
			}
		}
		p.steps = append(p.steps, assignment)
		return p, nil
	}
	if err := p.planComposition(target, &chart.Participants, path, composeBase); err != nil {
		return nil, err
	}
	return p, nil
}
