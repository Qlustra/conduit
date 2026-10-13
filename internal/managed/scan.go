package managed

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Scan refreshes disk-state metadata for declared participants and cached members.
// Sail owns traversal. Local Scannable hooks own their branches, and declared
// member sources must implement Enterable.
// Scan does not load content, discover members, or write to the filesystem.
func (l *Layout[T]) Scan(ctx layout.Context) (layout.ResultCode, error) {
	if l == nil || l.layoutVessel == nil {
		return layout.ScanFailed, fmt.Errorf("scan: a composed layout is required")
	}

	result := layout.ScanTraversed
	finish := func(node any, at subject.Encounter, code layout.ResultCode, err error) (subject.Route, error) {
		if at.Depth() == 0 {
			result = code
		}
		recordScan(ctx, node, code, err)
		if err != nil {
			return subject.Abort(err)
		}
		return subject.Prune(), nil
	}

	// Successful completions pop sources in depth-first order. An error cancels
	// completions, leaving the enclosing sources to receive failure reports below.
	var pending []layout.Pather
	err := l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			// A declared source owns descent even when its carrier has a local
			// Scan method. Other nodes reach the local scanner below.
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			pending = append(pending, node)
			return subject.Descend(subject.DepthFirst, func() error {
				pending = pending[:len(pending)-1]
				recordScan(ctx, node, layout.ScanTraversed, nil)
				return nil
			}), nil
		}),
		subject.Intercept[layout.Scannable](func(node layout.Scannable, at subject.Encounter) (subject.Route, error) {
			state, err := node.Scan()
			code := layout.ScanTraversed
			switch state {
			case layout.DiskPresent:
				code = layout.ScanPresent
			case layout.DiskMissing:
				code = layout.ScanMissing
			}
			if err != nil {
				code = layout.ScanFailed
			}
			return finish(node, at, code, err)
		}),
		subject.Intercept[layout.Pather](func(node layout.Pather, at subject.Encounter) (subject.Route, error) {
			if isFilesystemLeaf(node, at) {
				return finish(node, at, layout.ScanNotApplicable, nil)
			}
			return subject.Pass(), nil
		}),
	})
	if err != nil {
		for i := len(pending) - 1; i >= 0; i-- {
			recordScan(ctx, pending[i], layout.ScanFailed, err)
		}
		result = layout.ScanFailed
	}
	return result, err
}

func recordScan(ctx layout.Context, node any, result layout.ResultCode, err error) {
	if ctx.Reporter == nil {
		return
	}
	if pather, ok := node.(layout.Pather); ok {
		ctx.Reporter.Record(layout.Entry{
			Op:     layout.OpScan,
			Path:   pather.Path(),
			Result: result,
			Err:    err,
		})
	}
}
