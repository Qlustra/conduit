package pipeline

import (
	"context"
	"fmt"
	"sync"
)

type stepperSinkRunner[I any, K comparable, N sink[K]] func(ctx context.Context, pctx Context, outputs []Item[I], sink N, result *TaskResult) error

type stepperRuntime[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K]] struct {
	snapshot *stepperState[I, S, P, K, N]
	runMu    *sync.Mutex
	runSink  stepperSinkRunner[I, K, N]
}

func newStepperRuntime[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K]](
	snapshot *stepperState[I, S, P, K, N],
	runMu *sync.Mutex,
	runSink stepperSinkRunner[I, K, N],
) stepperRuntime[I, S, P, K, N] {
	return stepperRuntime[I, S, P, K, N]{
		snapshot: snapshot,
		runMu:    runMu,
		runSink:  runSink,
	}
}

func (r stepperRuntime[I, S, P, K, N]) Name() string { return r.snapshot.name }

func (r stepperRuntime[I, S, P, K, N]) Run(ctx context.Context, contexts ...Context) (TaskResult, error) {
	pctx, err := runContext(contexts)
	if err != nil {
		return failTask(TaskResult{Name: r.snapshot.name}, err)
	}

	r.runMu.Lock()
	defer r.runMu.Unlock()

	return r.run(ctx, pctx)
}

func (r stepperRuntime[I, S, P, K, N]) run(ctx context.Context, pctx Context) (TaskResult, error) {
	result, inputs, err := r.prepareRuntime(pctx)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", r.snapshot.name, err))
	}

	outputs := inputs
	cardinality := r.snapshot.cardinality
	for _, step := range r.snapshot.steps {
		inputCardinality := step.inputCardinality(cardinality)
		if inputCardinality != cardinality {
			return failTask(result, fmt.Errorf("task %q: invalid state: step cardinality mismatch", r.snapshot.name))
		}
		if inputCardinality == singleTask && len(outputs) != 1 {
			return failTask(result, fmt.Errorf("task %q: invalid state: single task requires exactly one item", r.snapshot.name))
		}

		lctx := step.resolveLayoutContext(pctx.Layout)
		outs, err := step.runStep(ctx, lctx, outputs)
		if err != nil {
			return failTask(result, fmt.Errorf("task %q: %w", r.snapshot.name, err))
		}
		outputs = outs
		cardinality = step.outputCardinality(cardinality)
	}

	for _, sink := range r.snapshot.sinkOrder {
		if !sink.validateCardinality(cardinality) {
			return failTask(result, fmt.Errorf("task %q: invalid state: cardinality/sink mismatch", r.snapshot.name))
		}

		if cardinality == singleTask && len(outputs) != 1 {
			return failTask(result, fmt.Errorf("task %q: invalid state: single-item sink requires exactly one item", r.snapshot.name))
		}

		err := r.runSink(ctx, pctx, outputs, sink, &result)
		if err != nil {
			return failTask(result, fmt.Errorf("task %q: %w", r.snapshot.name, err))
		}
	}

	return result, nil
}

func (r stepperRuntime[I, S, P, K, N]) prepareRuntime(pctx Context) (TaskResult, []Item[I], error) {
	result := TaskResult{Name: r.snapshot.taskName(), Status: StatusRan}

	if err := pctx.validate(); err != nil {
		return result, nil, err
	}

	if err := r.snapshot.validate(); err != nil {
		return result, nil, err
	}

	inputs, err := r.snapshot.snapshotItems()
	if err != nil {
		return result, nil, err
	}

	return result, inputs, nil
}
