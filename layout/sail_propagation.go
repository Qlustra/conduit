package layout

import (
	"fmt"
	"iter"

	"github.com/qlustra/sail/subject"
)

// Enter supplies cached directory members in sorted order for Sail propagation.
func (s *Slot[T]) Enter() (subject.Children, error) {
	return subject.Iterate(s.All()), nil
}

// Enter supplies cached file members in sorted order for Sail propagation.
func (s *FileSlot[T]) Enter() (subject.Children, error) {
	return subject.Iterate(s.All()), nil
}

// Enter supplies cached link members in sorted order for Sail propagation.
func (s *LinkSlot[T]) Enter() (subject.Children, error) {
	return subject.Iterate(s.All()), nil
}

// ComposeMembers lazily composes and supplies typed directory members for the supplied names.
// The error callback must be checked after iteration and before completion.
func (s *Slot[T]) ComposeMembers(names iter.Seq2[string, error]) (subject.Children, func() error) {
	return composeSlotMembers(names, s.ComposeMember)
}

// ComposeMembers lazily composes and supplies typed file members for the supplied names.
// The error callback must be checked after iteration and before completion.
func (s *FileSlot[T]) ComposeMembers(names iter.Seq2[string, error]) (subject.Children, func() error) {
	return composeSlotMembers(names, s.ComposeMember)
}

// ComposeMembers lazily composes and supplies typed link members for the supplied names.
// The error callback must be checked after iteration and before completion.
func (s *LinkSlot[T]) ComposeMembers(names iter.Seq2[string, error]) (subject.Children, func() error) {
	return composeSlotMembers(names, s.ComposeMember)
}

func composeSlotMembers[T any](names iter.Seq2[string, error], composeMember func(string) (T, error)) (subject.Children, func() error) {
	var compositionErr error
	members := func(yield func(string, T) bool) {
		for name, err := range names {
			if err != nil {
				compositionErr = err
				return
			}
			item, err := composeMember(name)
			if err != nil {
				compositionErr = fmt.Errorf("compose member %q: %w", name, err)
				return
			}
			if !yield(name, item) {
				return
			}
		}
	}
	return subject.Iterate(members), func() error { return compositionErr }
}
