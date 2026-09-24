// Package spec supplies declaration tokens for conduit-gen. Tokens describe a
// named management topology; they do not compose layouts, hold runtime state, or
// perform filesystem operations. Use package mgmt directly without generation.
//
// A topology struct has one root Project[L], Space[L], or Scope[L] field. Other
// fields bind relative to it using scope:"from=RootField;bind=Layout.Field".
// Bind "." to reuse the same supplied layout. Document[T] binds a supported
// typed content node. Collection[M] binds a directory slot whose child layout
// matches the root layout of topology M. An ordinary topology field binds a
// nested management surface, including a nested Space.
//
// conduit-gen emits concrete wrappers whose methods delegate to mgmt. An ops
// tag restricts the operation set; unsupported explicit operations are errors.
// See cmd/conduit-gen for the initial generator's supported declaration syntax.
package spec

// Project declares a recognition-capable management root over layout L.
type Project[L any] struct{}

// Space declares a chosen management root over layout L, without recognition.
type Space[L any] struct{}

// Scope declares an operational view over an already-bound layout L.
type Scope[L any] struct{}

// Document declares a typed content handle with value T.
type Document[T any] struct{}

// Collection declares repeated children with management topology M.
// M is a topology declaration, not a layout or a runtime wrapper type.
type Collection[M any] struct{}
