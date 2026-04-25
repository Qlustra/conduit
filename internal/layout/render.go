package layout

import (
	"fmt"
	"reflect"
)

// RenderDeep
// Derives renderable text content into memory for already composed/cached items.
// Reflective, render state -> text cache
// - walks already composed/cached hierarchy
// - renders text-backed template wrappers
// - does not discover slot entries
// - does not write to disk
func RenderDeep(target any) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	return renderDeepValue(reflect.ValueOf(target))
}

func renderDeepValue(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(deepRendererType) {
			return v.Interface().(DeepRenderer).RenderDeep()
		}

		if v.Type().Implements(renderableType) {
			return renderValue(v)
		}

		if v.Type().Implements(templatableType) {
			return renderTemplateValue(v)
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepRendererType) {
			return ptr.Interface().(DeepRenderer).RenderDeep()
		}

		if ptr.Type().Implements(renderableType) {
			return renderValue(ptr)
		}

		if ptr.Type().Implements(templatableType) {
			return renderTemplateValue(ptr)
		}
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if sf.PkgPath != "" {
			continue
		}

		if sf.Tag.Get("layout") == "" && !sf.Anonymous {
			continue
		}

		if err := renderDeepValue(field); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}

func renderValue(v reflect.Value) error {
	rendered, err := v.Interface().(Renderable).Render()
	if err != nil {
		return err
	}

	v.Interface().(Renderable).SetRendered(rendered)
	return nil
}

func renderTemplateValue(v reflect.Value) error {
	templated := v.Interface().(Templatable)

	rendered, err := templated.RenderTemplate(templated.Template())
	if err != nil {
		return err
	}

	templated.SetRendered(rendered)
	return nil
}
