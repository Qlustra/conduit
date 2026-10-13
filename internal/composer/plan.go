package composer

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/qlustra/conduit/internal/contracts"
	legend "github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail"
)

// Plan resolves all declared paths and checks receiver availability against
// a fresh Subject before any composition setters run. Sail allocates missing
// pointer storage along the declared participant paths.
func Plan[T any](root string, chart *legend.Layout, engine *Engine) (*Composition[T], error) {
	if chart == nil {
		return nil, fmt.Errorf("compose: a populated Layout Chart is required")
	}

	prototype := new(T)
	if err := sail.Allocate(prototype, chart); err != nil {
		return nil, fmt.Errorf("compose: allocating new Subject: %w", err)
	}
	if _, _, err := sail.Bind[any](prototype, chart); err != nil {
		return nil, fmt.Errorf("compose: new Subject: %w", err)
	}
	root = filepath.Clean(root)
	composition := &Composition[T]{plan: plan{engine: engine, allocations: []participantMarker{chart}}}
	if err := composition.planComposition(prototype, &chart.Participants, root, root); err != nil {
		return nil, err
	}
	return composition, nil
}

func (p *plan) planComposition(prototype any, participants *legend.Participants, base, composeBase string) error {
	for _, declaration := range declarations(participants) {
		path, err := resolvePath(base, composeBase, declaration.path)
		if err != nil {
			return fmt.Errorf("compose: participant at %v: %w", declaration.marker.CarrierPath(), err)
		}
		if err := sail.Allocate(prototype, declaration.marker); err != nil {
			return fmt.Errorf("compose: allocating participant at %v: %w", declaration.marker.CarrierPath(), err)
		}
		p.allocations = append(p.allocations, declaration.marker)

		if declaration.children != nil {
			if _, _, err := sail.Bind[any](prototype, declaration.marker); err != nil {
				return fmt.Errorf("compose: group at %v: %w", declaration.marker.CarrierPath(), err)
			}
			if err := p.planComposition(prototype, declaration.children, path, composeBase); err != nil {
				return err
			}
			continue
		}

		_, present, err := sail.Bind[contracts.Composable](prototype, declaration.marker)
		if err != nil {
			return fmt.Errorf("compose: participant at %v: %w", declaration.marker.CarrierPath(), err)
		}
		if !present {
			return fmt.Errorf("compose: participant at %v does not implement Composable", declaration.marker.CarrierPath())
		}
		var members contracts.MemberComposer
		if declaration.memberKind != "" {
			source, present, err := sail.Bind[layout.CollectionCarrier](prototype, declaration.marker)
			if err != nil {
				return fmt.Errorf("compose: collection at %v: %w", declaration.marker.CarrierPath(), err)
			}
			if !present {
				return fmt.Errorf("compose: participant at %v is not a collection source", declaration.marker.CarrierPath())
			}
			if kind := source.ChildKind(); kind != declaration.memberKind {
				return fmt.Errorf("compose: collection at %v has child kind %q, expected %q", declaration.marker.CarrierPath(), kind, declaration.memberKind)
			}
			members, err = p.engine.prepareMembers(prototype, declaration.marker, source)
			if err != nil {
				return err
			}
		}

		p.steps = append(p.steps, step{
			marker:       declaration.marker,
			path:         path,
			composeBase:  composeBase,
			declaredPath: declaration.path,
			setDeclared:  true,
			members:      members,
		})
	}
	return nil
}

type declaration struct {
	marker     participantMarker
	path       string
	children   *legend.Participants
	memberKind string
}

// declarations projects the known Legend roles into one band. Sail records
// carrier field paths, which preserve source order across role-specific
// spots. No Subject fields or tags are inspected here.
func declarations(participants *legend.Participants) []declaration {
	var result []declaration
	for i := range participants.Directories.Value {
		marker := &participants.Directories.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get()})
	}
	for i := range participants.Content.Value {
		marker := &participants.Content.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get()})
	}
	for i := range participants.Files.Value {
		marker := &participants.Files.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get()})
	}
	for i := range participants.Executables.Value {
		marker := &participants.Executables.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get()})
	}
	for i := range participants.Groups.Value {
		marker := &participants.Groups.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get(), children: &marker.Participants})
	}
	for i := range participants.DirectoryMembers.Value {
		marker := &participants.DirectoryMembers.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get(), memberKind: "dir"})
	}
	for i := range participants.DocumentMembers.Value {
		marker := &participants.DocumentMembers.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get(), memberKind: "file"})
	}
	for i := range participants.LinkMembers.Value {
		marker := &participants.LinkMembers.Value[i]
		result = append(result, declaration{marker: marker, path: marker.Path.Get(), memberKind: "link"})
	}
	slices.SortFunc(result, func(a, b declaration) int {
		return slices.Compare(a.marker.CarrierPath(), b.marker.CarrierPath())
	})
	return result
}
