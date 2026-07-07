package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/qlustra/conduit/layout"
)

type byteStepOp uint8

const (
	byteStepTransform byteStepOp = iota + 1
	byteStepSplit
	byteStepFilter
	byteStepSort
	byteStepConcat
	byteStepPick
	byteStepSelect
)

type byteStep struct {
	step[byteStepOp]

	transform TransformFunc
	split     SplitContentFunc
	filter    FilterContentFunc
	sort      SortContentFunc
	pick      PickContentFunc
	selectFn  SelectContentFunc

	concatOptions layout.ConcatOptions
}

type byteStepper struct {
	*stepperTask[Blob, inputSource[Blob], byteStep, byteSinkOperation, byteSink, byteTaskRuntime]
}

func (p byteStep) inputCardinality(activeCardinality taskCardinality) taskCardinality {
	switch p.kind {
	case byteStepTransform:
		return activeCardinality
	case byteStepSplit:
		return singleTask
	case byteStepFilter:
		return multiTask
	case byteStepSort:
		return multiTask
	case byteStepConcat:
		return multiTask
	case byteStepPick:
		return multiTask
	case byteStepSelect:
		return multiTask
	default:
		return taskCardinalityUnknown
	}
}

func (p byteStep) outputCardinality(activeCardinality taskCardinality) taskCardinality {
	switch p.kind {
	case byteStepTransform:
		return activeCardinality
	case byteStepSplit:
		return multiTask
	case byteStepFilter:
		return multiTask
	case byteStepSort:
		return multiTask
	case byteStepConcat:
		return singleTask
	case byteStepPick:
		return singleTask
	case byteStepSelect:
		return singleTask
	default:
		return taskCardinalityUnknown
	}
}

func (p byteStep) runStep(ctx context.Context, lctx layout.Context, items []Item[Blob]) ([]Item[Blob], error) {
	switch p.kind {
	case byteStepTransform:
		if p.transform == nil {
			return nil, fmt.Errorf("transform step has nil function")
		}
		out := make([]Item[Blob], 0, len(items))
		for _, item := range items {
			data, err := materializeByteItem(ctx, lctx, &item)
			if err != nil {
				return nil, err
			}
			transformed, err := layout.TransformBytes(data, p.transform)
			if err != nil {
				return nil, err
			}
			item.Data = transformed
			out = append(out, item)
		}
		items = out

	case byteStepSplit:
		if p.split == nil {
			return nil, fmt.Errorf("split step has nil function")
		}
		collector := &byteSplitter{ctx: ctx, lctx: lctx, item: &items[0]}
		if err := p.split(ctx, lctx, collector, items[0]); err != nil {
			return nil, err
		}
		items = collector.items

	case byteStepFilter:
		if p.filter == nil {
			return nil, fmt.Errorf("filter step has nil function")
		}
		out := make([]Item[Blob], 0, len(items))
		for _, item := range items {
			filter := byteFilter{ctx: ctx, lctx: lctx, item: &item}
			keep, err := p.filter(ctx, lctx, filter, item)
			if err != nil {
				return nil, err
			}
			if keep {
				out = append(out, item)
			}
		}
		items = out

	case byteStepSort:
		if p.sort == nil {
			return nil, fmt.Errorf("sort step has nil function")
		}
		sort.SliceStable(items, func(i int, j int) bool { return p.sort(items[i], items[j]) })

	case byteStepConcat:
		readers := make([]io.Reader, 0, len(items))
		for i := range items {
			data, err := materializeByteItem(ctx, lctx, &items[i])
			if err != nil {
				return nil, err
			}
			readers = append(readers, bytes.NewReader(data))
		}
		data, err := layout.ConcatReaders(p.concatOptions, readers...)
		if err != nil {
			return nil, err
		}
		items = []Item[Blob]{{Key: "concat", Name: "concat", Path: "concat", Value: Blob{Key: "concat", Name: "concat", Path: "concat"}, Data: data}}

	case byteStepPick:
		if p.pick == nil {
			return nil, fmt.Errorf("pick step has nil function")
		}
		for _, item := range items {
			if p.pick(item) {
				return []Item[Blob]{item}, nil
			}
		}
		return nil, fmt.Errorf("pick step matched no items")

	case byteStepSelect:
		if p.selectFn == nil {
			return nil, fmt.Errorf("select step has nil function")
		}
		return []Item[Blob]{p.selectFn(items)}, nil

	default:
		return nil, fmt.Errorf("unsupported step kind %d", p.kind)
	}

	return items, nil
}
