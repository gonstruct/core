package authorization

import "fmt"

type Session[T comparable] struct {
	key  string
	role string

	// only is what the session is limited to, whatever its role allows: a
	// token that was given less than the person behind it. nil is no limit.
	only map[string]bool
}

func NewSession[T comparable](key, role string) Session[T] {
	return Session[T]{
		key:  key,
		role: role,
	}
}

// Only limits the session to the permissions given: it can do what its role
// allows and the list names, nothing else. Limiting twice keeps what both
// lists name.
func (session Session[T]) Only(permissions ...T) Session[T] {
	only := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		name := fmt.Sprint(permission)
		if session.only == nil || session.only[name] {
			only[name] = true
		}
	}

	session.only = only

	return session
}

func (session Session[T]) allows(permission string) bool {
	return session.only == nil || session.only[permission]
}
