package layout

import (
	"fmt"
	"reflect"
)

// DefaultDeep applies in-memory defaults across a composed or cached layout.
//
// It calls Default behavior on visited nodes without reading from disk,
// discovering new slot entries, or writing anything back to disk.
func DefaultDeep(target any) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	return defaultDeepValue(reflect.ValueOf(target))
}

func defaultDeepValue(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(deepDefaulterType) {
			return v.Interface().(DeepDefaulter).DefaultDeep()
		}

		if v.Type().Implements(defaulterType) {
			return v.Interface().(Defaulter).Default()
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepDefaulterType) {
			return ptr.Interface().(DeepDefaulter).DefaultDeep()
		}

		if ptr.Type().Implements(defaulterType) {
			return ptr.Interface().(Defaulter).Default()
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

		if err := defaultDeepValue(field); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
