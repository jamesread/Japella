/** Window event fired after health issues change (run, acknowledge) so navigation badges refresh. */
export const HEALTH_CHANGED_EVENT = 'japella:health-changed';

export function notifyHealthChanged() {
	window.dispatchEvent(new Event(HEALTH_CHANGED_EVENT));
}

/** Sort order and labels for issue severities. */
export const HEALTH_SEVERITIES = [
	{ id: 'error', one: 'error', many: 'errors' },
	{ id: 'warning', one: 'warning', many: 'warnings' },
	{ id: 'info', one: 'info', many: 'info' },
];

export function canViewHealthFromStatus(st) {
	if (!st?.isLoggedIn) {
		return false;
	}
	if (st.rbacIsSuperuser) {
		return true;
	}
	const perms = st.rbacPermissions || [];
	return perms.includes('system.settings');
}

export async function fetchHealthSummaryCount(status) {
	const canView = status
		? canViewHealthFromStatus(status)
		: Boolean(
				window.userRbacIsSuperuser ||
				(Array.isArray(window.userRbacPermissions) &&
					window.userRbacPermissions.includes('system.settings')),
			);
	if (!canView || !window.client) {
		return 0;
	}
	try {
		const res = await window.client.getHealthSummary({});
		return (Number(res.errors) || 0) + (Number(res.warnings) || 0);
	} catch (e) {
		console.warn('Failed to load health summary for navigation', e);
		return 0;
	}
}
