package layout

import (
	"fmt"
	"reflect"
)

// LoadDeep
// Populates cached content from disk. Reflects filesystem into memory.
// Reflective, filesystem content -> memory cache
// - scans slots
// - loads discovered typed files
// - populates caches
// - does not create missing files
func LoadDeep(target any) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	return loadDeepValue(reflect.ValueOf(target))
}

func loadDeepValue(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(deepLoaderType) {
			return v.Interface().(DeepLoader).LoadDeep()
		}

		if v.Type().Implements(loaderType) {
			_, err := v.Interface().(Loadable).Load()
			return err
		}
		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepLoaderType) {
			return ptr.Interface().(DeepLoader).LoadDeep()
		}

		if ptr.Type().Implements(loaderType) {
			_, err := ptr.Interface().(Loadable).Load()
			return err
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

		if err := loadDeepValue(field); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
