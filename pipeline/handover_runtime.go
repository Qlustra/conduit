package pipeline

import (
	"context"
	"fmt"
	"sort"

	"github.com/qlustra/conduit/layout"
)

type handoverOriginStepKind uint8

const (
	handoverOriginStepFilter handoverOriginStepKind = iota + 1
	handoverOriginStepSort
)

type handoverOriginStep[O any] struct {
	kind   handoverOriginStepKind
	lctx   layout.Context
	filter TypedFilterFunc[O]
	sort   TypedSortFunc[O]
}

type handoverTargetPlan[O any] struct {
	key    string
	origin Item[O]
}

type handoverEmission[O, T any] struct {
	key      string
	origin   Item[O]
	populate func(target *Item[T]) error
}

type extractEmitter[T any] struct {
	emit func(key string, populate func(target *Item[T]) error)
}

func (e extractEmitter[T]) Emit(key string, populate func(target *Item[T]) error) {
	e.emit(key, populate)
}

func runHandoverOriginSteps[O any](ctx context.Context, fallback layout.Context, items []Item[O], steps []handoverOriginStep[O]) ([]Item[O], error) {
	for _, step := range steps {
		lctx := resolveLayoutContext(step.lctx, fallback)
		switch step.kind {
		case handoverOriginStepFilter:
			if step.filter == nil {
				return nil, fmt.Errorf("filter step has nil function")
			}
			out := make([]Item[O], 0, len(items))
			for _, item := range items {
				keep, err := step.filter(ctx, lctx, item)
				if err != nil {
					return nil, err
				}
				if keep {
					out = append(out, item)
				}
			}
			items = out
		case handoverOriginStepSort:
			if step.sort == nil {
				return nil, fmt.Errorf("sort step has nil function")
			}
			sort.SliceStable(items, func(i int, j int) bool { return step.sort(items[i], items[j]) })
		default:
			return nil, fmt.Errorf("unsupported handover origin step kind %d", step.kind)
		}
	}
	return items, nil
}

func applyHandoverDuplicatePolicy[O any](plans []handoverTargetPlan[O], policy DuplicateOutputPolicy) ([]handoverTargetPlan[O], error) {
	seen := make(map[string]int, len(plans))
	for i, plan := range plans {
		if plan.key == "" {
			return nil, fmt.Errorf("origin item %q has no target key", plan.origin.Name)
		}
		if prev, ok := seen[plan.key]; ok && policy == DuplicateOutputFail {
			return nil, fmt.Errorf("duplicate target key %q for items %q and %q", plan.key, plans[prev].origin.Name, plan.origin.Name)
		}
		seen[plan.key] = i
	}

	switch policy {
	case DuplicateOutputFail:
		return plans, nil
	case DuplicateOutputLastWins:
		filtered := make([]handoverTargetPlan[O], 0, len(seen))
		for i, plan := range plans {
			if seen[plan.key] == i {
				filtered = append(filtered, plan)
			}
		}
		return filtered, nil
	default:
		return nil, fmt.Errorf("unsupported duplicate output policy %d", policy)
	}
}

func applyHandoverEmissionDuplicatePolicy[O, T any](emissions []handoverEmission[O, T], policy DuplicateOutputPolicy) ([]handoverEmission[O, T], error) {
	seen := make(map[string]int, len(emissions))
	for i, emission := range emissions {
		if emission.key == "" {
			return nil, fmt.Errorf("origin item %q emitted no target key", emission.origin.Name)
		}
		if prev, ok := seen[emission.key]; ok && policy == DuplicateOutputFail {
			return nil, fmt.Errorf("duplicate target key %q for items %q and %q", emission.key, emissions[prev].origin.Name, emission.origin.Name)
		}
		seen[emission.key] = i
	}

	switch policy {
	case DuplicateOutputFail:
		return emissions, nil
	case DuplicateOutputLastWins:
		filtered := make([]handoverEmission[O, T], 0, len(seen))
		for i, emission := range emissions {
			if seen[emission.key] == i {
				filtered = append(filtered, emission)
			}
		}
		return filtered, nil
	default:
		return nil, fmt.Errorf("unsupported duplicate output policy %d", policy)
	}
}

func runHandoverSinks[T any](result *TaskResult, fallback layout.Context, items []Item[T], sinks []typedSink) error {
	for _, sink := range sinks {
		sink.lctx = resolveLayoutContext(sink.lctx, fallback)
		for i := range items {
			entry, err := runTypedSink(&items[i], sink)
			result.recordDeepOperation(sink.kind, entry)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func handoverItemResult[O, T any](origin Item[O], target Item[T]) HandoverItemResult {
	return HandoverItemResult{
		OriginKey:  origin.Key,
		OriginName: origin.Name,
		OriginPath: origin.Path,
		OriginFile: filePath(origin.File),
		OriginDir:  dirPath(origin.Dir),
		TargetKey:  target.Key,
		TargetName: target.Name,
		TargetPath: target.Path,
		TargetFile: filePath(target.File),
		TargetDir:  dirPath(target.Dir),
	}
}

func compileHandoverItemResult[O, T any](origins []Item[O], target Item[T]) HandoverItemResult {
	entry := HandoverItemResult{
		TargetKey:  target.Key,
		TargetName: target.Name,
		TargetPath: target.Path,
		TargetFile: filePath(target.File),
		TargetDir:  dirPath(target.Dir),
		OriginKeys: make([]string, 0, len(origins)),
	}
	for _, origin := range origins {
		entry.OriginKeys = append(entry.OriginKeys, origin.Key)
	}
	return entry
}

func filePath(file layout.File) string {
	if hasFile(file) {
		return file.Path()
	}
	return ""
}

func dirPath(dir layout.Dir) string {
	if hasDir(dir) {
		return dir.Path()
	}
	return ""
}
