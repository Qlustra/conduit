package conduit

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

// Validate checks declared participants and cached members without changing disk
// or memory state. It validates collection directories before entering members.
// Local Validator hooks own their branches; Sail owns traversal.
func (l *Layout[T]) Validate(opts layout.ValidateOptions) (layout.ResultCode, error) {
	if l == nil || l.layoutVessel == nil {
		return layout.ValidateFailed, fmt.Errorf("validate: a composed layout is required")
	}

	result := layout.ValidateTraversed
	finish := func(node any, at subject.Encounter, code layout.ResultCode, err error) (subject.Route, error) {
		if at.Depth() == 0 {
			result = code
		}
		recordValidation(opts, node, code, err)
		if err != nil {
			return subject.Abort(err)
		}
		return subject.Prune(), nil
	}

	// Completion reports follow the members. Canceled completions leave the
	// enclosing sources here so failures can be reported from inner to outer.
	var pending []layout.Pather
	err := l.layoutVessel.Propagate(subject.DepthFirst, subject.Propagation{
		subject.Intercept[layout.CollectionCarrier](func(node layout.CollectionCarrier, at subject.Encounter) (subject.Route, error) {
			if !isMemberSource(at) {
				return subject.Pass(), nil
			}
			if err := layout.NewDir(node.Path()).Validate(opts); err != nil {
				return finish(node, at, layout.ValidateFailed, err)
			}
			pending = append(pending, node)
			return subject.Descend(subject.DepthFirst, func() error {
				pending = pending[:len(pending)-1]
				recordValidation(opts, node, layout.ValidateTraversed, nil)
				return nil
			}), nil
		}),
		subject.Intercept[layout.Validator](func(node layout.Validator, at subject.Encounter) (subject.Route, error) {
			err := node.Validate(opts)
			code := layout.ValidateOK
			if err != nil {
				code = layout.ValidateFailed
			}
			return finish(node, at, code, err)
		}),
	})
	if err != nil {
		for i := len(pending) - 1; i >= 0; i-- {
			recordValidation(opts, pending[i], layout.ValidateFailed, err)
		}
		result = layout.ValidateFailed
	}
	return result, err
}

func recordValidation(opts layout.ValidateOptions, node any, result layout.ResultCode, err error) {
	if opts.Reporter == nil {
		return
	}
	if pather, ok := node.(layout.Pather); ok {
		opts.Reporter.Record(layout.Entry{
			Op:     layout.OpValidate,
			Path:   pather.Path(),
			Result: result,
			Err:    err,
		})
	}
}
