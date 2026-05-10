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
func LoadDeep(target any, ctx Context) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	return loadDeepValue(reflect.ValueOf(target), ctx)
}

func loadDeepValue(v reflect.Value, ctx Context) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(reflect.TypeOf((*reportDeepLoader)(nil)).Elem()) {
			return v.Interface().(reportDeepLoader).loadDeepReport(ctx)
		}

		if v.Type().Implements(deepLoaderType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportLoad(ctx, path, func() (ResultCode, error) {
					err := v.Interface().(DeepLoader).LoadDeep(ctx)
					if err != nil {
						return LoadFailed, err
					}
					return LoadTraversed, nil
				})
			}
			return v.Interface().(DeepLoader).LoadDeep(ctx)
		}

		if v.Type().Implements(reflect.TypeOf((*reportLoader)(nil)).Elem()) {
			return v.Interface().(reportLoader).loadReport(ctx)
		}

		if v.Type().Implements(loaderType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportLoad(ctx, path, func() (ResultCode, error) {
					loaded, err := v.Interface().(Loadable).Load()
					if err != nil {
						return LoadFailed, err
					}
					if loaded {
						return LoadLoaded, nil
					}
					return LoadMissing, nil
				})
			}
			_, err := v.Interface().(Loadable).Load()
			return err
		}
		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(reflect.TypeOf((*reportDeepLoader)(nil)).Elem()) {
			return ptr.Interface().(reportDeepLoader).loadDeepReport(ctx)
		}

		if ptr.Type().Implements(deepLoaderType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportLoad(ctx, path, func() (ResultCode, error) {
					err := ptr.Interface().(DeepLoader).LoadDeep(ctx)
					if err != nil {
						return LoadFailed, err
					}
					return LoadTraversed, nil
				})
			}
			return ptr.Interface().(DeepLoader).LoadDeep(ctx)
		}

		if ptr.Type().Implements(reflect.TypeOf((*reportLoader)(nil)).Elem()) {
			return ptr.Interface().(reportLoader).loadReport(ctx)
		}

		if ptr.Type().Implements(loaderType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportLoad(ctx, path, func() (ResultCode, error) {
					loaded, err := ptr.Interface().(Loadable).Load()
					if err != nil {
						return LoadFailed, err
					}
					if loaded {
						return LoadLoaded, nil
					}
					return LoadMissing, nil
				})
			}
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

		if err := loadDeepValue(field, ctx); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
