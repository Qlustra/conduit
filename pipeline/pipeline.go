package pipeline

import (
	"context"
	"sync"
)

// Runnable is a task that can be executed by a Pipeline.
type Runnable interface {
	Name() string
	Run(ctx context.Context, opts RunOptions) (TaskResult, error)
}

type runnableSnapshotter interface {
	snapshotRunnable() Runnable
}

// Pipeline owns a set of processing tasks.
type Pipeline struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	tasks []Runnable
}

// New returns a pipeline initialized with tasks.
func New(tasks ...Runnable) *Pipeline {
	p := &Pipeline{}
	p.Add(tasks...)
	return p
}

// Add registers tasks in declaration order.
func (p *Pipeline) Add(tasks ...Runnable) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tasks = append(p.tasks, tasks...)
}

// Run executes all registered tasks in declaration order.
func (p *Pipeline) Run(ctx context.Context, opts RunOptions) (Result, error) {
	p.runMu.Lock()
	defer p.runMu.Unlock()

	var result Result
	for _, task := range p.snapshotTasks() {
		taskResult, err := task.Run(ctx, opts)
		result.Tasks = append(result.Tasks, taskResult)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (p *Pipeline) snapshotTasks() []Runnable {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tasks := make([]Runnable, len(p.tasks))
	for i, task := range p.tasks {
		if snapshotter, ok := task.(runnableSnapshotter); ok {
			tasks[i] = snapshotter.snapshotRunnable()
			continue
		}
		tasks[i] = task
	}
	return tasks
}

// RunOptions configures pipeline execution.
type RunOptions struct {
	Context Context
}
