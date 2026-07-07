package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

type testWorkspace struct {
	Root     layout.Dir                                `layout:"."`
	Services layout.Slot[*testService]                 `layout:"services"`
	Servers  layout.Slot[*testServer]                  `layout:"servers"`
	Configs  layout.FileSlot[formats.JSONFile[config]] `layout:"configs"`
	Routes   layout.FileSlot[formats.JSONFile[route]]  `layout:"routes"`
	Bundles  layout.FileSlot[formats.JSONFile[bundle]] `layout:"bundles"`
}

type testService struct {
	Root   layout.Dir                  `layout:"."`
	Config formats.JSONFile[svcConfig] `layout:"config.json"`
}

type svcConfig struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type config struct {
	Name string `json:"name"`
}

type testServer struct {
	Root   layout.Dir                     `layout:"."`
	Config formats.JSONFile[serverConfig] `layout:"config.json"`
}

type serverConfig struct {
	Source  string `json:"source"`
	Enabled bool   `json:"enabled"`
}

type route struct {
	Service string `json:"service"`
	Path    string `json:"path"`
}

type bundle struct {
	Services []string `json:"services"`
}

func TestPipelineByteFileTransformToFile(t *testing.T) {
	base := t.TempDir()
	src := layout.NewFile(filepath.Join(base, "in.txt"))
	dst := layout.NewFile(filepath.Join(base, "out.txt"))
	writeTestFile(t, src, "alpha")

	p := New(TaskFromFile("upper", src).Transform(layout.DefaultContext, upperTransform).To(layout.DefaultContext, dst))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Tasks) != 1 || result.Tasks[0].Status != StatusRan {
		t.Fatalf("Run() result = %+v, want one ran task", result)
	}
	if len(result.Tasks[0].To.Items) != 1 {
		t.Fatalf("To result items = %+v, want one item", result.Tasks[0].To.Items)
	}
	if got := result.Tasks[0].To.Items[0]; got.File != dst.Path() || got.Bytes != len("ALPHA") {
		t.Fatalf("To result item = %+v, want file %q and %d bytes", got, dst.Path(), len("ALPHA"))
	}
	if got := readTestFile(t, dst); got != "ALPHA" {
		t.Fatalf("output = %q, want %q", got, "ALPHA")
	}
}

func TestPipelineRunRequiresContext(t *testing.T) {
	base := t.TempDir()
	src := layout.NewFile(filepath.Join(base, "in.txt"))
	dst := layout.NewFile(filepath.Join(base, "out.txt"))
	writeTestFile(t, src, "alpha")

	p := New(TaskFromFile("upper", src).Transform(layout.DefaultContext, upperTransform).To(layout.DefaultContext, dst))

	_, err := p.Run(context.Background(), RunOptions{})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil for missing pipeline context")
	}
	if !strings.Contains(err.Error(), "pipeline context layout is required") {
		t.Fatalf("Run() error = %v, want missing context error", err)
	}
	if dst.Exists() {
		t.Fatal("destination was written with missing pipeline context")
	}
}

func TestPipelineByteFilesWriteBack(t *testing.T) {
	base := t.TempDir()
	first := layout.NewFile(filepath.Join(base, "a.txt"))
	second := layout.NewFile(filepath.Join(base, "b.txt"))
	writeTestFile(t, first, "alpha")
	writeTestFile(t, second, "beta")

	p := New(TaskFromFiles("format", first, second).Transform(layout.DefaultContext, upperTransform).WriteBack(layout.DefaultContext))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Tasks[0].WriteBack.Items) != 2 {
		t.Fatalf("WriteBack result items = %+v, want two items", result.Tasks[0].WriteBack.Items)
	}
	if got := readTestFile(t, first); got != "ALPHA" {
		t.Fatalf("first = %q, want %q", got, "ALPHA")
	}
	if got := readTestFile(t, second); got != "BETA" {
		t.Fatalf("second = %q, want %q", got, "BETA")
	}
}

func TestPipelineByteConcatReducesToSingleOutput(t *testing.T) {
	base := t.TempDir()
	first := layout.NewFile(filepath.Join(base, "a.txt"))
	second := layout.NewFile(filepath.Join(base, "b.txt"))
	dst := layout.NewFile(filepath.Join(base, "bundle.txt"))
	writeTestFile(t, first, "alpha")
	writeTestFile(t, second, "beta")

	p := New(TaskFromFiles("bundle", first, second).
		Concat(layout.DefaultContext, layout.ConcatOptions{Header: []byte("// generated\n"), Separator: []byte("\n")}).
		To(layout.DefaultContext, dst))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := "// generated\nalpha\nbeta"
	if got := readTestFile(t, dst); got != want {
		t.Fatalf("bundle = %q, want %q", got, want)
	}
}

