package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// ExpandTask extracts zero or more target entries from each origin entry.
type ExpandTask[O, T any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name   string
	origin Entries[O]
	target Entries[T]

	originSteps []handoverOriginStep[O]

	extract     ExtractFunc[O, T]
	extractLctx layout.Context

	sinks     []typedSink
	configErr error
}

type expandTaskSnapshot[O, T any] struct {
	name        string
	origin      Entries[O]
	target      Entries[T]
	originSteps []handoverOriginStep[O]
	extract     ExtractFunc[O, T]
	extractLctx layout.Context
	sinks       []typedSink
	configErr   error
}

type expandTaskRunSnapshot[O, T any] struct {
	task  expandTaskSnapshot[O, T]
	runMu *sync.Mutex
}

// Expand returns a one-to-many typed handover task.
func Expand[O, T any](name string, origin Entries[O], target Entries[T]) *ExpandTask[O, T] {
	return &ExpandTask[O, T]{name: name, origin: origin, target: target}
}

func (t *ExpandTask[O, T]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.name
}

func (t *ExpandTask[O, T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *ExpandTask[O, T]) snapshotRunnable() Runnable {
	return expandTaskRunSnapshot[O, T]{task: t.snapshot(), runMu: &t.runMu}
}

func (s expandTaskRunSnapshot[O, T]) Name() string { return s.task.name }

func (s expandTaskRunSnapshot[O, T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	return runExpandTask(ctx, opts, s.task)
}

func (t *ExpandTask[O, T]) Filter(lctx layout.Context, fn TypedFilterFunc[O]) *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepFilter, lctx: lctx, filter: fn})
	return t
}

func (t *ExpandTask[O, T]) Sort(fn TypedSortFunc[O]) *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepSort, sort: fn})
	return t
}

func (t *ExpandTask[O, T]) Extract(lctx layout.Context, fn ExtractFunc[O, T]) *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.configErr != nil {
		return t
	}
	if t.extract != nil {
		t.configErr = fmt.Errorf("task %q already has Extract", t.name)
		return t
	}
	t.extract = fn
	t.extractLctx = lctx
	return t
}

func (t *ExpandTask[O, T]) EnsureDeep(lctx layout.Context) *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkEnsureDeep, lctx: lctx})
	return t
}
func (t *ExpandTask[O, T]) DefaultDeep() *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkDefaultDeep})
	return t
}
func (t *ExpandTask[O, T]) RenderDeep() *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkRenderDeep})
	return t
}
func (t *ExpandTask[O, T]) SyncDeep(lctx layout.Context) *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkSyncDeep, lctx: lctx})
	return t
}
func (t *ExpandTask[O, T]) ValidateDeep(opts layout.ValidateOptions) *ExpandTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkValidateDeep, validate: opts})
	return t
}

func (t *ExpandTask[O, T]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	return runExpandTask(ctx, opts, t.snapshot())
}

func (t *ExpandTask[O, T]) snapshot() expandTaskSnapshot[O, T] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	originSteps := make([]handoverOriginStep[O], len(t.originSteps))
	copy(originSteps, t.originSteps)
	sinks := make([]typedSink, len(t.sinks))
	copy(sinks, t.sinks)
	return expandTaskSnapshot[O, T]{
		name:        t.name,
		origin:      t.origin,
		target:      t.target,
		originSteps: originSteps,
		extract:     t.extract,
		extractLctx: t.extractLctx,
		sinks:       sinks,
		configErr:   t.configErr,
	}
}

func runExpandTask[O, T any](ctx context.Context, opts RunOptions, task expandTaskSnapshot[O, T]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	result.Handover.Kind = HandoverExpand
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
	if task.extract == nil {
		return failTask(result, fmt.Errorf("task %q has no Extract", task.name))
	}

	origins, err := runHandoverOriginSteps(ctx, opts.Context.Layout, task.origin.snapshot(), task.originSteps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}

	extractContext := resolveLayoutContext(task.extractLctx, opts.Context.Layout)
	emissions := make([]handoverEmission[O, T], 0)
	for _, origin := range origins {
		emitter := extractEmitter[T]{emit: func(key string, populate func(target *Item[T]) error) {
			emissions = append(emissions, handoverEmission[O, T]{key: key, origin: origin, populate: populate})
		}}
		if err := task.extract(ctx, extractContext, origin, emitter); err != nil {
			return failTask(result, fmt.Errorf("task %q: extract %q: %w", task.name, origin.Name, err))
		}
	}

	emissions, err = applyHandoverEmissionDuplicatePolicy(emissions, opts.Context.DuplicateOutputs)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}

	targets := make([]Item[T], 0, len(emissions))
	for _, emission := range emissions {
		if emission.populate == nil {
			err := fmt.Errorf("emitted target %q has nil populate function", emission.key)
			result.Handover.Items = append(result.Handover.Items, HandoverItemResult{OriginKey: emission.origin.Key, OriginName: emission.origin.Name, OriginPath: emission.origin.Path, Err: err})
			return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
		}
		target, err := task.target.target(emission.key)
		entry := HandoverItemResult{}
		if err == nil {
			entry = handoverItemResult(emission.origin, target)
		}
		if err != nil {
			entry.OriginKey = emission.origin.Key
			entry.OriginName = emission.origin.Name
			entry.OriginPath = emission.origin.Path
			entry.Err = err
			result.Handover.Items = append(result.Handover.Items, entry)
			return failTask(result, fmt.Errorf("task %q: target %q: %w", task.name, emission.key, err))
		}
		if err := emission.populate(&target); err != nil {
			entry.Err = err
			result.Handover.Items = append(result.Handover.Items, entry)
			return failTask(result, fmt.Errorf("task %q: populate %q: %w", task.name, emission.key, err))
		}
		task.target.put(emission.key, target.Value)
		entry = handoverItemResult(emission.origin, target)
		result.Handover.Items = append(result.Handover.Items, entry)
		targets = append(targets, target)
	}

	if err := runHandoverSinks(&result, opts.Context.Layout, targets, task.sinks); err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	return result, nil
}
