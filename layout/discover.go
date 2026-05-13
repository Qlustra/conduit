package layout

import (
	"fmt"
	"reflect"
)

// DiscoverDeep discovers composed layout structure from disk without loading
// typed file content.
//
// It discovers slot-backed children from disk, composes them, and refreshes
// disk-state metadata for discovered typed files. In-memory typed content is
// preserved. DiscoverDeep does not create files, write files, or replace
// cached typed values from disk.
func DiscoverDeep(target any, ctx Context) (ResultCode, error) {
	if target == nil {
		return DiscoverFailed, fmt.Errorf("target must not be nil")
	}

	return discoverDeepValue(reflect.ValueOf(target), ctx)
}

func discoverDeepValue(v reflect.Value, ctx Context) (ResultCode, error) {
	if !v.IsValid() {
		return 0, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}

		if v.Type().Implements(deepDiscovererType) {
			result, err := v.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, result, err)
			}
			return result, err
		}

		if v.Type().Implements(discovererType) {
			state, err := v.Interface().(Discoverable).Discover()
			result := resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state)
			if err != nil {
				result = DiscoverFailed
			}
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, result, err)
			}
			return result, err
		}

		if v.Type().Implements(deepScannerType) {
			result, err := v.Interface().(DeepScanner).ScanDeep(ctx)
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, resultForDiscoverFromScanResult(result, err), err)
			}
			return resultForDiscoverFromScanResult(result, err), err
		}

		if v.Type().Implements(scannerType) {
			state, err := v.Interface().(Scannable).Scan()
			result := resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state)
			if err != nil {
				result = DiscoverFailed
			}
			if path, ok := pathOf(v.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, result, err)
			}
			return result, err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepDiscovererType) {
			result, err := ptr.Interface().(DeepDiscoverer).DiscoverDeep(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, result, err)
			}
			return result, err
		}

		if ptr.Type().Implements(discovererType) {
			state, err := ptr.Interface().(Discoverable).Discover()
			result := resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state)
			if err != nil {
				result = DiscoverFailed
			}
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, result, err)
			}
			return result, err
		}

		if ptr.Type().Implements(deepScannerType) {
			result, err := ptr.Interface().(DeepScanner).ScanDeep(ctx)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, resultForDiscoverFromScanResult(result, err), err)
			}
			return resultForDiscoverFromScanResult(result, err), err
		}

		if ptr.Type().Implements(scannerType) {
			state, err := ptr.Interface().(Scannable).Scan()
			result := resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state)
			if err != nil {
				result = DiscoverFailed
			}
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordResult(ctx, OpDiscover, path, result, err)
			}
			return result, err
		}
	}

	switch v.Type() {
	case dirType, fileType:
		return recordResult(ctx, OpDiscover, v.Interface().(Pather).Path(), DiscoverNotApplicable, nil)
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

		if _, err := discoverDeepValue(field, ctx); err != nil {
			return DiscoverFailed, fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return DiscoverTraversed, nil
}