func TestPipelineBlobSplitFilterSortToDir(t *testing.T) {
	out := layout.NewDir(filepath.Join(t.TempDir(), "out"))

	p := New(TaskFromBlob("split", Blob{Name: "lines.txt", Data: []byte("b\nskip\na")}).
		Split(layout.DefaultContext, splitLines).
		Filter(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, filter Filter, item Item[Blob]) (bool, error) {
			data, err := filter.Read()
			if err != nil {
				return false, err
			}
			return !strings.Contains(string(data), "skip"), nil
		}).
		Sort(func(a Item[Blob], b Item[Blob]) bool { return a.Name > b.Name }).
		Transform(layout.DefaultContext, upperTransform).
		ToDir(layout.DefaultContext, out, Flatten()))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, out.File("002.txt")); got != "A" {
		t.Fatalf("002.txt = %q, want %q", got, "A")
	}
	if got := readTestFile(t, out.File("000.txt")); got != "B" {
		t.Fatalf("000.txt = %q, want %q", got, "B")
	}
	if out.File("001.txt").Exists() {
		t.Fatal("filtered item was written")
	}
}

func TestPipelineToDirCanPreserveItemPathStructure(t *testing.T) {
	out := layout.NewDir(filepath.Join(t.TempDir(), "out"))

	p := New(TaskFromBlob("preserve", Blob{Name: "ignored.txt", Path: "nested/result.txt", Data: []byte("payload")}).
		Split(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, split Split, item Item[Blob]) error {
			data, err := split.Read()
			if err != nil {
				return err
			}
			split.EmitBlob(Blob{Name: item.Name, Path: item.Path, Data: data})
			return nil
		}).
		ToDir(layout.DefaultContext, out, PreserveStructure()))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, out.File("nested/result.txt")); got != "payload" {
		t.Fatalf("preserved output = %q, want %q", got, "payload")
	}
}

func TestPipelineRejectsUnsafeItemPathsBeforeWriting(t *testing.T) {
	out := layout.NewDir(filepath.Join(t.TempDir(), "out"))
	writeTestFile(t, out.File("valid.txt"), "original")

	p := New(TaskFromBlobs("unsafe",
		Blob{Name: "valid.txt", Path: "valid.txt", Data: []byte("updated")},
		Blob{Name: "escape.txt", Path: "../escape.txt", Data: []byte("escape")},
	).ToDir(layout.DefaultContext, out, PreserveStructure()))

	_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil for unsafe item path")
	}
	if got := readTestFile(t, out.File("valid.txt")); got != "original" {
		t.Fatalf("valid output after failure = %q, want %q", got, "original")
	}
}

func TestPipelineMapperFailureDoesNotWriteAnyMappedDestinations(t *testing.T) {
	base := t.TempDir()
	out := layout.NewDir(filepath.Join(base, "out"))
	writeTestFile(t, out.File("a.txt.out"), "original")
	wantErr := errors.New("bad mapping")

	p := New(TaskFromBlobs("map-fail",
		Blob{Name: "a.txt", Data: []byte("alpha")},
		Blob{Name: "b.txt", Data: []byte("beta")},
	).Transform(layout.DefaultContext, upperTransform).
		ToFiles(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[Blob]) (layout.File, error) {
			if item.Name == "b.txt" {
				return layout.File{}, wantErr
			}
			return out.File(item.Name + ".out"), nil
		}))

	_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if got := readTestFile(t, out.File("a.txt.out")); got != "original" {
		t.Fatalf("mapped destination after failure = %q, want %q", got, "original")
	}
}

func TestPipelineDuplicateOutputPolicies(t *testing.T) {
	out := layout.NewDir(filepath.Join(t.TempDir(), "out"))
	failing := New(TaskFromBlobs("duplicate-fail",
		Blob{Name: "a/same.txt", Data: []byte("first")},
		Blob{Name: "b/same.txt", Data: []byte("second")},
	).ToDir(layout.DefaultContext, out, Flatten()))

	_, err := failing.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "duplicate output path") {
		t.Fatalf("Run() error = %v, want duplicate output error", err)
	}

	pctx := DefaultContext
	pctx.DuplicateOutputs = DuplicateOutputLastWins
	lastWins := New(TaskFromBlobs("duplicate-last-wins",
		Blob{Name: "a/same.txt", Data: []byte("first")},
		Blob{Name: "b/same.txt", Data: []byte("second")},
	).ToDir(layout.DefaultContext, out, Flatten()))

	if _, err := lastWins.Run(context.Background(), RunOptions{Context: pctx}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, out.File("same.txt")); got != "second" {
		t.Fatalf("output = %q, want %q", got, "second")
	}
}

func TestPipelineRejectsDuplicateByteSink(t *testing.T) {
	base := t.TempDir()
	src := layout.NewFile(filepath.Join(base, "in.txt"))
	first := layout.NewFile(filepath.Join(base, "first.txt"))
	second := layout.NewFile(filepath.Join(base, "second.txt"))
	writeTestFile(t, src, "alpha")

	p := New(TaskFromFile("duplicate-sink", src).
		To(layout.DefaultContext, first).
		To(layout.DefaultContext, second))

	_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "already has a byte sink") {
		t.Fatalf("Run() error = %v, want duplicate byte sink error", err)
	}
	if first.Exists() || second.Exists() {
		t.Fatal("duplicate sink task wrote output")
	}
}

