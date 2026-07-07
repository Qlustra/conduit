package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// CompileToFileTask builds one byte subject from many typed origin entries.
type CompileToFileTask[O any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name   string
	origin Entries[O]
	target *BlobSubject

	originSteps []handoverOriginStep[O]

	build     BuildFunc[O, Blob]
	buildLctx layout.Context

	sink      byteSink
	configErr error
}

type compileToFileTaskSnapshot[O any] struct {
	name        string
	origin      Entries[O]
	target      *BlobSubject
	originSteps []handoverOriginStep[O]
	build       BuildFunc[O, Blob]
	buildLctx   layout.Context
	sink        byteSink
	configErr   error
}

type compileToFileTaskRunSnapshot[O any] struct {
	task  compileToFileTaskSnapshot[O]
	runMu *sync.Mutex
}

// CompileToFile returns a many-to-one byte-producing handover task.
func CompileToFile[O any](name string, origin Entries[O], target *BlobSubject) *CompileToFileTask[O] {
	return &CompileToFileTask[O]{name: name, origin: origin, target: target}
}

func (t *CompileToFileTask[O]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

func (t *CompileToFileTask[O]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *CompileToFileTask[O]) snapshotRunnable() Runnable {
	return compileToFileTaskRunSnapshot[O]{task: t.snapshot(), runMu: &t.runMu}
}

func (s compileToFileTaskRunSnapshot[O]) Name() string { return s.task.name }

func (s compileToFileTaskRunSnapshot[O]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return runCompileToFileTask(ctx, opts, s.task)
}

func (t *CompileToFileTask[O]) Filter(lctx layout.Context, fn TypedFilterFunc[O]) *CompileToFileTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepFilter, lctx: lctx, filter: fn})
	return t
}

func (t *CompileToFileTask[O]) Sort(fn TypedSortFunc[O]) *CompileToFileTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepSort, sort: fn})
	return t
}

func (t *CompileToFileTask[O]) Build(lctx layout.Context, fn BuildFunc[O, Blob]) *CompileToFileTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.configErr != nil {
		return t
	}
	if t.build != nil {
		t.configErr = fmt.Errorf("task %q already has Build", t.name)
		return t
	}
	t.build = fn
	t.buildLctx = lctx
	return t
}

func (t *CompileToFileTask[O]) ToTarget(lctx layout.Context) *CompileToFileTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToTarget, lctx: lctx})
	return t
}

func (t *CompileToFileTask[O]) To(lctx layout.Context, dest layout.File) *CompileToFileTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToFile, lctx: lctx, file: dest})
	return t
}

func (t *CompileToFileTask[O]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()
	return runCompileToFileTask(ctx, opts, t.snapshot())
}

func (t *CompileToFileTask[O]) snapshot() compileToFileTaskSnapshot[O] {
	t.mu.RLock()
	defer t.mu.RUnlock()
	originSteps := make([]handoverOriginStep[O], len(t.originSteps))
	copy(originSteps, t.originSteps)
	return compileToFileTaskSnapshot[O]{
		name:        t.name,
		origin:      t.origin,
		target:      t.target,
		originSteps: originSteps,
		build:       t.build,
		buildLctx:   t.buildLctx,
		sink:        t.sink,
		configErr:   t.configErr,
	}
}

