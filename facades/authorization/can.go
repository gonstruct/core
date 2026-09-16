package authorization

import "fmt"

func (session Session[T]) Can(permission T) bool {
	rolesWithPermissions := Use(session.key)
	if permissions, exists := rolesWithPermissions[session.role]; exists {
		if value, exists := permissions[fmt.Sprint(permission)]; exists {
			return value
		}
	}
	return false
}

func (session Session[T]) CanAny(permissions ...T) bool {
	rolesWithPermissions := Use(session.key)
	if rolePermissions, exists := rolesWithPermissions[session.role]; exists {
		for _, permission := range permissions {
			if value, exists := rolePermissions[fmt.Sprint(permission)]; exists && value {
				return true
			}
		}
	}
	return false
}

func (session Session[T]) CanAll(permissions ...T) bool {
	rolesWithPermissions := Use(session.key)
	if rolePermissions, exists := rolesWithPermissions[session.role]; exists {
		for _, permission := range permissions {
			if value, exists := rolePermissions[fmt.Sprint(permission)]; !exists || !value {
				return false
			}
		}
		return true
	}
	return false
}

func (session Session[T]) Cannot(permission T) bool {
	rolesWithPermissions := Use(session.key)
	if permissions, exists := rolesWithPermissions[session.role]; exists {
		if value, exists := permissions[fmt.Sprint(permission)]; exists {
			return !value
		}
	}
	return true
}
