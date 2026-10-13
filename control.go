package conduit

import (
	"fmt"
	"reflect"

	"github.com/qlustra/conduit/control"
	"github.com/qlustra/conduit/internal/controlbinding"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
)

// ControlPath composes an independent live Layout and binds a control surface
// to it. C names the authored surface; L is inferred from structure.
func ControlPath[C, L any](structure *Structure[L], rootPath string) (*C, error) {
	if structure == nil {
		return nil, fmt.Errorf("control: a compiled Structure is required")
	}
	live, err := structure.Compose(rootPath)
	if err != nil {
		return nil, err
	}
	return ControlLayout[C](structure, live)
}

// ControlLayout binds a surface to existing live layout state. Whole-layout
// scopes retain live itself; focused scopes retain views of its selected trees.
// Binding neither recomposes the tree nor reconciles independent mutations.
func ControlLayout[C, L any](structure *Structure[L], live *Layout[L]) (*C, error) {
	if structure == nil || structure.layoutCharter == nil || structure.layoutChart == nil {
		return nil, fmt.Errorf("control: a compiled Structure is required")
	}
	if live == nil || live.Tree() == nil {
		return nil, fmt.Errorf("control: a composed Layout is required")
	}
	charter, err := sail.CompileLegend[legend.ControlLegend]()
	if err != nil {
		return nil, err
	}
	builder := newControlBuilder(charter, structure.layoutCharter)
	context := controlContext{tree: live.Tree(), handler: live, spec: layoutSpec{treeType: reflect.TypeFor[*L](), chart: structure.layoutChart}}
	// A scope can be the supplied surface itself, as in ProgramControl. An
	// envelope can instead expose several independent projects/scopes.
	if reflect.TypeFor[*C]().Implements(reflect.TypeFor[control.ScopeCarrier]()) {
		chart, err := sail.Populate[controlRoot[C]](charter)
		if err != nil {
			return nil, err
		}
		root := new(controlRoot[C])
		plan, err := builder.prepare(root, chart, context.spec)
		if err != nil {
			return nil, err
		}
		register, err := plan.bind(root, context)
		if err != nil {
			return nil, err
		}
		if err := register(); err != nil {
			return nil, err
		}
		return &root.Scope, nil
	}
	chart, err := sail.Populate[C](charter)
	if err != nil {
		return nil, err
	}
	surface := new(C)
	plan, err := builder.prepare(surface, chart, context.spec)
	if err != nil {
		return nil, err
	}
	register, err := plan.bind(surface, context)
	if err != nil {
		return nil, err
	}
	if err := register(); err != nil {
		return nil, err
	}
	return surface, nil
}

type controlRoot[C any] struct {
	Scope C `control:"scope"`
}

// Keep the binding protocols checked alongside the public construction API.
var _ controlbinding.Scope = (*control.ScopeCtrl[struct{}])(nil)
