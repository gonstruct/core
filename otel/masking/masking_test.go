package masking_test

import (
	"encoding/json"
	"github.com/gonstruct/core/otel/masking"
	"strings"
	"testing"
)

func TestRedactsSensitiveKeys(t *testing.T) {
	masked := masking.New().Mask([]byte(`{"email":"a@b.com","name":"Ada","user_password":"hunter2"}`))

	var result map[string]any
	if err := json.Unmarshal([]byte(masked), &result); err != nil {
		t.Fatalf("masked output is not valid JSON: %v", err)
	}

	if result["email"] != masking.Redacted {
		t.Errorf("email not redacted: %v", result["email"])
	}

	// Substring matching: "user_password" contains "password".
	if result["user_password"] != masking.Redacted {
		t.Errorf("user_password not redacted: %v", result["user_password"])
	}

	if result["name"] != "Ada" {
		t.Errorf("non-sensitive field was altered: %v", result["name"])
	}
}

func TestRecursesIntoNestedStructures(t *testing.T) {
	masked := masking.New().Mask([]byte(`{"user":{"token":"abc"},"items":[{"cvv":"123"}]}`))

	if strings.Contains(masked, "abc") {
		t.Errorf("nested object secret leaked: %s", masked)
	}

	if strings.Contains(masked, "123") {
		t.Errorf("secret inside array leaked: %s", masked)
	}
}

func TestAcceptsExtraKeys(t *testing.T) {
	masked := masking.New("internal_ref").Mask([]byte(`{"internal_ref":"xyz"}`))

	if strings.Contains(masked, "xyz") {
		t.Errorf("custom sensitive key leaked: %s", masked)
	}
}

// A non-JSON body cannot be redacted field by field, so it must never be
// emitted verbatim.
func TestReplacesNonJSONBodies(t *testing.T) {
	masked := masking.New().Mask([]byte("password=hunter2&card_number=4111111111111111"))

	if masked != masking.UnparseableBody {
		t.Errorf("non-JSON body was not replaced: %s", masked)
	}
}

func TestReturnsEmptyForEmptyBody(t *testing.T) {
	if got := masking.New().Mask(nil); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
