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
func LoadDeep(target any, ctx Context) (ResultCode, error) {
	if target == nil {
		return LoadFailed, fmt.Errorf("target must not be nil")
	}
	return loadDeepValue(reflect.ValueOf(target), ctx)
}

func loadDeepValue(v reflect.Value, ctx Context) (ResultCode, error) {
	if !v.IsValid() {
		return 0, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}

		if v.Type().Implements(deepLoaderType) {
			result, err := v.Interface().(DeepLoader).LoadDeep(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpLoad, path, result, err)
			}
			return result, err
		}

		if v.Type().Implements(loaderType) {
			loaded, err := v.Interface().(Loadable).Load()
			result := LoadMissing
			if loaded {
				result = LoadLoaded
			}
			if err != nil {
				result = LoadFailed
			}
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpLoad, path, result, err)
			}
			return result, err
		}
		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepLoaderType) {
			result, err := ptr.Interface().(DeepLoader).LoadDeep(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpLoad, path, result, err)
			}
			return result, err
		}

		if ptr.Type().Implements(loaderType) {
			loaded, err := ptr.Interface().(Loadable).Load()
			result := LoadMissing
			if loaded {
				result = LoadLoaded
			}
			if err != nil {
				result = LoadFailed
			}
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpLoad, path, result, err)
			}
			return result, err
		}
	}

	switch v.Type() {
	case dirType, fileType:
		return recordResult(ctx, OpLoad, v.Interface().(Pather).Path(), LoadNotApplicable, nil)
	}

	if v.Kind() != reflect.Struct {
		return 0, nil
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

		if _, err := loadDeepValue(field, ctx); err != nil {
			return LoadFailed, fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return LoadTraversed, nil
}