func TestPipelineOperationContexts(t *testing.T) {
	out := layout.NewDir(filepath.Join(t.TempDir(), "out"))
	stepCtx := layout.DefaultContext
	stepCtx.SyncPolicy = layout.SyncIfDirty
	sinkCtx := layout.DefaultContext
	sinkCtx.SyncPolicy = layout.SyncIfUnsynced
	var sawStepContext bool
	var sawSinkContext bool

	p := New(TaskFromBlobs("contexts",
		Blob{Name: "a.txt", Data: []byte("alpha")},
		Blob{Name: "b.txt", Data: []byte("beta")},
	).
		Filter(stepCtx, func(ctx context.Context, lctx layout.Context, filter Filter, item Item[Blob]) (bool, error) {
			if lctx.SyncPolicy == layout.SyncIfDirty {
				sawStepContext = true
			}
			return true, nil
		}).
		ToFiles(sinkCtx, func(ctx context.Context, lctx layout.Context, item Item[Blob]) (layout.File, error) {
			if lctx.SyncPolicy == layout.SyncIfUnsynced {
				sawSinkContext = true
			}
			return out.File(item.Name), nil
		}))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !sawStepContext {
		t.Fatal("filter callback did not receive operation layout context")
	}
	if !sawSinkContext {
		t.Fatal("sink mapper did not receive operation layout context")
	}
}

func TestPipelineByteSinkUsesOperationWritePolicy(t *testing.T) {
	dst := layout.NewFile(filepath.Join(t.TempDir(), "out.txt"))
	writeTestFile(t, dst, "original")
	atomicCtx := layout.DefaultContext
	atomicCtx.WritePolicy = layout.WriteAtomicReplace
	atomicCtx.TempFilePlacement = layout.TempFileDir

	p := New(TaskFromBlob("atomic", Blob{Name: "in.txt", Data: []byte("new")}).To(atomicCtx, dst))
	_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "TempFileDir") {
		t.Fatalf("Run() error = %v, want TempFileDir error", err)
	}
	if got := readTestFile(t, dst); got != "original" {
		t.Fatalf("output after failed atomic write = %q, want original", got)
	}
}

func TestPipelineRunSnapshotsTaskList(t *testing.T) {
	first := &blockingRunnable{name: "first", started: make(chan struct{}), release: make(chan struct{})}
	second := &blockingRunnable{name: "second"}
	p := New(first)

	type runOutcome struct {
		result Result
		err    error
	}
	done := make(chan runOutcome, 1)
	go func() {
		result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
		done <- runOutcome{result: result, err: err}
	}()

	<-first.started
	p.Add(second)
	close(first.release)

	outcome := <-done
	if outcome.err != nil {
		t.Fatalf("Run() error = %v", outcome.err)
	}
	if len(outcome.result.Tasks) != 1 || outcome.result.Tasks[0].Name != "first" {
		t.Fatalf("first run tasks = %+v, want only first", outcome.result.Tasks)
	}

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if len(result.Tasks) != 2 || result.Tasks[0].Name != "first" || result.Tasks[1].Name != "second" {
		t.Fatalf("second run tasks = %+v, want first then second", result.Tasks)
	}
}

func TestByteTaskRunSnapshotsBuilderMutations(t *testing.T) {
	base := t.TempDir()
	dst := layout.NewFile(filepath.Join(base, "out.txt"))
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	task := TaskFromBlob("snapshot", Blob{Name: "in.txt", Data: []byte("alpha")}).
		Transform(layout.DefaultContext, func(dst io.Writer, src io.Reader) error {
			once.Do(func() { close(started) })
			<-release
			return upperTransform(dst, src)
		}).
		To(layout.DefaultContext, dst)

	done := make(chan error, 1)
	go func() {
		_, err := task.Run(context.Background(), RunOptions{Context: DefaultContext})
		done <- err
	}()

	<-started
	task.Transform(layout.DefaultContext, func(dst io.Writer, src io.Reader) error {
		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		_, err = dst.Write(append(data, '!'))
		return err
	})
	close(release)

	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, dst); got != "ALPHA" {
		t.Fatalf("first output = %q, want snapshot without later transform", got)
	}

	if _, err := task.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if got := readTestFile(t, dst); got != "ALPHA!" {
		t.Fatalf("second output = %q, want later transform applied", got)
	}
}

func TestPipelineRunSnapshotsBuiltInTaskDefinitionsAtStart(t *testing.T) {
	base := t.TempDir()
	dst := layout.NewFile(filepath.Join(base, "out.txt"))
	first := &blockingRunnable{name: "first", started: make(chan struct{}), release: make(chan struct{})}
	second := TaskFromBlob("second", Blob{Name: "in.txt", Data: []byte("alpha")}).
		Transform(layout.DefaultContext, upperTransform).
		To(layout.DefaultContext, dst)
	p := New(first, second)

	done := make(chan error, 1)
	go func() {
		_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
		done <- err
	}()

	<-first.started
	second.Transform(layout.DefaultContext, func(dst io.Writer, src io.Reader) error {
		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		_, err = dst.Write(append(data, '!'))
		return err
	})
	close(first.release)

	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, dst); got != "ALPHA" {
		t.Fatalf("first output = %q, want pipeline-start task snapshot", got)
	}

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if got := readTestFile(t, dst); got != "ALPHA!" {
		t.Fatalf("second output = %q, want later transform applied", got)
	}
}

