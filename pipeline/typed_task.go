package pipeline

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// Operation Callbacks

// ProcessFunc updates one typed item while preserving its type.
type ProcessFunc[T any] func(ctx context.Context, lctx layout.Context, item Item[T]) (T, error)

// TypedFilterFunc controls whether a typed item is retained.
type TypedFilterFunc[T any] func(ctx context.Context, lctx layout.Context, item Item[T]) (bool, error)

// TypedSortFunc orders two typed items.
type TypedSortFunc[T any] func(a Item[T], b Item[T]) bool

// TypedSplitFunc expands one typed item into zero or more same-typed items.
type TypedSplitFunc[T any] func(ctx context.Context, lctx layout.Context, split TypedSplit[T], item Item[T]) error

// TypedConcatFunc reduces multiple typed items into one value.
type TypedConcatFunc[T any] func(ctx context.Context, lctx layout.Context, items []Item[T]) (T, error)

// Types

type typedStepKind uint8

const (
	typedStepProcess typedStepKind = iota + 1
	typedStepFilter
	typedStepSort
	typedStepSplit
	typedStepConcat
)

type typedStep[T any] struct {
	kind typedStepKind
	lctx layout.Context

	process ProcessFunc[T]
	filter  TypedFilterFunc[T]
	sort    TypedSortFunc[T]
	split   TypedSplitFunc[T]
	concat  TypedConcatFunc[T]
}

// Typed Task

// TypedTask is a typed pipeline task.
type TypedTask[T any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name  string
	kind  subjectKind
	items []Item[T]
	steps []typedStep[T]
	sinks []typedSink

	configErr error
}

type typedTaskSnapshot[T any] struct {
	name      string
	kind      subjectKind
	items     []Item[T]
	steps     []typedStep[T]
	sinks     []typedSink
	configErr error
}

type typedTaskRunSnapshot[T any] struct {
	task  typedTaskSnapshot[T]
	runMu *sync.Mutex
}

// TypedSingleTask is a single-subject typed task.
type TypedSingleTask[T any] struct{ task *TypedTask[T] }

// TypedMultiTask is a multi-subject typed task.
type TypedMultiTask[T any] struct{ task *TypedTask[T] }

func newTypedSingleTask[T any](name string, item Item[T]) *TypedSingleTask[T] {
	return &TypedSingleTask[T]{task: &TypedTask[T]{name: name, kind: subjectSingle, items: []Item[T]{item}}}
}

func newTypedMultiTask[T any](name string, items []Item[T]) *TypedMultiTask[T] {
	return &TypedMultiTask[T]{task: &TypedTask[T]{name: name, kind: subjectMulti, items: items}}
}

func (t *TypedTask[T]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.name
}
func (t *TypedSingleTask[T]) Name() string { return t.task.Name() }
func (t *TypedMultiTask[T]) Name() string  { return t.task.Name() }

func (t *TypedTask[T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *TypedSingleTask[T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.task.run(ctx, opts)
}

func (t *TypedSingleTask[T]) snapshotRunnable() Runnable { return t.task.snapshotRunnable() }

func (t *TypedMultiTask[T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.task.run(ctx, opts)
}

func (t *TypedMultiTask[T]) snapshotRunnable() Runnable { return t.task.snapshotRunnable() }

func (t *TypedSingleTask[T]) Process(lctx layout.Context, fn ProcessFunc[T]) *TypedSingleTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepProcess, lctx: lctx, process: fn})
	return t
}

func (t *TypedSingleTask[T]) Split(lctx layout.Context, fn TypedSplitFunc[T]) *TypedMultiTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepSplit, lctx: lctx, split: fn})
	return &TypedMultiTask[T]{task: t.task}
}

