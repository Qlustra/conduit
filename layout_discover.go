package conduit

import (
	"fmt"
	"os"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Discover constructs physical collection members and refreshes their disk
// state without loading content. Sail owns traversal; local Discoverable hooks
// take precedence over Scannable.
func (l *Layout[T]) Discover(ctx layout.Context) (layout.ResultCode, error) {
	if l == nil || l.layoutVessel == nil {
		return layout.DiscoverFailed, fmt.Errorf("discover: a composed layout is required")
	}

	result := layout.DiscoverTraversed
	finish := func(node any, at subject.Encounter, code layout.ResultCode, err error) (subject.Route, error) {
		if at.Depth() == 0 {
			result = code
		}
		recordDiscovery(ctx, node, code, err)
		if err != nil {
			return subject.Abort(err)
		}
		return subject.Prune(), nil
	}
	observe := func(node any, at subject.Encounter, state layout.DiskState, err error) (subject.Route, error) {
		code := layout.DiscoverTraversed
		switch state {
		case layout.DiskPresent:
			code = layout.DiscoverPresent
		case layout.DiskMissing:
			code = layout.DiscoverMissing
		}
		if err != nil {
			code = layout.DiscoverFailed
		}
		return finish(node, at, code, err)
	}

	// Sources complete after their composed descendants. Member failures keep
	// their enclosing sources pending so failures can be reported inner to outer.
	var pending []layout.Pather
	err := l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			members, err := composeMembersFromFilesystem(node)
			if os.IsNotExist(err) {
				return finish(node, at, layout.DiscoverMissing, nil)
			}
			if err != nil {
				return finish(node, at, layout.DiscoverFailed, err)
			}
			pending = append(pending, node)
			return subject.DescendSelected(subject.DepthFirst, members.children, func() error {
				if err := members.err(); err != nil {
					return err
				}
				pending = pending[:len(pending)-1]
				recordDiscovery(ctx, node, layout.DiscoverTraversed, nil)
				return nil
			}), nil
		}),
		subject.Intercept[layout.Discoverable](func(node layout.Discoverable, at subject.Encounter) (subject.Route, error) {
			state, err := node.Discover()
			return observe(node, at, state, err)
		}),
		subject.Intercept[layout.Scannable](func(node layout.Scannable, at subject.Encounter) (subject.Route, error) {
			state, err := node.Scan()
			return observe(node, at, state, err)
		}),
		subject.Intercept[layout.Pather](func(node layout.Pather, at subject.Encounter) (subject.Route, error) {
			if isFilesystemLeaf(node, at) {
				return finish(node, at, layout.DiscoverNotApplicable, nil)
			}
			return subject.Pass(), nil
		}),
	})
	if err != nil {
		for i := len(pending) - 1; i >= 0; i-- {
			recordDiscovery(ctx, pending[i], layout.DiscoverFailed, err)
		}
		result = layout.DiscoverFailed
	}
	return result, err
}

func recordDiscovery(ctx layout.Context, node any, result layout.ResultCode, err error) {
	if ctx.Reporter == nil {
		return
	}
	if pather, ok := node.(layout.Pather); ok {
		ctx.Reporter.Record(layout.Entry{
			Op:     layout.OpDiscover,
			Path:   pather.Path(),
			Result: result,
			Err:    err,
		})
	}
}