func TestByteTaskRunSerializesDirectRuns(t *testing.T) {
	dst := layout.NewFile(filepath.Join(t.TempDir(), "out.txt"))
	var mu sync.Mutex
	active := 0
	maxActive := 0
	task := TaskFromBlob("serialized", Blob{Name: "in.txt", Data: []byte("alpha")}).
		Transform(layout.DefaultContext, func(dst io.Writer, src io.Reader) error {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)
			err := upperTransform(dst, src)

			mu.Lock()
			active--
			mu.Unlock()
			return err
		}).
		To(layout.DefaultContext, dst)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := task.Run(context.Background(), RunOptions{Context: DefaultContext})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
	if maxActive != 1 {
		t.Fatalf("max active runs = %d, want serialized direct task runs", maxActive)
	}
}

func TestTypedSlotEntriesProcessAndSyncDeep(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	api, err := ws.Services.Add("api", layout.DefaultContext)
	if err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	worker, err := ws.Services.Add("worker", layout.DefaultContext)
	if err != nil {
		t.Fatalf("Services.Add(worker) error = %v", err)
	}
	api.Config.Set(svcConfig{Name: "api", Enabled: false})
	worker.Config.Set(svcConfig{Name: "worker", Enabled: false})

	syncReport := &layout.Report{}
	syncCtx := layout.DefaultContext
	syncCtx.Reporter = syncReport
	p := New(TaskFromSlotEntries("enable", &ws.Services).
		Filter(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[*testService]) (bool, error) {
			return item.Key == "api", nil
		}).
		Process(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[*testService]) (*testService, error) {
			cfg := item.Value.Config.MustGet()
			cfg.Enabled = true
			item.Value.Config.Set(cfg)
			return item.Value, nil
		}).
		SyncDeep(syncCtx))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Tasks[0].SyncDeep.Items) != 1 {
		t.Fatalf("SyncDeep result items = %+v, want one item", result.Tasks[0].SyncDeep.Items)
	}
	if got := result.Tasks[0].SyncDeep.Items[0]; got.Result == 0 || len(got.Entries) == 0 || got.Err != nil {
		t.Fatalf("SyncDeep result item = %+v, want result and report entries", got)
	}
	if len(syncReport.Entries()) == 0 {
		t.Fatal("operation reporter did not receive SyncDeep entries")
	}
	if got := readTestFile(t, api.Config.File); !strings.Contains(got, `"enabled": true`) {
		t.Fatalf("api config = %s, want enabled true", got)
	}
	if got := readMaybeTestFile(t, worker.Config.File); strings.Contains(got, `"enabled": true`) {
		t.Fatalf("worker config = %s, want not synced", got)
	}
}

func TestPipelineRejectsDuplicateTypedDeepOperation(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}

	p := New(TaskFromSlotEntries("duplicate-deep", &ws.Services).
		SyncDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext))

	_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "already has SyncDeep") {
		t.Fatalf("Run() error = %v, want duplicate SyncDeep error", err)
	}
}

func TestTypedFileSlotEntriesSnapshotAndMetadata(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	alpha, err := ws.Configs.Add("alpha.json", layout.DefaultContext)
	if err != nil {
		t.Fatalf("Configs.Add(alpha) error = %v", err)
	}
	alpha.Set(config{Name: "old"})

	task := TaskFromFileSlotEntries("configs", &ws.Configs)
	if _, err := ws.Configs.Add("beta.json", layout.DefaultContext); err != nil {
		t.Fatalf("Configs.Add(beta) error = %v", err)
	}
	task.Process(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[formats.JSONFile[config]]) (formats.JSONFile[config], error) {
		if item.Key != "alpha.json" || item.Name != "alpha.json" || !hasFile(item.File) {
			t.Fatalf("item metadata = key %q name %q file %q", item.Key, item.Name, item.File.Path())
		}
		item.Value.Set(config{Name: "new"})
		return item.Value, nil
	}).SyncDeep(layout.DefaultContext)

	if _, err := New(task).Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, alpha.File); !strings.Contains(got, `"name": "new"`) {
		t.Fatalf("alpha config = %s, want new", got)
	}
	if got := readMaybeTestFile(t, ws.Configs.MustAt("beta.json").File); strings.Contains(got, `"name": "new"`) {
		t.Fatalf("beta config = %s, want not processed", got)
	}
}

