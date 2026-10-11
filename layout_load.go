package conduit

import (
	"fmt"
	"os"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Load discovers physical collection members and loads declared content from
// disk. Missing content is cleared by its local Load hook. Sail owns traversal,
// and absent cached members are skipped.
func (l *Layout[T]) Load(ctx layout.Context) (layout.ResultCode, error) {
	if l == nil || l.layoutVessel == nil {
		return layout.LoadFailed, fmt.Errorf("load: a composed layout is required")
	}

	result := layout.LoadTraversed
	finish := func(node any, at subject.Encounter, code layout.ResultCode, err error) (subject.Route, error) {
		if at.Depth() == 0 {
			result = code
		}
		recordLoad(ctx, node, code, err)
		if err != nil {
			return subject.Abort(err)
		}
		return subject.Prune(), nil
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
				return finish(node, at, layout.LoadMissing, nil)
			}
			if err != nil {
				return finish(node, at, layout.LoadFailed, err)
			}
			pending = append(pending, node)
			return subject.DescendSelected(subject.DepthFirst, members.children, func() error {
				if err := members.err(); err != nil {
					return err
				}
				pending = pending[:len(pending)-1]
				recordLoad(ctx, node, layout.LoadTraversed, nil)
				return nil
			}), nil
		}),
		subject.Intercept[layout.Loadable](func(node layout.Loadable, at subject.Encounter) (subject.Route, error) {
			loaded, err := node.Load()
			code := layout.LoadMissing
			if loaded {
				code = layout.LoadLoaded
			}
			if err != nil {
				code = layout.LoadFailed
			}
			return finish(node, at, code, err)
		}),
		subject.Intercept[layout.Pather](func(node layout.Pather, at subject.Encounter) (subject.Route, error) {
			if isFilesystemLeaf(node, at) {
				return finish(node, at, layout.LoadNotApplicable, nil)
			}
			return subject.Pass(), nil
		}),
	})
	if err != nil {
		for i := len(pending) - 1; i >= 0; i-- {
			recordLoad(ctx, pending[i], layout.LoadFailed, err)
		}
		result = layout.LoadFailed
	}
	return result, err
}

func recordLoad(ctx layout.Context, node any, result layout.ResultCode, err error) {
	if ctx.Reporter == nil {
		return
	}
	if pather, ok := node.(layout.Pather); ok {
		ctx.Reporter.Record(layout.Entry{
			Op:     layout.OpLoad,
			Path:   pather.Path(),
			Result: result,
			Err:    err,
		})
	}
}
