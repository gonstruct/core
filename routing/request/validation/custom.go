package validation

import (
	"reflect"
	"regexp"
	"time"

	v10Validator "github.com/go-playground/validator/v10"
	xlanguage "golang.org/x/text/language"
)

func init() {
	validatorV10.RegisterValidation("date_ltefield", dateLteFieldValidation)
	validatorV10.RegisterValidation("date_gtefield", dateGteFieldValidation)
	validatorV10.RegisterValidation("username", usernameValidation)
	validatorV10.RegisterValidation("iso639_1", iso6391Validation)
}

func iso6391Validation(fl v10Validator.FieldLevel) bool {
	value := fl.Field().String()
	if len(value) != 2 {
		return false
	}

	base, err := xlanguage.ParseBase(value)
	return err == nil && base.String() == value
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,255}$`)

func usernameValidation(fl v10Validator.FieldLevel) bool {
	return usernameRegex.MatchString(fl.Field().String())
}

func dateLteFieldValidation(fl v10Validator.FieldLevel) bool {
	currentValue, ok := validationTime(fl.Field())
	if !ok {
		return true
	}

	otherValue, ok := validationTime(fl.Parent().FieldByName(fl.Param()))
	if !ok {
		return true
	}

	return !currentValue.After(*otherValue)
}

func dateGteFieldValidation(fl v10Validator.FieldLevel) bool {
	currentValue, ok := validationTime(fl.Field())
	if !ok {
		return true
	}

	otherValue, ok := validationTime(fl.Parent().FieldByName(fl.Param()))
	if !ok {
		return true
	}

	return !currentValue.Before(*otherValue)
}

func validationTime(field reflect.Value) (*time.Time, bool) {
	if !field.IsValid() {
		return nil, false
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil, false
		}

		value, ok := field.Interface().(*time.Time)
		if !ok {
			return nil, false
		}

		return value, true
	}

	value, ok := field.Interface().(time.Time)
	if !ok {
		return nil, false
	}

	return &value, true
}
