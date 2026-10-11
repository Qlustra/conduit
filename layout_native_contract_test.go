package conduit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/sail/subject"
)

type nativeContractValue struct {
	Name string `json:"name"`
}

type nativeContractFile struct {
	layout.File
	ensures int
}

func (f *nativeContractFile) Ensure(ctx layout.Context) error {
	f.ensures++
	return f.File.Ensure(ctx)
}

type nativeContractDirectoryBase = layout.Dir

type nativeContractDir struct {
	nativeContractDirectoryBase
	ensures int
}

func (d *nativeContractDir) Ensure(ctx layout.Context) error {
	d.ensures++
	return d.nativeContractDirectoryBase.Ensure(ctx)
}

type nativeContractExec struct {
	layout.Exec
	ensures int
}

func (e *nativeContractExec) Ensure(ctx layout.Context) error {
	e.ensures++
	return e.Exec.Ensure(ctx)
}

type nativeContractFormat struct {
	formats.JSONFile[nativeContractValue]
	loads int
}

func (f *nativeContractFormat) Load() (bool, error) {
	f.loads++
	return f.JSONFile.Load()
}

type nativeContractTemplate struct {
	layout.TextTemplate[nativeContractValue]
}
type nativeContractLink struct{ layout.Link }
type nativeContractFileLink struct{ layout.FileLink }
type nativeContractDirLink struct{ layout.DirLink }

type nativeContractComposite struct {
	Root   nativeContractDir    `layout:"dir" path:"."`
	Config nativeContractFormat `layout:"content" path:"config.json"`
}

type nativeContractDirectories struct {
	layout.Slot[*nativeContractComposite]
}
type nativeContractDocuments struct {
	layout.FileSlot[*nativeContractFormat]
}
type nativeContractLinks struct {
	layout.LinkSlot[*layout.Link]
}

func TestNativeContractsComposeExtendedCarrierFamilies(t *testing.T) {
	type tree struct {
		Root        nativeContractDir         `layout:"dir" path:"."`
		Directory   *nativeContractDir        `layout:"dir" path:"assets"`
		Executable  *nativeContractExec       `layout:"exec" path:"run.sh"`
		File        nativeContractFile        `layout:"file" path:"main.go"`
		Format      nativeContractFormat      `layout:"content" path:"config.json"`
		Template    nativeContractTemplate    `layout:"content" path:"readme.txt"`
		Link        nativeContractLink        `layout:"content" path:"alias"`
		FileLink    nativeContractFileLink    `layout:"content" path:"file-alias"`
		DirLink     nativeContractDirLink     `layout:"content" path:"dir-alias"`
		Directories nativeContractDirectories `layout:"directories" path:"services"`
		Documents   nativeContractDocuments   `layout:"documents" path:"configs"`
		Links       nativeContractLinks       `layout:"links" path:"links"`
	}
	base := t.TempDir()
	_, model := composeTree[tree](t, base)
	member := model.Directories.MustComposeMember("api")
	document := model.Documents.MustComposeMember("api.json")
	link := model.Links.MustComposeMember("api")
	for _, node := range []struct {
		carrier layout.Pather
		path    string
	}{
		{&model.Root, "."},
		{model.Directory, "assets"},
		{model.Executable, "run.sh"},
		{&model.File, "main.go"},
		{&model.Format, "config.json"},
		{&model.Template, "readme.txt"},
		{&model.Link, "alias"},
		{&model.FileLink, "file-alias"},
		{&model.DirLink, "dir-alias"},
		{&member.Root, "services/api"},
		{&member.Config, "services/api/config.json"},
		{document, "configs/api.json"},
		{link, "links/api"},
	} {
		if got, want := node.carrier.Path(), filepath.Join(base, node.path); got != want {
			t.Errorf("%T.Path() = %q, want %q", node.carrier, got, want)
		}
	}
	if _, err := os.Stat(model.Directories.Path()); !os.IsNotExist(err) {
		t.Fatalf("composition created a collection directory: %v", err)
	}
	if got, _ := model.Directories.Get("api"); got != member {
		t.Fatal("directory wrapper did not retain its composed member")
	}
	document.Set(nativeContractValue{Name: "cached"})
	link.SetTarget("../target")
	if got, _ := model.Documents.Get("api.json"); got != document || got.MustGet().Name != "cached" {
		t.Fatal("document wrapper did not retain its member and typed state")
	}
	if got, _ := model.Links.Get("api"); got != link || got.MustTarget() != "../target" {
		t.Fatal("link wrapper did not retain its member and target state")
	}
}

