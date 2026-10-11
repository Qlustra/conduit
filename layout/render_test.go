package layout

import (
	"strings"
	"testing"
)

type testRenderContext struct {
	Name  string
	Items []string
}

type testTemplatedFile struct {
	TextTemplate[testRenderContext]
}

func (f *testTemplatedFile) Template() string {
	return "{{ .Name }}:{{ range $i, $item := .Items }}{{ if $i }},{{ end }}{{ $item }}{{ end }}"
}

func TestTextTemplateRenderTemplateReportsTemplateErrors(t *testing.T) {
	var f testTemplatedFile
	f.SetContext(testRenderContext{Name: "root"})

	_, err := f.RenderTemplate("{{ .Missing }}")
	if err == nil {
		t.Fatal("RenderTemplate() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "can't evaluate field Missing") {
		t.Fatalf("RenderTemplate() error = %q, want message containing %q", got, "can't evaluate field Missing")
	}
}
