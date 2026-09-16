package resource

import "encoding/json"

type Resource map[string]any

func (resource Resource) MarshalJSON() ([]byte, error) {
	for key, value := range resource {
		if missingValue, ok := value.(potentiallyMissing); ok && missingValue.IsMissing() {
			delete(resource, key)
		}
	}

	type alias map[string]any
	return json.Marshal(alias(resource))
}

type ResourceInterface interface {
	Attributes() Resource
}

type Collection []Resource

func NewCollection[T ResourceInterface, I any](converter func(I) T, items []I) Collection {
	collection := make(Collection, len(items))
	for i, item := range items {
		collection[i] = converter(item).Attributes()
	}
	return collection
}