func TestTypedSplitAndConcat(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}

	var got svcConfig
	p := New(TaskFromSlotEntries("split-concat", &ws.Services).
		Split(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, split TypedSplit[*testService], item Item[*testService]) error {
			split.Emit(item)
			split.Emit(item)
			return nil
		}).
		Concat(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, items []Item[*testService]) (*testService, error) {
			if len(items) != 2 {
				t.Fatalf("len(items) = %d, want 2", len(items))
			}
			items[0].Value.Config.Set(svcConfig{Name: fmt.Sprintf("count-%d", len(items))})
			return items[0].Value, nil
		}).
		Process(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[*testService]) (*testService, error) {
			got = item.Value.Config.MustGet()
			return item.Value, nil
		}).
		DefaultDeep())

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Name != "count-2" {
		t.Fatalf("concat result name = %q, want count-2", got.Name)
	}
}

func TestBridgePopulatesTargetSlotEntries(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	api, err := ws.Services.Add("api", layout.DefaultContext)
	if err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	api.Config.Set(svcConfig{Name: "api", Enabled: false})

	p := New(
		TaskFromSlotEntries("enable", &ws.Services).
			Process(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[*testService]) (*testService, error) {
				cfg := item.Value.Config.MustGet()
				cfg.Enabled = true
				item.Value.Config.Set(cfg)
				return item.Value, nil
			}).
			DefaultDeep(),
		Bridge("configure", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)).
			Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[*testServer]) error {
				cfg := origin.Value.Config.MustGet()
				target.Value.Config.Set(serverConfig{Source: cfg.Name, Enabled: cfg.Enabled})
				return nil
			}).
			EnsureDeep(layout.DefaultContext).
			SyncDeep(layout.DefaultContext),
	)

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.Tasks[1].Handover; got.Kind != HandoverBridge || len(got.Items) != 1 {
		t.Fatalf("bridge handover = %+v, want one bridge item", got)
	}
	server := ws.Servers.MustAt("api")
	if got := readTestFile(t, server.Config.File); !strings.Contains(got, `"source": "api"`) || !strings.Contains(got, `"enabled": true`) {
		t.Fatalf("server config = %s, want bridged enabled api", got)
	}
}

func TestCompileBuildsSingleFileSlotEntry(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	if _, err := ws.Services.Add("worker", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(worker) error = %v", err)
	}

	p := New(Compile("bundle", SlotEntries(&ws.Services), FileSlotEntry(&ws.Bundles, "services.json")).
		Build(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origins []Item[*testService], target *Item[formats.JSONFile[bundle]]) error {
			out := bundle{Services: make([]string, 0, len(origins))}
			for _, origin := range origins {
				out.Services = append(out.Services, origin.Key)
			}
			target.Value.Set(out)
			return nil
		}).
		EnsureDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.Tasks[0].Handover; got.Kind != HandoverCompile || len(got.Items) != 1 || len(got.Items[0].OriginKeys) != 2 {
		t.Fatalf("compile handover = %+v, want one compile item with two origins", got)
	}
	if got := readTestFile(t, ws.Bundles.MustAt("services.json").File); !strings.Contains(got, `"api"`) || !strings.Contains(got, `"worker"`) {
		t.Fatalf("bundle = %s, want api and worker", got)
	}
}

func TestExpandExtractsFileSlotEntries(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}

	p := New(Expand("routes", SlotEntries(&ws.Services), FileSlotEntries(&ws.Routes)).
		Extract(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], emit EntryEmitter[formats.JSONFile[route]]) error {
			emit.Emit(origin.Key+"-root.json", func(target *Item[formats.JSONFile[route]]) error {
				target.Value.Set(route{Service: origin.Key, Path: "/"})
				return nil
			})
			emit.Emit(origin.Key+"-health.json", func(target *Item[formats.JSONFile[route]]) error {
				target.Value.Set(route{Service: origin.Key, Path: "/health"})
				return nil
			})
			return nil
		}).
		EnsureDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.Tasks[0].Handover; got.Kind != HandoverExpand || len(got.Items) != 2 {
		t.Fatalf("expand handover = %+v, want two expanded items", got)
	}
	if got := readTestFile(t, ws.Routes.MustAt("api-root.json").File); !strings.Contains(got, `"path": "/"`) {
		t.Fatalf("root route = %s, want root path", got)
	}
	if got := readTestFile(t, ws.Routes.MustAt("api-health.json").File); !strings.Contains(got, `"path": "/health"`) {
		t.Fatalf("health route = %s, want health path", got)
	}
}

func TestCompileToFileSharesBlobSubjectWithLaterByteTask(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	for _, name := range []string{"api", "worker"} {
		if _, err := ws.Services.Add(name, layout.DefaultContext); err != nil {
			t.Fatalf("Services.Add(%s) error = %v", name, err)
		}
	}

	bundle := BlobSubjectFromBlob(Blob{Key: "bundle", Name: "bundle.txt", Path: "bundle.txt"})
	out := layout.NewFile(filepath.Join(ws.Root.Path(), "bundle.upper.txt"))

	p := New(
		CompileToFile("bundle", SlotEntries(&ws.Services), bundle).
			Build(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origins []Item[*testService], target *Item[Blob]) error {
				names := make([]string, 0, len(origins))
				for _, origin := range origins {
					names = append(names, origin.Key)
				}
				target.Data = []byte(strings.Join(names, ","))
				return nil
			}),
		TaskFromBlobSubject("bundle-upper", bundle).
			Transform(layout.DefaultContext, upperTransform).
			To(layout.DefaultContext, out),
	)

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := string(bundle.Snapshot().Data); got != "api,worker" {
		t.Fatalf("bundle subject = %q, want %q", got, "api,worker")
	}
	if got := readTestFile(t, out); got != "API,WORKER" {
		t.Fatalf("upper bundle = %q, want %q", got, "API,WORKER")
	}
}

