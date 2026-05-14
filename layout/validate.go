package layout

import (
	"fmt"
	"reflect"
)

// ValidateDeep validates a composed or cached layout without mutating disk or
// memory state.
//
// It walks already composed or cached children, invokes Validate or
// ValidateDeep when implemented, and records path-level validation outcomes
// through opts.Reporter when provided. ValidateDeep does not ensure files,
// discover new slot entries, load content, render templates, or sync changes.
func ValidateDeep(target any, opts ValidateOptions) (ResultCode, error) {
	if target == nil {
		return ValidateFailed, fmt.Errorf("target must not be nil")
	}

	return validateDeepValue(reflect.ValueOf(target), opts)
}

func validateDeepValue(v reflect.Value, opts ValidateOptions) (ResultCode, error) {
	if !v.IsValid() {
		return 0, nil
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}

		if v.Type().Implements(deepValidatorType) {
			result, err := v.Interface().(DeepValidator).ValidateDeep(opts)
			if path, ok := pathOf(v.Interface()); ok {
				return recordValidateResult(opts, OpValidate, path, result, err)
			}
			return result, err
		}

		if v.Type().Implements(validatorType) {
			err := v.Interface().(Validator).Validate(opts)
			result := ValidateOK
			if err != nil {
				result = ValidateFailed
			}
			if path, ok := pathOf(v.Interface()); ok {
				return recordValidateResult(opts, OpValidate, path, result, err)
			}
			return result, err
		}

		v = v.Elem()
	}

	if v.CanAddr() {
		ptr := v.Addr()

		if ptr.Type().Implements(deepValidatorType) {
			result, err := ptr.Interface().(DeepValidator).ValidateDeep(opts)
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordValidateResult(opts, OpValidate, path, result, err)
			}
			return result, err
		}

		if ptr.Type().Implements(validatorType) {
			err := ptr.Interface().(Validator).Validate(opts)
			result := ValidateOK
			if err != nil {
				result = ValidateFailed
			}
			if path, ok := pathOf(ptr.Interface()); ok {
				return recordValidateResult(opts, OpValidate, path, result, err)
			}
			return result, err
		}
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

		if _, err := validateDeepValue(field, opts); err != nil {
			return ValidateFailed, fmt.Errorf("field %q: %w", sf.Name, err)
		}
	}

	return ValidateTraversed, nil
}
