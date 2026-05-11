package layout

import (
	"fmt"
	"reflect"
)

// ScanDeep
// Scans filesystem according to in-memory semantic structures and compares observed state.
// Observes both sides without mutating either.
// Observational, filesystem presence -> handler state/cache metadata
func ScanDeep(target any, ctx Context) (ResultCode, error) {
	if target == nil {
		return ScanFailed, fmt.Errorf("target must not be nil")
	}

	return scanDeepValue(reflect.ValueOf(target), ctx)
}

func scanDeepValue(v reflect.Value, ctx Context) (ResultCode, error) {
	if !v.IsValid() {
		return 0, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}

		if v.Type().Implements(deepScannerType) {
			result, err := v.Interface().(DeepScanner).ScanDeep(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpScan, path, result, err)
			}
			return result, err
		}

		if v.Type().Implements(scannerType) {
			state, err := v.Interface().(Scannable).Scan()
			result := resultFromDiskState(ScanPresent, ScanMissing, ScanTraversed, state)
			if err != nil {
				result = ScanFailed
			}
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpScan, path, result, err)
			}
			return result, err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepScannerType) {
			result, err := ptr.Interface().(DeepScanner).ScanDeep(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpScan, path, result, err)
			}
			return result, err
		}

		if ptr.Type().Implements(scannerType) {
			state, err := ptr.Interface().(Scannable).Scan()
			result := resultFromDiskState(ScanPresent, ScanMissing, ScanTraversed, state)
			if err != nil {
				result = ScanFailed
			}
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpScan, path, result, err)
			}
			return result, err
		}
	}

	switch v.Type() {
	case dirType, fileType:
		return recordResult(ctx, OpScan, v.Interface().(Pather).Path(), ScanNotApplicable, nil)
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

		if _, err := scanDeepValue(field, ctx); err != nil {
			return ScanFailed, fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return ScanTraversed, nil
}
