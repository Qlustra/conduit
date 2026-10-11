package layout

import (
	"fmt"

	"github.com/qlustra/conduit/internal/contracts"
)

func composeSlotMember[T any](composer contracts.MemberComposer, root Dir, path string) (T, error) {
	if composer == nil {
		var zero T
		return zero, fmt.Errorf("compose slot member at %s: no member composer configured", path)
	}

	composeBase := root.Path()
	if base, ok := root.ComposedBaseDir(); ok {
		composeBase = base.Path()
	}

	storage := new(T)
	if err := composer.ComposeMember(storage, path, composeBase); err != nil {
		var zero T
		return zero, err
	}
	return *storage, nil
}
