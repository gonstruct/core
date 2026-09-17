package validation

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gonstruct/core/routing/response"

	v10Validator "github.com/go-playground/validator/v10"
	"github.com/iancoleman/strcase"
)

var validatorV10 = v10Validator.New(v10Validator.WithRequiredStructEnabled())

func init() {
	validatorV10.RegisterTagNameFunc(func(field reflect.StructField) string {
		if value, ok := field.Tag.Lookup("json"); ok {
			return value
		}

		if value, ok := field.Tag.Lookup("form"); ok {
			return value
		}

		if value, ok := field.Tag.Lookup("uri"); ok {
			return value
		}

		return strcase.ToDelimited(field.Name, '_')
	})
}

type requestContext interface {
	BindJson(obj any) error
	BindUri(obj any) error
	BindQuery(obj any) error
	GetRequest() *http.Request
}

type Result[T any] func(binder string) (T, response.Response)

func Validator[T any, C requestContext](request C) Result[T] {
	return func(binder string) (T, response.Response) {
		return validator_v10[T](request, binder)
	}
}

func validator_v10[T any, C requestContext](request C, binder string) (T, response.Response) {
	var result T

	// A body that will not bind is the client's mistake, not ours. Left as
	// the default 500 it marks the span failed, force-keeps the trace and
	// files an error issue — for a bot sending a letter where a number
	// goes.
	switch binder {
	case "json":
		if err := request.BindJson(&result); err != nil {
			return result, response.Error(err, response.WithMessage("No valid json body provided"), response.WithStatusCode(http.StatusBadRequest))
		}
	case "params":
		if err := request.BindUri(&result); err != nil {
			return result, response.Error(err, response.WithMessage("No valid uri parameters provided"), response.WithStatusCode(http.StatusBadRequest))
		}
	case "query":
		if err := request.BindQuery(&result); err != nil {
			return result, response.Error(err, response.WithMessage("No valid query parameters provided"), response.WithStatusCode(http.StatusBadRequest))
		}
	default:
		return result, response.Error(fmt.Errorf("unknown binder: %s", binder),
			response.WithMessage("Failed to validate request object"),
		)
	}

	if err := validatorV10.Struct(result); err != nil {
		var validationErrors v10Validator.ValidationErrors
		if !errors.As(err, &validationErrors) {
			return result, response.Error(err, response.WithMessage("Failed to validate request object"))
		}

		return result, BuildValidationErrorResponse(validationErrors)
	}

	return result, nil
}
