// Package composer prepares and instantiates layouts described by Conduit's Sail
// Legend. Sail allocates and binds declared participants; Conduit resolves their
// paths without touching the filesystem. Collections receive member composers
// whose Charts are reused for lazy member construction.
package composer

import (
	"github.com/qlustra/conduit/internal/contracts"
	saillegend "github.com/qlustra/sail/legend"
)

// Composition holds validated assignments in declaration order. Its populated Chart
// must remain unchanged after preparation.
type Composition[T any] struct {
	plan
}

type plan struct {
	engine      *Engine
	allocations []participantMarker
	steps       []step
}

type participantMarker interface {
	saillegend.MarkerDeclaration
	CarrierPath() []int
}

type step struct {
	marker       participantMarker
	path         string
	composeBase  string
	declaredPath string
	setDeclared  bool
	members      contracts.MemberComposer
}
