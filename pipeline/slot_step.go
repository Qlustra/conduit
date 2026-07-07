package pipeline

import (
	"context"
	"fmt"
	"sort"

	"github.com/qlustra/conduit/layout"
)

type slotStepOp uint8

const (
	slotStepProcess slotStepOp = iota + 1
	slotStepFilter
	slotStepSort
	slotStepSplit
)

type slotStep[I any] struct {
	step[slotStepOp]

	process ProcessTypedFunc[I]
	filter  FilterTypedFunc[I]
	sort    SortTypedFunc[I]
	split   SplitTypedFunc[I]
}

func (p slotStep[I]) inputCardinality(_ taskCardinality) taskCardinality {
	return multiTask
}

func (p slotStep[I]) outputCardinality(_ taskCardinality) taskCardinality {
	return multiTask
}

func (p slotStep[I]) runStep(ctx context.Context, lctx layout.Context, items []Item[I]) ([]Item[I], error) {
	switch p.kind {
	case slotStepProcess:
		if p.process == nil {
			return nil, fmt.Errorf("process step has nil function")
		}
		out := make([]Item[I], 0, len(items))
		for _, item := range items {
			value, err := p.process(ctx, lctx, item)
			if err != nil {
				return nil, err
			}
			item.Value = value
			out = append(out, item)
		}
		items = out

	case slotStepFilter:
		if p.filter == nil {
			return nil, fmt.Errorf("filter step has nil function")
		}
		out := make([]Item[I], 0, len(items))
		for _, item := range items {
			keep, err := p.filter(ctx, lctx, item)
			if err != nil {
				return nil, err
			}
			if keep {
				out = append(out, item)
			}
		}
		items = out

	case slotStepSort:
		if p.sort == nil {
			return nil, fmt.Errorf("sort step has nil function")
		}
		sort.SliceStable(items, func(i int, j int) bool { return p.sort(items[i], items[j]) })

	case slotStepSplit:
		if p.split == nil {
			return nil, fmt.Errorf("split step has nil function")
		}
		out := make([]Item[I], 0)
		for _, item := range items {
			collector := &typedSplitter[I]{}
			if err := p.split(ctx, lctx, collector, item); err != nil {
				return nil, err
			}
			out = append(out, collector.items...)
		}
		items = out

	default:
		return nil, fmt.Errorf("unsupported typed step kind %d", p.kind)
	}

	return items, nil
}