func (t *TypedSingleTask[T]) EnsureDeep(lctx layout.Context) *TypedSingleTask[T] {
	t.task.addSink(typedSink{kind: typedSinkEnsureDeep, lctx: lctx})
	return t
}
func (t *TypedSingleTask[T]) DefaultDeep() *TypedSingleTask[T] {
	t.task.addSink(typedSink{kind: typedSinkDefaultDeep})
	return t
}
func (t *TypedSingleTask[T]) RenderDeep() *TypedSingleTask[T] {
	t.task.addSink(typedSink{kind: typedSinkRenderDeep})
	return t
}
func (t *TypedSingleTask[T]) SyncDeep(lctx layout.Context) *TypedSingleTask[T] {
	t.task.addSink(typedSink{kind: typedSinkSyncDeep, lctx: lctx})
	return t
}
func (t *TypedSingleTask[T]) ValidateDeep(opts layout.ValidateOptions) *TypedSingleTask[T] {
	t.task.addSink(typedSink{kind: typedSinkValidateDeep, validate: opts})
	return t
}

func (t *TypedMultiTask[T]) Process(lctx layout.Context, fn ProcessFunc[T]) *TypedMultiTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepProcess, lctx: lctx, process: fn})
	return t
}

func (t *TypedMultiTask[T]) Filter(lctx layout.Context, fn TypedFilterFunc[T]) *TypedMultiTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepFilter, lctx: lctx, filter: fn})
	return t
}

func (t *TypedMultiTask[T]) Sort(fn TypedSortFunc[T]) *TypedMultiTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepSort, sort: fn})
	return t
}

func (t *TypedMultiTask[T]) Split(lctx layout.Context, fn TypedSplitFunc[T]) *TypedMultiTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepSplit, lctx: lctx, split: fn})
	return t
}

func (t *TypedMultiTask[T]) Concat(lctx layout.Context, fn TypedConcatFunc[T]) *TypedSingleTask[T] {
	t.task.addStep(typedStep[T]{kind: typedStepConcat, lctx: lctx, concat: fn})
	return &TypedSingleTask[T]{task: t.task}
}

func (t *TypedMultiTask[T]) EnsureDeep(lctx layout.Context) *TypedMultiTask[T] {
	t.task.addSink(typedSink{kind: typedSinkEnsureDeep, lctx: lctx})
	return t
}
func (t *TypedMultiTask[T]) DefaultDeep() *TypedMultiTask[T] {
	t.task.addSink(typedSink{kind: typedSinkDefaultDeep})
	return t
}
func (t *TypedMultiTask[T]) RenderDeep() *TypedMultiTask[T] {
	t.task.addSink(typedSink{kind: typedSinkRenderDeep})
	return t
}
func (t *TypedMultiTask[T]) SyncDeep(lctx layout.Context) *TypedMultiTask[T] {
	t.task.addSink(typedSink{kind: typedSinkSyncDeep, lctx: lctx})
	return t
}
func (t *TypedMultiTask[T]) ValidateDeep(opts layout.ValidateOptions) *TypedMultiTask[T] {
	t.task.addSink(typedSink{kind: typedSinkValidateDeep, validate: opts})
	return t
}

func (t *TypedTask[T]) addStep(step typedStep[T]) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.steps = append(t.steps, step)
}

func (t *TypedTask[T]) addSink(sink typedSink) {
	t.mu.Lock()
	defer t.mu.Unlock()

	addTypedSink(t.name, &t.configErr, &t.sinks, sink)
}

func (t *TypedTask[T]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	return runTypedTask(ctx, opts, t.snapshot())
}

func (t *TypedTask[T]) snapshot() typedTaskSnapshot[T] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	steps := make([]typedStep[T], len(t.steps))
	copy(steps, t.steps)
	sinks := make([]typedSink, len(t.sinks))
	copy(sinks, t.sinks)
	return typedTaskSnapshot[T]{
		name:      t.name,
		kind:      t.kind,
		items:     cloneItems(t.items),
		steps:     steps,
		sinks:     sinks,
		configErr: t.configErr,
	}
}

func (t *TypedTask[T]) snapshotRunnable() Runnable {
	return typedTaskRunSnapshot[T]{task: t.snapshot(), runMu: &t.runMu}
}

func (s typedTaskRunSnapshot[T]) Name() string { return s.task.name }

func (s typedTaskRunSnapshot[T]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	return runTypedTask(ctx, opts, s.task)
}

