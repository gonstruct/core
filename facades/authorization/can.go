package authorization

import "fmt"

func (session Session[T]) Can(permission T) bool {
	name := fmt.Sprint(permission)
	if !session.allows(name) {
		return false
	}

	rolesWithPermissions := Use(session.key)
	if permissions, exists := rolesWithPermissions[session.role]; exists {
		if value, exists := permissions[name]; exists {
			return value
		}
	}
	return false
}

func (session Session[T]) CanAny(permissions ...T) bool {
	for _, permission := range permissions {
		if session.Can(permission) {
			return true
		}
	}
	return false
}

func (session Session[T]) CanAll(permissions ...T) bool {
	if len(permissions) == 0 {
		return session.hasRole()
	}

	for _, permission := range permissions {
		if !session.Can(permission) {
			return false
		}
	}
	return true
}

func (session Session[T]) Cannot(permission T) bool {
	return !session.Can(permission)
}

func (session Session[T]) hasRole() bool {
	_, exists := Use(session.key)[session.role]

	return exists
}
