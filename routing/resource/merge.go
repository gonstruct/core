package resource

func (resource Resource) MergeWith(resources ...ResourceInterface) Resource {
	for _, res := range resources {
		for key, value := range res.Attributes() {
			resource[key] = value // Explicitly overwrite existing keys
		}
	}

	return resource
}

func (resource Resource) Merge(resources ...Resource) Resource {
	for _, res := range resources {
		for key, value := range res {
			resource[key] = value // Explicitly overwrite existing keys
		}
	}

	return resource
}

func (resource Resource) MergeWhen(value bool, then func() Resource, fallback ...Resource) Resource {
	if value {
		return resource.Merge(then())
	}

	if len(fallback) == 1 {
		return resource.Merge(fallback[0])
	}

	return resource
}
