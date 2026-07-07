package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

type taskCardinality uint8

const (
	taskCardinalityUnknown taskCardinality = iota
	singleTask
	multiTask
)

type runnableStep[I any] interface {
	runStep(ctx context.Context, layout layout.Context, input []Item[I]) ([]Item[I], error)
	resolveLayoutContext(layout layout.Context) layout.Context
	inputCardinality(activeCardinality taskCardinality) taskCardinality
	outputCardinality(activeCardinality taskCardinality) taskCardinality
}

type step[OP any] struct {
	kind                OP
	lctx                layout.Context
	expectedCardinality taskCardinality
	producedCardinality taskCardinality
}

func (p step[OP]) inputCardinality(activeCardinality taskCardinality) taskCardinality {
	if p.expectedCardinality == taskCardinalityUnknown {
		return activeCardinality
	}
	return p.expectedCardinality
}

func (p step[OP]) outputCardinality(activeCardinality taskCardinality) taskCardinality {
	if p.producedCardinality == taskCardinalityUnknown {
		return activeCardinality
	}
	return p.producedCardinality
}

func (s step[OP]) resolveLayoutContext(fallback layout.Context) layout.Context {
	return resolveLayoutContext(s.lctx, fallback)
}

// Runtime is a task that can be executed by a Pipeline.
type Runtime interface {
	// Name returns the task name used in results and errors.
	Name() string

	// Run executes the task with the supplied context. If no pipeline context is
	// supplied, DefaultContext is used.
	Run(ctx context.Context, contexts ...Context) (TaskResult, error)
}

type runtimeSnapshotter interface {
	snapshotRuntime() Runtime
}
