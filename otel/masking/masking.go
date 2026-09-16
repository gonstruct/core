// Package masking redacts sensitive values out of request and response bodies
// before they are attached to a span.
package masking

import (
	"encoding/json"
	"strings"
)

const (
	Redacted        = "[REDACTED]"
	UnparseableBody = "[unparseable body]"
)

// Masker walks JSON and replaces every value whose key looks sensitive.
//
// A body that is not JSON cannot be redacted field by field. Emitting it raw
// would be the exact leak this type exists to prevent, so it is replaced
// wholesale with UnparseableBody.
type Masker struct {
	keys []string
}

// New returns a Masker matching DefaultKeys plus any extra keys.
func New(extra ...string) *Masker {
	keys := make([]string, 0, len(DefaultKeys)+len(extra))
	keys = append(keys, DefaultKeys...)

	for _, key := range extra {
		keys = append(keys, strings.ToLower(key))
	}

	return &Masker{keys: keys}
}

// Mask returns the redacted body, or an empty string when there is nothing to
// record.
func (self *Masker) Mask(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var object map[string]any
	if err := json.Unmarshal(body, &object); err == nil {
		self.maskMap(object)

		return marshal(object)
	}

	var array []any
	if err := json.Unmarshal(body, &array); err == nil {
		self.maskSlice(array)

		return marshal(array)
	}

	return UnparseableBody
}

func (self *Masker) maskMap(data map[string]any) {
	for key, value := range data {
		if self.isSensitive(key) {
			data[key] = Redacted

			continue
		}

		switch typed := value.(type) {
		case map[string]any:
			self.maskMap(typed)
		case []any:
			self.maskSlice(typed)
		}
	}
}

func (self *Masker) maskSlice(data []any) {
	for _, item := range data {
		switch typed := item.(type) {
		case map[string]any:
			self.maskMap(typed)
		case []any:
			self.maskSlice(typed)
		}
	}
}

// IsSensitive reports whether a key looks like it holds a secret.
//
// Separators are normalised because the same field arrives spelled differently
// depending on where it came from: a JSON body says "api_key", an HTTP header
// says "X-Api-Key", and a query parameter might say "api-key". Matching the raw
// string would redact the first and leak the other two.
func (self *Masker) IsSensitive(key string) bool {
	return self.isSensitive(key)
}

func (self *Masker) isSensitive(key string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")

	for _, sensitive := range self.keys {
		if strings.Contains(normalized, sensitive) {
			return true
		}
	}

	return false
}

func marshal(value any) string {
	masked, err := json.Marshal(value)
	if err != nil {
		return UnparseableBody
	}

	return string(masked)
}