func TestCompileToFileToTargetWritesAttachedFile(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	for _, name := range []string{"api", "worker"} {
		if _, err := ws.Services.Add(name, layout.DefaultContext); err != nil {
			t.Fatalf("Services.Add(%s) error = %v", name, err)
		}
	}

	targetFile := layout.NewFile(filepath.Join(ws.Root.Path(), "artifacts", "services.txt"))
	target := BlobSubjectForFile(targetFile)
	p := New(CompileToFile("bundle", SlotEntries(&ws.Services), target).
		Build(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origins []Item[*testService], target *Item[Blob]) error {
			target.Data = []byte(fmt.Sprintf("services=%d", len(origins)))
			return nil
		}).
		ToTarget(layout.DefaultContext))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.Tasks[0].ToTarget.Items; len(got) != 1 || got[0].File != targetFile.Path() {
		t.Fatalf("ToTarget result = %+v, want one write to %q", got, targetFile.Path())
	}
	if got := readTestFile(t, targetFile); got != "services=2" {
		t.Fatalf("target file = %q, want %q", got, "services=2")
	}
}

func TestBridgeToFilesToTargetsWritesOnlyProducedSubjects(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	for _, name := range []string{"api", "worker"} {
		if _, err := ws.Services.Add(name, layout.DefaultContext); err != nil {
			t.Fatalf("Services.Add(%s) error = %v", name, err)
		}
	}

	targets := BlobSubjects()
	outDir := layout.NewDir(filepath.Join(ws.Root.Path(), "out"))
	p := New(BridgeToFiles("bridge", SlotEntries(&ws.Services), targets).
		Filter(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[*testService]) (bool, error) {
			return item.Key == "api", nil
		}).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[Blob]) error {
			target.File = outDir.File(origin.Key + ".txt")
			target.Path = origin.Key + ".txt"
			target.Data = []byte("service=" + origin.Key)
			return nil
		}).
		ToTargets(layout.DefaultContext))

	result, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.Tasks[0].ToTargets.Items; len(got) != 1 || got[0].File != outDir.File("api.txt").Path() {
		t.Fatalf("ToTargets result = %+v, want one api.txt write", got)
	}
	if got := readTestFile(t, outDir.File("api.txt")); got != "service=api" {
		t.Fatalf("api output = %q, want %q", got, "service=api")
	}
	if got := readMaybeTestFile(t, outDir.File("worker.txt")); got != "" {
		t.Fatalf("worker output = %q, want empty", got)
	}
	if _, ok := targets.Get("worker"); ok {
		t.Fatal("worker target subject was created despite filtering")
	}
}

func TestExpandToFilesSharesBlobSubjectSetWithLaterByteTask(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}

	routes := BlobSubjects()
	out := layout.NewDir(filepath.Join(ws.Root.Path(), "rendered"))
	p := New(
		ExpandToFiles("routes", SlotEntries(&ws.Services), routes).
			Extract(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], emit EntryEmitter[Blob]) error {
				emit.Emit(origin.Key+"/root.txt", func(target *Item[Blob]) error {
					target.Path = origin.Key + "/root.txt"
					target.Data = []byte("root:" + origin.Key)
					return nil
				})
				emit.Emit(origin.Key+"/health.txt", func(target *Item[Blob]) error {
					target.Path = origin.Key + "/health.txt"
					target.Data = []byte("health:" + origin.Key)
					return nil
				})
				return nil
			}),
		TaskFromBlobSubjectSet("upper-routes", routes).
			Transform(layout.DefaultContext, upperTransform).
			ToDir(layout.DefaultContext, out, PreserveStructure()),
	)

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := routes.Keys(); strings.Join(got, ",") != "api/health.txt,api/root.txt" {
		t.Fatalf("route keys = %v, want [api/health.txt api/root.txt]", got)
	}
	if got := readTestFile(t, out.File("api/root.txt")); got != "ROOT:API" {
		t.Fatalf("root output = %q, want %q", got, "ROOT:API")
	}
	if got := readTestFile(t, out.File("api/health.txt")); got != "HEALTH:API" {
		t.Fatalf("health output = %q, want %q", got, "HEALTH:API")
	}
}

