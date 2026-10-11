package composer

import (
	"fmt"

	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

type childKindDeclaration interface {
	ChildKind() string
}

// ValidateChart checks collection roles against their declared carrier kinds.
// Sail reads ChildKind on a fresh zero-valued surrogate; no live tree is needed.
func ValidateChart(chart *legend.Layout) error {
	if chart == nil {
		return fmt.Errorf("compile: a populated Layout Chart is required")
	}
	return validateParticipants(&chart.Participants)
}

func validateParticipants(participants *legend.Participants) error {
	for _, declaration := range declarations(participants) {
		if declaration.children != nil {
			if err := validateParticipants(declaration.children); err != nil {
				return err
			}
		}
		if declaration.memberKind == "" {
			continue
		}
		var kind string
		var present bool
		var err error
		switch marker := declaration.marker.(type) {
		case *legend.DirectoryMembers:
			kind, present, err = sail.ReadCapability[string, childKindDeclaration](marker.CarrierType(), "ChildKind")
		case *legend.DocumentMembers:
			kind, present, err = sail.ReadCapability[string, childKindDeclaration](marker.CarrierType(), "ChildKind")
		case *legend.LinkMembers:
			kind, present, err = sail.ReadCapability[string, childKindDeclaration](marker.CarrierType(), "ChildKind")
		default:
			return fmt.Errorf("compile: collection at %v has an unsupported declaration", declaration.marker.CarrierPath())
		}
		if err != nil {
			return fmt.Errorf("compile: collection at %v child kind: %w", declaration.marker.CarrierPath(), err)
		}
		if !present {
			return fmt.Errorf("compile: collection at %v cannot declare its child kind", declaration.marker.CarrierPath())
		}
		if kind != declaration.memberKind {
			return fmt.Errorf("compile: collection at %v has child kind %q, expected %q", declaration.marker.CarrierPath(), kind, declaration.memberKind)
		}
	}
	return nil
}
