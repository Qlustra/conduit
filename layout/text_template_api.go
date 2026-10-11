package layout

// TextTemplateAPI is the complete exported API of TextTemplate for render context C.
type TextTemplateAPI[C any] interface {
	FormatAPI[string]
	ClearContext()
	GetContext() (C, bool)
	HasContext() bool
	MustContext() C
	RenderTemplate(tpl string) (string, error)
	SetContext(ctx C)
	SetDefaultContext(ctx C) bool
	SetRendered(value string)
}