func TestBridgeDuplicateTargetKeyPolicies(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	if _, err := ws.Services.Add("worker", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(worker) error = %v", err)
	}

	populateCalls := 0
	failing := New(Bridge("duplicate-bridge", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)).
		Rekey(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService]) (string, error) {
			return "same", nil
		}).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[*testServer]) error {
			populateCalls++
			return nil
		}))

	_, err := failing.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "duplicate target key") {
		t.Fatalf("Run() error = %v, want duplicate target key error", err)
	}
	if populateCalls != 0 {
		t.Fatalf("populate calls = %d, want 0 before duplicate failure", populateCalls)
	}

	pctx := DefaultContext
	pctx.DuplicateOutputs = DuplicateOutputLastWins
	lastWins := New(Bridge("last-wins-bridge", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)).
		Rekey(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService]) (string, error) {
			return "same", nil
		}).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[*testServer]) error {
			target.Value.Config.Set(serverConfig{Source: origin.Key})
			return nil
		}).
		EnsureDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext))

	if _, err := lastWins.Run(context.Background(), RunOptions{Context: pctx}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, ws.Servers.MustAt("same").Config.File); !strings.Contains(got, `"source": "worker"`) {
		t.Fatalf("last-wins server config = %s, want worker", got)
	}
}

func TestBridgeToFilesDuplicateTargetKeyPolicies(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	for _, name := range []string{"api", "worker"} {
		if _, err := ws.Services.Add(name, layout.DefaultContext); err != nil {
			t.Fatalf("Services.Add(%s) error = %v", name, err)
		}
	}

	targets := BlobSubjects()
	populateCalls := 0
	failing := New(BridgeToFiles("duplicate-bridge-files", SlotEntries(&ws.Services), targets).
		Rekey(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService]) (string, error) {
			return "same", nil
		}).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[Blob]) error {
			populateCalls++
			return nil
		}))

	_, err := failing.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "duplicate target key") {
		t.Fatalf("Run() error = %v, want duplicate target key error", err)
	}
	if populateCalls != 0 {
		t.Fatalf("populate calls = %d, want 0 before duplicate failure", populateCalls)
	}

	pctx := DefaultContext
	pctx.DuplicateOutputs = DuplicateOutputLastWins
	lastWinsTargets := BlobSubjects()
	lastWins := New(BridgeToFiles("last-wins-bridge-files", SlotEntries(&ws.Services), lastWinsTargets).
		Rekey(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService]) (string, error) {
			return "same", nil
		}).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[Blob]) error {
			target.Data = []byte(origin.Key)
			return nil
		}))

	if _, err := lastWins.Run(context.Background(), RunOptions{Context: pctx}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	subject, ok := lastWinsTargets.Get("same")
	if !ok {
		t.Fatal("missing last-wins target subject")
	}
	if got := string(subject.Snapshot().Data); got != "worker" {
		t.Fatalf("last-wins subject = %q, want %q", got, "worker")
	}
}

func TestExpandDuplicateTargetKeyPolicies(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	if _, err := ws.Services.Add("worker", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(worker) error = %v", err)
	}

	populateCalls := 0
	failing := New(Expand("duplicate-expand", SlotEntries(&ws.Services), FileSlotEntries(&ws.Routes)).
		Extract(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], emit EntryEmitter[formats.JSONFile[route]]) error {
			emit.Emit("same.json", func(target *Item[formats.JSONFile[route]]) error {
				populateCalls++
				return nil
			})
			return nil
		}))

	_, err := failing.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "duplicate target key") {
		t.Fatalf("Run() error = %v, want duplicate target key error", err)
	}
	if populateCalls != 0 {
		t.Fatalf("populate calls = %d, want 0 before duplicate failure", populateCalls)
	}

	pctx := DefaultContext
	pctx.DuplicateOutputs = DuplicateOutputLastWins
	lastWins := New(Expand("last-wins-expand", SlotEntries(&ws.Services), FileSlotEntries(&ws.Routes)).
		Extract(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], emit EntryEmitter[formats.JSONFile[route]]) error {
			emit.Emit("same.json", func(target *Item[formats.JSONFile[route]]) error {
				target.Value.Set(route{Service: origin.Key, Path: "/" + origin.Key})
				return nil
			})
			return nil
		}).
		EnsureDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext))

	if _, err := lastWins.Run(context.Background(), RunOptions{Context: pctx}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := readTestFile(t, ws.Routes.MustAt("same.json").File); !strings.Contains(got, `"service": "worker"`) {
		t.Fatalf("last-wins route = %s, want worker", got)
	}
}

func TestCompileFilterAndSortOrdering(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	for _, name := range []string{"api", "worker", "admin"} {
		if _, err := ws.Services.Add(name, layout.DefaultContext); err != nil {
			t.Fatalf("Services.Add(%s) error = %v", name, err)
		}
	}

	var gotOrder []string
	p := New(Compile("ordered", SlotEntries(&ws.Services), FileSlotEntry(&ws.Bundles, "ordered.json")).
		Filter(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item Item[*testService]) (bool, error) {
			return item.Key != "worker", nil
		}).
		Sort(func(a Item[*testService], b Item[*testService]) bool { return a.Key > b.Key }).
		Build(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origins []Item[*testService], target *Item[formats.JSONFile[bundle]]) error {
			for _, origin := range origins {
				gotOrder = append(gotOrder, origin.Key)
			}
			target.Value.Set(bundle{Services: gotOrder})
			return nil
		}))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := strings.Join(gotOrder, ","); got != "api,admin" {
		t.Fatalf("build origin order = %q, want api,admin", got)
	}
}

