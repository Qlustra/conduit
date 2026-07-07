package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/qlustra/conduit/layout"
)

// Operation Callbacks

// TransformFunc is the byte transform callback used by byte tasks.
type TransformFunc = layout.TransformFunc

// SplitFunc expands one byte item into zero or more byte items.
type SplitFunc func(ctx context.Context, lctx layout.Context, split Split, item Item[Blob]) error

// FilterFunc controls whether a byte item is retained.
type FilterFunc func(ctx context.Context, lctx layout.Context, filter Filter, item Item[Blob]) (bool, error)

// SortFunc orders two byte items.
type SortFunc func(a Item[Blob], b Item[Blob]) bool

// FileMapper maps one byte item to its destination file.
type FileMapper func(ctx context.Context, lctx layout.Context, item Item[Blob]) (layout.File, error)

type byteStepKind uint8

const (
	byteStepTransform byteStepKind = iota + 1
	byteStepSplit
	byteStepFilter
	byteStepSort
	byteStepConcat
)

type byteSinkKind uint8

const (
	byteSinkUnknown byteSinkKind = iota
	byteSinkWriteBack
	byteSinkToFile
	byteSinkToDir
	byteSinkToFiles
)

type byteStep struct {
	kind byteStepKind
	lctx layout.Context

	transform TransformFunc
	split     SplitFunc
	filter    FilterFunc
	sort      SortFunc

	concatOptions layout.ConcatOptions
}

type byteSink struct {
	kind byteSinkKind
	lctx layout.Context

	file        layout.File
	destination Destination
	mapper      FileMapper
}

type byteTask struct {
	mu    sync.RWMutex
	runMu sync.Mutex

	name  string
	kind  subjectKind
	items []Item[Blob]
	steps []byteStep
	sink  byteSink

	configErr error
}

type byteTaskSnapshot struct {
	name      string
	kind      subjectKind
	items     []Item[Blob]
	steps     []byteStep
	sink      byteSink
	configErr error
}

type byteTaskRunSnapshot struct {
	task  byteTaskSnapshot
	runMu *sync.Mutex
}

// Tasks

// ByteSingleTask is a single-subject byte task.
type ByteSingleTask struct{ task *byteTask }

// ByteMultiTask is a multi-subject byte task.
type ByteMultiTask struct{ task *byteTask }

func newByteSingleTask(name string, item Item[Blob]) *ByteSingleTask {
	return &ByteSingleTask{task: &byteTask{name: name, kind: subjectSingle, items: []Item[Blob]{item}}}
}

func newByteMultiTask(name string, items []Item[Blob]) *ByteMultiTask {
	return &ByteMultiTask{task: &byteTask{name: name, kind: subjectMulti, items: items}}
}

func (t *byteTask) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.name
}

// Name returns the task name.
func (t *ByteSingleTask) Name() string { return t.task.Name() }

// Name returns the task name.
func (t *ByteMultiTask) Name() string { return t.task.Name() }

// Run executes the byte task.
func (t *ByteSingleTask) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.task.run(ctx, opts)
}

func (t *ByteSingleTask) snapshotRunnable() Runnable { return t.task.snapshotRunnable() }

// Run executes the byte task.
func (t *ByteMultiTask) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	return t.task.run(ctx, opts)
}

func (t *ByteMultiTask) snapshotRunnable() Runnable { return t.task.snapshotRunnable() }

// Transform rewrites the single byte subject in memory using fn.
func (t *ByteSingleTask) Transform(lctx layout.Context, fn TransformFunc) *ByteSingleTask {
	t.task.addStep(byteStep{kind: byteStepTransform, lctx: lctx, transform: fn})
	return t
}

// Split expands the single byte subject into a multi-subject byte task.
func (t *ByteSingleTask) Split(lctx layout.Context, fn SplitFunc) *ByteMultiTask {
	t.task.addStep(byteStep{kind: byteStepSplit, lctx: lctx, split: fn})
	return &ByteMultiTask{task: t.task}
}

// WriteBack writes the final byte subject back to its source file.
func (t *ByteSingleTask) WriteBack(lctx layout.Context) *ByteSingleTask {
	t.task.setSink(byteSink{kind: byteSinkWriteBack, lctx: lctx})
	return t
}

// To writes the final single byte subject to dest.
func (t *ByteSingleTask) To(lctx layout.Context, dest layout.File) *ByteSingleTask {
	t.task.setSink(byteSink{kind: byteSinkToFile, lctx: lctx, file: dest})
	return t
}

// Transform rewrites each byte subject in memory using fn.
func (t *ByteMultiTask) Transform(lctx layout.Context, fn TransformFunc) *ByteMultiTask {
	t.task.addStep(byteStep{kind: byteStepTransform, lctx: lctx, transform: fn})
	return t
}

