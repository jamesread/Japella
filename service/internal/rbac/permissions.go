package rbac

import armrbac "github.com/jamesread/armature-iam/rbac"

// Core permission names from armature-iam (must match rbac_permissions.name in the database).
const (
	PermissionAppAccess          = armrbac.PermissionAppAccess
	PermissionUsersView          = armrbac.PermissionUsersView
	PermissionUsersCreate        = armrbac.PermissionUsersCreate
	PermissionUsersDelete        = armrbac.PermissionUsersDelete
	PermissionUsersResetPassword = armrbac.PermissionUsersResetPassword
	PermissionUserGroupsView     = armrbac.PermissionUserGroupsView
	PermissionUserGroupsManage   = armrbac.PermissionUserGroupsManage
	PermissionRbacView           = armrbac.PermissionRbacView
	PermissionRbacManage         = armrbac.PermissionRbacManage
	PermissionSystemSettings     = armrbac.PermissionSystemSettings
	PermissionSystemLogs         = armrbac.PermissionSystemLogs
	PermissionSystemImpersonate  = armrbac.PermissionSystemImpersonate
)

// Japella-specific permissions (not in armature-iam core catalog).
const (
	PermissionSocialAccountsViewAll = "social-accounts.view-all"
	PermissionSystemConnectors      = "system.connectors"
	PermissionAccountPoliciesManage = "account-policies.manage"
)

const RoleSuperuser = armrbac.RoleSuperuser
const RoleMember = armrbac.RoleMember

const GroupEveryone = armrbac.GroupEveryone
const GroupAdministrators = armrbac.GroupAdministrators
