package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinEnsurePreservesExplicitNoneAndZeroDefault(t *testing.T) {
	type node interface {
		Node
		Ensure(Context) error
	}
	carriers := []struct {
		name string
		new  func(string) node
	}{
		{"directory", func(path string) node { d := NewDir(path); return &d }},
		{"file", func(path string) node { f := NewFile(path); return &f }},
		{"executable", func(path string) node { e := NewExec(path); return &e }},
		{"format", func(path string) node { var f testMapFile; f.ComposePath(path); return &f }},
	}
	for _, policy := range []struct {
		name   string
		value  EnsurePolicy
		create bool
	}{
		{"none", EnsureNone, false},
		{"zero default", 0, true},
	} {
		for _, carrier := range carriers {
			t.Run(policy.name+"/"+carrier.name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "uncreated", "node")
				n := carrier.new(path)
				ctx := DefaultContext
				ctx.EnsurePolicy = policy.value
				if err := n.Ensure(ctx); err != nil {
					t.Fatal(err)
				}
				if got := n.Exists(); got != policy.create {
					t.Fatalf("Ensure created node = %v, want %v", got, policy.create)
				}
				if !policy.create {
					if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
						t.Fatalf("EnsureNone created a parent directory: %v", err)
					}
				}
			})
		}
	}
}

func TestEnsurePolicyHelpers(t *testing.T) {
	policy := EnsureNone.
		Allow(EnsureDirs | EnsureSyncables).
		Deny(EnsureDirs)

	if policy != EnsureSyncables {
		t.Fatalf("policy = %v, want EnsureSyncables", policy)
	}
	if policy.Has(EnsureSyncables) != true {
		t.Fatal("policy.Has(EnsureSyncables) = false, want true")
	}
	if policy.Has(EnsureDirs) != false {
		t.Fatal("policy.Has(EnsureDirs) = true, want false")
	}

	policy = EnsureScaffold.Deny(EnsureDirs | EnsureFiles | EnsureExecs)
	if policy != EnsureNone {
		t.Fatalf("policy after Deny(all scaffold bits) = %v, want EnsureNone", policy)
	}

	policy = EnsureNone.Allow(EnsureDirs | EnsurePolicy(1<<7))
	if policy != EnsureDirs {
		t.Fatalf("policy with unknown bits allowed = %v, want EnsureDirs", policy)
	}
}
