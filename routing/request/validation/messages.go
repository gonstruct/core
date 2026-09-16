package validation

import (
	"fmt"
	"github.com/gonstruct/core/routing/response"
	"reflect"
	"strings"

	v10Validator "github.com/go-playground/validator/v10"
	"github.com/iancoleman/strcase"
)

var defaultMessages = map[string]string{
	"date_gtefield":    "The from and to fields must be valid dates in YYYY-MM-DD format, and from must be before or equal to to.",
	"date_ltefield":    "The from and to fields must be valid dates in YYYY-MM-DD format, and from must be before or equal to to.",
	"required":         "The {field} field is required",
	"email":            "The {field} field must be a valid email address",
	"min":              "The {field} field must be at least {param} characters long",
	"max":              "The {field} field must be at most {param} characters long",
	"len":              "The {field} field must be exactly {param} characters long",
	"url":              "The {field} field must be a valid URL",
	"numeric":          "The {field} field must be a valid number",
	"uuid":             "The {field} field must be a valid UUID",
	"eq":               "The {field} field must be equal to {param}",
	"ne":               "The {field} field must not be equal to {param}",
	"gte":              "The {field} field must be greater than or equal to {param}",
	"lte":              "The {field} field must be less than or equal to {param}",
	"required_if":      "The {field} field is required when {param}",
	"required_with":    "The {field} field is required when {param} is present",
	"required_without": "The {field} field is required when {param} is missing",
	"oneof":            "The {field} field must be one of [{param}]",
	"uuid4":            "The {field} field must be a valid UUIDv4",
	"alphanum":         "The {field} field must only contain alphanumeric characters",
}

var arrayMessages = map[string]string{
	"min": "The {field} field must have at least {param} items",
	"max": "The {field} field must have at most {param} items",
}

func getValidationMessage(fieldErr v10Validator.FieldError) (field string, message string) {
	field = fieldErr.Field()

	fieldNamespace := strings.Split(fieldErr.Namespace(), ".")
	if len(fieldNamespace) > 1 {
		field = strings.Join(fieldNamespace[1:], ".")
	}

	formattedField := strcase.ToDelimited(field, ' ')
	formattedParam := strcase.ToDelimited(fieldErr.Param(), ' ')

	message, ok := defaultMessages[fieldErr.Tag()]
	if !ok {
		message = fmt.Sprintf("The %s field must be valid", formattedField)
	}

	if fieldErr.Kind() == reflect.Slice {
		if arrayMessage, ok := arrayMessages[fieldErr.Tag()]; ok {
			message = arrayMessage
		}
	}

	if param := fieldErr.Param(); param != "" {
		message = strings.ReplaceAll(message, "{param}", formattedParam)
	}
	message = strings.ReplaceAll(message, "{field}", formattedField)

	return field, message
}

func BuildValidationErrorResponse(validationErrors v10Validator.ValidationErrors) response.Response {
	errors := make(response.ErrorsMap, len(validationErrors))
	for _, err := range validationErrors {
		field, message := getValidationMessage(err)
		errors[field] = message
	}

	return response.Errors(errors)
}
