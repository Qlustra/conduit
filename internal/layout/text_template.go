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

type TextTemplate[C any] struct {
	Format[string, textCodec]
	context *C
}

func (f *TextTemplate[C]) ComposePath(path string) {
	f.Format.ComposePath(path)
	f.context = nil
}

func (f *TextTemplate[C]) SetContext(ctx C) {
	f.context = &ctx
}

func (f TextTemplate[C]) GetContext() (C, bool) {
	if f.context == nil {
		var zero C
		return zero, false
	}
	return *f.context, true
}

func (f *TextTemplate[C]) MustContext() C {
	if f.context == nil {
		panic("render context is not set")
	}
	return *f.context
}

func (f TextTemplate[C]) HasContext() bool {
	return f.context != nil
}

func (f *TextTemplate[C]) ClearContext() {
	f.context = nil
}

func (f *TextTemplate[C]) SetRendered(value string) {
	f.Set(value)
}

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