func runCompileToFileTask[O any](ctx context.Context, opts RunOptions, task compileToFileTaskSnapshot[O]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	if err := opts.Context.validate(); err != nil {
		return failTask(result, err)
	}
	if task.configErr != nil {
		return failTask(result, task.configErr)
	}
	if task.origin == nil {
		return failTask(result, fmt.Errorf("task %q has no origin", task.name))
	}
	if task.target == nil {
		return failTask(result, fmt.Errorf("task %q has no target", task.name))
	}
	if task.build == nil {
		return failTask(result, fmt.Errorf("task %q has no Build", task.name))
	}
	origins, err := runHandoverOriginSteps(ctx, opts.Context.Layout, task.origin.snapshot(), task.originSteps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	target := task.target.Snapshot()
	buildContext := resolveLayoutContext(task.buildLctx, opts.Context.Layout)
	if err := task.build(ctx, buildContext, origins, &target); err != nil {
		return failTask(result, fmt.Errorf("task %q: build: %w", task.name, err))
	}
	target = normalizeBlobItem(target)
	task.target.put(target)
	if err := runProducedByteSink(ctx, opts, subjectSingle, []Item[Blob]{target}, task.sink, &result); err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	return result, nil
}

// BridgeToFilesTask maps typed origin entries to byte subjects one-to-one.
type BridgeToFilesTask[O any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name   string
	origin Entries[O]
	target *BlobSubjectSet

	originSteps []handoverOriginStep[O]

	rekey     HandoverKeyFunc[O]
	rekeyLctx layout.Context

	populate     BridgeFunc[O, Blob]
	populateLctx layout.Context

	sink      byteSink
	configErr error
}

type bridgeToFilesTaskSnapshot[O any] struct {
	name         string
	origin       Entries[O]
	target       *BlobSubjectSet
	originSteps  []handoverOriginStep[O]
	rekey        HandoverKeyFunc[O]
	rekeyLctx    layout.Context
	populate     BridgeFunc[O, Blob]
	populateLctx layout.Context
	sink         byteSink
	configErr    error
}

type bridgeToFilesTaskRunSnapshot[O any] struct {
	task  bridgeToFilesTaskSnapshot[O]
	runMu *sync.Mutex
}

// BridgeToFiles returns a same-cardinality byte-producing handover task.
func BridgeToFiles[O any](name string, origin Entries[O], target *BlobSubjectSet) *BridgeToFilesTask[O] {
	return &BridgeToFilesTask[O]{name: name, origin: origin, target: target}
}

func (t *BridgeToFilesTask[O]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

func (t *BridgeToFilesTask[O]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *BridgeToFilesTask[O]) snapshotRunnable() Runnable {
	return bridgeToFilesTaskRunSnapshot[O]{task: t.snapshot(), runMu: &t.runMu}
}

func (s bridgeToFilesTaskRunSnapshot[O]) Name() string { return s.task.name }

func (s bridgeToFilesTaskRunSnapshot[O]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return runBridgeToFilesTask(ctx, opts, s.task)
}

func (t *BridgeToFilesTask[O]) Filter(lctx layout.Context, fn TypedFilterFunc[O]) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepFilter, lctx: lctx, filter: fn})
	return t
}

func (t *BridgeToFilesTask[O]) Sort(fn TypedSortFunc[O]) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepSort, sort: fn})
	return t
}

func (t *BridgeToFilesTask[O]) Rekey(lctx layout.Context, fn HandoverKeyFunc[O]) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.configErr != nil {
		return t
	}
	if t.rekey != nil {
		t.configErr = fmt.Errorf("task %q already has Rekey", t.name)
		return t
	}
	t.rekey = fn
	t.rekeyLctx = lctx
	return t
}

func (t *BridgeToFilesTask[O]) Populate(lctx layout.Context, fn BridgeFunc[O, Blob]) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.configErr != nil {
		return t
	}
	if t.populate != nil {
		t.configErr = fmt.Errorf("task %q already has Populate", t.name)
		return t
	}
	t.populate = fn
	t.populateLctx = lctx
	return t
}

func (t *BridgeToFilesTask[O]) ToTargets(lctx layout.Context) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToTargets, lctx: lctx})
	return t
}

func (t *BridgeToFilesTask[O]) ToDir(lctx layout.Context, dest layout.Dir, opt DestinationOption) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToDir, lctx: lctx, destination: newDestination(dest, opt)})
	return t
}

func (t *BridgeToFilesTask[O]) ToFiles(lctx layout.Context, mapper FileMapper) *BridgeToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToFiles, lctx: lctx, mapper: mapper})
	return t
}

func (t *BridgeToFilesTask[O]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()
	return runBridgeToFilesTask(ctx, opts, t.snapshot())
}

func (t *BridgeToFilesTask[O]) snapshot() bridgeToFilesTaskSnapshot[O] {
	t.mu.RLock()
	defer t.mu.RUnlock()
	originSteps := make([]handoverOriginStep[O], len(t.originSteps))
	copy(originSteps, t.originSteps)
	return bridgeToFilesTaskSnapshot[O]{
		name:         t.name,
		origin:       t.origin,
		target:       t.target,
		originSteps:  originSteps,
		rekey:        t.rekey,
		rekeyLctx:    t.rekeyLctx,
		populate:     t.populate,
		populateLctx: t.populateLctx,
		sink:         t.sink,
		configErr:    t.configErr,
	}
}

