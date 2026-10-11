package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"path/filepath"
	"testing"
)

func TestComposedPathsTrackComposeBaseAcrossTree(t *testing.T) {
	type service struct {
		Root   Dir  `layout:"dir" path:"."`
		Config File `layout:"file" path:"config.yaml"`
		Logs   Dir  `layout:"dir" path:"logs"`
	}

	type workspace struct {
		Root     Dir                    `layout:"dir" path:"."`
		Services Slot[*service]         `layout:"directories" path:"services"`
		Configs  FileSlot[testJSONFile] `layout:"documents" path:"configs"`
	}

	_, ws := composeTree[workspace](t, "workspace")

	base, ok := ws.Root.ComposedBaseDir()
	if !ok {
		t.Fatal("Root.ComposedBaseDir() ok = false, want true")
	}
	if got := base.Path(); got != "workspace" {
		t.Fatalf("Root.ComposedBaseDir().Path() = %q, want %q", got, "workspace")
	}

	if got, ok := ws.Root.ComposedRelativePath(); !ok || got != "." {
		t.Fatalf("Root.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, ".")
	}
	if got, ok := ws.Root.JoinComposedPath("bin", "tool"); !ok || got != filepath.Join("bin", "tool") {
		t.Fatalf("Root.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("bin", "tool"))
	}

	svc := ws.Services.MustComposeMember("api")

	if got, ok := ws.Services.ComposedRelativePath(); !ok || got != "services" {
		t.Fatalf("Services.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, "services")
	}
	if got, ok := ws.Services.JoinComposedPath("api"); !ok || got != filepath.Join("services", "api") {
		t.Fatalf("Services.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api"))
	}
	if got, ok := ws.Configs.ComposedRelativePath(); !ok || got != "configs" {
		t.Fatalf("Configs.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, "configs")
	}
	if got, ok := ws.Configs.JoinComposedPath("app.json"); !ok || got != filepath.Join("configs", "app.json") {
		t.Fatalf("Configs.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("configs", "app.json"))
	}

	if got, ok := svc.Root.ComposedRelativePath(); !ok || got != filepath.Join("services", "api") {
		t.Fatalf("service.Root.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api"))
	}
	if got, ok := svc.Config.ComposedRelativePath(); !ok || got != filepath.Join("services", "api", "config.yaml") {
		t.Fatalf("service.Config.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api", "config.yaml"))
	}

	logs := svc.Root.Dir("logs")
	if got, ok := logs.ComposedRelativePath(); !ok || got != filepath.Join("services", "api", "logs") {
		t.Fatalf("derived logs.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api", "logs"))
	}
	parent := svc.Config.ParentDir()
	if got, ok := parent.ComposedRelativePath(); !ok || got != filepath.Join("services", "api") {
		t.Fatalf("Config.ParentDir().ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api"))
	}
	if got, ok := svc.Config.JoinComposedPath("bak"); !ok || got != filepath.Join("services", "api", "config.yaml", "bak") {
		t.Fatalf("Config.JoinComposedPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api", "config.yaml", "bak"))
	}

	config := ws.Configs.MustComposeMember("app.json")
	if got, ok := config.ComposedRelativePath(); !ok || got != filepath.Join("configs", "app.json") {
		t.Fatalf("config.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("configs", "app.json"))
	}
}

