package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// ContentCollectionTask is a multi-item byte task.
type ContentCollectionTask struct {
	stepper *byteStepper
}

// Name returns the task name.
func (t *ContentCollectionTask) Name() string {
	return t.stepper.Name()
}

// Run executes the byte task.
func (t *ContentCollectionTask) Run(ctx context.Context, contexts ...Context) (TaskResult, error) {
	return t.stepper.run(ctx, contexts...)
}

func (t *ContentCollectionTask) snapshotRuntime() Runtime { return t.stepper.snapshotRuntime() }

// Transform rewrites each byte item in memory using fn.
func (t *ContentCollectionTask) Transform(fn TransformFunc) *ContentCollectionTask {
	return t.TransformWith(layout.Context{}, fn)
}

// TransformWith rewrites each byte item using a step-specific layout context.
func (t *ContentCollectionTask) TransformWith(lctx layout.Context, fn TransformFunc) *ContentCollectionTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepTransform, lctx: lctx}, transform: fn})
	return t
}

// Filter keeps only byte items for which fn returns true.
func (t *ContentCollectionTask) Filter(fn FilterContentFunc) *ContentCollectionTask {
	return t.FilterWith(layout.Context{}, fn)
}

// FilterWith keeps byte items using a step-specific layout context.
func (t *ContentCollectionTask) FilterWith(lctx layout.Context, fn FilterContentFunc) *ContentCollectionTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepFilter, lctx: lctx}, filter: fn})
	return t
}

// Sort orders byte items using fn.
func (t *ContentCollectionTask) Sort(fn SortContentFunc) *ContentCollectionTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepSort}, sort: fn})
	return t
}

// Concat reduces all byte items into one single byte item.
func (t *ContentCollectionTask) Concat(opts layout.ConcatOptions) *SingleContentTask {
	return t.ConcatWith(layout.Context{}, opts)
}

// ConcatWith reduces all byte items using a step-specific layout context.
func (t *ContentCollectionTask) ConcatWith(lctx layout.Context, opts layout.ConcatOptions) *SingleContentTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepConcat, lctx: lctx}, concatOptions: opts})
	return &SingleContentTask{stepper: t.stepper}
}

// Pick keeps the first item that matches fn.
func (t *ContentCollectionTask) Pick(fn PickContentFunc) *SingleContentTask {
	return t.PickWith(layout.Context{}, fn)
}

// PickWith keeps the first item that matches fn using a step-specific layout context.
func (t *ContentCollectionTask) PickWith(lctx layout.Context, fn PickContentFunc) *SingleContentTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepPick, lctx: lctx}, pick: fn})
	return &SingleContentTask{stepper: t.stepper}
}

// Select selects one item from the available items.
func (t *ContentCollectionTask) Select(fn SelectContentFunc) *SingleContentTask {
	return t.SelectWith(layout.Context{}, fn)
}

// SelectWith selects one item from the available items using a step-specific layout context.
func (t *ContentCollectionTask) SelectWith(lctx layout.Context, fn SelectContentFunc) *SingleContentTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepSelect, lctx: lctx}, selectFn: fn})
	return &SingleContentTask{stepper: t.stepper}
}

// WriteToSources writes each final byte item back to its source file.
//
// It requires file-backed source items; blob-only inputs cannot be written back.
func (t *ContentCollectionTask) WriteToSources() *ContentCollectionTask {
	return t.WriteToSourcesWith(layout.Context{})
}

// WriteToSourcesWith writes each final byte item back using a sink-specific layout context.
//
// It requires file-backed source items; blob-only inputs cannot be written back.
func (t *ContentCollectionTask) WriteToSourcesWith(lctx layout.Context) *ContentCollectionTask {
	t.stepper.addOnlySink(newByteSink("writeBack", byteSinkWriteBack, lctx))
	return t
}

// WriteToDir writes final byte items under dest using flattened item paths.
func (t *ContentCollectionTask) WriteToDir(dest layout.Dir) *ContentCollectionTask {
	return t.WriteToDirWith(layout.Context{}, dest)
}

// WriteToDirWith writes final byte items using a sink-specific layout context.
func (t *ContentCollectionTask) WriteToDirWith(lctx layout.Context, dest layout.Dir) *ContentCollectionTask {
	t.stepper.addOnlySink(newByteSinkWithDestination("byteToDir:"+dest.Path(), byteSinkToDir, lctx, newDestination(dest, destinationFlatten)))
	return t
}

// WriteToDirPreserve writes final byte items under dest preserving item paths.
func (t *ContentCollectionTask) WriteToDirPreserve(dest layout.Dir) *ContentCollectionTask {
	return t.WriteToDirPreserveWith(layout.Context{}, dest)
}

// WriteToDirPreserveWith preserves item paths using a sink-specific layout context.
func (t *ContentCollectionTask) WriteToDirPreserveWith(lctx layout.Context, dest layout.Dir) *ContentCollectionTask {
	t.stepper.addOnlySink(newByteSinkWithDestination("ToDirPreserve:"+dest.Path(), byteSinkToDir, lctx, newDestination(dest, destinationPreserveStructure)))
	return t
}

// WriteToFiles writes final byte items to files returned by mapper.
func (t *ContentCollectionTask) WriteToFiles(sinkLabel string, mapper MapContentFunc) *ContentCollectionTask {
	return t.WriteToFilesWith(layout.Context{}, sinkLabel, mapper)
}

// WriteToFilesWith writes final byte items using a sink-specific layout context.
func (t *ContentCollectionTask) WriteToFilesWith(lctx layout.Context, sinkLabel string, mapper MapContentFunc) *ContentCollectionTask {
	t.stepper.addOnlySink(newByteSinkWithMapper("ToFiles:"+sinkLabel, byteSinkToFiles, lctx, mapper))
	return t
}
