// Package controlbinding connects declaration interpretation with typed carriers.
// Binding is explicit; it does not monitor independently mutated layouts.
package controlbinding

import (
	"github.com/qlustra/conduit/internal/layoutdecl"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail"
	"github.com/qlustra/sail/subject"
)

type Operations interface {
	Ensure(layout.Context) (layout.ResultCode, error)
	Load(layout.Context) (layout.ResultCode, error)
	Discover(layout.Context) (layout.ResultCode, error)
	Scan(layout.Context) (layout.ResultCode, error)
	Sync(layout.Context) (layout.ResultCode, error)
	Validate(layout.ValidateOptions) (layout.ResultCode, error)
	Default() error
	Render() error
	Unload() error
}

type ScopePlan struct {
	Chart *layoutdecl.Layout
	Bind  func(tree any) (any, error)
}

type Scope interface {
	PrepareLayout(*sail.Charter[layoutdecl.Layout]) (ScopePlan, error)
	BindScope(handler any, register func() error) error
	RegisterControls() error
}

type Document interface {
	BindDocument(tree any) error
}

type Collection interface {
	MemberSubjects() subject.Children
	NewMemberSubject() any
	MemberControl(subject any) any
	BindCollection(source any, makeControl func(any) (any, error)) error
	RegisterControls() error
}
