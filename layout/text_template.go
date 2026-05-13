package layout

import (
	"bytes"
	texttemplate "text/template"
)

type textCodec struct{}

func (textCodec) Marshal(value string) ([]byte, error) {
	return []byte(value), nil
}

func (textCodec) Unmarshal(data []byte) (string, error) {
	return string(data), nil
}

// TextTemplate is a text-producing file wrapper with cached render context.
//
// It uses the same disk and memory state model as Format[string, ...] and adds
// a separate in-memory render context used by RenderDeep or custom render
// flows.
type TextTemplate[C any] struct {
	Format[string, textCodec]
	context *C
}

// ComposePath binds the template file to path and clears cached render
// context.
func (f *TextTemplate[C]) ComposePath(path string) {
	f.Format.ComposePath(path)
	f.context = nil
}

// SetContext stores the render context in memory.
func (f *TextTemplate[C]) SetContext(ctx C) {
	f.context = &ctx
}

// SetDefaultContext stores ctx only when no render context is currently set.
//
// It returns whether the default was applied.
func (f *TextTemplate[C]) SetDefaultContext(ctx C) bool {
	if f.context != nil {
		return false
	}
	f.SetContext(ctx)
	return true
}

// GetContext returns the cached render context, if any.
func (f TextTemplate[C]) GetContext() (C, bool) {
	if f.context == nil {
		var zero C
		return zero, false
	}
	return *f.context, true
}

// MustContext returns the cached render context or panics when it is absent.
func (f *TextTemplate[C]) MustContext() C {
	if f.context == nil {
		panic("render context is not set")
	}
	return *f.context
}

// HasContext reports whether a render context is currently cached.
func (f TextTemplate[C]) HasContext() bool {
	return f.context != nil
}

// ClearContext removes the cached render context.
func (f *TextTemplate[C]) ClearContext() {
	f.context = nil
}

// SetRendered stores rendered text as the cached file content.
func (f *TextTemplate[C]) SetRendered(value string) {
	f.Set(value)
}

// RenderTemplate renders tpl using the cached context through text/template.
//
// It uses the template option missingkey=error.
func (f *TextTemplate[C]) RenderTemplate(tpl string) (string, error) {
	parsed, err := texttemplate.New("conduit").Option("missingkey=error").Parse(tpl)
	if err != nil {
		return "", err
	}

	var data any
	if f.context != nil {
		data = *f.context
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
