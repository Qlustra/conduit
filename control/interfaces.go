package control

import "reflect"

// ControlCarrier identifies the layout declaration controlled by a carrier.
// Declaration metadata must be safe to read from a zero-valued carrier.
type ControlCarrier interface {
	LayoutCarrierType() reflect.Type
}

// ProjectCarrier identifies project controls, including authored types that
// embed ProjectCtrl.
type ProjectCarrier interface {
	ControlCarrier
	isProjectControl()
}

// DocumentCarrier identifies controls over a declared layout content carrier.
type DocumentCarrier interface {
	ControlCarrier
	isDocumentControl()
}

// ScopeCarrier identifies controls over a declared layout subtree.
type ScopeCarrier interface {
	ControlCarrier
	isScopeControl()
}

// CollectionCarrier identifies a collection source and its member control type.
// MemberControlType must be safe to read from a zero-valued carrier.
type CollectionCarrier interface {
	ControlCarrier
	MemberControlType() reflect.Type
	isCollectionControl()
}
