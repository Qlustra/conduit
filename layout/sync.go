package layout

import (
	"fmt"
	"reflect"
)

// SyncDeep
// Writes sync-eligible cached content back to disk. Projects memory onto filesystem.
// Projective, memory cache -> filesystem content
// - walks already composed/cached hierarchy
// - writes typed file contents when the node's sync policy allows its memory state
// - does not ensure raw Dir/File fields outside sync-capable wrappers
// - does not invent missing slot entries unless they are already cached in the slot
// - does not delete anything absent from memory
func SyncDeep(target any, ctx Context) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	return syncDeepValue(reflect.ValueOf(target), ctx)
}

func syncDeepValue(v reflect.Value, ctx Context) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(reflect.TypeOf((*reportDeepSyncer)(nil)).Elem()) {
			return v.Interface().(reportDeepSyncer).syncDeepReport(ctx)
		}

		if v.Type().Implements(deepSyncerType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportSync(ctx, path, func() (ResultCode, error) {
					err := v.Interface().(DeepSyncer).SyncDeep(ctx)
					if err != nil {
						return SyncFailed, err
					}
					return SyncTraversed, nil
				})
			}
			return v.Interface().(DeepSyncer).SyncDeep(ctx)
		}

		if v.Type().Implements(reflect.TypeOf((*reportSyncer)(nil)).Elem()) {
			return v.Interface().(reportSyncer).syncReport(ctx)
		}

		if v.Type().Implements(syncerType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportSync(ctx, path, func() (ResultCode, error) {
					err := v.Interface().(Syncer).Sync(ctx)
					if err != nil {
						return SyncFailed, err
					}
					return SyncTraversed, nil
				})
			}
			return v.Interface().(Syncer).Sync(ctx)
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(reflect.TypeOf((*reportDeepSyncer)(nil)).Elem()) {
			return ptr.Interface().(reportDeepSyncer).syncDeepReport(ctx)
		}

		if ptr.Type().Implements(deepSyncerType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportSync(ctx, path, func() (ResultCode, error) {
					err := ptr.Interface().(DeepSyncer).SyncDeep(ctx)
					if err != nil {
						return SyncFailed, err
					}
					return SyncTraversed, nil
				})
			}
			return ptr.Interface().(DeepSyncer).SyncDeep(ctx)
		}

		if ptr.Type().Implements(reflect.TypeOf((*reportSyncer)(nil)).Elem()) {
			return ptr.Interface().(reportSyncer).syncReport(ctx)
		}

		if ptr.Type().Implements(syncerType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportSync(ctx, path, func() (ResultCode, error) {
					err := ptr.Interface().(Syncer).Sync(ctx)
					if err != nil {
						return SyncFailed, err
					}
					return SyncTraversed, nil
				})
			}
			return ptr.Interface().(Syncer).Sync(ctx)
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

		if err := syncDeepValue(field, ctx); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