// Filter keeps only byte subjects for which fn returns true.
func (t *ByteMultiTask) Filter(lctx layout.Context, fn FilterFunc) *ByteMultiTask {
	t.task.addStep(byteStep{kind: byteStepFilter, lctx: lctx, filter: fn})
	return t
}

// Sort orders byte subjects using fn.
func (t *ByteMultiTask) Sort(fn SortFunc) *ByteMultiTask {
	t.task.addStep(byteStep{kind: byteStepSort, sort: fn})
	return t
}

// Concat reduces all byte subjects into one single byte subject.
func (t *ByteMultiTask) Concat(lctx layout.Context, opts layout.ConcatOptions) *ByteSingleTask {
	t.task.addStep(byteStep{kind: byteStepConcat, lctx: lctx, concatOptions: opts})
	return &ByteSingleTask{task: t.task}
}

// WriteBack writes each final byte subject back to its source file.
func (t *ByteMultiTask) WriteBack(lctx layout.Context) *ByteMultiTask {
	t.task.setSink(byteSink{kind: byteSinkWriteBack, lctx: lctx})
	return t
}

// ToDir writes final byte subjects under dest according to opt.
func (t *ByteMultiTask) ToDir(lctx layout.Context, dest layout.Dir, opt DestinationOption) *ByteMultiTask {
	t.task.setSink(byteSink{kind: byteSinkToDir, lctx: lctx, destination: newDestination(dest, opt)})
	return t
}

// ToFiles writes final byte subjects to files returned by mapper.
func (t *ByteMultiTask) ToFiles(lctx layout.Context, mapper FileMapper) *ByteMultiTask {
	t.task.setSink(byteSink{kind: byteSinkToFiles, lctx: lctx, mapper: mapper})
	return t
}

func (t *byteTask) addStep(step byteStep) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.steps = append(t.steps, step)
}

func (t *byteTask) setSink(sink byteSink) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.configErr != nil {
		return
	}
	if t.sink.kind != byteSinkUnknown {
		t.configErr = fmt.Errorf("task %q already has a byte sink", t.name)
		return
	}
	t.sink = sink
}

func (t *byteTask) run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	t.runMu.Lock()
	defer t.runMu.Unlock()

	return runByteTask(ctx, opts, t.snapshot())
}

func (t *byteTask) snapshot() byteTaskSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	steps := make([]byteStep, len(t.steps))
	copy(steps, t.steps)
	return byteTaskSnapshot{
		name:      t.name,
		kind:      t.kind,
		items:     cloneItems(t.items),
		steps:     steps,
		sink:      t.sink,
		configErr: t.configErr,
	}
}

func (t *byteTask) snapshotRunnable() Runnable {
	return byteTaskRunSnapshot{task: t.snapshot(), runMu: &t.runMu}
}

func (s byteTaskRunSnapshot) Name() string { return s.task.name }

func (s byteTaskRunSnapshot) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	return runByteTask(ctx, opts, s.task)
}

// Split

// Split is the operation-scoped helper passed to SplitFunc callbacks.
type Split interface {
	// Read returns the current item bytes, materializing file-backed data when
	// necessary.
	Read() ([]byte, error)

	// Emit appends item to the split output.
	Emit(item Item[Blob])

	// EmitBytes appends an in-memory byte blob named name.
	EmitBytes(name string, data []byte)

	// EmitString appends an in-memory string blob named name.
	EmitString(name string, data string)

	// EmitFile appends a file-backed byte item.
	EmitFile(file layout.File)

	// EmitBlob appends blob as an in-memory byte item.
	EmitBlob(blob Blob)
}

type byteSplitCollector struct {
	ctx   context.Context
	lctx  layout.Context
	item  *Item[Blob]
	items []Item[Blob]
}

func (s *byteSplitCollector) Read() ([]byte, error) {
	return materializeByteItem(s.ctx, s.lctx, s.item)
}

func (s *byteSplitCollector) Emit(item Item[Blob]) { s.items = append(s.items, item) }
func (s *byteSplitCollector) EmitBytes(name string, data []byte) {
	s.EmitBlob(Blob{Name: name, Data: data})
}
func (s *byteSplitCollector) EmitString(name string, data string) { s.EmitBytes(name, []byte(data)) }
func (s *byteSplitCollector) EmitFile(file layout.File)           { s.Emit(itemFromFile(file)) }
func (s *byteSplitCollector) EmitBlob(blob Blob)                  { s.Emit(itemFromBlob(blob)) }

// Filter

// Filter is the operation-scoped helper passed to FilterFunc callbacks.
type Filter interface {
	// Read returns the current item bytes, materializing file-backed data when
	// necessary.
	Read() ([]byte, error)
}

type byteFilterScope struct {
	ctx  context.Context
	lctx layout.Context
	item *Item[Blob]
}

func (f byteFilterScope) Read() ([]byte, error) { return materializeByteItem(f.ctx, f.lctx, f.item) }
