package managed

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Render derives text into cached content on declared participants and cached
// members. Renderable takes precedence over Templatable, and local rendering
// hooks own their branches. Member sources must implement Enterable.
// Render does not discover members or write to the filesystem.
func (l *Layout[T]) Render() error {
	if l == nil || l.layoutVessel == nil {
		return fmt.Errorf("render: a composed layout is required")
	}

	return l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			return subject.Descend(subject.DepthFirst), nil
		}),
		subject.Intercept[layout.Renderable](func(node layout.Renderable, _ subject.Encounter) (subject.Route, error) {
			rendered, err := node.Render()
			if err != nil {
				return subject.Abort(err)
			}
			node.SetRendered(rendered)
			return subject.Prune(), nil
		}),
		subject.Intercept[layout.Templatable](func(node layout.Templatable, _ subject.Encounter) (subject.Route, error) {
			rendered, err := node.RenderTemplate(node.Template())
			if err != nil {
				return subject.Abort(err)
			}
			node.SetRendered(rendered)
			return subject.Prune(), nil
		}),
	})
}