func runTypedTask[T any](ctx context.Context, opts RunOptions, task typedTaskSnapshot[T]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	if err := opts.Context.validate(); err != nil {
		return failTask(result, err)
	}
	if task.configErr != nil {
		return failTask(result, task.configErr)
	}
	if len(task.sinks) == 0 {
		return failTask(result, fmt.Errorf("task %q has no sink", task.name))
	}
	items, _, err := runTypedSteps(ctx, opts.Context.Layout, task.kind, cloneItems(task.items), task.steps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	for _, sink := range task.sinks {
		sink.lctx = resolveLayoutContext(sink.lctx, opts.Context.Layout)
		for i := range items {
			entry, err := runTypedSink(&items[i], sink)
			result.recordDeepOperation(sink.kind, entry)
			if err != nil {
				return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
			}
		}
	}
	return result, nil
}

func runTypedSteps[T any](ctx context.Context, fallback layout.Context, kind subjectKind, items []Item[T], steps []typedStep[T]) ([]Item[T], subjectKind, error) {
	for _, step := range steps {
		lctx := resolveLayoutContext(step.lctx, fallback)
		switch step.kind {
		case typedStepProcess:
			if step.process == nil {
				return nil, subjectUnknown, fmt.Errorf("process step has nil function")
			}
			out := make([]Item[T], 0, len(items))
			for _, item := range items {
				value, err := step.process(ctx, lctx, item)
				if err != nil {
					return nil, subjectUnknown, err
				}
				item.Value = value
				out = append(out, item)
			}
			items = out

		case typedStepFilter:
			if kind != subjectMulti {
				return nil, subjectUnknown, fmt.Errorf("filter requires a multi subject")
			}
			if step.filter == nil {
				return nil, subjectUnknown, fmt.Errorf("filter step has nil function")
			}
			out := make([]Item[T], 0, len(items))
			for _, item := range items {
				keep, err := step.filter(ctx, lctx, item)
				if err != nil {
					return nil, subjectUnknown, err
				}
				if keep {
					out = append(out, item)
				}
			}
			items = out

		case typedStepSort:
			if kind != subjectMulti {
				return nil, subjectUnknown, fmt.Errorf("sort requires a multi subject")
			}
			if step.sort == nil {
				return nil, subjectUnknown, fmt.Errorf("sort step has nil function")
			}
			sort.SliceStable(items, func(i int, j int) bool { return step.sort(items[i], items[j]) })

		case typedStepSplit:
			if step.split == nil {
				return nil, subjectUnknown, fmt.Errorf("split step has nil function")
			}
			out := make([]Item[T], 0)
			for _, item := range items {
				collector := &typedSplitCollector[T]{}
				if err := step.split(ctx, lctx, collector, item); err != nil {
					return nil, subjectUnknown, err
				}
				out = append(out, collector.items...)
			}
			items = out
			kind = subjectMulti

		case typedStepConcat:
			if kind != subjectMulti {
				return nil, subjectUnknown, fmt.Errorf("concat requires a multi subject")
			}
			if step.concat == nil {
				return nil, subjectUnknown, fmt.Errorf("concat step has nil function")
			}
			value, err := step.concat(ctx, lctx, items)
			if err != nil {
				return nil, subjectUnknown, err
			}
			items = []Item[T]{{Key: "concat", Name: "concat", Path: "concat", Value: value}}
			kind = subjectSingle

		default:
			return nil, subjectUnknown, fmt.Errorf("unsupported typed step kind %d", step.kind)
		}
	}
	return items, kind, nil
}

// Split

// TypedSplit emits same-typed items from a typed split callback.
type TypedSplit[T any] interface {
	Emit(item Item[T])
	EmitValue(key string, value T)
}

type typedSplitCollector[T any] struct{ items []Item[T] }

func (s *typedSplitCollector[T]) Emit(item Item[T]) { s.items = append(s.items, item) }
func (s *typedSplitCollector[T]) EmitValue(key string, value T) {
	s.Emit(Item[T]{Key: key, Name: key, Path: key, Value: value})
}
