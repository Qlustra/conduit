package mgmt_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/mgmt"
)

type documentValue struct {
	Title   string            `json:"title" yaml:"title" toml:"title"`
	Labels  map[string]string `json:"labels" yaml:"labels" toml:"labels"`
	Extra   any               `json:"extra,omitempty" yaml:"extra,omitempty" toml:"extra,omitempty"`
	Ignored string            `json:"-" yaml:"-" toml:"-"`
}

type documentTree struct {
	Root layout.Dir                      `layout:"."`
	Data formats.JSONFile[documentValue] `layout:"document.json" manage:"required"`
}

var errDocumentRule = errors.New("document title is invalid")

func documentRule(_ context.Context, value documentValue) error {
	if value.Title == "invalid" {
		return errDocumentRule
	}
	return nil
}

func documentFixture(t *testing.T, io layout.Context, options mgmt.DocumentOptions[documentValue]) (*documentTree, mgmt.Document[documentValue]) {
	t.Helper()
	root := t.TempDir()
	var tree documentTree
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tree.Data.Path(), []byte(`{"title":"disk","labels":{"initial":"kept"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.Data.Load(); err != nil {
		t.Fatal(err)
	}
	space, err := mgmt.BindSpace(tree.Root, &tree, mgmt.SpaceOptions{Context: io})
	if err != nil {
		t.Fatal(err)
	}
	document, err := mgmt.BindDocument(space, &tree.Data, options)
	if err != nil {
		t.Fatal(err)
	}
	return &tree, document
}

func TestDocumentReadKeepsStateAndReturnsIsolatedMemory(t *testing.T) {
	tree, document := documentFixture(t, layout.Context{}, mgmt.DocumentOptions[documentValue]{})
	original := documentValue{Title: "authored", Labels: map[string]string{"initial": "memory"}, Ignored: "preserve me"}
	tree.Data.Set(original)
	disk, err := document.Read(context.Background(), mgmt.ReadPolicy{})
	if err != nil || disk.Title != "disk" {
		t.Fatalf("disk read = %+v, %v", disk, err)
	}
	memory, err := document.Read(context.Background(), mgmt.ReadPolicy{Source: mgmt.Memory})
	if err != nil || memory.Title != "authored" {
		t.Fatalf("memory read = %+v, %v", memory, err)
	}
	memory.Labels["initial"] = "mutated result"
	assertDocumentState(t, &tree.Data, original, layout.MemoryDirty)
	if document.Binding().Path() != tree.Data.Path() || !document.Capabilities().Has(mgmt.OpRead) || !document.Capabilities().Has(mgmt.OpUpdate) {
		t.Fatalf("document binding/capabilities = %+v / %+v", document.Binding(), document.Capabilities())
	}
}

func TestDocumentUpdatePrecommitFailuresPreserveCacheAndDisk(t *testing.T) {
	for _, stage := range []string{"transform", "domain validation", "encoding", "cancellation", "missing", "malformed"} {
		t.Run(stage, func(t *testing.T) {
			tree, document := documentFixture(t, layout.Context{}, mgmt.DocumentOptions[documentValue]{Validate: documentRule})
			original := documentValue{Title: "authored", Labels: map[string]string{"initial": "memory"}, Ignored: "preserve me"}
			tree.Data.Set(original)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if stage == "missing" {
				if err := os.Remove(tree.Data.Path()); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "malformed" {
				if err := os.WriteFile(tree.Data.Path(), []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, beforeErr := os.ReadFile(tree.Data.Path())
			policy := mgmt.UpdatePolicy{Dirty: mgmt.ReplaceDirty}
			if stage != "missing" && stage != "malformed" {
				policy.Source = mgmt.Memory
			}
			out, err := document.Update(ctx, func(value *documentValue) error {
				value.Labels["initial"] = "candidate"
				switch stage {
				case "transform":
					return errors.New("transform failed")
				case "domain validation":
					value.Title = "invalid"
				case "encoding":
					value.Extra = make(chan struct{})
				case "cancellation":
					cancel()
				}
				return nil
			}, policy)
			if err == nil || out.Applied() {
				t.Fatalf("failed update = %+v, %v", out, err)
			}
			assertDocumentState(t, &tree.Data, original, layout.MemoryDirty)
			// Restoration keeps original codec-ignored fields and original map
			// identity, rather than restoring a lossy codec reconstruction.
			original.Labels["after"] = "same map"
			cached, _ := tree.Data.Get()
			if cached.Labels["after"] != "same map" {
				t.Fatal("precommit failure replaced the original cache identity")
			}
			after, afterErr := os.ReadFile(tree.Data.Path())
			if string(after) != string(before) || os.IsNotExist(beforeErr) != os.IsNotExist(afterErr) {
				t.Fatalf("failure changed disk from %q (%v) to %q (%v)", before, beforeErr, after, afterErr)
			}
		})
	}
}

func TestDocumentUpdateDirtyPolicyAndMemoryOnly(t *testing.T) {
	tree, document := documentFixture(t, layout.Context{}, mgmt.DocumentOptions[documentValue]{Validate: documentRule})
	tree.Data.Set(documentValue{Title: "authored", Labels: map[string]string{"memory": "yes"}})
	called := false
	if _, err := document.Update(context.Background(), func(value *documentValue) error {
		called = true
		return nil
	}, mgmt.UpdatePolicy{}); !errors.Is(err, mgmt.ErrDirty) || called {
		t.Fatalf("dirty update = %v; transform called = %v", err, called)
	}
	var escaped map[string]string
	out, err := document.Update(context.Background(), func(value *documentValue) error {
		if value.Title != "authored" {
			t.Fatalf("memory source used %+v", value)
		}
		value.Title = "edited"
		escaped = value.Labels
		return nil
	}, mgmt.UpdatePolicy{Source: mgmt.Memory, Effects: mgmt.MemoryOnly})
	if err != nil || out.Applied() {
		t.Fatalf("memory update = %+v, %v", out, err)
	}
	escaped["memory"] = "must not leak"
	assertDocumentState(t, &tree.Data, documentValue{Title: "edited", Labels: map[string]string{"memory": "yes"}}, layout.MemoryDirty)
	disk, err := tree.Data.Read()
	if err != nil || disk.Title != "disk" {
		t.Fatalf("memory update wrote disk: %+v, %v", disk, err)
	}

	out, err = document.Update(context.Background(), func(value *documentValue) error {
		if value.Title != "disk" {
			t.Fatalf("disk source used %+v", value)
		}
		value.Title = "persisted"
		return nil
	}, mgmt.UpdatePolicy{Dirty: mgmt.ReplaceDirty})
	if err != nil || !out.Applied() {
		t.Fatalf("persisted update = %+v, %v", out, err)
	}
	assertDocumentState(t, &tree.Data, documentValue{Title: "persisted", Labels: map[string]string{"initial": "kept"}}, layout.MemorySynced)
	if tree.Data.DiskState() != layout.DiskPresent {
		t.Fatalf("disk state = %v", tree.Data.DiskState())
	}
}

func TestDocumentUpdateOptimisticConflictPreservesPrecommitCache(t *testing.T) {
	tree, document := documentFixture(t, layout.Context{}, mgmt.DocumentOptions[documentValue]{})
	before, _ := tree.Data.Get()
	out, err := document.Update(context.Background(), func(value *documentValue) error {
		value.Title = "candidate"
		return os.WriteFile(tree.Data.Path(), []byte(`{"title":"external","labels":{}}`), 0600)
	}, mgmt.UpdatePolicy{CheckConflict: true})
	if !errors.Is(err, mgmt.ErrConflict) || out.Applied() {
		t.Fatalf("conflicting update = %+v, %v", out, err)
	}
	assertDocumentState(t, &tree.Data, before, layout.MemoryLoaded)
	disk, err := tree.Data.Read()
	if err != nil || disk.Title != "external" {
		t.Fatalf("external change overwritten: %+v, %v", disk, err)
	}
}

func TestDocumentWriteFailureRetainsCandidateDirty(t *testing.T) {
	badTemp := filepath.Join(t.TempDir(), "file-not-directory")
	if err := os.WriteFile(badTemp, []byte("blocker"), 0600); err != nil {
		t.Fatal(err)
	}
	tree, document := documentFixture(t, layout.Context{
		WritePolicy: layout.WriteAtomicReplace, TempFilePlacement: layout.TempFileDir, TempDir: layout.NewDir(badTemp),
	}, mgmt.DocumentOptions[documentValue]{})
	out, err := document.Update(context.Background(), func(value *documentValue) error {
		value.Title = "retry me"
		return nil
	}, mgmt.UpdatePolicy{})
	if err == nil || out.Applied() || !out.MayHaveApplied() || len(out.Effects) == 0 || out.Effects[len(out.Effects)-1].Phase != "write" {
		t.Fatalf("failed write = %+v, %v", out, err)
	}
	assertDocumentState(t, &tree.Data, documentValue{Title: "retry me", Labels: map[string]string{"initial": "kept"}}, layout.MemoryDirty)
	disk, err := tree.Data.Read()
	if err != nil || disk.Title != "disk" {
		t.Fatalf("failed atomic write changed disk: %+v, %v", disk, err)
	}
}

type uncertainDocumentNode struct {
	formats.JSONFile[documentValue]
	failure error
	partial bool
}

func (n *uncertainDocumentNode) PrepareWrite() (layout.PreparedWrite, error) {
	prepared, err := n.JSONFile.PrepareWrite()
	if err != nil {
		return nil, err
	}
	return uncertainDocumentWrite{PreparedWrite: prepared, failure: n.failure, partial: n.partial}, nil
}

type uncertainDocumentWrite struct {
	layout.PreparedWrite
	failure error
	partial bool
}

func (w uncertainDocumentWrite) Write(_ layout.Context) error {
	if w.partial {
		if err := os.WriteFile(w.Path(), []byte("{"), 0600); err != nil {
			return err
		}
	}
	return w.failure
}

func TestDocumentFailedWritesDistinguishUncertainEffectsFromStalePreparation(t *testing.T) {
	for _, partial := range []bool{true, false} {
		t.Run(map[bool]string{true: "partial write", false: "stale preparation"}[partial], func(t *testing.T) {
			var tree struct {
				Root layout.Dir            `layout:"."`
				Data uncertainDocumentNode `layout:"data.json"`
			}
			if err := layout.Compose(t.TempDir(), &tree); err != nil {
				t.Fatal(err)
			}
			tree.Data.partial = partial
			tree.Data.failure = layout.ErrPreparedWriteStale
			if partial {
				tree.Data.failure = errors.New("write interrupted after truncation")
			}
			if err := os.WriteFile(tree.Data.Path(), []byte(`{"title":"before","labels":{}}`), 0600); err != nil {
				t.Fatal(err)
			}
			space, err := mgmt.BindSpace(tree.Root, &tree, mgmt.SpaceOptions{})
			if err != nil {
				t.Fatal(err)
			}
			document, err := mgmt.BindDocument(space, &tree.Data, mgmt.DocumentOptions[documentValue]{})
			if err != nil {
				t.Fatal(err)
			}
			out, err := document.Update(context.Background(), func(value *documentValue) error {
				value.Title = "candidate"
				return nil
			}, mgmt.UpdatePolicy{})
			if !errors.Is(err, tree.Data.failure) || out.Applied() || out.MayHaveApplied() != partial {
				t.Fatalf("failed write reporting = %+v, %v", out, err)
			}
			if len(out.Effects) != 1 || out.Effects[0].MayHaveApplied != partial {
				t.Fatalf("effect uncertainty = %+v", out.Effects)
			}
			cached, _ := tree.Data.Get()
			if cached.Title != "candidate" || tree.Data.MemoryState() != layout.MemoryDirty {
				t.Fatalf("failed write lost candidate: %+v (%v)", cached, tree.Data.MemoryState())
			}
		})
	}
	known := mgmt.Outcome{Effects: []mgmt.Effect{{Applied: true}}}
	if !known.Applied() || !known.MayHaveApplied() {
		t.Fatal("known effects must also count as possibly applied")
	}
}

func TestDocumentUpdateHonorsAtomicReplacement(t *testing.T) {
	tree, document := documentFixture(t, layout.Context{
		WritePolicy: layout.WriteAtomicReplace, TempFilePlacement: layout.TempFileAdjacent,
	}, mgmt.DocumentOptions[documentValue]{})
	before, err := os.Stat(tree.Data.Path())
	if err != nil {
		t.Fatal(err)
	}
	out, err := document.Update(context.Background(), func(value *documentValue) error {
		value.Title = "atomic"
		return nil
	}, mgmt.UpdatePolicy{CheckConflict: true})
	if err != nil || !out.Applied() {
		t.Fatalf("atomic update = %+v, %v", out, err)
	}
	after, err := os.Stat(tree.Data.Path())
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("atomic replacement rewrote the original inode")
	}
	assertDocumentState(t, &tree.Data, documentValue{Title: "atomic", Labels: map[string]string{"initial": "kept"}}, layout.MemorySynced)
}

func TestDocumentDomainRulesCoverLoadSaveAndValidationSources(t *testing.T) {
	tree, document := documentFixture(t, layout.Context{}, mgmt.DocumentOptions[documentValue]{Validate: documentRule})
	original, _ := tree.Data.Get()
	if err := os.WriteFile(tree.Data.Path(), []byte(`{"title":"invalid","labels":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := document.Load(context.Background(), mgmt.LoadPolicy{}); !errors.Is(err, errDocumentRule) {
		t.Fatalf("load bypassed domain rule: %v", err)
	}
	assertDocumentState(t, &tree.Data, original, layout.MemoryLoaded)
	if diagnostics, err := document.Validate(context.Background(), mgmt.ValidationPolicy{Source: mgmt.Disk}); !errors.Is(err, errDocumentRule) || !diagnostics.HasErrors() {
		t.Fatalf("disk validation = %+v, %v", diagnostics, err)
	}
	if _, err := document.Validate(context.Background(), mgmt.ValidationPolicy{Source: mgmt.Memory}); err != nil {
		t.Fatalf("memory validation read invalid disk value: %v", err)
	}
	assertDocumentState(t, &tree.Data, original, layout.MemoryLoaded)

	tree.Data.Set(documentValue{Title: "invalid", Labels: map[string]string{}})
	if _, err := document.Save(context.Background(), mgmt.SavePolicy{}); !errors.Is(err, errDocumentRule) {
		t.Fatalf("save bypassed domain rule: %v", err)
	}
	if _, err := document.Load(context.Background(), mgmt.LoadPolicy{}); !errors.Is(err, mgmt.ErrDirty) {
		t.Fatalf("load discarded dirty content: %v", err)
	}
	if err := os.WriteFile(tree.Data.Path(), []byte(`{"title":"valid","labels":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := document.Load(context.Background(), mgmt.LoadPolicy{Dirty: mgmt.ReplaceDirty}); err != nil {
		t.Fatal(err)
	}
	assertDocumentState(t, &tree.Data, documentValue{Title: "valid", Labels: map[string]string{}}, layout.MemoryLoaded)

	// Validators cannot leak accidental map edits into the candidate or cache.
	space, err := mgmt.BindSpace(tree.Root, tree, mgmt.SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mutatingRule, err := mgmt.BindDocument(space, &tree.Data, mgmt.DocumentOptions[documentValue]{
		Validate: func(_ context.Context, value documentValue) error {
			value.Labels["leaked"] = "no"
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mutatingRule.Update(context.Background(), func(value *documentValue) error {
		value.Title = "clean candidate"
		return nil
	}, mgmt.UpdatePolicy{}); err != nil {
		t.Fatal(err)
	}
	assertDocumentState(t, &tree.Data, documentValue{Title: "clean candidate", Labels: map[string]string{}}, layout.MemorySynced)
}

func TestDocumentLoadMissingAndMalformedPreservesAuthoredState(t *testing.T) {
	for _, missing := range []bool{true, false} {
		tree, document := documentFixture(t, layout.Context{}, mgmt.DocumentOptions[documentValue]{})
		original := documentValue{Title: "authored", Labels: map[string]string{"initial": "kept"}, Ignored: "keep"}
		tree.Data.Set(original)
		if missing {
			if err := os.Remove(tree.Data.Path()); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(tree.Data.Path(), []byte("{"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := document.Load(context.Background(), mgmt.LoadPolicy{Dirty: mgmt.ReplaceDirty}); err == nil {
			t.Fatalf("accepted missing=%v input", missing)
		}
		assertDocumentState(t, &tree.Data, original, layout.MemoryDirty)
	}
}

type ruleDocumentNode struct {
	formats.JSONFile[documentValue]
}

func (n *ruleDocumentNode) Validate(options layout.ValidateOptions) error {
	if value, ok := n.Get(); ok && value.Title == "invalid" {
		return errors.New("wrapper validation failed")
	}
	return n.JSONFile.Validate(options)
}

func TestDocumentCollectsWrapperAndTypedDiagnostics(t *testing.T) {
	var tree struct {
		Root layout.Dir       `layout:"."`
		Data ruleDocumentNode `layout:"data.json"`
	}
	root := t.TempDir()
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	tree.Data.Set(documentValue{Title: "invalid", Labels: map[string]string{}})
	space, err := mgmt.BindSpace(tree.Root, &tree, mgmt.SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	document, err := mgmt.BindDocument(space, &tree.Data, mgmt.DocumentOptions[documentValue]{Validate: documentRule})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := document.Validate(context.Background(), mgmt.ValidationPolicy{Diagnostics: mgmt.CollectAll})
	if err == nil || len(diagnostics.Entries) != 2 || !errors.Is(err, errDocumentRule) {
		t.Fatalf("collected validation = %+v, %v", diagnostics, err)
	}
	if diagnostics, err := document.Validate(context.Background(), mgmt.ValidationPolicy{}); err == nil || len(diagnostics.Entries) != 1 {
		t.Fatalf("fail-fast validation = %+v, %v", diagnostics, err)
	}
}

func TestDocumentFormatAdaptersAreStructural(t *testing.T) {
	var tree struct {
		Root layout.Dir                      `layout:"."`
		JSON formats.JSONFile[documentValue] `layout:"data.json"`
		YAML formats.YAMLFile[documentValue] `layout:"data.yaml"`
		TOML formats.TOMLFile[documentValue] `layout:"data.toml"`
	}
	if err := layout.Compose(t.TempDir(), &tree); err != nil {
		t.Fatal(err)
	}
	space, err := mgmt.BindSpace(tree.Root, &tree, mgmt.SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range []mgmt.DocumentNode[documentValue]{&tree.JSON, &tree.YAML, &tree.TOML} {
		document, err := mgmt.BindDocument(space, node, mgmt.DocumentOptions[documentValue]{})
		if err != nil {
			t.Fatal(err)
		}
		node.Set(documentValue{Title: "typed", Labels: map[string]string{"format": "supported"}})
		if _, err := document.Save(context.Background(), mgmt.SavePolicy{Missing: mgmt.CreateMissing}); err != nil {
			t.Fatal(err)
		}
		value, err := document.Read(context.Background(), mgmt.ReadPolicy{})
		if err != nil || value.Title != "typed" || value.Labels["format"] != "supported" || node.MemoryState() != layout.MemorySynced {
			t.Fatalf("structural document = %+v, %v, state %v", value, err, node.MemoryState())
		}
	}
}

func assertDocumentState(t *testing.T, node *formats.JSONFile[documentValue], want documentValue, state layout.MemoryState) {
	t.Helper()
	got, ok := node.Get()
	if !ok || !reflect.DeepEqual(got, want) || node.MemoryState() != state {
		t.Fatalf("cached document = %+v (present %v, state %v), want %+v (state %v)", got, ok, node.MemoryState(), want, state)
	}
}
