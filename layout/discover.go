package layout

import (
	"fmt"
	"reflect"
)

// DiscoverDeep
// Discovers filesystem-backed layout structure without loading typed file content.
// Reflective, filesystem presence -> composed structure and handler state/cache metadata
// - discovers slots from disk
// - scans discovered typed files
// - populates slot caches
// - does not load file content into memory
func DiscoverDeep(target any, ctx Context) error {
	if target == nil {
		return fmt.Errorf("target must not be nil")
	}

	return discoverDeepValue(reflect.ValueOf(target), ctx)
}

func discoverDeepValue(v reflect.Value, ctx Context) error {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		if v.Type().Implements(deepDiscovererType) {
			return v.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
		}

		if v.Type().Implements(discovererType) {
			_, err := v.Interface().(Discoverable).Discover()
			return err
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

		if ptr.Type().Implements(deepDiscovererType) {
			return ptr.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
		}

		if ptr.Type().Implements(discovererType) {
			_, err := ptr.Interface().(Discoverable).Discover()
			return err
		}

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

		if err := discoverDeepValue(field, ctx); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
