package composer

import (
	"fmt"

	"github.com/qlustra/conduit/internal/contracts"
	"github.com/qlustra/sail"
)

// Compose constructs an independent Subject and applies the plan.
// Allocation and binding were checked against a fresh Subject during preparation.
// A later failure indicates an invalid or changed plan and panics; receiver method
// panics propagate normally.
func (p *Composition[T]) Compose() *T {
	if p == nil {
		panic("compose: a prepared plan is required")
	}

	view := new(T)
	if err := p.plan.apply(view); err != nil {
		panic(err)
	}
	return view
}

func (p *plan) apply(view any) error {
	for _, marker := range p.allocations {
		if err := sail.Allocate(view, marker); err != nil {
			return fmt.Errorf("compose: allocating a prepared participant at %v: %w", marker.CarrierPath(), err)
		}
	}
	receivers := make([]contracts.Composable, len(p.steps))
	sources := make([]contracts.MemberCompositionAware, len(p.steps))
	for i, assignment := range p.steps {
		receiver, present, err := sail.Bind[contracts.Composable](view, assignment.marker)
		if err != nil {
			return fmt.Errorf("compose: binding a prepared participant at %v: %w", assignment.marker.CarrierPath(), err)
		}
		if !present {
			return fmt.Errorf("compose: prepared participant at %v no longer supports composition", assignment.marker.CarrierPath())
		}
		receivers[i] = receiver
		if assignment.members != nil {
			source, present, err := sail.Bind[contracts.MemberCompositionAware](view, assignment.marker)
			if err != nil {
				return fmt.Errorf("compose: binding a prepared collection at %v: %w", assignment.marker.CarrierPath(), err)
			}
			if !present {
				return fmt.Errorf("compose: prepared collection at %v no longer accepts a member composer", assignment.marker.CarrierPath())
			}
			sources[i] = source
		}
	}

	for i, assignment := range p.steps {
		applyComposition(receivers[i], assignment)
		if sources[i] != nil {
			sources[i].SetMemberComposer(assignment.members)
		}
	}
	return nil
}

func applyComposition(receiver contracts.Composable, assignment step) {
	receiver.ComposePath(assignment.path)
	if aware, ok := receiver.(contracts.ComposeBaseAware); ok {
		aware.SetComposeBase(assignment.composeBase)
	}
	if assignment.setDeclared {
		if aware, ok := receiver.(contracts.DeclaredPathAware); ok {
			aware.SetDeclaredPath(assignment.declaredPath)
		}
	}
}