func TestNativeContentContractPreservesWrapperLoadHook(t *testing.T) {
	type tree struct {
		Documents nativeContractDocuments `layout:"documents" path:"configs"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	member := model.Documents.MustComposeMember("api.json")
	if err := os.MkdirAll(model.Documents.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(member.Path(), []byte(`{"name":"disk"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Load(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if got, _ := model.Documents.Get("api.json"); got != member || got.loads != 1 || got.MustGet().Name != "disk" {
		t.Fatal("managed load lost the wrapper hook, member identity, or decoded state")
	}
}

type nativeContractScannedFile struct {
	layout.File
	scans int
}

func (f *nativeContractScannedFile) Scan() (layout.DiskState, error) {
	f.scans++
	return layout.DiskPresent, nil
}

func TestNativeRawCarrierCustomHookPrecedesNonApplicableClassification(t *testing.T) {
	type tree struct {
		File    nativeContractScannedFile                   `layout:"file" path:"main.go"`
		Members layout.FileSlot[*nativeContractScannedFile] `layout:"documents" path:"members"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	member := model.Members.MustComposeMember("test.go")
	var report layout.Report
	ctx := layout.DefaultContext
	ctx.Reporter = &report
	if _, err := manager.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if model.File.scans != 1 || member.scans != 1 {
		t.Fatal("raw carrier classification bypassed a wrapper's custom Scan hook")
	}
	assertEntries(t, report.Entries(), []layout.Entry{
		{Op: layout.OpScan, Path: model.File.Path(), Result: layout.ScanPresent},
		{Op: layout.OpScan, Path: member.Path(), Result: layout.ScanPresent},
		{Op: layout.OpScan, Path: model.Members.Path(), Result: layout.ScanTraversed},
	})
}

type nativeContractShallowContent struct{}

func (nativeContractShallowContent) Path() string                    { return "" }
func (nativeContractShallowContent) HasContent() bool                { return false }
func (nativeContractShallowContent) DiskState() layout.DiskState     { return layout.DiskUnknown }
func (nativeContractShallowContent) MemoryState() layout.MemoryState { return layout.MemoryUnknown }

type nativeContractShallowCollection struct{}

func (nativeContractShallowCollection) Path() string      { return "" }
func (nativeContractShallowCollection) ChildKind() string { return "dir" }
func (nativeContractShallowCollection) Enter() (subject.Children, error) {
	return subject.Children{}, nil
}

type nativeContractBadGet struct {
	formats.JSONFile[nativeContractValue]
}

func (nativeContractBadGet) Get() (string, bool) { return "", false }

type nativeContractBadSet struct {
	formats.JSONFile[nativeContractValue]
}

func (*nativeContractBadSet) Set(string) {}

type nativeContractBadDir struct{ nativeContractDirectoryBase }

func (nativeContractBadDir) Dir(string) string { return "" }

type nativeContractBadExec struct{ layout.Exec }

func (nativeContractBadExec) Output(context.Context, layout.RunOptions) (string, error) {
	return "", nil
}

type nativeContractBadEnsureDir struct{ nativeContractDirectoryBase }

func (*nativeContractBadEnsureDir) Ensure() error { return nil }

type nativeContractHiddenDir struct{ layout.Dir }

type nativeContractBadRunExec struct{ layout.Exec }

func (nativeContractBadRunExec) Run(context.Context, layout.RunOptions) (int, error) {
	return 0, nil
}

type nativeContractBadState struct {
	formats.JSONFile[nativeContractValue]
}

func (nativeContractBadState) MemoryState() string { return "" }

type nativeContractBadContext struct {
	layout.TextTemplate[nativeContractValue]
}

func (*nativeContractBadContext) SetContext(string) {}

type nativeContractBadTarget struct{ layout.Link }

func (*nativeContractBadTarget) SetTarget(int) {}

type nativeContractBadTargetFile struct{ layout.FileLink }

func (nativeContractBadTargetFile) TargetFile() (layout.Dir, bool) { return layout.Dir{}, false }

type nativeContractBadTargetDir struct{ layout.DirLink }

func (nativeContractBadTargetDir) TargetDir() (layout.File, bool) { return layout.File{}, false }

type nativeContractBadEntries struct {
	layout.FileSlot[*nativeContractFormat]
}

func (*nativeContractBadEntries) Entries() []string { return nil }

type nativeContractBadMember struct {
	layout.Slot[*nativeContractComposite]
}

func (*nativeContractBadMember) ComposeMember(string) (*nativeContractFormat, error) { return nil, nil }

type nativeContractBadLinkGet struct {
	layout.LinkSlot[*layout.Link]
}

func (*nativeContractBadLinkGet) Get(string) (*nativeContractFile, bool) { return nil, false }

type nativeContractBadEnter struct {
	layout.FileSlot[*nativeContractFormat]
}

func (*nativeContractBadEnter) Enter() string { return "" }

func TestNativeContractsRejectIncompleteAndIncompatibleCarriers(t *testing.T) {
	type shallowContent struct {
		Node nativeContractShallowContent `layout:"content" path:"node"`
	}
	type shallowCollection struct {
		Node nativeContractShallowCollection `layout:"directories" path:"node"`
	}
	type badGet struct {
		Node nativeContractBadGet `layout:"content" path:"node"`
	}
	type badSet struct {
		Node nativeContractBadSet `layout:"content" path:"node"`
	}
	type badDir struct {
		Node nativeContractBadDir `layout:"dir" path:"node"`
	}
	type hiddenDir struct {
		Node nativeContractHiddenDir `layout:"dir" path:"node"`
	}
	type badExec struct {
		Node nativeContractBadExec `layout:"exec" path:"node"`
	}
	type badContext struct {
		Node nativeContractBadContext `layout:"content" path:"node"`
	}
	type badTarget struct {
		Node nativeContractBadTarget `layout:"content" path:"node"`
	}
	type badTargetFile struct {
		Node nativeContractBadTargetFile `layout:"content" path:"node"`
	}
	type badTargetDir struct {
		Node nativeContractBadTargetDir `layout:"content" path:"node"`
	}
	type badEntries struct {
		Node nativeContractBadEntries `layout:"documents" path:"node"`
	}
	type badMember struct {
		Node nativeContractBadMember `layout:"directories" path:"node"`
	}
	type badLinkGet struct {
		Node nativeContractBadLinkGet `layout:"links" path:"node"`
	}
	for _, test := range []struct {
		name    string
		compile func() error
	}{
		{"shallow content", compileRoleError[shallowContent]},
		{"shallow collection", compileRoleError[shallowCollection]},
		{"content Get value type", compileRoleError[badGet]},
		{"content Set value type", compileRoleError[badSet]},
		{"directory child handle", compileRoleError[badDir]},
		{"directory field hides child handle method", compileRoleError[hiddenDir]},
		{"executable output", compileRoleError[badExec]},
		{"template context type", compileRoleError[badContext]},
		{"link target type", compileRoleError[badTarget]},
		{"file link target handle", compileRoleError[badTargetFile]},
		{"directory link target handle", compileRoleError[badTargetDir]},
		{"document collection entry type", compileRoleError[badEntries]},
		{"directory collection member type", compileRoleError[badMember]},
		{"link collection member type", compileRoleError[badLinkGet]},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compile(); err == nil {
				t.Fatal("Compile accepted a carrier without its family's complete compatible API")
			}
		})
	}
}

func TestNativeContractsRejectIncompatibleMemberRootCarriers(t *testing.T) {
	type badFile struct {
		Members layout.FileSlot[*contractIncompatibleFile] `layout:"documents" path:"members"`
	}
	type badDir struct {
		Members layout.Slot[*nativeContractBadEnsureDir] `layout:"directories" path:"members"`
	}
	type badExec struct {
		Members layout.FileSlot[*nativeContractBadRunExec] `layout:"documents" path:"members"`
	}
	type badContentGet struct {
		Members layout.FileSlot[*nativeContractBadGet] `layout:"documents" path:"members"`
	}
	type badContentState struct {
		Members layout.FileSlot[*nativeContractBadState] `layout:"documents" path:"members"`
	}
	type badCollectionEnter struct {
		Members layout.Slot[*nativeContractBadEnter] `layout:"directories" path:"members"`
	}
	type badCollectionEntries struct {
		Members layout.Slot[*nativeContractBadEntries] `layout:"directories" path:"members"`
	}
	for _, test := range []struct {
		name    string
		compose func(*testing.T, string) error
	}{
		{"file WriteBytes", composeNativeContractError[badFile]},
		{"directory Ensure", composeNativeContractError[badDir]},
		{"executable Run", composeNativeContractError[badExec]},
		{"content Get", composeNativeContractError[badContentGet]},
		{"content MemoryState", composeNativeContractError[badContentState]},
		{"nested collection Enter", composeNativeContractError[badCollectionEnter]},
		{"nested collection Entries", composeNativeContractError[badCollectionEntries]},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compose(t, t.TempDir()); err == nil {
				t.Fatal("Compose accepted an incompatible carrier as a member root")
			}
		})
	}
}

func composeNativeContractError[T any](t *testing.T, path string) error {
	t.Helper()
	structure, err := conduit.Compile[T]()
	if err != nil {
		t.Fatalf("Compile() rejected the outer collection declaration before preparing its member chart: %v", err)
	}
	_, err = structure.Compose(path)
	return err
}

type nativeContractRawMembers struct {
	Files       layout.FileSlot[*nativeContractFile] `layout:"documents" path:"files"`
	Directories layout.Slot[*nativeContractDir]      `layout:"directories" path:"directories"`
	Executables layout.FileSlot[*nativeContractExec] `layout:"documents" path:"executables"`
}

func TestNativeCarrierEnsurePoliciesGateDeclaredAndMemberHooks(t *testing.T) {
	type tree struct {
		File       nativeContractFile       `layout:"file" path:"file.txt"`
		Directory  nativeContractDir        `layout:"dir" path:"directory"`
		Executable nativeContractExec       `layout:"exec" path:"run.sh"`
		Members    nativeContractRawMembers `layout:"group" path:"members"`
	}
	for _, test := range []struct {
		name                        string
		policy                      layout.EnsurePolicy
		file, directory, executable int
	}{
		{"none", layout.EnsureNone, 0, 0, 0},
		{"files", layout.EnsureFiles, 1, 0, 0},
		{"directories", layout.EnsureDirs, 0, 1, 0},
		{"executables", layout.EnsureExecs, 0, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager, model := composeTree[tree](t, t.TempDir())
			file := model.Members.Files.MustComposeMember("file.txt")
			directory := model.Members.Directories.MustComposeMember("directory")
			executable := model.Members.Executables.MustComposeMember("run.sh")
			var report layout.Report
			ctx := layout.DefaultContext
			ctx.EnsurePolicy, ctx.Reporter = test.policy, &report
			if _, err := manager.Ensure(ctx); err != nil {
				t.Fatal(err)
			}
			for _, node := range []struct {
				carrier     layout.Node
				calls, want int
			}{
				{&model.File, model.File.ensures, test.file},
				{file, file.ensures, test.file},
				{&model.Directory, model.Directory.ensures, test.directory},
				{directory, directory.ensures, test.directory},
				{&model.Executable, model.Executable.ensures, test.executable},
				{executable, executable.ensures, test.executable},
			} {
				if node.calls != node.want || node.carrier.Exists() != (node.want != 0) {
					t.Errorf("%T at %q: hook calls %d and Exists %v, want %d and %v", node.carrier, node.carrier.Path(), node.calls, node.carrier.Exists(), node.want, node.want != 0)
				}
				code := layout.EnsureSkippedPolicy
				if node.want != 0 {
					code = layout.EnsureEnsured
				}
				assertHasEntry(t, report.Entries(), layout.Entry{Op: layout.OpEnsure, Path: node.carrier.Path(), Result: code})
			}
		})
	}
}

func TestNativeRawMemberCarriersReportNonApplicableOperations(t *testing.T) {
	manager, model := composeTree[nativeContractRawMembers](t, t.TempDir())
	file := model.Files.MustComposeMember("main.go")
	directory := model.Directories.MustComposeMember("assets")
	executable := model.Executables.MustComposeMember("run.sh")
	if _, err := manager.Ensure(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		run          func(layout.Context) (layout.ResultCode, error)
		op           layout.Operation
		node, source layout.ResultCode
	}{
		{"scan", manager.Scan, layout.OpScan, layout.ScanNotApplicable, layout.ScanTraversed},
		{"load", manager.Load, layout.OpLoad, layout.LoadNotApplicable, layout.LoadTraversed},
		{"discover", manager.Discover, layout.OpDiscover, layout.DiscoverNotApplicable, layout.DiscoverTraversed},
		{"sync", manager.Sync, layout.OpSync, layout.SyncNotApplicable, layout.SyncTraversed},
	} {
		t.Run(test.name, func(t *testing.T) {
			var report layout.Report
			ctx := layout.DefaultContext
			ctx.Reporter = &report
			if _, err := test.run(ctx); err != nil {
				t.Fatal(err)
			}
			assertEntries(t, report.Entries(), []layout.Entry{
				{Op: test.op, Path: file.Path(), Result: test.node},
				{Op: test.op, Path: model.Files.Path(), Result: test.source},
				{Op: test.op, Path: directory.Path(), Result: test.node},
				{Op: test.op, Path: model.Directories.Path(), Result: test.source},
				{Op: test.op, Path: executable.Path(), Result: test.node},
				{Op: test.op, Path: model.Executables.Path(), Result: test.source},
			})
		})
	}
}

func TestNativeContractsKeepCompositeMemberDeclarations(t *testing.T) {
	type tree struct {
		Directories nativeContractDirectories `layout:"directories" path:"services"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	member := model.Directories.MustComposeMember("api")
	member.Config.Set(nativeContractValue{Name: "api"})
	ctx := layout.DefaultContext
	ctx.EnsurePolicy = layout.EnsureDirs
	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if member.Root.ensures != 1 || !member.Root.Exists() || member.Config.Exists() {
		t.Fatal("directory-only Ensure did not honor the composite member's declared nodes")
	}
	if _, err := manager.Sync(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if member.Config.MustGet().Name != "api" || !member.Config.Exists() {
		t.Fatal("Sync did not reach the composite member's declared content")
	}
}
