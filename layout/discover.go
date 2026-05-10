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

		if v.Type().Implements(reflect.TypeOf((*reportDeepDiscoverer)(nil)).Elem()) {
			return v.Interface().(reportDeepDiscoverer).discoverDeepReport(ctx)
		}

		if v.Type().Implements(deepDiscovererType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					err := v.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
					if err != nil {
						return DiscoverFailed, err
					}
					return DiscoverTraversed, nil
				})
			}
			return v.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
		}

		if v.Type().Implements(reflect.TypeOf((*reportDiscoverer)(nil)).Elem()) {
			return v.Interface().(reportDiscoverer).discoverReport(ctx)
		}

		if v.Type().Implements(discovererType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					state, err := v.Interface().(Discoverable).Discover()
					if err != nil {
						return DiscoverFailed, err
					}
					return resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state), nil
				})
			}
			_, err := v.Interface().(Discoverable).Discover()
			return err
		}

		if v.Type().Implements(deepScannerType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					err := v.Interface().(DeepScanner).ScanDeep(ctx)
					if err != nil {
						return DiscoverFailed, err
					}
					return DiscoverTraversed, nil
				})
			}
			return v.Interface().(DeepScanner).ScanDeep(ctx)
		}

		if v.Type().Implements(scannerType) {
			if path, ok := pathOf(v.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					state, err := v.Interface().(Scannable).Scan()
					if err != nil {
						return DiscoverFailed, err
					}
					return resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state), nil
				})
			}
			_, err := v.Interface().(Scannable).Scan()
			return err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(reflect.TypeOf((*reportDeepDiscoverer)(nil)).Elem()) {
			return ptr.Interface().(reportDeepDiscoverer).discoverDeepReport(ctx)
		}

		if ptr.Type().Implements(deepDiscovererType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					err := ptr.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
					if err != nil {
						return DiscoverFailed, err
					}
					return DiscoverTraversed, nil
				})
			}
			return ptr.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
		}

		if ptr.Type().Implements(reflect.TypeOf((*reportDiscoverer)(nil)).Elem()) {
			return ptr.Interface().(reportDiscoverer).discoverReport(ctx)
		}

		if ptr.Type().Implements(discovererType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					state, err := ptr.Interface().(Discoverable).Discover()
					if err != nil {
						return DiscoverFailed, err
					}
					return resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state), nil
				})
			}
			_, err := ptr.Interface().(Discoverable).Discover()
			return err
		}

		if ptr.Type().Implements(deepScannerType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					err := ptr.Interface().(DeepScanner).ScanDeep(ctx)
					if err != nil {
						return DiscoverFailed, err
					}
					return DiscoverTraversed, nil
				})
			}
			return ptr.Interface().(DeepScanner).ScanDeep(ctx)
		}

		if ptr.Type().Implements(scannerType) {
			if path, ok := pathOf(ptr.Interface()); ok {
				return reportDiscover(ctx, path, func() (ResultCode, error) {
					state, err := ptr.Interface().(Scannable).Scan()
					if err != nil {
						return DiscoverFailed, err
					}
					return resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state), nil
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

		if err := discoverDeepValue(field, ctx); err != nil {
			return fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return nil
}
