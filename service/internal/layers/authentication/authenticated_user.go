package authentication

import (
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/rbac"
)

type AuthenticatedUser struct {
	User *db.UserAccount
	RBAC *db.EffectiveRBAC
}

func (a *AuthenticatedUser) HasPermission(p string) bool {
	if a == nil || a.RBAC == nil {
		return false
	}
	return a.RBAC.Has(p)
}

// CanAccessControlPanel is true when the user may use the admin control panel (any linked capability).
func (a *AuthenticatedUser) CanAccessControlPanel() bool {
	if a == nil || a.RBAC == nil {
		return false
	}
	if a.RBAC.IsSuperuser {
		return true
	}
	return a.HasPermission(rbac.PermissionUsersView) ||
		a.HasPermission(rbac.PermissionRbacView) ||
		a.HasPermission(rbac.PermissionUserGroupsView) ||
		a.HasPermission(rbac.PermissionSystemConnectors) ||
		a.HasPermission(rbac.PermissionSystemSettings) ||
		a.HasPermission(rbac.PermissionSystemLogs) ||
		a.HasPermission(rbac.PermissionSystemImpersonate)
}

// CanViewSystemDiagnostics is true when GetStatus may include infrastructure and host details.
func (a *AuthenticatedUser) CanViewSystemDiagnostics() bool {
	if a == nil || a.RBAC == nil {
		return false
	}
	if a.RBAC.IsSuperuser {
		return true
	}
	return a.HasPermission(rbac.PermissionSystemConnectors) ||
		a.HasPermission(rbac.PermissionSystemSettings) ||
		a.HasPermission(rbac.PermissionSystemLogs)
}