func runBridgeToFilesTask[O any](ctx context.Context, opts RunOptions, task bridgeToFilesTaskSnapshot[O]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	if err := opts.Context.validate(); err != nil {
		return failTask(result, err)
	}
	if task.configErr != nil {
		return failTask(result, task.configErr)
	}
	if task.origin == nil {
		return failTask(result, fmt.Errorf("task %q has no origin", task.name))
	}
	if task.target == nil {
		return failTask(result, fmt.Errorf("task %q has no target", task.name))
	}
	if task.populate == nil {
		return failTask(result, fmt.Errorf("task %q has no Populate", task.name))
	}
	origins, err := runHandoverOriginSteps(ctx, opts.Context.Layout, task.origin.snapshot(), task.originSteps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	plans := make([]handoverTargetPlan[O], 0, len(origins))
	rekeyContext := resolveLayoutContext(task.rekeyLctx, opts.Context.Layout)
	for _, origin := range origins {
		key := origin.Key
		if task.rekey != nil {
			key, err = task.rekey(ctx, rekeyContext, origin)
			if err != nil {
				return failTask(result, fmt.Errorf("task %q: rekey %q: %w", task.name, origin.Name, err))
			}
		}
		plans = append(plans, handoverTargetPlan[O]{key: key, origin: origin})
	}
	plans, err = applyHandoverDuplicatePolicy(plans, opts.Context.DuplicateOutputs)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	populateContext := resolveLayoutContext(task.populateLctx, opts.Context.Layout)
	targets := make([]Item[Blob], 0, len(plans))
	for _, plan := range plans {
		subject := task.target.At(plan.key)
		target := subject.Snapshot()
		target.Key = plan.key
		target.Value.Key = plan.key
		if err := task.populate(ctx, populateContext, plan.origin, &target); err != nil {
			return failTask(result, fmt.Errorf("task %q: populate %q: %w", task.name, plan.key, err))
		}
		target.Key = plan.key
		target.Value.Key = plan.key
		target = normalizeBlobItem(target)
		subject.put(target)
		targets = append(targets, target)
	}
	if err := runProducedByteSink(ctx, opts, subjectMulti, targets, task.sink, &result); err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	return result, nil
}

// ExpandToFilesTask extracts zero or more byte subjects from each typed origin entry.
type ExpandToFilesTask[O any] struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name   string
	origin Entries[O]
	target *BlobSubjectSet

	originSteps []handoverOriginStep[O]

	extract     ExtractFunc[O, Blob]
	extractLctx layout.Context

	sink      byteSink
	configErr error
}

type expandToFilesTaskSnapshot[O any] struct {
	name        string
	origin      Entries[O]
	target      *BlobSubjectSet
	originSteps []handoverOriginStep[O]
	extract     ExtractFunc[O, Blob]
	extractLctx layout.Context
	sink        byteSink
	configErr   error
}

type expandToFilesTaskRunSnapshot[O any] struct {
	task  expandToFilesTaskSnapshot[O]
	runMu *sync.Mutex
}

// ExpandToFiles returns a one-to-many byte-producing handover task.
func ExpandToFiles[O any](name string, origin Entries[O], target *BlobSubjectSet) *ExpandToFilesTask[O] {
	return &ExpandToFilesTask[O]{name: name, origin: origin, target: target}
}

func (t *ExpandToFilesTask[O]) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

func (t *ExpandToFilesTask[O]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.run(ctx, opts)
}

func (t *ExpandToFilesTask[O]) snapshotRunnable() Runnable {
	return expandToFilesTaskRunSnapshot[O]{task: t.snapshot(), runMu: &t.runMu}
}

func (s expandToFilesTaskRunSnapshot[O]) Name() string { return s.task.name }

func (s expandToFilesTaskRunSnapshot[O]) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return runExpandToFilesTask(ctx, opts, s.task)
}

func (t *ExpandToFilesTask[O]) Filter(lctx layout.Context, fn TypedFilterFunc[O]) *ExpandToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepFilter, lctx: lctx, filter: fn})
	return t
}

func (t *ExpandToFilesTask[O]) Sort(fn TypedSortFunc[O]) *ExpandToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.originSteps = append(t.originSteps, handoverOriginStep[O]{kind: handoverOriginStepSort, sort: fn})
	return t
}

func (t *ExpandToFilesTask[O]) Extract(lctx layout.Context, fn ExtractFunc[O, Blob]) *ExpandToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.configErr != nil {
		return t
	}
	if t.extract != nil {
		t.configErr = fmt.Errorf("task %q already has Extract", t.name)
		return t
	}
	t.extract = fn
	t.extractLctx = lctx
	return t
}

func (t *ExpandToFilesTask[O]) ToTargets(lctx layout.Context) *ExpandToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToTargets, lctx: lctx})
	return t
}

func (t *ExpandToFilesTask[O]) ToDir(lctx layout.Context, dest layout.Dir, opt DestinationOption) *ExpandToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToDir, lctx: lctx, destination: newDestination(dest, opt)})
	return t
}

func (t *ExpandToFilesTask[O]) ToFiles(lctx layout.Context, mapper FileMapper) *ExpandToFilesTask[O] {
	t.mu.Lock()
	defer t.mu.Unlock()
	setByteSink(t.name, &t.configErr, &t.sink, byteSink{kind: byteSinkToFiles, lctx: lctx, mapper: mapper})
	return t
}

