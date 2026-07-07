package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// CompileTask builds one target entry from many origin entries.
type CompileTask[O, T any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name   string
	origin Entries[O]
	target Entry[T]

	originSteps []handoverOriginStep[O]

	build     BuildFunc[O, T]
	buildLctx layout.Context

	sinks     []typedSink
	configErr error
}

type compileTaskSnapshot[O, T any] struct {
	name        string
	origin      Entries[O]
	target      Entry[T]
	originSteps []handoverOriginStep[O]
	build       BuildFunc[O, T]
	buildLctx   layout.Context
	sinks       []typedSink
	configErr   error
}

type compileTaskRunSnapshot[O, T any] struct {
	task  compileTaskSnapshot[O, T]
	runMu *sync.Mutex
}

// Compile returns a many-to-one typed handover task.
func Compile[O, T any](name string, origin Entries[O], target Entry[T]) *CompileTask[O, T] {
	return &CompileTask[O, T]{name: name, origin: origin, target: target}
}

// Name returns the task name.
func (t *CompileTask[O, T]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.name
}

// Run executes the compile task.
func (t *CompileTask[O, T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *CompileTask[O, T]) snapshotRunnable() Runnable {
	return compileTaskRunSnapshot[O, T]{task: t.snapshot(), runMu: &t.runMu}
}

func (s compileTaskRunSnapshot[O, T]) Name() string { return s.task.name }

func (s compileTaskRunSnapshot[O, T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	return runCompileTask(ctx, opts, s.task)
}

// Filter keeps only origin items for which fn returns true.
func (t *CompileTask[O, T]) Filter(lctx layout.Context, fn TypedFilterFunc[O]) *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepFilter, lctx: lctx, filter: fn})
	return t
}

// Sort orders origin items before Build receives them.
func (t *CompileTask[O, T]) Sort(fn TypedSortFunc[O]) *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepSort, sort: fn})
	return t
}

// Build installs the required callback that builds the single target item from
// all origin items.
func (t *CompileTask[O, T]) Build(lctx layout.Context, fn BuildFunc[O, T]) *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.configErr != nil {
		return t
	}
	if t.build != nil {
		t.configErr = fmt.Errorf("task %q already has Build", t.name)
		return t
	}
	t.build = fn
	t.buildLctx = lctx
	return t
}

// EnsureDeep runs layout.EnsureDeep on the built target item.
func (t *CompileTask[O, T]) EnsureDeep(lctx layout.Context) *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkEnsureDeep, lctx: lctx})
	return t
}

// DefaultDeep runs layout.DefaultDeep on the built target item.
func (t *CompileTask[O, T]) DefaultDeep() *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkDefaultDeep})
	return t
}

// RenderDeep runs layout.RenderDeep on the built target item.
func (t *CompileTask[O, T]) RenderDeep() *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkRenderDeep})
	return t
}

// SyncDeep runs layout.SyncDeep on the built target item.
func (t *CompileTask[O, T]) SyncDeep(lctx layout.Context) *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkSyncDeep, lctx: lctx})
	return t
}

// ValidateDeep runs layout.ValidateDeep on the built target item.
func (t *CompileTask[O, T]) ValidateDeep(opts layout.ValidateOptions) *CompileTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkValidateDeep, validate: opts})
	return t
}

func (t *CompileTask[O, T]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	return runCompileTask(ctx, opts, t.snapshot())
}

func (t *CompileTask[O, T]) snapshot() compileTaskSnapshot[O, T] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	originSteps := make([]handoverOriginStep[O], len(t.originSteps))
	copy(originSteps, t.originSteps)
	sinks := make([]typedSink, len(t.sinks))
	copy(sinks, t.sinks)
	return compileTaskSnapshot[O, T]{
		name:        t.name,
		origin:      t.origin,
		target:      t.target,
		originSteps: originSteps,
		build:       t.build,
		buildLctx:   t.buildLctx,
		sinks:       sinks,
		configErr:   t.configErr,
	}
}

func runCompileTask[O, T any](ctx context.Context, opts RunOptions, task compileTaskSnapshot[O, T]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	result.Handover.Kind = HandoverCompile
	if err := opts.Context.validate(); err != nil {
		return failTask(result, err)
	}
	if task.configErr != nil {
		return failTask(result, task.configErr)
	}
	if task.origin == nil {
		return failTask(result, fmt.Errorf("task %q has no origin", task.name))
	}
	if task.target == nil {
		return failTask(result, fmt.Errorf("task %q has no target", task.name))
	}
	if task.build == nil {
		return failTask(result, fmt.Errorf("task %q has no Build", task.name))
	}

	origins, err := runHandoverOriginSteps(ctx, opts.Context.Layout, task.origin.snapshot(), task.originSteps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	target, err := task.target.target()
	entry := HandoverItemResult{}
	if err == nil {
		entry = compileHandoverItemResult(origins, target)
	}
	if err != nil {
		entry.Err = err
		result.Handover.Items = append(result.Handover.Items, entry)
		return failTask(result, fmt.Errorf("task %q: target %q: %w", task.name, task.target.key(), err))
	}
	buildContext := resolveLayoutContext(task.buildLctx, opts.Context.Layout)
	if err := task.build(ctx, buildContext, origins, &target); err != nil {
		entry.Err = err
		result.Handover.Items = append(result.Handover.Items, entry)
		return failTask(result, fmt.Errorf("task %q: build %q: %w", task.name, task.target.key(), err))
	}
	task.target.put(target.Value)
	entry = compileHandoverItemResult(origins, target)
	result.Handover.Items = append(result.Handover.Items, entry)

	items := []Item[T]{target}
	if err := runHandoverSinks(&result, opts.Context.Layout, items, task.sinks); err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	return result, nil
}
