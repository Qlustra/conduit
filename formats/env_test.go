package formats

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/qlustra/conduit/layout"
)

func TestEnvFileSaveAndLoad(t *testing.T) {
	var f EnvFile
	f.ComposePath(filepath.Join(t.TempDir(), ".env"))
	f.Set(map[string]string{
		"COUNT":   "2",
		"MESSAGE": "hello world",
	})

	if err := f.Save(layout.DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != "COUNT=2\nMESSAGE=\"hello world\"\n" {
		t.Fatalf("file contents = %q", got)
	}

	var loaded EnvFile
	loaded.ComposePath(f.Path())
	ok, err := loaded.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok {
		t.Fatalf("Load() ok = false, want true")
	}
	if got := loaded.MustGet()["COUNT"]; got != "2" {
		t.Fatalf("MustGet()[\"COUNT\"] = %q, want %q", got, "2")
	}
	if got := loaded.MustGet()["MESSAGE"]; got != "hello world" {
		t.Fatalf("MustGet()[\"MESSAGE\"] = %q, want %q", got, "hello world")
	}
}

func TestEnvFileLoadAcceptsRelaxedSyntaxAndNormalizesOnSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# comment\nexport NAME=api\nPORT: 8080\nFOO=bar # trailing comment\nNAME=worker\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var f EnvFile
	f.ComposePath(path)

	ok, err := f.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok {
		t.Fatalf("Load() ok = false, want true")
	}

	got := f.MustGet()
	if got["NAME"] != "worker" {
		t.Fatalf("MustGet()[\"NAME\"] = %q, want %q", got["NAME"], "worker")
	}
	if got["PORT"] != "8080" {
		t.Fatalf("MustGet()[\"PORT\"] = %q, want %q", got["PORT"], "8080")
	}
	if got["FOO"] != "bar" {
		t.Fatalf("MustGet()[\"FOO\"] = %q, want %q", got["FOO"], "bar")
	}

	if err := f.Save(layout.DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != "FOO=\"bar\"\nNAME=\"worker\"\nPORT=8080\n" {
		t.Fatalf("normalized file contents = %q", got)
	}
}

func TestEnvFilePreservesLiteralExpressionsAcrossRoundTrip(t *testing.T) {
	var f EnvFile
	f.ComposePath(filepath.Join(t.TempDir(), ".env"))
	f.Set(map[string]string{
		"EXAMPLE_VAR": "${EXAMPLE_VAR:-default}",
	})

	if err := f.Save(layout.DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var loaded EnvFile
	loaded.ComposePath(f.Path())
	ok, err := loaded.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok {
		t.Fatalf("Load() ok = false, want true")
	}
	if got := loaded.MustGet()["EXAMPLE_VAR"]; got != "${EXAMPLE_VAR:-default}" {
		t.Fatalf("MustGet()[\"EXAMPLE_VAR\"] = %q, want %q", got, "${EXAMPLE_VAR:-default}")
	}
}

func TestEnvMapParsesEntriesWithLastWins(t *testing.T) {
	got, err := EnvMap([]string{
		"PORT=8080",
		"DATABASE_URL=postgres://db/app?sslmode=disable",
		"PORT=9000",
	})
	if err != nil {
		t.Fatalf("EnvMap() error = %v", err)
	}

	want := map[string]string{
		"DATABASE_URL": "postgres://db/app?sslmode=disable",
		"PORT":         "9000",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvMap() = %#v, want %#v", got, want)
	}
}

func TestEnvMapRejectsInvalidEntries(t *testing.T) {
	t.Run("missing separator", func(t *testing.T) {
		if _, err := EnvMap([]string{"PORT"}); err == nil {
			t.Fatal("EnvMap() error = nil, want non-nil")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		if _, err := EnvMap([]string{"=value"}); err == nil {
			t.Fatal("EnvMap() error = nil, want non-nil")
		}
	})
}

func TestResolveEnvAppliesBaselinePrecedenceAndScope(t *testing.T) {
	defaults := map[string]string{
		"APP_MODE":  "default",
		"EMPTY_OK":  "",
		"LOG_LEVEL": "info",
	}
	processEnv := map[string]string{
		"APP_MODE":      "process",
		"PROCESS_EXTRA": "from-process",
	}
	fileEnv := map[string]string{
		"APP_MODE":   "file",
		"FILE_EXTRA": "from-file",
		"LOG_LEVEL":  "debug",
	}

	tests := []struct {
		name string
		opts EnvResolveOptions
		want map[string]string
	}{
		{
			name: "baseline only process wins",
			opts: EnvResolveOptions{Precedence: EnvProcessWins},
			want: map[string]string{
				"APP_MODE":  "process",
				"EMPTY_OK":  "",
				"LOG_LEVEL": "debug",
			},
		},
		{
			name: "baseline only file wins",
			opts: EnvResolveOptions{Precedence: EnvFileWins},
			want: map[string]string{
				"APP_MODE":  "file",
				"EMPTY_OK":  "",
				"LOG_LEVEL": "debug",
			},
		},
		{
			name: "keep process extras",
			opts: EnvResolveOptions{Precedence: EnvProcessWins, Scope: EnvKeepProcess},
			want: map[string]string{
				"APP_MODE":      "process",
				"EMPTY_OK":      "",
				"LOG_LEVEL":     "debug",
				"PROCESS_EXTRA": "from-process",
			},
		},
		{
			name: "keep file extras",
			opts: EnvResolveOptions{Precedence: EnvProcessWins, Scope: EnvKeepFile},
			want: map[string]string{
				"APP_MODE":   "process",
				"EMPTY_OK":   "",
				"FILE_EXTRA": "from-file",
				"LOG_LEVEL":  "debug",
			},
		},
		{
			name: "keep all extras file wins",
			opts: EnvResolveOptions{Precedence: EnvFileWins, Scope: EnvKeepAll},
			want: map[string]string{
				"APP_MODE":      "file",
				"EMPTY_OK":      "",
				"FILE_EXTRA":    "from-file",
				"LOG_LEVEL":     "debug",
				"PROCESS_EXTRA": "from-process",
			},
		},
		{
			name: "zero precedence defaults to process wins",
			opts: EnvResolveOptions{Scope: EnvKeepAll},
			want: map[string]string{
				"APP_MODE":      "process",
				"EMPTY_OK":      "",
				"FILE_EXTRA":    "from-file",
				"LOG_LEVEL":     "debug",
				"PROCESS_EXTRA": "from-process",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveEnv(defaults, processEnv, fileEnv, tt.opts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ResolveEnv() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEnvListReturnsSortedEntries(t *testing.T) {
	got := EnvList(map[string]string{
		"LOG_LEVEL": "debug",
		"APP_MODE":  "local",
	})

	want := []string{
		"APP_MODE=local",
		"LOG_LEVEL=debug",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvList() = %#v, want %#v", got, want)
	}
}

func TestEnvFileResolveWithProcess(t *testing.T) {
	t.Run("unloaded returns false", func(t *testing.T) {
		var f EnvFile
		f.ComposePath(filepath.Join(t.TempDir(), ".env"))

		if _, ok := f.ResolveWithProcess(map[string]string{"A": "default"}, map[string]string{"A": "process"}, EnvResolveOptions{}); ok {
			t.Fatal("ResolveWithProcess() ok = true, want false")
		}
	})

	t.Run("loaded resolves cached file content", func(t *testing.T) {
		var f EnvFile
		f.ComposePath(filepath.Join(t.TempDir(), ".env"))
		f.Set(map[string]string{
			"APP_MODE":    "file",
			"FILE_SECRET": "present",
		})

		got, ok := f.ResolveWithProcess(
			map[string]string{"APP_MODE": "default", "EMPTY_OK": ""},
			map[string]string{"APP_MODE": "process", "PROCESS_ONLY": "x"},
			EnvResolveOptions{Precedence: EnvFileWins, Scope: EnvKeepFile},
		)
		if !ok {
			t.Fatal("ResolveWithProcess() ok = false, want true")
		}

		want := map[string]string{
			"APP_MODE":    "file",
			"EMPTY_OK":    "",
			"FILE_SECRET": "present",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ResolveWithProcess() = %#v, want %#v", got, want)
		}
	})
}
