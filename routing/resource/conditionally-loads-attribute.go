package resource

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

func When[T any](value bool, then func() T, fallback ...any) any {
	if value {
		return then()
	}
	if len(fallback) == 1 {
		return fallback[0]
	}

	return missingValue{}
}

func WhenLoaded[T, R any](getter func() T, converter func(T) R, fallback ...any) any {
	value := getter()

	v := reflect.ValueOf(value)
	if v.IsNil() {
		if len(fallback) == 1 {
			return fallback[0]
		}

		if v.Type().Kind() == reflect.Slice {
			return []any{}
		}

		return missingValue{}
	}

	converted := converter(value)

	if value, ok := any(converted).(ResourceInterface); ok {
		return value.Attributes()
	}

	return converted
}

func WhenLoadedV2[T, R any](context *gin.Context, getter func() T, converter func(T) R, fallback ...any) any {
	value := getter()

	v := reflect.ValueOf(value)
	if v.IsNil() {
		if len(fallback) == 1 {
			return fallback[0]
		}

		if v.Type().Kind() == reflect.Slice {
			return []any{}
		}

		return missingValue{}
	}

	converted := converter(value)

	if value, ok := any(converted).(V2ResourceInterface); ok {
		return value.Attributes(context)
	}

	return converted
}

func WhenNotNil[T any](value T, do func(option T) any, fallback ...any) any {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.IsZero() {
		if len(fallback) == 1 {
			return fallback[0]
		}
		return missingValue{}
	}

	isNotEmpty := func() bool {
		switch reflected.Kind() {
		case reflect.String:
			return reflected.Len() > 0
		case reflect.Slice, reflect.Map, reflect.Array, reflect.Chan:
			return !reflected.IsNil() && reflected.Len() > 0
		case reflect.Pointer, reflect.Interface:
			return !reflected.IsNil()
		default:
			return true
		}
	}()

	if isNotEmpty {
		return do(value)
	}

	if len(fallback) == 1 {
		return fallback[0]
	}

	return missingValue{}
}