func TestBridgeRekeyReceivesOperationLayoutContext(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	rekeyCtx := layout.DefaultContext
	rekeyCtx.SyncPolicy = layout.SyncIfDirty
	var sawRekeyContext bool

	p := New(Bridge("rekey-context", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)).
		Rekey(rekeyCtx, func(ctx context.Context, lctx layout.Context, origin Item[*testService]) (string, error) {
			if lctx.SyncPolicy == layout.SyncIfDirty {
				sawRekeyContext = true
			}
			return origin.Key + "-server", nil
		}).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[*testServer]) error {
			return nil
		}))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !sawRekeyContext {
		t.Fatal("rekey callback did not receive operation layout context")
	}
}

func TestHandoverTasksRequireExecutionVerb(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	tests := []struct {
		name string
		task Runnable
		want string
	}{
		{name: "bridge", task: Bridge("missing-populate", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)), want: "no Populate"},
		{name: "compile", task: Compile("missing-build", SlotEntries(&ws.Services), FileSlotEntry(&ws.Bundles, "bundle.json")), want: "no Build"},
		{name: "expand", task: Expand("missing-extract", SlotEntries(&ws.Services), FileSlotEntries(&ws.Routes)), want: "no Extract"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.task).Run(context.Background(), RunOptions{Context: DefaultContext})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Run() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestHandoverRejectsDuplicateTypedDeepOperation(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if _, err := ws.Services.Add("api", layout.DefaultContext); err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}

	p := New(Bridge("duplicate-handover-deep", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[*testServer]) error {
			return nil
		}).
		SyncDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext))

	_, err := p.Run(context.Background(), RunOptions{Context: DefaultContext})
	if err == nil || !strings.Contains(err.Error(), "already has SyncDeep") {
		t.Fatalf("Run() error = %v, want duplicate SyncDeep error", err)
	}
}

func TestHandoverWithoutDeepOpsUpdatesCacheOnly(t *testing.T) {
	var ws testWorkspace
	if err := layout.Compose(t.TempDir(), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	api, err := ws.Services.Add("api", layout.DefaultContext)
	if err != nil {
		t.Fatalf("Services.Add(api) error = %v", err)
	}
	api.Config.Set(svcConfig{Name: "api", Enabled: true})

	p := New(Bridge("cache-only", SlotEntries(&ws.Services), SlotEntries(&ws.Servers)).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin Item[*testService], target *Item[*testServer]) error {
			cfg := origin.Value.Config.MustGet()
			target.Value.Config.Set(serverConfig{Source: cfg.Name, Enabled: cfg.Enabled})
			return nil
		}))

	if _, err := p.Run(context.Background(), RunOptions{Context: DefaultContext}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	server, ok := ws.Servers.Get("api")
	if !ok {
		t.Fatal("target server was not cached")
	}
	if got := server.Config.MustGet(); got.Source != "api" || !got.Enabled {
		t.Fatalf("cached server config = %+v, want bridged api config", got)
	}
	if got := readMaybeTestFile(t, server.Config.File); got != "" {
		t.Fatalf("server config was written to disk without deep ops: %s", got)
	}
}

type blockingRunnable struct {
	name    string
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *blockingRunnable) Name() string { return r.name }

func (r *blockingRunnable) Run(ctx context.Context, opts RunOptions) (TaskResult, error) {
	if r.started != nil {
		r.once.Do(func() { close(r.started) })
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return failTask(TaskResult{Name: r.name}, ctx.Err())
		}
	}
	return TaskResult{Name: r.name, Status: StatusRan}, nil
}

func splitLines(ctx context.Context, lctx layout.Context, split Split, item Item[Blob]) error {
	data, err := split.Read()
	if err != nil {
		return err
	}
	for i, line := range bytes.Split(data, []byte("\n")) {
		split.EmitBytes(fmt.Sprintf("%03d.txt", i), line)
	}
	return nil
}

func upperTransform(dst io.Writer, src io.Reader) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	_, err = dst.Write([]byte(strings.ToUpper(string(data))))
	return err
}

func writeTestFile(t *testing.T, file layout.File, content string) {
	t.Helper()
	if err := file.WriteBytes([]byte(content), layout.DefaultContext); err != nil {
		t.Fatalf("WriteBytes(%s) error = %v", file.Path(), err)
	}
}

func readTestFile(t *testing.T, file layout.File) string {
	t.Helper()
	data, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", file.Path(), err)
	}
	return string(data)
}

func readMaybeTestFile(t *testing.T, file layout.File) string {
	t.Helper()
	data, err := os.ReadFile(file.Path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ""
		}
		t.Fatalf("os.ReadFile(%s) error = %v", file.Path(), err)
	}
	return string(data)
}
