package rbac

// Permission names (must match rbac_permissions.name in the database).
const (
	PermissionAppAccess          = "app.access"
	PermissionUsersView          = "users.view"
	PermissionUsersCreate        = "users.create"
	PermissionUsersDelete        = "users.delete"
	PermissionUsersResetPassword = "users.reset-password"
	PermissionSocialAccountsViewAll = "social-accounts.view-all"
	PermissionSystemSettings       = "system.settings"
	PermissionSystemConnectors     = "system.connectors"
	PermissionSystemLogs           = "system.logs"
	PermissionSystemImpersonate    = "system.impersonate"
	PermissionRbacView             = "rbac.view"
	PermissionRbacManage           = "rbac.manage"
	PermissionUserGroupsView       = "usergroups.view"
	PermissionUserGroupsManage     = "usergroups.manage"
)

const RoleSuperuser = "superuser"
const RoleMember    = "member"
