package authorization

type Session[T comparable] struct {
	key  string
	role string
}

func NewSession[T comparable](key, role string) Session[T] {
	return Session[T]{
		key:  key,
		role: role,
	}
}
