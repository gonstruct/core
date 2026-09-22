package exception_test

import (
	"errors"
	"testing"

	"github.com/gonstruct/core/routing/response/exception"
)

func TestValidationErrorErrIsNilWhenNothingWasRecorded(t *testing.T) {
	problems := exception.ValidationError{}
	problems.Add("name", "")

	if err := problems.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	problems.Add("name", "The name is taken.")
	var found exception.ValidationError
	if err := problems.Err(); !errors.As(err, &found) || found["name"] != "The name is taken." {
		t.Fatalf("expected the recorded problem back, got %v", err)
	}
}

func TestValidationIsOneField(t *testing.T) {
	err := exception.Validation("pipelineId", "No such pipeline.")
	if err["pipelineId"] != "No such pipeline." || len(err) != 1 {
		t.Fatalf("expected one field, got %v", err)
	}
}
