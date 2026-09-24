package mgmt

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

type deepContractValue struct {
	Value string `json:"value"`
}

type deepContractDefaultFile struct {
	formats.JSONFile[deepContractValue]
	calls int
}

func (f *deepContractDefaultFile) Default() error {
	f.calls++
	f.Set(deepContractValue{Value: "child default"})
	return nil
}

type deepContractDefaultLayout struct {
	Root   layout.Dir              `layout:"."`
	Config deepContractDefaultFile `layout:"config.json"`
	calls  int
}

// A deep contract owns its subtree. Running child defaults again would replace
// the parent-authored value and differs from existing layout.DefaultDeep.
func (l *deepContractDefaultLayout) DefaultDeep() error {
	l.calls++
	l.Config.Set(deepContractValue{Value: "deep default"})
	return nil
}

func TestBootstrapHonorsDeepDefaultBoundary(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new")
	var tree deepContractDefaultLayout
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Bootstrap(context.Background(), space.Scope(), struct{}{}, Initialization[deepContractDefaultLayout, struct{}]{}, BootstrapPolicy{}); err != nil {
		t.Fatal(err)
	}
	value, err := tree.Config.Read()
	if err != nil || value.Value != "deep default" {
		t.Fatalf("initialized content = %+v, %v", value, err)
	}
	if tree.calls != 1 || tree.Config.calls != 0 {
		t.Fatalf("default calls: root=%d child=%d; want 1,0", tree.calls, tree.Config.calls)
	}
}

type deepContractValidatorFile struct {
	formats.JSONFile[deepContractValue]
	calls int
	err   error
}

// JSONFile promotes File.Validate, so DeepValidator must take precedence over
// the ordinary Validator contract, just as layout.ValidateDeep does.
func (f *deepContractValidatorFile) ValidateDeep(layout.ValidateOptions) (layout.ResultCode, error) {
	f.calls++
	if f.err != nil {
		return layout.ValidateFailed, f.err
	}
	return layout.ValidateOK, nil
}

type deepContractValidationLayout struct {
	Root   layout.Dir                `layout:"."`
	First  deepContractValidatorFile `layout:"first.json"`
	Second deepContractValidatorFile `layout:"second.json"`
}

func TestValidateHonorsDeepValidatorsAndCollectsIndependentErrors(t *testing.T) {
	root := t.TempDir()
	firstErr, secondErr := errors.New("first domain error"), errors.New("second domain error")
	tree := deepContractValidationLayout{}
	tree.First.err, tree.Second.err = firstErr, secondErr
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := space.Scope().Validate(context.Background(), ValidationPolicy{Diagnostics: CollectAll})
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("validation dropped deep errors: %v", err)
	}
	if len(diagnostics.Entries) != 2 || tree.First.calls != 1 || tree.Second.calls != 1 {
		t.Fatalf("diagnostics=%+v; calls=%d,%d", diagnostics, tree.First.calls, tree.Second.calls)
	}
}

type deepContractValidationGroup struct {
	Root  layout.Dir                `layout:"."`
	Child deepContractValidatorFile `layout:"child.json"`
	calls int
}

func (l *deepContractValidationGroup) ValidateDeep(options layout.ValidateOptions) (layout.ResultCode, error) {
	l.calls++
	return l.Child.ValidateDeep(options)
}

func TestValidateDoesNotRepeatChildrenOfDeepValidator(t *testing.T) {
	root := t.TempDir()
	var tree deepContractValidationGroup
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := space.Scope().Validate(context.Background(), ValidationPolicy{}); err != nil {
		t.Fatal(err)
	}
	if tree.calls != 1 || tree.Child.calls != 1 {
		t.Fatalf("validation calls: group=%d child=%d; want 1,1", tree.calls, tree.Child.calls)
	}
}
