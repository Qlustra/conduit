package control

import "reflect"

type controlCarrier[LayoutLegend any] struct {
}

func (controlCarrier[LayoutLegend]) LayoutCarrierType() reflect.Type {
	return reflect.TypeFor[LayoutLegend]()
}

type DocumentCtrl[LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
	layout *LayoutLegend
}

func (DocumentCtrl[LayoutLegend]) isDocumentControl() {}

// ScopeCtrl controls a chosen layout segment, including an entire layout.
// Its deep operations will operate through that segment's live Layout.
type ScopeCtrl[LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
	state scopeState[LayoutLegend]
}

func (ScopeCtrl[LayoutLegend]) isScopeControl() {}

// ProjectCtrl is a ScopeCtrl with additional project meaning, including
// recognition. It retains the scope contract and controlled layout type.
type ProjectCtrl[LayoutLegend any] struct {
	ScopeCtrl[LayoutLegend]
}

func (ProjectCtrl[LayoutLegend]) isProjectControl() {}

type CollectionCtrl[MemberScopeControl any, LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
	state *collectionState[MemberScopeControl, LayoutLegend]
}

func (CollectionCtrl[MemberScopeControl, LayoutLegend]) isCollectionControl() {}

func (CollectionCtrl[MemberScopeControl, LayoutLegend]) MemberControlType() reflect.Type {
	return reflect.TypeFor[MemberScopeControl]()
}
