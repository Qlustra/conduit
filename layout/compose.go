package layout

import (
	"fmt"
	"path/filepath"
	"reflect"
)

// Compose
// Builds path-bound semantic objects.
func Compose(root string, target any) error {
	root = filepath.Clean(root)
	return compose(root, root, target)
}

func compose(root string, composeBase string, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to struct")
	}

	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct")
	}

	return composeInto(root, composeBase, elem)
}

func composeInto(base string, composeBase string, dst reflect.Value) error {
	t := dst.Type()
	composeBaseAwareType := reflect.TypeOf((*composeBaseAware)(nil)).Elem()
	declaredPathAwareType := reflect.TypeOf((*declaredPathAware)(nil)).Elem()

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

		if err := assignPath(path, composeBase, tag, field, composeBaseAwareType, declaredPathAwareType); err != nil {
			return fmt.Errorf("field %q: %w", structField.Name, err)
		}
	}

	return nil
}

func assignPath(path string, composeBase string, declaredPath string, field reflect.Value, composeBaseAwareType reflect.Type, declaredPathAwareType reflect.Type) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	if field.Kind() == reflect.Struct {
		ptr := field.Addr()

		if ptr.Type().Implements(composableEntryType) {
			ptr.Interface().(Composable).ComposePath(path)
			if ptr.Type().Implements(composeBaseAwareType) {
				ptr.Interface().(composeBaseAware).setComposeBase(composeBase)
			}
			if ptr.Type().Implements(declaredPathAwareType) {
				ptr.Interface().(declaredPathAware).setDeclaredPath(declaredPath)
			}
			return nil
		}

		return composeInto(path, composeBase, field)
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
			if field.Type().Implements(composeBaseAwareType) {
				field.Interface().(composeBaseAware).setComposeBase(composeBase)
			}
			if field.Type().Implements(declaredPathAwareType) {
				field.Interface().(declaredPathAware).setDeclaredPath(declaredPath)
			}
			return nil
		}

		return composeInto(path, composeBase, field.Elem())
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

func composePathAs[T any](path string, composeBase string) (T, error) {
	var zero T

	typ := reflect.TypeOf((*T)(nil)).Elem()
	composeBaseAwareType := reflect.TypeOf((*composeBaseAware)(nil)).Elem()

	if typ.Kind() == reflect.Pointer {
		elem := typ.Elem()
		if elem.Kind() != reflect.Struct {
			return zero, fmt.Errorf("T must be struct or pointer to struct, got %s", typ)
		}

		v := reflect.New(elem)
		if v.Type().Implements(composableEntryType) {
			v.Interface().(Composable).ComposePath(path)
			if v.Type().Implements(composeBaseAwareType) {
				v.Interface().(composeBaseAware).setComposeBase(composeBase)
			}
			return v.Interface().(T), nil
		}

		if err := compose(path, composeBase, v.Interface()); err != nil {
			return zero, err
		}

		return v.Interface().(T), nil
	}

	if typ.Kind() != reflect.Struct {
		return zero, fmt.Errorf("T must be struct or pointer to struct, got %s", typ)
	}

	v := reflect.New(typ)
	if v.Type().Implements(composableEntryType) {
		v.Interface().(Composable).ComposePath(path)
		if v.Type().Implements(composeBaseAwareType) {
			v.Interface().(composeBaseAware).setComposeBase(composeBase)
		}
		return v.Elem().Interface().(T), nil
	}

	if err := compose(path, composeBase, v.Interface()); err != nil {
		return zero, err
	}

	return v.Elem().Interface().(T), nil
}

func ComposeAs[T any](root Dir) (T, error) {
	composeBase := root.Path()
	if base, ok := root.ComposedBaseDir(); ok {
		composeBase = base.Path()
	}
	return composePathAs[T](root.Path(), composeBase)
}
