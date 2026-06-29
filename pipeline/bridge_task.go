package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// BridgeTask maps origin entries to target entries one-to-one.
type BridgeTask[O, T any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name   string
	origin Entries[O]
	target Entries[T]

	originSteps []handoverOriginStep[O]

	rekey     HandoverKeyFunc[O]
	rekeyLctx layout.Context

	populate     BridgeFunc[O, T]
	populateLctx layout.Context

	sinks     []typedSink
	configErr error
}

type bridgeTaskSnapshot[O, T any] struct {
	name         string
	origin       Entries[O]
	target       Entries[T]
	originSteps  []handoverOriginStep[O]
	rekey        HandoverKeyFunc[O]
	rekeyLctx    layout.Context
	populate     BridgeFunc[O, T]
	populateLctx layout.Context
	sinks        []typedSink
	configErr    error
}

type bridgeTaskRunSnapshot[O, T any] struct {
	task  bridgeTaskSnapshot[O, T]
	runMu *sync.Mutex
}

// Bridge returns a same-cardinality typed handover task.
func Bridge[O, T any](name string, origin Entries[O], target Entries[T]) *BridgeTask[O, T] {
	return &BridgeTask[O, T]{name: name, origin: origin, target: target}
}

func (t *BridgeTask[O, T]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.name
}

func (t *BridgeTask[O, T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *BridgeTask[O, T]) snapshotRunnable() Runnable {
	return bridgeTaskRunSnapshot[O, T]{task: t.snapshot(), runMu: &t.runMu}
}

func (s bridgeTaskRunSnapshot[O, T]) Name() string { return s.task.name }

func (s bridgeTaskRunSnapshot[O, T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	return runBridgeTask(ctx, opts, s.task)
}

func (t *BridgeTask[O, T]) Filter(lctx layout.Context, fn TypedFilterFunc[O]) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepFilter, lctx: lctx, filter: fn})
	return t
}

func (t *BridgeTask[O, T]) Sort(fn TypedSortFunc[O]) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepSort, sort: fn})
	return t
}

func (t *BridgeTask[O, T]) Rekey(lctx layout.Context, fn HandoverKeyFunc[O]) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.configErr != nil {
		return t
	}
	if t.rekey != nil {
		t.configErr = fmt.Errorf("task %q already has Rekey", t.name)
		return t
	}
	t.rekey = fn
	t.rekeyLctx = lctx
	return t
}

func (t *BridgeTask[O, T]) Populate(lctx layout.Context, fn BridgeFunc[O, T]) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.configErr != nil {
		return t
	}
	if t.populate != nil {
		t.configErr = fmt.Errorf("task %q already has Populate", t.name)
		return t
	}
	t.populate = fn
	t.populateLctx = lctx
	return t
}

func (t *BridgeTask[O, T]) EnsureDeep(lctx layout.Context) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkEnsureDeep, lctx: lctx})
	return t
}
func (t *BridgeTask[O, T]) DefaultDeep() *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkDefaultDeep})
	return t
}
func (t *BridgeTask[O, T]) RenderDeep() *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkRenderDeep})
	return t
}
func (t *BridgeTask[O, T]) SyncDeep(lctx layout.Context) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkSyncDeep, lctx: lctx})
	return t
}
func (t *BridgeTask[O, T]) ValidateDeep(opts layout.ValidateOptions) *BridgeTask[O, T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, typedSink{kind: typedSinkValidateDeep, validate: opts})
	return t
}

func (t *BridgeTask[O, T]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	return runBridgeTask(ctx, opts, t.snapshot())
}

func (t *BridgeTask[O, T]) snapshot() bridgeTaskSnapshot[O, T] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	originSteps := make([]handoverOriginStep[O], len(t.originSteps))
	copy(originSteps, t.originSteps)
	sinks := make([]typedSink, len(t.sinks))
	copy(sinks, t.sinks)
	return bridgeTaskSnapshot[O, T]{
		name:         t.name,
		origin:       t.origin,
		target:       t.target,
		originSteps:  originSteps,
		rekey:        t.rekey,
		rekeyLctx:    t.rekeyLctx,
		populate:     t.populate,
		populateLctx: t.populateLctx,
		sinks:        sinks,
		configErr:    t.configErr,
	}
}

func runBridgeTask[O, T any](ctx context.Context, opts RunOptions, task bridgeTaskSnapshot[O, T]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	result.Handover.Kind = HandoverBridge
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
	if task.populate == nil {
		return failTask(result, fmt.Errorf("task %q has no Populate", task.name))
	}

	origins, err := runHandoverOriginSteps(ctx, opts.Context.Layout, task.origin.snapshot(), task.originSteps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}

	plans := make([]handoverTargetPlan[O], 0, len(origins))
	rekeyContext := resolveLayoutContext(task.rekeyLctx, opts.Context.Layout)
	for _, origin := range origins {
		key := origin.Key
		if task.rekey != nil {
			var err error
			key, err = task.rekey(ctx, rekeyContext, origin)
			if err != nil {
				return failTask(result, fmt.Errorf("task %q: rekey %q: %w", task.name, origin.Name, err))
			}
		}
		plans = append(plans, handoverTargetPlan[O]{key: key, origin: origin})
	}

	plans, err = applyHandoverDuplicatePolicy(plans, opts.Context.DuplicateOutputs)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}

	targets := make([]Item[T], 0, len(plans))
	populateContext := resolveLayoutContext(task.populateLctx, opts.Context.Layout)
	for _, plan := range plans {
		target, err := task.target.target(plan.key)
		entry := HandoverItemResult{}
		if err == nil {
			entry = handoverItemResult(plan.origin, target)
		}
		if err != nil {
			entry.OriginKey = plan.origin.Key
			entry.OriginName = plan.origin.Name
			entry.OriginPath = plan.origin.Path
			entry.Err = err
			result.Handover.Items = append(result.Handover.Items, entry)
			return failTask(result, fmt.Errorf("task %q: target %q: %w", task.name, plan.key, err))
		}
		if err := task.populate(ctx, populateContext, plan.origin, &target); err != nil {
			entry.Err = err
			result.Handover.Items = append(result.Handover.Items, entry)
			return failTask(result, fmt.Errorf("task %q: populate %q: %w", task.name, plan.key, err))
		}
		task.target.put(plan.key, target.Value)
		entry = handoverItemResult(plan.origin, target)
		result.Handover.Items = append(result.Handover.Items, entry)
		targets = append(targets, target)
	}

	if err := runHandoverSinks(&result, opts.Context.Layout, targets, task.sinks); err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	return result, nil
}
