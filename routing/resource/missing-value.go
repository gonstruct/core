package resource

type potentiallyMissing interface {
	IsMissing() bool
}

type missingValue struct{}

func (missingValue) IsMissing() bool {
	return true
}
