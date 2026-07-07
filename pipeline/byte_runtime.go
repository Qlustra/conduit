package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/qlustra/conduit/layout"
)

func runByteTask(ctx context.Context, opts RunOptions, task byteTaskSnapshot) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	if err := opts.Context.validate(); err != nil {
		return failTask(result, err)
	}
	if task.configErr != nil {
		return failTask(result, task.configErr)
	}
	if task.sink.kind == byteSinkUnknown {
		return failTask(result, fmt.Errorf("task %q has no sink", task.name))
	}

	items := task.items
	if task.source != nil {
		items = task.source.snapshotItems()
	}
	items, kind, err := runByteSteps(ctx, opts.Context.Layout, task.kind, cloneItems(items), task.steps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	sinkContext := resolveLayoutContext(task.sink.lctx, opts.Context.Layout)
	sink := task.sink
	sink.lctx = sinkContext
	plans, err := planByteSink(ctx, kind, items, sink, opts.Context.DuplicateOutputs)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	writes, err := stageByteWrites(ctx, sinkContext, plans)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	for _, write := range writes {
		entry := byteWriteResultEntry(write)
		if err := write.file.WriteBytes(write.data, sinkContext); err != nil {
			entry.Err = err
			result.recordByteWrite(task.sink.kind, entry)
			return failTask(result, fmt.Errorf("task %q: write %s: %w", task.name, write.file.Path(), err))
		}
		result.recordByteWrite(sink.kind, entry)
	}
	return result, nil
}

func runByteSteps(ctx context.Context, fallback layout.Context, kind subjectKind, items []Item[Blob], steps []byteStep) ([]Item[Blob], subjectKind, error) {
	for _, step := range steps {
		lctx := resolveLayoutContext(step.lctx, fallback)
		switch step.kind {
		case byteStepTransform:
			if step.transform == nil {
				return nil, subjectUnknown, fmt.Errorf("transform step has nil function")
			}
			out := make([]Item[Blob], 0, len(items))
			for _, item := range items {
				data, err := materializeByteItem(ctx, lctx, &item)
				if err != nil {
					return nil, subjectUnknown, err
				}
				transformed, err := layout.TransformBytes(data, step.transform)
				if err != nil {
					return nil, subjectUnknown, err
				}
				item.Data = transformed
				out = append(out, item)
			}
			items = out

		case byteStepSplit:
			if kind != subjectSingle || len(items) != 1 {
				return nil, subjectUnknown, fmt.Errorf("split requires a single subject")
			}
			if step.split == nil {
				return nil, subjectUnknown, fmt.Errorf("split step has nil function")
			}
			collector := &byteSplitCollector{ctx: ctx, lctx: lctx, item: &items[0]}
			if err := step.split(ctx, lctx, collector, items[0]); err != nil {
				return nil, subjectUnknown, err
			}
			items = collector.items
			kind = subjectMulti

		case byteStepFilter:
			if kind != subjectMulti {
				return nil, subjectUnknown, fmt.Errorf("filter requires a multi subject")
			}
			if step.filter == nil {
				return nil, subjectUnknown, fmt.Errorf("filter step has nil function")
			}
			out := make([]Item[Blob], 0, len(items))
			for _, item := range items {
				filter := byteFilterScope{ctx: ctx, lctx: lctx, item: &item}
				keep, err := step.filter(ctx, lctx, filter, item)
				if err != nil {
					return nil, subjectUnknown, err
				}
				if keep {
					out = append(out, item)
				}
			}
			items = out

		case byteStepSort:
			if kind != subjectMulti {
				return nil, subjectUnknown, fmt.Errorf("sort requires a multi subject")
			}
			if step.sort == nil {
				return nil, subjectUnknown, fmt.Errorf("sort step has nil function")
			}
			sort.SliceStable(items, func(i int, j int) bool { return step.sort(items[i], items[j]) })

		case byteStepConcat:
			if kind != subjectMulti {
				return nil, subjectUnknown, fmt.Errorf("concat requires a multi subject")
			}
			readers := make([]io.Reader, 0, len(items))
			for i := range items {
				data, err := materializeByteItem(ctx, lctx, &items[i])
				if err != nil {
					return nil, subjectUnknown, err
				}
				readers = append(readers, bytes.NewReader(data))
			}
			data, err := layout.ConcatReaders(step.concatOptions, readers...)
			if err != nil {
				return nil, subjectUnknown, err
			}
			items = []Item[Blob]{{Key: "concat", Name: "concat", Path: "concat", Value: Blob{Key: "concat", Name: "concat", Path: "concat"}, Data: data}}
			kind = subjectSingle

		default:
			return nil, subjectUnknown, fmt.Errorf("unsupported step kind %d", step.kind)
		}
	}
	return items, kind, nil
}
