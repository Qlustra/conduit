package layout

import (
	"fmt"
	"reflect"
)

// ScanDeep
// Scans filesystem according to in-memory semantic structures and compares observed state.
// Observes both sides without mutating either.
// Observational, filesystem presence -> handler state/cache metadata
func ScanDeep(target any, ctx Context) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}

	return scanDeepValue(reflect.ValueOf(target), ctx)
}

func scanDeepValue(v reflect.Value, ctx Context) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(reflect.TypeOf((*reportDeepScanner)(nil)).Elem()) {
			return v.Interface().(reportDeepScanner).scanDeepReport(ctx)
		}

		if v.Type().Implements(deepScannerType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportScan(ctx, path, func() (ResultCode, error) {
					err := v.Interface().(DeepScanner).ScanDeep(ctx)
					if err != nil {
						return ScanFailed, err
					}
					return ScanTraversed, nil
				})
			}
			return v.Interface().(DeepScanner).ScanDeep(ctx)
		}

		if v.Type().Implements(reflect.TypeOf((*reportScanner)(nil)).Elem()) {
			return v.Interface().(reportScanner).scanReport(ctx)
		}

		if v.Type().Implements(scannerType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportScan(ctx, path, func() (ResultCode, error) {
					state, err := v.Interface().(Scannable).Scan()
					if err != nil {
						return ScanFailed, err
					}
					return resultFromDiskState(ScanPresent, ScanMissing, ScanTraversed, state), nil
				})
			}
			_, err := v.Interface().(Scannable).Scan()
			return err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(reflect.TypeOf((*reportDeepScanner)(nil)).Elem()) {
			return ptr.Interface().(reportDeepScanner).scanDeepReport(ctx)
		}

		if ptr.Type().Implements(deepScannerType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportScan(ctx, path, func() (ResultCode, error) {
					err := ptr.Interface().(DeepScanner).ScanDeep(ctx)
					if err != nil {
						return ScanFailed, err
					}
					return ScanTraversed, nil
				})
			}
			return ptr.Interface().(DeepScanner).ScanDeep(ctx)
		}

		if ptr.Type().Implements(reflect.TypeOf((*reportScanner)(nil)).Elem()) {
			return ptr.Interface().(reportScanner).scanReport(ctx)
		}

		if ptr.Type().Implements(scannerType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportScan(ctx, path, func() (ResultCode, error) {
					state, err := ptr.Interface().(Scannable).Scan()
					if err != nil {
						return ScanFailed, err
					}
					return resultFromDiskState(ScanPresent, ScanMissing, ScanTraversed, state), nil
				})
			}
			_, err := ptr.Interface().(Scannable).Scan()
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

		if err := scanDeepValue(field, ctx); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
