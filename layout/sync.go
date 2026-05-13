package layout

import (
	"fmt"
	"reflect"
)

// SyncDeep writes sync-eligible cached content from memory back to disk.
//
// It recurses through already composed or cached children and delegates to
// stateful nodes whose Sync behavior is allowed by ctx.SyncPolicy. SyncDeep
// does not ensure standalone raw Dir or File fields, discover uncached slot
// entries, or delete anything absent from memory.
func SyncDeep(target any, ctx Context) (ResultCode, error) {
	if target == nil {
		return SyncFailed, fmt.Errorf("target must not be nil")
	}
	return syncDeepValue(reflect.ValueOf(target), ctx)
}

func syncDeepValue(v reflect.Value, ctx Context) (ResultCode, error) {
	if !v.IsValid() {
		return 0, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}

		if v.Type().Implements(deepSyncerType) {
			result, err := v.Interface().(DeepSyncer).SyncDeep(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpSync, path, result, err)
			}
			return result, err
		}

		if v.Type().Implements(syncerType) {
			result, err := v.Interface().(Syncer).Sync(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpSync, path, result, err)
			}
			return result, err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepSyncerType) {
			result, err := ptr.Interface().(DeepSyncer).SyncDeep(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpSync, path, result, err)
			}
			return result, err
		}

		if ptr.Type().Implements(syncerType) {
			result, err := ptr.Interface().(Syncer).Sync(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpSync, path, result, err)
			}
			return result, err
		}
	}

	switch v.Type() {
	case dirType, fileType:
		return recordResult(ctx, OpSync, v.Interface().(Pather).Path(), SyncNotApplicable, nil)
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

		if _, err := syncDeepValue(field, ctx); err != nil {
			return SyncFailed, fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return SyncTraversed, nil
}
