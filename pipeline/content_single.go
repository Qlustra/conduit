package pipeline

import (
	"context"

	"github.com/qlustra/conduit/layout"
)

// SingleContentTask is a single-item byte task.
type SingleContentTask struct {
	stepper *byteStepper
}

// Name returns the task name.
func (t *SingleContentTask) Name() string {
	return t.stepper.Name()
}

// Run executes the byte task.
func (t *SingleContentTask) Run(ctx context.Context, contexts ...Context) (TaskResult, error) {
	return t.stepper.run(ctx, contexts...)
}

func (t *SingleContentTask) snapshotRuntime() Runtime { return t.stepper.snapshotRuntime() }

// Transform rewrites the single byte item in memory using fn.
func (t *SingleContentTask) Transform(fn TransformFunc) *SingleContentTask {
	return t.TransformWith(layout.Context{}, fn)
}

// TransformWith rewrites the single byte item using a step-specific layout context.
func (t *SingleContentTask) TransformWith(lctx layout.Context, fn TransformFunc) *SingleContentTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepTransform, lctx: lctx}, transform: fn})
	return t
}

// Split expands the single byte item into a multi-item byte task.
func (t *SingleContentTask) Split(fn SplitContentFunc) *ContentCollectionTask {
	return t.SplitWith(layout.Context{}, fn)
}

// SplitWith expands the single byte item using a step-specific layout context.
func (t *SingleContentTask) SplitWith(lctx layout.Context, fn SplitContentFunc) *ContentCollectionTask {
	t.stepper.addStep(byteStep{step: step[byteStepOp]{kind: byteStepSplit, lctx: lctx}, split: fn})
	return &ContentCollectionTask{stepper: t.stepper}
}

// WriteToSource writes the final byte item back to its source file.
//
// It requires a file-backed source item; blob-only inputs cannot be written back.
func (t *SingleContentTask) WriteToSource() *SingleContentTask {
	return t.WriteToSourceWith(layout.Context{})
}

// WriteToSourceWith writes the final byte item back using a sink-specific layout context.
//
// It requires a file-backed source item; blob-only inputs cannot be written back.
func (t *SingleContentTask) WriteToSourceWith(lctx layout.Context) *SingleContentTask {
	t.stepper.addOnlySink(newByteSink("ToSource", byteSinkWriteBack, lctx))
	return t
}

// WriteToFile writes the final single byte item to dest.
func (t *SingleContentTask) WriteToFile(dest layout.File) *SingleContentTask {
	return t.WriteToFileWith(layout.Context{}, dest)
}

// WriteToFileWith writes the final single byte item using a sink-specific layout context.
func (t *SingleContentTask) WriteToFileWith(lctx layout.Context, dest layout.File) *SingleContentTask {
	t.stepper.addOnlySink(newByteSinkWithFile("ToFile:"+dest.Path(), byteSinkToFile, lctx, dest))
	return t
}
