import {
	UserMultiple02Icon,
	UserGroupIcon,
	SecurityValidationIcon,
	WebSecurityIcon,
	ShieldKeyIcon,
} from '@hugeicons/core-free-icons'

function hasPermission(st, permission) {
	return (
		st?.rbacIsSuperuser ||
		(Array.isArray(st?.rbacPermissions) && st.rbacPermissions.includes(permission))
	)
}

/**
 * Populate a picocrank Navigation with IAM hub tiles (Users, groups, policies, RBAC).
 * @param {import('vue').Ref} navRef - Navigation component ref (.value exposes addCallback)
 * @param {(path: string) => void} goToRoute
 * @param {object} st - getStatus() payload
 */
export function setupIamNavigationGrid(navRef, goToRoute, st) {
	const nav = navRef?.value ?? navRef
	if (!nav) {
		return
	}

	if (hasPermission(st, 'users.view')) {
		nav.addCallback('Users', () => goToRoute('/users'), {
			icon: UserMultiple02Icon,
			name: 'iam-users',
			description: 'Manage system users',
		})
	}

	if (hasPermission(st, 'usergroups.view')) {
		nav.addCallback('User Groups', () => goToRoute('/user-groups'), {
			icon: UserGroupIcon,
			name: 'iam-user-groups',
			description: 'Manage user groups and membership',
		})
	}

	if (hasPermission(st, 'account-policies.manage')) {
		nav.addCallback('Account Policies', () => goToRoute('/account-policies'), {
			icon: SecurityValidationIcon,
			name: 'iam-account-policies',
			description: 'Approval workflows for social accounts',
		})
	}

	if (hasPermission(st, 'rbac.view')) {
		nav.addCallback('Permissions', () => goToRoute('/settings/rbac/permissions'), {
			icon: ShieldKeyIcon,
			name: 'iam-permissions',
			description: 'View all RBAC permissions in the system',
		})
		nav.addCallback('Roles & Permissions', () => goToRoute('/settings/rbac'), {
			icon: WebSecurityIcon,
			name: 'iam-rbac',
			description: 'Manage RBAC roles and assign them to groups',
		})
	}
}
