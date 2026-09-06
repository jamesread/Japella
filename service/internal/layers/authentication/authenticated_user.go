package authentication

import (
	"context"
	"time"

	armlayer "github.com/jamesread/armature-iam/layer"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/rbac"
)

// AuthenticatedUser is the Connect authn context for Japella requests.
type AuthenticatedUser struct {
	User     *db.UserAccount
	RBAC     *db.EffectiveRBAC
	ReadOnly bool
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
		a.HasPermission(rbac.PermissionAccountPoliciesManage) ||
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

// FromContext returns the Japella authenticated user from a Connect request context.
func FromContext(ctx context.Context) *AuthenticatedUser {
	au := armlayer.UserFromContext(ctx)
	if au == nil {
		return nil
	}
	return armatureUserToJapella(au)
}

func armatureUserToJapella(au *armlayer.AuthenticatedUser) *AuthenticatedUser {
	if au == nil || au.User == nil {
		return nil
	}
	out := &AuthenticatedUser{
		User: &db.UserAccount{
			Model: db.Model{
				ID:        uint32(au.User.ID),
				CreatedAt: parseArmatureTime(au.User.CreatedAt),
				UpdatedAt: parseArmatureTime(au.User.UpdatedAt),
			},
			Username:     au.User.Username,
			PasswordHash: au.User.PasswordHash,
			CreatedBy:    au.User.CreatedBy,
		},
		ReadOnly: au.ReadOnly,
	}
	if au.RBAC != nil {
		perms := make(map[string]bool, len(au.RBAC.Permissions))
		for k, v := range au.RBAC.Permissions {
			perms[k] = v
		}
		out.RBAC = &db.EffectiveRBAC{
			IsSuperuser: au.RBAC.IsSuperuser,
			Permissions: perms,
			RoleNames:   append([]string(nil), au.RBAC.RoleNames...),
		}
	}
	return out
}

func parseArmatureTime(s string) (t time.Time) {
	if s == "" {
		return
	}
	t, _ = time.Parse("2006-01-02 15:04:05", s)
	return
}
