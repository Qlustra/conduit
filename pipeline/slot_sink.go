package pipeline

import (
	"context"
	"fmt"
	"reflect"

	"github.com/qlustra/conduit/layout"
)

type slotSinkOperation uint8

const (
	slotSinkUnknown slotSinkOperation = iota
	slotSinkEnsureDeep
	slotSinkDefaultDeep
	slotSinkRenderDeep
	slotSinkSyncDeep
	slotSinkValidateDeep
)

type slotSink struct {
	baseSink[slotSinkOperation]

	validate layout.ValidateOptions
}

func (s slotSink) getKind() slotSinkOperation {
	return s.kind
}

func (s slotSink) isKind(kind slotSinkOperation) bool {
	return s.kind == kind
}

func (s slotSink) getName() string {
	switch s.kind {
	case slotSinkEnsureDeep:
		return "EnsureDeep"
	case slotSinkDefaultDeep:
		return "DefaultDeep"
	case slotSinkRenderDeep:
		return "RenderDeep"
	case slotSinkSyncDeep:
		return "SyncDeep"
	case slotSinkValidateDeep:
		return "ValidateDeep"
	default:
		return fmt.Sprintf("slot sink %d", s.kind)
	}
}

func sinkSlotItems[T any](_ context.Context, pctx Context, outputs []Item[T], sink slotSink, result *TaskResult) error {
	layoutContext := sink.resolveLayoutContext(pctx.Layout)
	for i := range outputs {
		entry, err := runSlotSink(&outputs[i], sink, layoutContext)
		result.recordDeepOperation(sink.kind, entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func runSlotSink[T any](item *Item[T], sink slotSink, layoutContext layout.Context) (SlotOperationResult, error) {
	entry := slotOperationResult(item)
	target := slotOperationTarget(item)
	switch sink.kind {
	case slotSinkEnsureDeep:
		report := &layout.Report{}
		lctx := layoutContext
		lctx.Reporter = joinReporter(lctx.Reporter, report)
		result, err := layout.EnsureDeep(target, lctx)
		entry.Result = result
		entry.Entries = report.Entries()
		entry.Err = err
		return entry, err
	case slotSinkDefaultDeep:
		err := layout.DefaultDeep(target)
		entry.Err = err
		return entry, err
	case slotSinkRenderDeep:
		err := layout.RenderDeep(target)
		entry.Err = err
		return entry, err
	case slotSinkSyncDeep:
		report := &layout.Report{}
		lctx := layoutContext
		lctx.Reporter = joinReporter(lctx.Reporter, report)
		result, err := layout.SyncDeep(target, lctx)
		entry.Result = result
		entry.Entries = report.Entries()
		entry.Err = err
		return entry, err
	case slotSinkValidateDeep:
		report := &layout.Report{}
		opts := sink.validate
		opts.Reporter = joinReporter(opts.Reporter, report)
		result, err := layout.ValidateDeep(target, opts)
		entry.Result = result
		entry.Entries = report.Entries()
		entry.Err = err
		return entry, err
	default:
		err := fmt.Errorf("unsupported typed sink kind %d", sink.kind)
		entry.Err = err
		return entry, err
	}
}

func slotOperationResult[T any](item *Item[T]) SlotOperationResult {
	result := SlotOperationResult{Key: item.Key, Name: item.Name, Path: item.Path}
	if hasFile(item.File) {
		result.File = item.File.Path()
	}
	if hasDir(item.Dir) {
		result.Dir = item.Dir.Path()
	}
	return result
}

type joinedReporter []layout.Reporter

func joinReporter(reporters ...layout.Reporter) layout.Reporter {
	joined := make(joinedReporter, 0, len(reporters))
	for _, reporter := range reporters {
		if reporter != nil {
			joined = append(joined, reporter)
		}
	}
	if len(joined) == 0 {
		return nil
	}
	return joined
}

func (r joinedReporter) Record(entry layout.Entry) {
	for _, reporter := range r {
		reporter.Record(entry)
	}
}

func slotOperationTarget[T any](item *Item[T]) any {
	value := any(item.Value)
	if value == nil {
		return &item.Value
	}
	if reflect.TypeOf(value).Kind() == reflect.Pointer {
		return item.Value
	}
	return &item.Value
}
