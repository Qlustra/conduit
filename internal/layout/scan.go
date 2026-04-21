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

		if v.Type().Implements(deepScannerType) {
			return v.Interface().(DeepScanner).ScanDeep(ctx)
		}

		if v.Type().Implements(scannerType) {
			_, err := v.Interface().(Scannable).Scan()
			return err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepScannerType) {
			return ptr.Interface().(DeepScanner).ScanDeep(ctx)
		}

		if ptr.Type().Implements(scannerType) {
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
