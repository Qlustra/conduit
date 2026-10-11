package conduit

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

type ensurer interface {
	Ensure(layout.Context) error
}

type memberKeys interface {
	Keys() []string
}

// Ensure prepares declared filesystem structure and currently cached members.
// Collection roots are prepared before their members are visited, including
// directory member roots whose trees have no declared nodes. Local Ensure hooks
// own their branches; Sail owns traversal.
// Ensure does not discover members, load content, or persist cached content.
func (l *Layout[T]) Ensure(ctx layout.Context) (layout.ResultCode, error) {
	if l == nil || l.layoutVessel == nil {
		return layout.EnsureFailed, fmt.Errorf("ensure: a composed layout is required")
	}

	result := layout.EnsureEnsured
	finish := func(node any, at subject.Encounter, code layout.ResultCode, err error) (subject.Route, error) {
		if at.Depth() == 0 {
			result = code
		}
		recordEnsure(ctx, node, code, err)
		if err != nil {
			return subject.Abort(err)
		}
		return subject.Prune(), nil
	}

	// Canceled completions leave enclosing sources pending for failure reports.
	var pending []layout.Pather
	err := l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			if err := ensureCollectionRoots(node, ctx); err != nil {
				return finish(node, at, layout.EnsureFailed, err)
			}
			pending = append(pending, node)
			return subject.Descend(subject.DepthFirst, func() error {
				pending = pending[:len(pending)-1]
				recordEnsure(ctx, node, layout.EnsureEnsured, nil)
				return nil
			}), nil
		}),
		subject.Intercept[ensurer](func(node ensurer, at subject.Encounter) (subject.Route, error) {
			if _, ok := node.(layout.Syncer); ok && !ctx.EnsurePolicy.Has(layout.EnsureSyncables) {
				return finish(node, at, layout.EnsureSkippedPolicy, nil)
			}
			if policy, known := ensureCarrierPolicy(node); known && !ctx.EnsurePolicy.Has(policy) {
				return finish(node, at, layout.EnsureSkippedPolicy, nil)
			}
			err := node.Ensure(ctx)
			code := layout.EnsureEnsured
			if err != nil {
				code = layout.EnsureFailed
			}
			return finish(node, at, code, err)
		}),
	})
	if err != nil {
		for i := len(pending) - 1; i >= 0; i-- {
			recordEnsure(ctx, pending[i], layout.EnsureFailed, err)
		}
		result = layout.EnsureFailed
	}
	return result, err
}

func ensureCollectionRoots(node layout.CollectionCarrier, ctx layout.Context) error {
	kind := node.ChildKind()
	switch kind {
	case "dir", "file", "link":
	default:
		return fmt.Errorf("ensure: member source at %q has unsupported child kind %q", node.Path(), kind)
	}
	var names []string
	if kind == "dir" {
		keys, ok := node.(memberKeys)
		if !ok {
			return fmt.Errorf("ensure: directory member source at %q must supply cached keys", node.Path())
		}
		names = keys.Keys()
	}
	root := layout.NewDir(node.Path())
	if err := root.Ensure(ctx); err != nil {
		return err
	}
	for _, name := range names {
		if err := root.Dir(name).Ensure(ctx); err != nil {
			return fmt.Errorf("ensure member root %q: %w", name, err)
		}
	}
	return nil
}

func recordEnsure(ctx layout.Context, node any, result layout.ResultCode, err error) {
	if ctx.Reporter == nil {
		return
	}
	if pather, ok := node.(layout.Pather); ok {
		ctx.Reporter.Record(layout.Entry{
			Op:     layout.OpEnsure,
			Path:   pather.Path(),
			Result: result,
			Err:    err,
		})
	}
}