func (t *ExpandToFilesTask[O]) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()
	return runExpandToFilesTask(ctx, opts, t.snapshot())
}

func (t *ExpandToFilesTask[O]) snapshot() expandToFilesTaskSnapshot[O] {
	t.mu.RLock()
	defer t.mu.RUnlock()
	originSteps := make([]handoverOriginStep[O], len(t.originSteps))
	copy(originSteps, t.originSteps)
	return expandToFilesTaskSnapshot[O]{
		name:        t.name,
		origin:      t.origin,
		target:      t.target,
		originSteps: originSteps,
		extract:     t.extract,
		extractLctx: t.extractLctx,
		sink:        t.sink,
		configErr:   t.configErr,
	}
}

func runExpandToFilesTask[O any](ctx context.Context, opts RunOptions, task expandToFilesTaskSnapshot[O]) (TaskResult, error) {
	result := TaskResult{Name: task.name, Status: StatusRan}
	if err := opts.Context.validate(); err != nil {
		return failTask(result, err)
	}
	if task.configErr != nil {
		return failTask(result, task.configErr)
	}
	if task.origin == nil {
		return failTask(result, fmt.Errorf("task %q has no origin", task.name))
	}
	if task.target == nil {
		return failTask(result, fmt.Errorf("task %q has no target", task.name))
	}
	if task.extract == nil {
		return failTask(result, fmt.Errorf("task %q has no Extract", task.name))
	}
	origins, err := runHandoverOriginSteps(ctx, opts.Context.Layout, task.origin.snapshot(), task.originSteps)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	extractContext := resolveLayoutContext(task.extractLctx, opts.Context.Layout)
	emissions := make([]handoverEmission[O, Blob], 0)
	for _, origin := range origins {
		emitter := extractEmitter[Blob]{emit: func(key string, populate func(target *Item[Blob]) error) {
			emissions = append(emissions, handoverEmission[O, Blob]{key: key, origin: origin, populate: populate})
		}}
		if err := task.extract(ctx, extractContext, origin, emitter); err != nil {
			return failTask(result, fmt.Errorf("task %q: extract %q: %w", task.name, origin.Name, err))
		}
	}
	emissions, err = applyHandoverEmissionDuplicatePolicy(emissions, opts.Context.DuplicateOutputs)
	if err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	targets := make([]Item[Blob], 0, len(emissions))
	for _, emission := range emissions {
		if emission.populate == nil {
			return failTask(result, fmt.Errorf("task %q: emitted target %q has nil populate function", task.name, emission.key))
		}
		subject := task.target.At(emission.key)
		target := subject.Snapshot()
		target.Key = emission.key
		target.Value.Key = emission.key
		if err := emission.populate(&target); err != nil {
			return failTask(result, fmt.Errorf("task %q: populate %q: %w", task.name, emission.key, err))
		}
		target.Key = emission.key
		target.Value.Key = emission.key
		target = normalizeBlobItem(target)
		subject.put(target)
		targets = append(targets, target)
	}
	if err := runProducedByteSink(ctx, opts, subjectMulti, targets, task.sink, &result); err != nil {
		return failTask(result, fmt.Errorf("task %q: %w", task.name, err))
	}
	return result, nil
}

func setByteSink(taskName string, configErr *error, sink *byteSink, next byteSink) {
	if *configErr != nil {
		return
	}
	if sink.kind != byteSinkUnknown {
		*configErr = fmt.Errorf("task %q already has a byte sink", taskName)
		return
	}
	*sink = next
}

func runProducedByteSink(ctx context.Context, opts RunOptions, kind subjectKind, items []Item[Blob], sink byteSink, result *TaskResult) error {
	if sink.kind == byteSinkUnknown {
		return nil
	}
	sinkContext := resolveLayoutContext(sink.lctx, opts.Context.Layout)
	sink.lctx = sinkContext
	plans, err := planByteSink(ctx, kind, items, sink, opts.Context.DuplicateOutputs)
	if err != nil {
		return err
	}
	writes, err := stageByteWrites(ctx, sinkContext, plans)
	if err != nil {
		return err
	}
	for _, write := range writes {
		entry := byteWriteResultEntry(write)
		if err := write.file.WriteBytes(write.data, sinkContext); err != nil {
			entry.Err = err
			result.recordByteWrite(sink.kind, entry)
			return fmt.Errorf("write %s: %w", write.file.Path(), err)
		}
		result.recordByteWrite(sink.kind, entry)
	}
	return nil
}
