package conduit

import (
	"fmt"
	"reflect"

	"github.com/qlustra/conduit/control"
	"github.com/qlustra/conduit/internal/controlbinding"
	"github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/legend"
	"github.com/qlustra/sail"
	saillegend "github.com/qlustra/sail/legend"
)

type layoutSpec struct {
	treeType reflect.Type // Exact pointer passed into selectors and typed binding.
	chart    *layoutdecl.Layout
}
type controlMarker interface {
	saillegend.MarkerDeclaration
	CarrierType() reflect.Type
}
type controlContext struct {
	spec    layoutSpec
	tree    any
	handler any
}
type registration func() error
type controlBinding func(surface any, context controlContext) (registration, error)
type surfacePlan struct{ bindings []controlBinding }

func (p *surfacePlan) bind(surface any, context controlContext) (registration, error) {
	var registrations []registration
	for _, binding := range p.bindings {
		register, err := binding(surface, context)
		if err != nil {
			return nil, err
		}
		if register != nil {
			registrations = append(registrations, register)
		}
	}
	return func() error {
		for _, register := range registrations {
			if err := register(); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

type controlPlanKey struct{ surface, layout reflect.Type }
type controlBuilder struct {
	charter *sail.Charter[legend.ControlLegend]
	layouts *sail.Charter[layoutdecl.Layout]
	plans   map[controlPlanKey]*surfacePlan
}

func newControlBuilder(charter *sail.Charter[legend.ControlLegend], layouts *sail.Charter[layoutdecl.Layout]) *controlBuilder {
	return &controlBuilder{charter: charter, layouts: layouts, plans: make(map[controlPlanKey]*surfacePlan)}
}

// Preparing the complete declaration before live binding makes errors in
// recursive member surfaces visible before a member is composed. Placeholders
// let finite live trees reuse recursive declaration types.
func (b *controlBuilder) prepare(surface any, chart *legend.ControlLegend, parent layoutSpec) (*surfacePlan, error) {
	key := controlPlanKey{reflect.TypeOf(surface), parent.treeType}
	if plan := b.plans[key]; plan != nil {
		return plan, nil
	}
	plan := new(surfacePlan)
	b.plans[key] = plan
	for i := range chart.Lane.ProjectsSpot.Value {
		marker := &chart.Lane.ProjectsSpot.Value[i]
		binding, err := b.scope(surface, marker, &marker.Lane, parent)
		if err != nil {
			return nil, err
		}
		plan.bindings = append(plan.bindings, binding)
	}
	for i := range chart.Lane.ScopesSpot.Value {
		marker := &chart.Lane.ScopesSpot.Value[i]
		binding, err := b.scope(surface, marker, &marker.Lane, parent)
		if err != nil {
			return nil, err
		}
		plan.bindings = append(plan.bindings, binding)
	}
	if len(plan.bindings) == 0 {
		return nil, fmt.Errorf("control: %T declares no project or scope", surface)
	}
	return plan, nil
}

func controlReceiver(surface any, marker controlMarker) (control.ControlCarrier, error) {
	if err := sail.Allocate(surface, marker); err != nil {
		return nil, err
	}
	carrier, present, err := sail.Bind[control.ControlCarrier](surface, marker)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, fmt.Errorf("control: %v does not provide a control carrier", marker.CarrierType())
	}
	return carrier, nil
}

func (b *controlBuilder) scope(surface any, marker controlMarker, lane *legend.ScopesLane, parent layoutSpec) (controlBinding, error) {
	carrier, err := controlReceiver(surface, marker)
	if err != nil {
		return nil, err
	}
	scope, ok := carrier.(controlbinding.Scope)
	if !ok {
		return nil, fmt.Errorf("control: %T does not provide scope binding", carrier)
	}
	selection, err := selectControl(carrier, parent)
	if err != nil {
		return nil, err
	}
	factory, err := scope.PrepareLayout(b.layouts)
	if err != nil {
		return nil, err
	}
	spec := layoutSpec{reflect.PointerTo(carrier.LayoutCarrierType()), factory.Chart}
	children, err := b.members(surface, lane, spec)
	if err != nil {
		return nil, err
	}
	return func(live any, context controlContext) (registration, error) {
		carrier, err := controlReceiver(live, marker)
		if err != nil {
			return nil, err
		}
		tree, err := selection(carrier, context.tree)
		if err != nil {
			return nil, err
		}
		handler := context.handler
		if tree != context.tree || handler == nil {
			handler, err = factory.Bind(tree)
			if err != nil {
				return nil, err
			}
		}
		register, err := children.bind(live, controlContext{spec: spec, tree: tree, handler: handler})
		if err != nil {
			return nil, err
		}
		scope := carrier.(controlbinding.Scope)
		if err := scope.BindScope(handler, register); err != nil {
			return nil, err
		}
		return scope.RegisterControls, nil
	}, nil
}

func (b *controlBuilder) members(surface any, lane *legend.ScopesLane, parent layoutSpec) (*surfacePlan, error) {
	plan := new(surfacePlan)
	for i := range lane.DocumentsSpot.Value {
		marker := &lane.DocumentsSpot.Value[i]
		carrier, err := controlReceiver(surface, marker)
		if err != nil {
			return nil, err
		}
		selection, err := selectControl(carrier, parent)
		if err != nil {
			return nil, err
		}
		plan.bindings = append(plan.bindings, func(live any, context controlContext) (registration, error) {
			carrier, err := controlReceiver(live, marker)
			if err != nil {
				return nil, err
			}
			tree, err := selection(carrier, context.tree)
			if err != nil {
				return nil, err
			}
			document, ok := carrier.(controlbinding.Document)
			if !ok {
				return nil, fmt.Errorf("control: %T does not provide document binding", carrier)
			}
			return nil, document.BindDocument(tree)
		})
	}
	for i := range lane.CollectionsSpot.Value {
		binding, err := b.collection(surface, &lane.CollectionsSpot.Value[i], parent)
		if err != nil {
			return nil, err
		}
		plan.bindings = append(plan.bindings, binding)
	}
	for i := range lane.ScopesSpot.Value {
		marker := &lane.ScopesSpot.Value[i]
		binding, err := b.scope(surface, marker, &marker.Lane, parent)
		if err != nil {
			return nil, err
		}
		plan.bindings = append(plan.bindings, binding)
	}
	return plan, nil
}

func (b *controlBuilder) collection(surface any, marker *legend.CollectionLandmark, parent layoutSpec) (controlBinding, error) {
	carrier, err := controlReceiver(surface, marker)
	if err != nil {
		return nil, err
	}
	collection, ok := carrier.(controlbinding.Collection)
	if !ok {
		return nil, fmt.Errorf("control: %T does not provide collection binding", carrier)
	}
	selection, err := selectControl(carrier, parent)
	if err != nil {
		return nil, err
	}
	sourceType := reflect.PointerTo(carrier.LayoutCarrierType())
	if !sourceType.Implements(reflect.TypeFor[layout.CollectionCarrier]()) {
		return nil, fmt.Errorf("control: %v is not a layout member source", sourceType)
	}
	get, ok := sourceType.MethodByName("Get")
	if !ok || get.Type.NumOut() != 2 || get.Type.Out(0).Kind() != reflect.Pointer {
		return nil, fmt.Errorf("control: %v must supply pointer members", sourceType)
	}
	memberType := get.Type.Out(0)
	chart, err := sail.PopulateMembers(b.charter, collection.MemberSubjects())
	if err != nil {
		return nil, err
	}
	// A member's root scope must control the source's actual member type. Its
	// own declaration supplies the typed Layout factory and nested chart.
	if len(chart.Lane.ScopesSpot.Value) != 1 {
		return nil, fmt.Errorf("control: collection member must declare one root scope")
	}
	memberSurface := collection.NewMemberSubject()
	memberMarker := &chart.Lane.ScopesSpot.Value[0]
	memberCarrier, err := controlReceiver(memberSurface, memberMarker)
	if err != nil {
		return nil, err
	}
	if reflect.PointerTo(memberCarrier.LayoutCarrierType()) != memberType {
		return nil, fmt.Errorf("control: member scope controls %v, source supplies %v", memberCarrier.LayoutCarrierType(), memberType)
	}
	memberScope, ok := memberCarrier.(controlbinding.Scope)
	if !ok {
		return nil, fmt.Errorf("control: %T does not provide member scope binding", memberCarrier)
	}
	factory, err := memberScope.PrepareLayout(b.layouts)
	if err != nil {
		return nil, err
	}
	memberSpec := layoutSpec{memberType, factory.Chart}
	memberPlan, err := b.prepare(memberSurface, chart, memberSpec)
	if err != nil {
		return nil, err
	}
	return func(live any, context controlContext) (registration, error) {
		carrier, err := controlReceiver(live, marker)
		if err != nil {
			return nil, err
		}
		source, err := selection(carrier, context.tree)
		if err != nil {
			return nil, err
		}
		collection := carrier.(controlbinding.Collection)
		makeControl := func(tree any) (any, error) {
			subject := collection.NewMemberSubject()
			register, err := memberPlan.bind(subject, controlContext{spec: memberSpec, tree: tree})
			if err != nil {
				return nil, err
			}
			if err := register(); err != nil {
				return nil, err
			}
			return collection.MemberControl(subject), nil
		}
		if err := collection.BindCollection(source, makeControl); err != nil {
			return nil, err
		}
		return collection.RegisterControls, nil
	}, nil
}

type controlSelection func(carrier control.ControlCarrier, parent any) (any, error)

func selectControl(carrier control.ControlCarrier, parent layoutSpec) (controlSelection, error) {
	if declared, selected := control.SelectionParent(carrier); selected {
		if declared != parent.treeType {
			return nil, fmt.Errorf("control: %T selector takes %v; enclosing scope supplies %v", carrier, declared, parent.treeType)
		}
		return func(carrier control.ControlCarrier, tree any) (any, error) {
			// Sail has already accepted the instantiated method contract. Here
			// Conduit only invokes it and checks that live binding has a target.
			result := reflect.ValueOf(carrier).MethodByName("Control").Call([]reflect.Value{reflect.ValueOf(tree)})[0]
			if result.IsNil() {
				return nil, fmt.Errorf("control: %T selector returned nil", carrier)
			}
			return result.Interface(), nil
		}, nil
	}
	if _, hasHook := reflect.TypeOf(carrier).MethodByName("Control"); hasHook {
		return nil, fmt.Errorf("control: %T declares Control without a Selected carrier's typed contract", carrier)
	}
	target := reflect.PointerTo(carrier.LayoutCarrierType())
	// The current scope's own root takes precedence. Descendant controls use
	// unique declared carrier types; dynamic members are selected by their keys.
	if target == parent.treeType {
		return func(_ control.ControlCarrier, tree any) (any, error) { return tree, nil }, nil
	}
	var matches []controlMarker
	for _, marker := range layoutMarkers(&parent.chart.Participants) {
		actual := marker.CarrierType()
		if actual.Kind() != reflect.Pointer {
			actual = reflect.PointerTo(actual)
		}
		if actual == target {
			matches = append(matches, marker)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("control: %T finds %d declared carriers of %v in %v; use a Selected carrier with Control to choose explicitly", carrier, len(matches), target, parent.treeType)
	}
	marker := matches[0]
	return func(_ control.ControlCarrier, tree any) (any, error) {
		value, present, err := sail.Bind[any](tree, marker)
		if err != nil {
			return nil, err
		}
		if !present {
			return nil, fmt.Errorf("control: missing selected layout carrier")
		}
		return value, nil
	}, nil
}

func layoutMarkers(participants *layoutdecl.Participants) []controlMarker {
	var markers []controlMarker
	for i := range participants.Directories.Value {
		markers = append(markers, &participants.Directories.Value[i])
	}
	for i := range participants.Content.Value {
		markers = append(markers, &participants.Content.Value[i])
	}
	for i := range participants.Files.Value {
		markers = append(markers, &participants.Files.Value[i])
	}
	for i := range participants.Executables.Value {
		markers = append(markers, &participants.Executables.Value[i])
	}
	for i := range participants.DirectoryMembers.Value {
		markers = append(markers, &participants.DirectoryMembers.Value[i])
	}
	for i := range participants.DocumentMembers.Value {
		markers = append(markers, &participants.DocumentMembers.Value[i])
	}
	for i := range participants.LinkMembers.Value {
		markers = append(markers, &participants.LinkMembers.Value[i])
	}
	for i := range participants.Groups.Value {
		group := &participants.Groups.Value[i]
		markers = append(markers, group)
		markers = append(markers, layoutMarkers(&group.Participants)...)
	}
	return markers
}
