package authorization

import "sync"

type RolesWithPermissions map[string]map[string]bool

var (
	getRolesWithPermissions   func() map[string]RolesWithPermissions
	rolesWithPermissionsCache = make(map[string]RolesWithPermissions)
	mu                        sync.RWMutex
)

func Register(value func() map[string]RolesWithPermissions) {
	mu.Lock()
	defer mu.Unlock()
	getRolesWithPermissions = value
}

func Use(key string) RolesWithPermissions {
	mu.RLock()
	if rolesWithPermissions, exists := rolesWithPermissionsCache[key]; exists {
		mu.RUnlock()
		return rolesWithPermissions
	}
	mu.RUnlock()

	if getRolesWithPermissions == nil {
		return make(RolesWithPermissions)
	}

	allRoles := getRolesWithPermissions()

	mu.Lock()
	rolesWithPermissionsCache = allRoles // Cache everything
	mu.Unlock()

	if permissions, exists := allRoles[key]; exists {
		return permissions
	}

	return make(RolesWithPermissions)
}