func TestDeclaredPathsTrackLocalLayoutFragments(t *testing.T) {
	type service struct {
		Root   Dir  `layout:"dir" path:"."`
		Config File `layout:"file" path:"config.yaml"`
		Logs   Dir  `layout:"dir" path:"logs"`
	}

	type workspace struct {
		Root     Dir                    `layout:"dir" path:"."`
		Services Slot[*service]         `layout:"directories" path:"services"`
		Configs  FileSlot[testJSONFile] `layout:"documents" path:"configs"`
	}

	_, ws := composeTree[workspace](t, "workspace")

	svc := ws.Services.MustComposeMember("api")

	if got, ok := ws.Root.DeclaredPath(); !ok || got != "." {
		t.Fatalf("Root.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, ".")
	}
	if got, ok := ws.Root.JoinDeclaredPath("config.yaml"); !ok || got != "config.yaml" {
		t.Fatalf("Root.JoinDeclaredPath() = (%q, %t), want (%q, true)", got, ok, "config.yaml")
	}

	if got, ok := ws.Services.DeclaredPath(); !ok || got != "services" {
		t.Fatalf("Services.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, "services")
	}
	if got, ok := ws.Services.JoinDeclaredPath("api"); !ok || got != filepath.Join("services", "api") {
		t.Fatalf("Services.JoinDeclaredPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("services", "api"))
	}
	if got, ok := ws.Configs.DeclaredPath(); !ok || got != "configs" {
		t.Fatalf("Configs.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, "configs")
	}
	if got, ok := ws.Configs.JoinDeclaredPath("app.json"); !ok || got != filepath.Join("configs", "app.json") {
		t.Fatalf("Configs.JoinDeclaredPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("configs", "app.json"))
	}

	if got, ok := svc.Root.DeclaredPath(); !ok || got != "." {
		t.Fatalf("service.Root.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, ".")
	}
	if got, ok := svc.Config.DeclaredPath(); !ok || got != "config.yaml" {
		t.Fatalf("service.Config.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, "config.yaml")
	}
	if got, ok := svc.Config.JoinDeclaredPath("bak"); !ok || got != filepath.Join("config.yaml", "bak") {
		t.Fatalf("Config.JoinDeclaredPath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("config.yaml", "bak"))
	}
	if got, ok := svc.Logs.DeclaredPath(); !ok || got != "logs" {
		t.Fatalf("service.Logs.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, "logs")
	}

	derived := svc.Root.Dir("logs")
	if _, ok := derived.DeclaredPath(); ok {
		t.Fatal("derived Dir.DeclaredPath() ok = true, want false")
	}
	if _, ok := derived.JoinDeclaredPath("child"); ok {
		t.Fatal("derived Dir.JoinDeclaredPath() ok = true, want false")
	}

	config := ws.Configs.MustComposeMember("app.json")
	if _, ok := config.DeclaredPath(); ok {
		t.Fatal("file slot item DeclaredPath() ok = true, want false")
	}
}

func TestRelHelpersUsePatherAndStringBases(t *testing.T) {
	type workspace struct {
		Root   Dir  `layout:"dir" path:"."`
		Config File `layout:"file" path:"config.yaml"`
		Logs   Dir  `layout:"dir" path:"logs"`
	}

	_, ws := composeTree[workspace](t, filepath.Join("workspace", "app"))

	if got, err := ws.Config.RelTo(ws.Root); err != nil || got != "config.yaml" {
		t.Fatalf("Config.RelTo(Root) = (%q, %v), want (%q, nil)", got, err, "config.yaml")
	}
	if got, err := ws.Logs.JoinRelTo(ws.Root, "current"); err != nil || got != filepath.Join("logs", "current") {
		t.Fatalf("Logs.JoinRelTo(Root) = (%q, %v), want (%q, nil)", got, err, filepath.Join("logs", "current"))
	}
	if got, err := ws.Config.RelToPath(filepath.Join("workspace")); err != nil || got != filepath.Join("app", "config.yaml") {
		t.Fatalf("Config.RelToPath() = (%q, %v), want (%q, nil)", got, err, filepath.Join("app", "config.yaml"))
	}
	if got, err := ws.Config.JoinRelToPath(filepath.Join("workspace"), "bak"); err != nil || got != filepath.Join("app", "config.yaml", "bak") {
		t.Fatalf("Config.JoinRelToPath() = (%q, %v), want (%q, nil)", got, err, filepath.Join("app", "config.yaml", "bak"))
	}
	if got, err := ws.Root.RelPathTo(ws.Config.Path()); err != nil || got != "config.yaml" {
		t.Fatalf("Root.RelPathTo(Config.Path()) = (%q, %v), want (%q, nil)", got, err, "config.yaml")
	}
	if got, err := ws.Config.RelPathTo(filepath.Join("workspace", "app")); err != nil || got != ".." {
		t.Fatalf("Config.RelPathTo(workspace/app) = (%q, %v), want (%q, nil)", got, err, "..")
	}
	if got, err := ws.Root.JoinRelPathTo(ws.Logs.Path(), "current"); err != nil || got != filepath.Join("logs", "current") {
		t.Fatalf("Root.JoinRelPathTo(Logs.Path(), current) = (%q, %v), want (%q, nil)", got, err, filepath.Join("logs", "current"))
	}
}

func TestRelHelpersWorkWithSlotAsBase(t *testing.T) {
	type service struct {
		Root   Dir  `layout:"dir" path:"."`
		Config File `layout:"file" path:"config.yaml"`
	}

	type workspace struct {
		Root     Dir            `layout:"dir" path:"."`
		Services Slot[*service] `layout:"directories" path:"services"`
	}

	_, ws := composeTree[workspace](t, "workspace")

	svc := ws.Services.MustComposeMember("api")
	if got, err := svc.Config.RelTo(&ws.Services); err != nil || got != filepath.Join("api", "config.yaml") {
		t.Fatalf("Config.RelTo(Services) = (%q, %v), want (%q, nil)", got, err, filepath.Join("api", "config.yaml"))
	}
}

func TestComposeRejectsAbsoluteLayoutTag(t *testing.T) {
	type workspace struct {
		Config File `layout:"file" path:"/etc/passwd"`
	}

	err := composeError[workspace]("/workspace")
	if err == nil {
		t.Fatal("Compose() error = nil, want non-nil")
	}
}

func TestComposeRejectsEscapingLayoutTag(t *testing.T) {
	type service struct {
		Config File `layout:"file" path:"../../../secrets/config.yaml"`
	}

	type workspace struct {
		Services struct {
			API service `layout:"group" path:"api"`
		} `layout:"group" path:"services"`
	}

	err := composeError[workspace]("/workspace")
	if err == nil {
		t.Fatal("Compose() error = nil, want non-nil")
	}
}

func TestComposeAllowsBacktrackingThatStaysWithinComposeRoot(t *testing.T) {
	type tooling struct {
		Bin struct {
			Hooks Dir `layout:"dir" path:"../hooks"`
		} `layout:"group" path:"bin"`
	}

	_, layout := composeTree[tooling](t, "/workspace")

	if got := layout.Bin.Hooks.Path(); got != filepath.Join("/workspace", "hooks") {
		t.Fatalf("Hooks.Path() = %q, want %q", got, filepath.Join("/workspace", "hooks"))
	}
}
func TestComposeAllocatesExplicitGroupsAndLeavesUntaggedFieldsAlone(t *testing.T) {
	type group struct {
		Config   *testMapFile `layout:"content" path:"config.json"`
		Untagged File
	}
	type tree struct {
		Declared *group `layout:"group" path:"nested"`
		Untagged *group
	}
	base := t.TempDir()
	first, model := composeTree[tree](t, base)
	if model.Declared == nil || model.Declared.Config == nil {
		t.Fatal("declared pointer path was not allocated")
	}
	if got := model.Declared.Config.Path(); got != filepath.Join(base, "nested", "config.json") {
		t.Fatalf("Config.Path() = %q", got)
	}
	if model.Untagged != nil || model.Declared.Untagged.Path() != "" {
		t.Fatal("untagged fields participated in composition")
	}
	if first.Tree() != model {
		t.Fatal("Tree() did not retain the live instance")
	}
	_, second := composeTree[tree](t, base)
	if second.Declared == model.Declared || second.Declared.Config == model.Declared.Config {
		t.Fatal("separate compositions share carrier instances")
	}
}
