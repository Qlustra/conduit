package managed

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

type unloader interface {
	Unload()
}

// Unload clears cached content on declared participants and cached members.
// Local Unload hooks own their branches. Member sources retain their cached
// membership and must implement Enterable so Sail can reach their members.
// Unload does not discover members or change the filesystem.
func (l *Layout[T]) Unload() error {
	if l == nil || l.layoutVessel == nil {
		return fmt.Errorf("unload: a composed layout is required")
	}

	return l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			return subject.Descend(subject.DepthFirst), nil
		}),
		subject.Intercept[unloader](func(node unloader, _ subject.Encounter) (subject.Route, error) {
			node.Unload()
			return subject.Prune(), nil
		}),
	})
}
