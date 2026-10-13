package managed

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Default applies in-memory defaults to declared participants and cached members.
// Local Defaulter hooks own their branches; member sources must implement
// Enterable so Sail can reach their members.
// Default does not load content, discover members, or write to disk.
func (l *Layout[T]) Default() error {
	if l == nil || l.layoutVessel == nil {
		return fmt.Errorf("default: a composed layout is required")
	}

	return l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			return subject.Descend(subject.DepthFirst), nil
		}),
		subject.Intercept[layout.Defaulter](func(node layout.Defaulter, _ subject.Encounter) (subject.Route, error) {
			if err := node.Default(); err != nil {
				return subject.Abort(err)
			}
			return subject.Prune(), nil
		}),
	})
}
