package layout

import (
	"path/filepath"
	"testing"
)

type testJSONFile struct {
	testMapFile
}

func TestDirPathHelpers(t *testing.T) {
	dir := NewDir(filepath.Join("workspace", "services", "api.v2"))

	if got := dir.Base(); got != "api.v2" {
		t.Fatalf("Base() = %q, want %q", got, "api.v2")
	}
	if got := dir.Stem(); got != "api" {
		t.Fatalf("Stem() = %q, want %q", got, "api")
	}
	if got := dir.ParentPath(); got != filepath.Join("workspace", "services") {
		t.Fatalf("ParentPath() = %q, want %q", got, filepath.Join("workspace", "services"))
	}
	if got := dir.ParentDir().Path(); got != filepath.Join("workspace", "services") {
		t.Fatalf("ParentDir().Path() = %q, want %q", got, filepath.Join("workspace", "services"))
	}
}

func TestFilePathHelpers(t *testing.T) {
	file := NewFile(filepath.Join("workspace", "configs", "archive.tar.gz"))

	if got := file.Base(); got != "archive.tar.gz" {
		t.Fatalf("Base() = %q, want %q", got, "archive.tar.gz")
	}
	if got := file.Ext(); got != ".gz" {
		t.Fatalf("Ext() = %q, want %q", got, ".gz")
	}
	if got := file.Stem(); got != "archive.tar" {
		t.Fatalf("Stem() = %q, want %q", got, "archive.tar")
	}
	if got := file.ParentPath(); got != filepath.Join("workspace", "configs") {
		t.Fatalf("ParentPath() = %q, want %q", got, filepath.Join("workspace", "configs"))
	}
	if got := file.ParentDir().Path(); got != filepath.Join("workspace", "configs") {
		t.Fatalf("ParentDir().Path() = %q, want %q", got, filepath.Join("workspace", "configs"))
	}
}

func TestPathHelpersPreserveDotfilesAndExtensionlessNames(t *testing.T) {
	file := NewFile(filepath.Join("workspace", ".env"))
	dir := NewDir(".config")

	if got := file.Ext(); got != "" {
		t.Fatalf("Ext() for dotfile = %q, want empty", got)
	}
	if got := file.Stem(); got != ".env" {
		t.Fatalf("Stem() for dotfile = %q, want %q", got, ".env")
	}
	if got := dir.Stem(); got != ".config" {
		t.Fatalf("Stem() for dot-dir = %q, want %q", got, ".config")
	}
}

func TestComposedPathsTrackComposeBaseAcrossTree(t *testing.T) {
	type service struct {
		Root   Dir  `layout:"."`
		Config File `layout:"config.yaml"`
		Logs   Dir  `layout:"logs"`
	}

	type workspace struct {
		Root     Dir                    `layout:"."`
		Services Slot[*service]         `layout:"services"`
		Configs  FileSlot[testJSONFile] `layout:"configs"`
	}

	var ws workspace
	if err := Compose("workspace", &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

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

	svc := ws.Services.MustAt("api")

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

	config := ws.Configs.MustAt("app.json")
	if got, ok := config.ComposedRelativePath(); !ok || got != filepath.Join("configs", "app.json") {
		t.Fatalf("config.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, filepath.Join("configs", "app.json"))
	}
}

func TestComposedPathsAreUnavailableWithoutComposition(t *testing.T) {
	dir := NewDir("workspace")
	file := NewFile(filepath.Join("workspace", "config.yaml"))
	slot := NewSlot[*struct{}](dir)
	fileSlot := NewFileSlot[testJSONFile](dir)

	if _, ok := dir.ComposedBaseDir(); ok {
		t.Fatal("Dir.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := dir.ComposedRelativePath(); ok {
		t.Fatal("Dir.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := dir.JoinComposedPath("child"); ok {
		t.Fatal("Dir.JoinComposedPath() ok = true, want false")
	}

	if _, ok := file.ComposedBaseDir(); ok {
		t.Fatal("File.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := file.ComposedRelativePath(); ok {
		t.Fatal("File.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := file.JoinComposedPath("child"); ok {
		t.Fatal("File.JoinComposedPath() ok = true, want false")
	}
	if _, ok := file.ParentDir().ComposedRelativePath(); ok {
		t.Fatal("File.ParentDir().ComposedRelativePath() ok = true, want false")
	}

	if _, ok := slot.ComposedBaseDir(); ok {
		t.Fatal("Slot.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := slot.ComposedRelativePath(); ok {
		t.Fatal("Slot.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := slot.JoinComposedPath("child"); ok {
		t.Fatal("Slot.JoinComposedPath() ok = true, want false")
	}
	if _, ok := fileSlot.ComposedBaseDir(); ok {
		t.Fatal("FileSlot.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := fileSlot.ComposedRelativePath(); ok {
		t.Fatal("FileSlot.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := fileSlot.JoinComposedPath("child"); ok {
		t.Fatal("FileSlot.JoinComposedPath() ok = true, want false")
	}
}

func TestDeclaredPathsTrackLocalLayoutFragments(t *testing.T) {
	type service struct {
		Root   Dir  `layout:"."`
		Config File `layout:"config.yaml"`
		Logs   Dir  `layout:"logs"`
	}

	type workspace struct {
		Root     Dir                    `layout:"."`
		Services Slot[*service]         `layout:"services"`
		Configs  FileSlot[testJSONFile] `layout:"configs"`
	}

	var ws workspace
	if err := Compose("workspace", &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	svc := ws.Services.MustAt("api")

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

	config := ws.Configs.MustAt("app.json")
	if _, ok := config.DeclaredPath(); ok {
		t.Fatal("file slot item DeclaredPath() ok = true, want false")
	}
}

func TestRelHelpersUsePatherAndStringBases(t *testing.T) {
	type workspace struct {
		Root   Dir  `layout:"."`
		Config File `layout:"config.yaml"`
		Logs   Dir  `layout:"logs"`
	}

	var ws workspace
	if err := Compose(filepath.Join("workspace", "app"), &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

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
		Root   Dir  `layout:"."`
		Config File `layout:"config.yaml"`
	}

	type workspace struct {
		Root     Dir            `layout:"."`
		Services Slot[*service] `layout:"services"`
	}

	var ws workspace
	if err := Compose("workspace", &ws); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	svc := ws.Services.MustAt("api")
	if got, err := svc.Config.RelTo(&ws.Services); err != nil || got != filepath.Join("api", "config.yaml") {
		t.Fatalf("Config.RelTo(Services) = (%q, %v), want (%q, nil)", got, err, filepath.Join("api", "config.yaml"))
	}
}
