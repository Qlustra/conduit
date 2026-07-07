package pipeline

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/qlustra/conduit/layout"
)

type byteSinkOperation uint8

const (
	byteSinkUnknown byteSinkOperation = iota
	byteSinkWriteBack
	byteSinkToFile
	byteSinkToDir
	byteSinkToFiles
)

type byteSink struct {
	baseSink[byteSinkOperation]

	file        layout.File
	destination destination
	mapper      MapContentFunc
}

func newByteSink(key string, kind byteSinkOperation, lctx layout.Context) byteSink {
	return byteSink{baseSink: baseSink[byteSinkOperation]{key: key, kind: kind, lctx: lctx}}
}

func newByteSinkWithFile(key string, kind byteSinkOperation, lctx layout.Context, file layout.File) byteSink {
	return byteSink{baseSink: baseSink[byteSinkOperation]{key: key, kind: kind, lctx: lctx}, file: file}
}

func newByteSinkWithDestination(key string, kind byteSinkOperation, lctx layout.Context, destination destination) byteSink {
	return byteSink{baseSink: baseSink[byteSinkOperation]{key: key, kind: kind, lctx: lctx}, destination: destination}
}

func newByteSinkWithMapper(key string, kind byteSinkOperation, lctx layout.Context, mapper MapContentFunc) byteSink {
	return byteSink{baseSink: baseSink[byteSinkOperation]{key: key, kind: kind, lctx: lctx}, mapper: mapper}
}

func (s byteSink) validateCardinality(cardinality taskCardinality) bool {
	k := s.kind
	if k == byteSinkWriteBack {
		return true
	}
	if cardinality == singleTask && k == byteSinkToFile {
		return true
	} else if cardinality == multiTask && (k == byteSinkToDir || k == byteSinkToFiles) {
		return true
	}

	return false
}

func (s byteSink) getKind() byteSinkOperation {
	return s.kind
}

func (s byteSink) isKind(kind byteSinkOperation) bool {
	return s.kind == kind
}

func (s byteSink) getName() string {
	switch s.kind {
	case byteSinkUnknown:
		return "Unknown"
	case byteSinkWriteBack:
		return "WriteBack"
	case byteSinkToFile:
		return "ToFile"
	case byteSinkToDir:
		return "ToDir"
	case byteSinkToFiles:
		return "ToFiles"
	default:
		return fmt.Sprintf("byte sink %d", s.kind)
	}
}

type byteWritePlan struct {
	file layout.File
	item Item[Blob]
}

func (plan byteWritePlan) outputPath() (string, error) {
	path := plan.file.Path()
	if path == "" || filepath.Clean(path) == "." {
		return "", fmt.Errorf("item %q has no output file path", plan.item.Name)
	}
	return filepath.Clean(path), nil
}

type byteWriteIntent struct {
	file layout.File
	item Item[Blob]
	data []byte
}

func planByteSink(ctx context.Context, items []Item[Blob], sink byteSink, duplicatePolicy DuplicateOutputPolicy) ([]byteWritePlan, error) {
	plans, err := planByteSinkDestinations(ctx, items, sink)
	if err != nil {
		return nil, err
	}
	return applyDuplicateOutputPolicy(plans, duplicatePolicy)
}

func planByteSinkDestinations(ctx context.Context, items []Item[Blob], sink byteSink) ([]byteWritePlan, error) {
	switch sink.kind {
	case byteSinkWriteBack:
		plans := make([]byteWritePlan, 0, len(items))
		for _, item := range items {
			if !hasFile(item.File) {
				return nil, fmt.Errorf("item %q cannot be written back", item.Name)
			}
			plans = append(plans, byteWritePlan{file: item.File, item: item})
		}
		return plans, nil

	case byteSinkToFile:
		return []byteWritePlan{{file: sink.file, item: items[0]}}, nil

	case byteSinkToDir:
		plans := make([]byteWritePlan, 0, len(items))
		for _, item := range items {
			file, err := sink.destination.resolveFile(item)
			if err != nil {
				return nil, err
			}
			plans = append(plans, byteWritePlan{file: file, item: item})
		}
		return plans, nil

	case byteSinkToFiles:
		if sink.mapper == nil {
			return nil, fmt.Errorf("mapped-file sink has nil mapper")
		}
		plans := make([]byteWritePlan, 0, len(items))
		for _, item := range items {
			file, err := sink.mapper(ctx, sink.lctx, item)
			if err != nil {
				return nil, err
			}
			plans = append(plans, byteWritePlan{file: file, item: item})
		}
		return plans, nil

	default:
		return nil, fmt.Errorf("unsupported sink kind %d", sink.kind)
	}
}

func stageByteWrites(ctx context.Context, lctx layout.Context, plans []byteWritePlan) ([]byteWriteIntent, error) {
	writes := make([]byteWriteIntent, 0, len(plans))
	for _, plan := range plans {
		data, err := materializeByteItem(ctx, lctx, &plan.item)
		if err != nil {
			return nil, err
		}
		writes = append(writes, byteWriteIntent{file: plan.file, item: plan.item, data: data})
	}
	return writes, nil
}

func byteWriteResultEntry(write byteWriteIntent) ByteWriteItemResult {
	return ByteWriteItemResult{
		Key:   write.item.Key,
		Name:  write.item.Name,
		Path:  write.item.Path,
		File:  write.file.Path(),
		Bytes: len(write.data),
	}
}

func applyDuplicateOutputPolicy(plans []byteWritePlan, policy DuplicateOutputPolicy) ([]byteWritePlan, error) {
	seen := make(map[string]int, len(plans))
	for i, plan := range plans {
		path, err := plan.outputPath()
		if err != nil {
			return nil, err
		}
		if prev, ok := seen[path]; ok && policy == DuplicateOutputFail {
			return nil, fmt.Errorf("duplicate output path %q for items %q and %q", path, plans[prev].item.Name, plan.item.Name)
		}
		seen[path] = i
	}

	switch policy {
	case DuplicateOutputFail:
		return plans, nil
	case DuplicateOutputLastWins:
		filtered := make([]byteWritePlan, 0, len(seen))
		for i, plan := range plans {
			path, err := plan.outputPath()
			if err != nil {
				return nil, err
			}
			if seen[path] == i {
				filtered = append(filtered, plan)
			}
		}
		return filtered, nil
	default:
		return nil, fmt.Errorf("unsupported duplicate output policy %d", policy)
	}
}

func sinkByteItems(ctx context.Context, pctx Context, outputs []Item[Blob], sink byteSink, result *TaskResult) error {
	if sink.kind == byteSinkUnknown {
		return nil
	}
	layoutContext := sink.resolveLayoutContext(pctx.Layout)
	plans, err := planByteSink(ctx, outputs, sink, pctx.DuplicateOutputs)
	if err != nil {
		return err
	}
	writes, err := stageByteWrites(ctx, layoutContext, plans)
	if err != nil {
		return err
	}
	for _, write := range writes {
		entry := byteWriteResultEntry(write)
		if err := write.file.WriteBytes(write.data, layoutContext); err != nil {
			entry.Err = err
			result.recordByteWrite(sink.kind, entry)
			return fmt.Errorf("write %s: %w", write.file.Path(), err)
		}
		result.recordByteWrite(sink.kind, entry)
	}
	return nil
}
