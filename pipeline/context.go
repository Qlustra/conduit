package pipeline

import (
	"fmt"

	"github.com/qlustra/conduit/layout"
)

// DuplicateOutputPolicy controls how a task handles multiple planned writes to
// the same output path.
type DuplicateOutputPolicy uint8

const (
	// DuplicateOutputUnset is invalid for execution. Use DefaultContext or set a
	// policy explicitly.
	DuplicateOutputUnset DuplicateOutputPolicy = iota

	// DuplicateOutputFail fails the task when multiple items map to one output.
	DuplicateOutputFail

	// DuplicateOutputLastWins keeps only the last item mapped to an output.
	DuplicateOutputLastWins
)

// Context configures pipeline execution.
type Context struct {
	Layout           layout.Context
	DuplicateOutputs DuplicateOutputPolicy
}

// DefaultContext is the recommended starting point for pipeline execution.
var DefaultContext = Context{
	Layout:           layout.DefaultContext,
	DuplicateOutputs: DuplicateOutputFail,
}

func (ctx Context) validate() error {
	if ctx.Layout.DirMode == 0 || ctx.Layout.FileMode == 0 {
		return fmt.Errorf("pipeline context layout is required; use pipeline.DefaultContext or set Context.Layout")
	}
	switch ctx.DuplicateOutputs {
	case DuplicateOutputFail, DuplicateOutputLastWins:
		return nil
	case DuplicateOutputUnset:
		return fmt.Errorf("pipeline duplicate output policy is required; use pipeline.DefaultContext or set Context.DuplicateOutputs")
	default:
		return fmt.Errorf("unsupported duplicate output policy %d", ctx.DuplicateOutputs)
	}
}

func resolveLayoutContext(op layout.Context, fallback layout.Context) layout.Context {
	if op.DirMode == 0 || op.FileMode == 0 {
		return fallback
	}
	return op
}
