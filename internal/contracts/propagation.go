package contracts

import (
	"iter"

	"github.com/qlustra/sail/subject"
)

// MembersComposer turns supplied names into typed members for one descent.
// Members are composed lazily, and a yielded copy retains ordinary Go value
// semantics. The error callback reports name or composition failures after
// iteration and must be checked before the descent is reported successful.
type MembersComposer interface {
	ComposeMembers(names iter.Seq2[string, error]) (subject.Children, func() error)
}
