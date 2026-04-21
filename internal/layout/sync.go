package layout

import (
	"fmt"
	"reflect"
)

// SyncDeep
// Writes cached content back to disk. Projects memory onto filesystem.
// Projective, memory cache -> filesystem content
// - walks already composed/cached hierarchy
// - writes loaded/cached typed file contents
// - ensures parent structure as needed
// - does not invent missing slot entries unless they are already cached in the slot
// - does not delete anything absent from memory
func SyncDeep(target any) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}
	return syncDeepValue(reflect.ValueOf(target))
}

func syncDeepValue(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(deepSyncerType) {
			return v.Interface().(DeepSyncer).SyncDeep()
		}

		if v.Type().Implements(syncerType) {
			return v.Interface().(Syncer).Sync()
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepSyncerType) {
			return ptr.Interface().(DeepSyncer).SyncDeep()
		}

		if ptr.Type().Implements(syncerType) {
			return ptr.Interface().(Syncer).Sync()
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

		if err := syncDeepValue(field); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
