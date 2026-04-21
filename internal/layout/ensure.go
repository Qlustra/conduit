package layout

import (
	"fmt"
	"reflect"
)

// EnsureDeep
// Materializes declared structure.
// Constructive, memory shape -> filesystem structure
func EnsureDeep(target any, ctx Context) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}

	return ensureDeepValue(reflect.ValueOf(target), ctx)
}

func ensureDeepValue(v reflect.Value, ctx Context) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		if v.Type().Implements(deepEnsurerType) {
			return v.Interface().(DeepEnsurer).EnsureDeep(ctx)
		}
		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()
		if ptr.Type().Implements(deepEnsurerType) {
			return ptr.Interface().(DeepEnsurer).EnsureDeep(ctx)
		}
	}

	switch v.Type() {
	case dirType:
		return v.Interface().(Dir).Ensure(ctx)

	case fileType:
		return v.Interface().(File).Ensure(ctx)
	}

	// If this is a struct that embeds Dir/File or wraps them, recurse into fields.
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		// Skip unexported fields.
		if sf.PkgPath != "" {
			continue
		}

		// Recurse only into declared layout fields, plus embedded fields.
		// That keeps us aligned with the composition model.
		if sf.Tag.Get("layout") == "" && !sf.Anonymous {
			continue
		}

		if err := ensureDeepValue(field, ctx); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
