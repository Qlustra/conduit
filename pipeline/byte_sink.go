package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qlustra/conduit/layout"
)

// DestinationMode controls how ToDir maps item paths into a destination root.
type DestinationMode uint8

const (
	// DestinationFlatten writes each item under the destination root using only
	// the item's final path element.
	DestinationFlatten DestinationMode = iota + 1

	// DestinationPreserveStructure writes each item under the destination root
	// using its relative item path.
	DestinationPreserveStructure
)

// Destination describes a directory sink root and path shaping mode.
type Destination struct {
	// Root is the directory under which ToDir writes outputs.
	Root layout.Dir

	// Mode controls whether item paths are flattened or preserved.
	Mode DestinationMode
}

// DestinationOption configures a ToDir destination.
type DestinationOption func(*Destination)

// Flatten configures ToDir to discard item path structure.
func Flatten() DestinationOption { return func(dest *Destination) { dest.Mode = DestinationFlatten } }

// PreserveStructure configures ToDir to keep each item's relative path under
// the destination root.
func PreserveStructure() DestinationOption {
	return func(dest *Destination) { dest.Mode = DestinationPreserveStructure }
}

func newDestination(root layout.Dir, opt DestinationOption) Destination {
	dest := Destination{Root: root}
	if opt != nil {
		opt(&dest)
	}
	return dest
}

type byteWritePlan struct {
	file layout.File
	item Item[Blob]
}

type byteWriteIntent struct {
	file layout.File
	item Item[Blob]
	data []byte
}

func planByteSink(ctx context.Context, kind subjectKind, items []Item[Blob], sink byteSink, duplicatePolicy DuplicateOutputPolicy) ([]byteWritePlan, error) {
	plans, err := planByteSinkDestinations(ctx, kind, items, sink)
	if err != nil {
		return nil, err
	}
	return applyDuplicateOutputPolicy(plans, duplicatePolicy)
}

func planByteSinkDestinations(ctx context.Context, kind subjectKind, items []Item[Blob], sink byteSink) ([]byteWritePlan, error) {
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
		if kind != subjectSingle || len(items) != 1 {
			return nil, fmt.Errorf("single-file sink requires a single subject")
		}
		return []byteWritePlan{{file: sink.file, item: items[0]}}, nil

	case byteSinkToDir:
		if kind != subjectMulti {
			return nil, fmt.Errorf("directory sink requires a multi subject")
		}
		plans := make([]byteWritePlan, 0, len(items))
		for _, item := range items {
			file, err := sink.destination.fileFor(item)
			if err != nil {
				return nil, err
			}
			plans = append(plans, byteWritePlan{file: file, item: item})
		}
		return plans, nil

	case byteSinkToFiles:
		if kind != subjectMulti {
			return nil, fmt.Errorf("mapped-file sink requires a multi subject")
		}
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

func (r *TaskResult) recordByteWrite(kind byteSinkKind, entry ByteWriteItemResult) {
	switch kind {
	case byteSinkWriteBack:
		r.WriteBack.Items = append(r.WriteBack.Items, entry)
	case byteSinkToFile:
		r.To.Items = append(r.To.Items, entry)
	case byteSinkToDir:
		r.ToDir.Items = append(r.ToDir.Items, entry)
	case byteSinkToFiles:
		r.ToFiles.Items = append(r.ToFiles.Items, entry)
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

func (plan byteWritePlan) outputPath() (string, error) {
	path := plan.file.Path()
	if path == "" || filepath.Clean(path) == "." {
		return "", fmt.Errorf("item %q has no output file path", plan.item.Name)
	}
	return filepath.Clean(path), nil
}

func (dest Destination) fileFor(item Item[Blob]) (layout.File, error) {
	rel, err := itemOutputPath(item)
	if err != nil {
		return layout.File{}, err
	}
	if err := validateRelativeOutputPath(rel); err != nil {
		return layout.File{}, err
	}

	switch dest.Mode {
	case DestinationFlatten:
		base := filepath.Base(rel)
		if err := validateRelativeOutputPath(base); err != nil {
			return layout.File{}, err
		}
		return dest.Root.File(base), nil
	case DestinationPreserveStructure:
		return dest.Root.File(rel), nil
	default:
		return layout.File{}, fmt.Errorf("unsupported destination mode %d", dest.Mode)
	}
}

func itemOutputPath(item Item[Blob]) (string, error) {
	rel := item.Path
	if rel == "" {
		rel = item.Name
	}
	if rel == "" {
		rel = item.Key
	}
	if rel == "" {
		return "", fmt.Errorf("item has no output path")
	}
	return filepath.Clean(rel), nil
}

func validateRelativeOutputPath(rel string) error {
	if rel == "" {
		return fmt.Errorf("output path must not be empty")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("output path %q must be relative", rel)
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." {
		return fmt.Errorf("output path %q is not valid", rel)
	}
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("output path %q must not escape destination", rel)
	}
	return nil
}
