package control

import "reflect"

type controlCarrier[LayoutLegend any] struct {
}

func (controlCarrier[LayoutLegend]) LayoutCarrierType() reflect.Type {
	return reflect.TypeFor[LayoutLegend]()
}

type ProjectCtrl[LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
}

func (ProjectCtrl[LayoutLegend]) isProjectControl() {}

type DocumentCtrl[LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
}

func (DocumentCtrl[LayoutLegend]) isDocumentControl() {}

type ScopeCtrl[LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
}

func (ScopeCtrl[LayoutLegend]) isScopeControl() {}

type CollectionCtrl[MemberScopeControl any, LayoutLegend any] struct {
	controlCarrier[LayoutLegend]
}

func (CollectionCtrl[MemberScopeControl, LayoutLegend]) isCollectionControl() {}

func (CollectionCtrl[MemberScopeControl, LayoutLegend]) MemberControlType() reflect.Type {
	return reflect.TypeFor[MemberScopeControl]()
}
