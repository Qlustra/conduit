package layout

import (
	"fmt"
	"reflect"
)

// EnsureDeep materializes declared filesystem structure for a composed layout.
//
// It creates raw Dir, File, and Exec nodes and recurses through already
// composed or cached children. ctx.EnsurePolicy can narrow which node kinds
// are materialized. EnsureDeep does not load typed content, discover new slot
// entries from disk, write typed file state, or delete anything.
func EnsureDeep(target any, ctx Context) (ResultCode, error) {
	if target == nil {
		return EnsureFailed, fmt.Errorf("target must not be nil")
	}

	return ensureDeepValue(reflect.ValueOf(target), ctx)
}

func ensureDeepValue(v reflect.Value, ctx Context) (ResultCode, error) {
	if !v.IsValid() {
		return 0, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}
		if v.Type().Implements(deepEnsurerType) {
			result, err := v.Interface().(DeepEnsurer).EnsureDeep(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpEnsure, path, result, err)
			}
			return result, err
		}
		if v.Type().Implements(syncerType) && !ctx.ensurePolicy().allowsSyncable() {
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpEnsure, path, EnsureSkippedPolicy, nil)
			}
			return EnsureSkippedPolicy, nil
		}
		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()
		if ptr.Type().Implements(deepEnsurerType) {
			result, err := ptr.Interface().(DeepEnsurer).EnsureDeep(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpEnsure, path, result, err)
			}
			return result, err
		}
		if ptr.Type().Implements(syncerType) && !ctx.ensurePolicy().allowsSyncable() {
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpEnsure, path, EnsureSkippedPolicy, nil)
			}
			return EnsureSkippedPolicy, nil
		}
	}

	switch v.Type() {
	case dirType:
		if !ctx.ensurePolicy().allowsDir() {
			return recordResult(ctx, OpEnsure, v.Interface().(Dir).Path(), EnsureSkippedPolicy, nil)
		}
		err := v.Interface().(Dir).Ensure(ctx)
		result := EnsureEnsured
		if err != nil {
			result = EnsureFailed
		}
		return recordResult(ctx, OpEnsure, v.Interface().(Dir).Path(), result, err)

	case fileType:
		if !ctx.ensurePolicy().allowsFile() {
			return recordResult(ctx, OpEnsure, v.Interface().(File).Path(), EnsureSkippedPolicy, nil)
		}
		err := v.Interface().(File).Ensure(ctx)
		result := EnsureEnsured
		if err != nil {
			result = EnsureFailed
		}
		return recordResult(ctx, OpEnsure, v.Interface().(File).Path(), result, err)
	}

	// If this is a struct that embeds Dir/File or wraps them, recurse into fields.
	if v.Kind() != reflect.Struct {
		return 0, nil
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

		if _, err := ensureDeepValue(field, ctx); err != nil {
			return EnsureFailed, fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return EnsureEnsured, nil
}
