package control

import "reflect"

// Selector is the live binding hook supplied by an authored control. Parent is
// the enclosing scope's layout, and Target is the selected layout carrier.
type Selector[Parent, Target any] interface {
	Control(*Parent) *Target
}

type selection[Parent, Target any] struct{}

func (selection[Parent, Target]) selectorContract() reflect.Type {
	return reflect.TypeFor[Selector[Parent, Target]]()
}
func (selection[Parent, Target]) selectorParent() reflect.Type {
	return reflect.TypeFor[*Parent]()
}

// SelectedDocumentCtrl declares a typed selector contract. The authored outer
// control supplies Control(*Parent) *Target; Sail validates that method before
// Conduit invokes it against the live layout.
type SelectedDocumentCtrl[Parent, Target any] struct {
	DocumentCtrl[Target]
	selection[Parent, Target]
}

// SelectedScopeCtrl selects a focused layout from its enclosing scope.
type SelectedScopeCtrl[Parent, Target any] struct {
	ScopeCtrl[Target]
	selection[Parent, Target]
}

// SelectedProjectCtrl gives a selected scope additional project meaning.
type SelectedProjectCtrl[Parent, Target any] struct {
	ProjectCtrl[Target]
	selection[Parent, Target]
}

// SelectedCollectionCtrl selects a member source. Its member controls retain
// the same construction and registry API as CollectionCtrl.
type SelectedCollectionCtrl[Parent, Member, Source any] struct {
	CollectionCtrl[Member, Source]
	selection[Parent, Source]
}

type selectorProvider interface {
	selectorContract() reflect.Type
	selectorParent() reflect.Type
}

// SelectionParent reports the declared selector input to Conduit's binder.
// Signature validation belongs to Sail's contract-family acceptance.
func SelectionParent(carrier ControlCarrier) (reflect.Type, bool) {
	selected, ok := carrier.(selectorProvider)
	if !ok {
		return nil, false
	}
	return selected.selectorParent(), true
}

func selectionContract(carrier ControlCarrier, ordinary reflect.Type) reflect.Type {
	if selected, ok := carrier.(selectorProvider); ok {
		return selected.selectorContract()
	}
	return ordinary
}

// These families keep the shared legend independent of concrete layout types.
// Ordinary carriers retain unique-type matching; selected carriers contribute
// their instantiated Control contract through sealed declaration metadata.
type DocumentFamily struct{}
type ScopeFamily struct{}
type ProjectFamily struct{}
type CollectionFamily struct{}

func (DocumentFamily) ContractOf(carrier DocumentCarrier) reflect.Type {
	return selectionContract(carrier, reflect.TypeFor[DocumentCarrier]())
}
func (ScopeFamily) ContractOf(carrier ScopeCarrier) reflect.Type {
	return selectionContract(carrier, reflect.TypeFor[ScopeCarrier]())
}
func (ProjectFamily) ContractOf(carrier ProjectCarrier) reflect.Type {
	return selectionContract(carrier, reflect.TypeFor[ProjectCarrier]())
}
func (CollectionFamily) ContractOf(carrier CollectionCarrier) reflect.Type {
	return selectionContract(carrier, reflect.TypeFor[CollectionCarrier]())
}
