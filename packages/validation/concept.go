package v_concept

import (
	"fmt"
	"time"
)

type Validator interface {
	String(key string, validators ...validator[string]) string
	Number(key string, validators ...validator[int]) int
	Date(key string, validators ...validator[time.Time]) time.Time
}

type validation struct {
	parsed map[string]any
	errors map[string]error
}

type validator[T comparable] interface {
	Validate(v T) error
}

func (v *validation) String(key string, validators ...validator[string]) string {
	value, ok := v.parsed[key]
	if !ok {
		v.errors[key] = fmt.Errorf("key %s not found", key)
		return ""
	}

	strValue, ok := value.(string)
	if !ok {
		v.errors[key] = fmt.Errorf("key %s is not a string", key)
		return ""
	}

	for _, validator := range validators {
		if err := validator.Validate(strValue); err != nil {
			v.errors[key] = err
			return ""
		}
	}

	return strValue
}

func (v *validation) Number(key string, validators ...validator[int]) int {
	return 42
}

func (v *validation) Date(key string, validators ...validator[time.Time]) time.Time {
	return time.Now()
}

func backendHandler() {
	validation := validation{
		parsed: map[string]any{
			"field":      "hello",
			"value":      3,
			"created_at": time.Now(),
		},
	}
	validated := rules(&validation)

	if len(validation.errors) > 0 {
		fmt.Printf("Validation errors: %+v\n", validation.errors)
		return
	}

	fmt.Printf("Validated: %+v\n", validated)
}

type myCustomType struct {
	Field     string
	Value     int
	CreatedAt time.Time
}

func rules(v Validator) myCustomType {
	return myCustomType{
		Field:     v.String("field"),
		Value:     v.Number("value"),
		CreatedAt: v.Date("created_at"),
	}
}
