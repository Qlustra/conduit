package pipeline

import (
	"context"
	"fmt"
	"sync"
)

type stepperRuntimeProvider[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K]] func(*stepperState[I, S, P, K, N], *sync.Mutex) Runtime

type stepperTask[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K], R Runtime] struct {
	mu            sync.RWMutex
	runMu         sync.Mutex
	state         *stepperState[I, S, P, K, N]
	createRuntime stepperRuntimeProvider[I, S, P, K, N]
}

func newStepperTask[I any, S inputSource[I], P runnableStep[I], K comparable, N sink[K], R Runtime](name string, cardinality taskCardinality, createRuntime stepperRuntimeProvider[I, S, P, K, N]) *stepperTask[I, S, P, K, N, R] {
	return &stepperTask[I, S, P, K, N, R]{
		state:         newStepperState[I, S, P, K, N](name, cardinality),
		createRuntime: createRuntime,
	}
}

func (t *stepperTask[I, S, P, K, N, R]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.state.name
}

func (t *stepperTask[I, S, P, K, N, R]) setSource(source S) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state.configErr != nil {
		return
	}
	if t.state.hasSource() {
		t.state.configErr = fmt.Errorf("task %q already has source", t.state.name)
		return
	}
	t.state.source = source
}

func (t *stepperTask[I, S, P, K, N, R]) addStep(step P) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state.steps = append(t.state.steps, step)
}

func (t *stepperTask[I, S, P, K, N, R]) addSink(sink N) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state.addSink(getSinkId(sink), sink)
}

func (t *stepperTask[I, S, P, K, N, R]) addExclusiveSink(sink N) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state.addExclusiveSink(getSinkId(sink), sink)
}

func (t *stepperTask[I, S, P, K, N, R]) addOnlySink(sink N) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state.addOnlySink(getSinkId(sink), sink)
}

func (t *stepperTask[I, S, P, K, N, R]) run(ctx context.Context, contexts ...Context) (TaskResult, error) {
	return t.snapshotRuntime().Run(ctx, contexts...)
}

func (t *stepperTask[I, S, P, K, N, R]) snapshotRuntime() Runtime {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.createRuntime(t.state.snapshotState(), &t.runMu)
}
