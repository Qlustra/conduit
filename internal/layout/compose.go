package layout

import (
	"fmt"
	"path/filepath"
	"reflect"
)

// Compose
// Builds path-bound semantic objects.
func Compose(root string, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to struct")
	}

	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct")
	}

	return composeInto(filepath.Clean(root), elem)
}

func composeInto(base string, dst reflect.Value) error {
	t := dst.Type()

	for i := 0; i < dst.NumField(); i++ {
		field := dst.Field(i)
		structField := t.Field(i)

		if structField.PkgPath != "" {
			continue
		}

		tag := structField.Tag.Get("layout")
		if tag == "" {
			continue
		}

		path := resolvePath(base, tag)

		if err := assignPath(path, field); err != nil {
			return fmt.Errorf("field %q: %w", structField.Name, err)
		}
	}

	return nil
}

func assignPath(path string, field reflect.Value) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	if field.Kind() == reflect.Struct {
		ptr := field.Addr()

		if ptr.Type().Implements(composableEntryType) {
			ptr.Interface().(Composable).ComposePath(path)
			return nil
		}

		return composeInto(path, field)
	}

	if field.Kind() == reflect.Pointer {
		elemType := field.Type().Elem()

		if elemType.Kind() != reflect.Struct {
			return fmt.Errorf("unsupported pointer target type %s", elemType)
		}

		if field.IsNil() {
			field.Set(reflect.New(elemType))
		}

		if field.Type().Implements(composableEntryType) {
			field.Interface().(Composable).ComposePath(path)
			return nil
		}

		return composeInto(path, field.Elem())
	}

	return fmt.Errorf("unsupported field type %s", field.Type())
}

func resolvePath(base string, tag string) string {
	switch tag {
	case ".", "":
		return base
	default:
		return filepath.Join(base, tag)
	}
}

func ComposeAs[T any](root Dir) (T, error) {
	var zero T

	typ := reflect.TypeOf((*T)(nil)).Elem()

	if typ.Kind() == reflect.Pointer {
		elem := typ.Elem()
		if elem.Kind() != reflect.Struct {
			return zero, fmt.Errorf("T must be struct or pointer to struct, got %s", typ)
		}

		v := reflect.New(elem)
		if err := Compose(root.Path(), v.Interface()); err != nil {
			return zero, err
		}

		return v.Interface().(T), nil
	}

	if typ.Kind() != reflect.Struct {
		return zero, fmt.Errorf("T must be struct or pointer to struct, got %s", typ)
	}

	v := reflect.New(typ)
	if err := Compose(root.Path(), v.Interface()); err != nil {
		return zero, err
	}

	return v.Elem().Interface().(T), nil
}
