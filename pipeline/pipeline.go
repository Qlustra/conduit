package pipeline

import (
	"context"
	"sync"
)

// Pipeline owns a set of processing tasks.
type Pipeline struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	tasks []Runtime
}

// New returns a pipeline initialized with tasks.
func New(tasks ...Runtime) *Pipeline {
	p := &Pipeline{}
	p.Add(tasks...)
	return p
}

// Add registers tasks in declaration order.
func (p *Pipeline) Add(tasks ...Runtime) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tasks = append(p.tasks, tasks...)
}

// Run executes all registered tasks in declaration order.
func (p *Pipeline) Run(ctx context.Context, contexts ...Context) (Result, error) {
	p.runMu.Lock()
	defer p.runMu.Unlock()

	pctx, err := runContext(contexts)
	if err != nil {
		return Result{}, err
	}
	if err := pctx.validate(); err != nil {
		return Result{}, err
	}

	var result Result
	for _, runner := range p.collectFrozenRunners() {
		taskResult, err := runner.Run(ctx, pctx)
		result.Tasks = append(result.Tasks, taskResult)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (p *Pipeline) collectFrozenRunners() []Runtime {
	p.mu.RLock()
	defer p.mu.RUnlock()

	runtimes := make([]Runtime, len(p.tasks))
	for i, task := range p.tasks {
		if snapshotter, ok := task.(runtimeSnapshotter); ok {
			runtimes[i] = snapshotter.snapshotRuntime()
			continue
		}
		runtimes[i] = task
	}
	return runtimes
}
