/**
 * RBAC helpers aligned with service/internal/layers/authentication/authenticated_user.go
 */

const CONTROL_PANEL_PERMISSIONS = [
	'users.view',
	'rbac.view',
	'usergroups.view',
	'account-policies.manage',
	'system.connectors',
	'system.settings',
	'system.logs',
	'system.impersonate',
]

export const IAM_PERMISSIONS = [
	'users.view',
	'usergroups.view',
	'account-policies.manage',
	'rbac.view',
]

const SYSTEM_DIAGNOSTICS_PERMISSIONS = [
	'system.connectors',
	'system.settings',
	'system.logs',
]

export function canAccessControlPanelFromStatus(st) {
	if (!st?.isLoggedIn) {
		return false
	}
	if (st.rbacIsSuperuser) {
		return true
	}
	const perms = st.rbacPermissions || []
	return CONTROL_PANEL_PERMISSIONS.some((p) => perms.includes(p))
}

export function canAccessIamFromStatus(st) {
	if (!st?.isLoggedIn) {
		return false
	}
	if (st.rbacIsSuperuser) {
		return true
	}
	const perms = st.rbacPermissions || []
	return IAM_PERMISSIONS.some((p) => perms.includes(p))
}

export function canViewSystemDiagnosticsFromStatus(st) {
	if (!st?.isLoggedIn) {
		return false
	}
	if (st.rbacIsSuperuser) {
		return true
	}
	const perms = st.rbacPermissions || []
	return SYSTEM_DIAGNOSTICS_PERMISSIONS.some((p) => perms.includes(p))
}
