package conduit_test

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/qlustra/conduit/layout"
	"strings"
	"testing"
)

type testMapCodec struct{}

func (testMapCodec) Marshal(value map[string]string) ([]byte, error) {
	return json.Marshal(value)
}

func (testMapCodec) Unmarshal(data []byte) (map[string]string, error) {
	var value map[string]string
	err := json.Unmarshal(data, &value)
	return value, err
}

type testMapFile struct {
	Format[map[string]string, testMapCodec]
}

type testDefaultFile struct {
	Format[string, textCodec]
}

func (f *testDefaultFile) Default() error {
	f.SetDefault("default")
	return nil
}

type testDefaultTemplate struct {
	TextTemplate[testRenderContext]
}

func (f *testDefaultTemplate) Default() error {
	f.SetDefaultContext(testRenderContext{
		Name:  "templated",
		Items: []string{"a"},
	})
	return nil
}

type testDefaultChild struct {
	Config testDefaultFile `layout:"content" path:"config.txt"`
}

type testDefaultLayout struct {
	Root     Dir                     `layout:"dir" path:"."`
	Config   testDefaultFile         `layout:"content" path:"config.txt"`
	Template testDefaultTemplate     `layout:"content" path:"template.txt"`
	Children Slot[*testDefaultChild] `layout:"directories" path:"children"`
}

type testDefaultErrorFile struct {
	Format[string, textCodec]
}

func (f *testDefaultErrorFile) Default() error {
	return fmt.Errorf("default failure")
}

type testDefaultErrorLayout struct {
	Root   Dir                  `layout:"dir" path:"."`
	Broken testDefaultErrorFile `layout:"content" path:"broken.txt"`
}

func assertHasEntry(t *testing.T, entries []Entry, want Entry) {
	t.Helper()

	for _, got := range entries {
		if got.Op == want.Op && got.Path == want.Path && got.Result == want.Result {
			if want.Err == nil || got.Err != nil {
				return
			}
		}
	}

	t.Fatalf("missing entry %#v in %#v", want, entries)
}

type testRenderContext struct {
	Name  string
	Items []string
}

type testTemplateFile struct {
	TextTemplate[testRenderContext]
}

func (f *testTemplateFile) Render() (string, error) {
	ctx, ok := f.GetContext()
	if !ok {
		return "", fmt.Errorf("render context is not set")
	}

	return ctx.Name + ":" + strings.Join(ctx.Items, ","), nil
}

type testTemplatedFile struct {
	TextTemplate[testRenderContext]
}

func (f *testTemplatedFile) Template() string {
	return "{{ .Name }}:{{ range $i, $item := .Items }}{{ if $i }},{{ end }}{{ $item }}{{ end }}"
}

type testBothFile struct {
	TextTemplate[testRenderContext]
}

func (f *testBothFile) Template() string {
	return "template-path"
}

func (f *testBothFile) Render() (string, error) {
	return "custom-path", nil
}

type testBrokenRenderable struct {
	Format[string, textCodec]
	rendered string
}

func (f *testBrokenRenderable) Render() (string, error) {
	return "broken", nil
}

func (f *testBrokenRenderable) SetRendered(value string) {
	f.rendered = value
}

type testRenderChild struct {
	Template testTemplateFile `layout:"content" path:"child.txt"`
}

type testRenderLayout struct {
	Root     Dir                    `layout:"dir" path:"."`
	Template testTemplateFile       `layout:"content" path:"README.md"`
	BuiltIn  testTemplatedFile      `layout:"content" path:"BUILTIN.md"`
	Both     testBothFile           `layout:"content" path:"BOTH.md"`
	Children Slot[*testRenderChild] `layout:"directories" path:"children"`
	Broken   *testBrokenRenderable  `layout:"content" path:"broken.txt"`
}

type testLoadErrorFile struct{ testMapFile }

func (f *testLoadErrorFile) Load() (bool, error) { return false, errors.New("load failed") }

func assertEntries(t *testing.T, got []Entry, want []Entry) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(entries) = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].Op != want[i].Op || got[i].Path != want[i].Path || got[i].Result != want[i].Result {
			t.Fatalf("entry[%d] = %#v, want %#v", i, got[i], want[i])
		}
		if (want[i].Err == nil) != (got[i].Err == nil) {
			t.Fatalf("entry[%d].Err = %v, expected error presence %t", i, got[i].Err, want[i].Err != nil)
		}
	}
}

type testValidatorFile struct {
	testMapFile
	calls *int
	err   error
}

func (f *testValidatorFile) Validate(opts ValidateOptions) error {
	if f.calls != nil {
		*f.calls = *f.calls + 1
	}
	return f.err
}

type testJSONFile struct {
	testMapFile
}
