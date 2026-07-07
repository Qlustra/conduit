package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// MultiSlotTask is a multi-item typed task.
type MultiSlotTask[I any] struct {
	stepper *slotStepper[I]
}

// Name returns the task name.
func (t *MultiSlotTask[I]) Name() string { return t.stepper.Name() }

// Run executes the typed task.
func (t *MultiSlotTask[I]) Run(ctx context.Context, contexts ...Context) (TaskResult, error) {
	return t.stepper.run(ctx, contexts...)
}

func (t *MultiSlotTask[I]) snapshotRuntime() Runtime { return t.stepper.snapshotRuntime() }

// Process updates each typed item while preserving its type.
func (t *MultiSlotTask[I]) Process(fn ProcessTypedFunc[I]) *MultiSlotTask[I] {
	return t.ProcessWith(layout.Context{}, fn)
}

// ProcessWith updates each typed item using a step-specific layout context.
func (t *MultiSlotTask[I]) ProcessWith(lctx layout.Context, fn ProcessTypedFunc[I]) *MultiSlotTask[I] {
	t.stepper.addStep(slotStep[I]{step: step[slotStepOp]{kind: slotStepProcess, lctx: lctx}, process: fn})
	return t
}

// Filter keeps only typed items for which fn returns true.
func (t *MultiSlotTask[I]) Filter(fn FilterTypedFunc[I]) *MultiSlotTask[I] {
	return t.FilterWith(layout.Context{}, fn)
}

// FilterWith keeps typed items using a step-specific layout context.
func (t *MultiSlotTask[I]) FilterWith(lctx layout.Context, fn FilterTypedFunc[I]) *MultiSlotTask[I] {
	t.stepper.addStep(slotStep[I]{step: step[slotStepOp]{kind: slotStepFilter, lctx: lctx}, filter: fn})
	return t
}

// Sort orders typed items using fn.
func (t *MultiSlotTask[I]) Sort(fn SortTypedFunc[I]) *MultiSlotTask[I] {
	t.stepper.addStep(slotStep[I]{step: step[slotStepOp]{kind: slotStepSort}, sort: fn})
	return t
}

// Split expands each typed item into zero or more same-typed items.
func (t *MultiSlotTask[I]) Split(fn SplitTypedFunc[I]) *MultiSlotTask[I] {
	return t.SplitWith(layout.Context{}, fn)
}

// SplitWith expands each typed item using a step-specific layout context.
func (t *MultiSlotTask[I]) SplitWith(lctx layout.Context, fn SplitTypedFunc[I]) *MultiSlotTask[I] {
	t.stepper.addStep(slotStep[I]{step: step[slotStepOp]{kind: slotStepSplit, lctx: lctx}, split: fn})
	return t
}

// EnsureDeep runs layout.EnsureDeep on each final typed item.
func (t *MultiSlotTask[I]) EnsureDeep() *MultiSlotTask[I] {
	return t.EnsureDeepWith(layout.Context{})
}

// EnsureDeepWith runs layout.EnsureDeep using a sink-specific layout context.
func (t *MultiSlotTask[I]) EnsureDeepWith(lctx layout.Context) *MultiSlotTask[I] {
	t.stepper.addExclusiveSink(slotSink{baseSink: baseSink[slotSinkOperation]{kind: slotSinkEnsureDeep, lctx: lctx}})
	return t
}

// DefaultDeep runs layout.DefaultDeep on each final typed item.
func (t *MultiSlotTask[I]) DefaultDeep() *MultiSlotTask[I] {
	t.stepper.addExclusiveSink(slotSink{baseSink: baseSink[slotSinkOperation]{kind: slotSinkDefaultDeep}})
	return t
}

// RenderDeep runs layout.RenderDeep on each final typed item.
func (t *MultiSlotTask[I]) RenderDeep() *MultiSlotTask[I] {
	t.stepper.addExclusiveSink(slotSink{baseSink: baseSink[slotSinkOperation]{kind: slotSinkRenderDeep}})
	return t
}

// SyncDeep runs layout.SyncDeep on each final typed item.
func (t *MultiSlotTask[I]) SyncDeep() *MultiSlotTask[I] {
	return t.SyncDeepWith(layout.Context{})
}

// SyncDeepWith runs layout.SyncDeep using a sink-specific layout context.
func (t *MultiSlotTask[I]) SyncDeepWith(lctx layout.Context) *MultiSlotTask[I] {
	t.stepper.addExclusiveSink(slotSink{baseSink: baseSink[slotSinkOperation]{kind: slotSinkSyncDeep, lctx: lctx}})
	return t
}

// ValidateDeep runs layout.ValidateDeep on each final typed item.
func (t *MultiSlotTask[I]) ValidateDeep(opts layout.ValidateOptions) *MultiSlotTask[I] {
	t.stepper.addExclusiveSink(slotSink{baseSink: baseSink[slotSinkOperation]{kind: slotSinkValidateDeep}, validate: opts})
	return t
}
