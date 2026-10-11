package conduit

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Sync writes eligible cached content on declared participants and cached
// members. Local Sync hooks own their branches and apply the context's sync,
// write, and path-safety policies. Sail owns traversal.
// Call Ensure separately when filesystem scaffolding is needed.
// Sync does not discover members or ensure raw filesystem nodes.
func (l *Layout[T]) Sync(ctx layout.Context) (layout.ResultCode, error) {
	if l == nil || l.layoutVessel == nil {
		return layout.SyncFailed, fmt.Errorf("sync: a composed layout is required")
	}

	result := layout.SyncTraversed
	finish := func(node any, at subject.Encounter, code layout.ResultCode, err error) (subject.Route, error) {
		if at.Depth() == 0 {
			result = code
		}
		recordSync(ctx, node, code, err)
		if err != nil {
			return subject.Abort(err)
		}
		return subject.Prune(), nil
	}

	// Sources complete after their cached members. Failures cancel completion
	// and leave the enclosing sources pending for inner-to-outer reporting.
	var pending []layout.Pather
	err := l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			pending = append(pending, node)
			return subject.Descend(subject.DepthFirst, func() error {
				pending = pending[:len(pending)-1]
				recordSync(ctx, node, layout.SyncTraversed, nil)
				return nil
			}), nil
		}),
		subject.Intercept[layout.Syncer](func(node layout.Syncer, at subject.Encounter) (subject.Route, error) {
			code, err := node.Sync(ctx)
			if err != nil {
				code = layout.SyncFailed
			}
			return finish(node, at, code, err)
		}),
		subject.Intercept[layout.Pather](func(node layout.Pather, at subject.Encounter) (subject.Route, error) {
			if isFilesystemLeaf(node, at) {
				return finish(node, at, layout.SyncNotApplicable, nil)
			}
			return subject.Pass(), nil
		}),
	})
	if err != nil {
		for i := len(pending) - 1; i >= 0; i-- {
			recordSync(ctx, pending[i], layout.SyncFailed, err)
		}
		result = layout.SyncFailed
	}
	return result, err
}

func recordSync(ctx layout.Context, node any, result layout.ResultCode, err error) {
	if ctx.Reporter == nil {
		return
	}
	if pather, ok := node.(layout.Pather); ok {
		ctx.Reporter.Record(layout.Entry{
			Op:     layout.OpSync,
			Path:   pather.Path(),
			Result: result,
			Err:    err,
		})
	}
}
