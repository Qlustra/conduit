package pipeline

import (
	"fmt"
	"reflect"

	"github.com/qlustra/conduit/layout"
)

type typedSinkKind uint8

const (
	typedSinkUnknown typedSinkKind = iota
	typedSinkEnsureDeep
	typedSinkDefaultDeep
	typedSinkRenderDeep
	typedSinkSyncDeep
	typedSinkValidateDeep
)

type typedSink struct {
	kind     typedSinkKind
	lctx     layout.Context
	validate layout.ValidateOptions
}

func addTypedSink(taskName string, configErr *error, sinks *[]typedSink, sink typedSink) {
	if *configErr != nil {
		return
	}
	for _, existing := range *sinks {
		if existing.kind == sink.kind {
			*configErr = fmt.Errorf("task %q already has %s", taskName, typedSinkName(sink.kind))
			return
		}
	}
	*sinks = append(*sinks, sink)
}

func runTypedSink[T any](item *Item[T], sink typedSink) (DeepItemResult, error) {
	entry := deepItemResult(item)
	target := deepTarget(item)
	switch sink.kind {
	case typedSinkEnsureDeep:
		report := &layout.Report{}
		lctx := sink.lctx
		lctx.Reporter = joinReporter(lctx.Reporter, report)
		result, err := layout.EnsureDeep(target, lctx)
		entry.Result = result
		entry.Entries = report.Entries()
		entry.Err = err
		return entry, err
	case typedSinkDefaultDeep:
		err := layout.DefaultDeep(target)
		entry.Err = err
		return entry, err
	case typedSinkRenderDeep:
		err := layout.RenderDeep(target)
		entry.Err = err
		return entry, err
	case typedSinkSyncDeep:
		report := &layout.Report{}
		lctx := sink.lctx
		lctx.Reporter = joinReporter(lctx.Reporter, report)
		result, err := layout.SyncDeep(target, lctx)
		entry.Result = result
		entry.Entries = report.Entries()
		entry.Err = err
		return entry, err
	case typedSinkValidateDeep:
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

func deepItemResult[T any](item *Item[T]) DeepItemResult {
	result := DeepItemResult{Key: item.Key, Name: item.Name, Path: item.Path}
	if hasFile(item.File) {
		result.File = item.File.Path()
	}
	if hasDir(item.Dir) {
		result.Dir = item.Dir.Path()
	}
	return result
}

func (r *TaskResult) recordDeepOperation(kind typedSinkKind, entry DeepItemResult) {
	switch kind {
	case typedSinkEnsureDeep:
		r.EnsureDeep.Items = append(r.EnsureDeep.Items, entry)
	case typedSinkDefaultDeep:
		r.DefaultDeep.Items = append(r.DefaultDeep.Items, entry)
	case typedSinkRenderDeep:
		r.RenderDeep.Items = append(r.RenderDeep.Items, entry)
	case typedSinkSyncDeep:
		r.SyncDeep.Items = append(r.SyncDeep.Items, entry)
	case typedSinkValidateDeep:
		r.ValidateDeep.Items = append(r.ValidateDeep.Items, entry)
	}
}

func typedSinkName(kind typedSinkKind) string {
	switch kind {
	case typedSinkEnsureDeep:
		return "EnsureDeep"
	case typedSinkDefaultDeep:
		return "DefaultDeep"
	case typedSinkRenderDeep:
		return "RenderDeep"
	case typedSinkSyncDeep:
		return "SyncDeep"
	case typedSinkValidateDeep:
		return "ValidateDeep"
	default:
		return fmt.Sprintf("typed sink %d", kind)
	}
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

func deepTarget[T any](item *Item[T]) any {
	value := any(item.Value)
	if value == nil {
		return &item.Value
	}
	if reflect.TypeOf(value).Kind() == reflect.Pointer {
		return item.Value
	}
	return &item.Value
}
