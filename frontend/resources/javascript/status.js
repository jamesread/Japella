import { waitForClient } from './util.js';

/** @type {import('./gen/japella/controlapi/v1/control_pb').GetStatusResponse | null} */
let cachedStatus = null;

/** @type {Promise<import('./gen/japella/controlapi/v1/control_pb').GetStatusResponse> | null} */
let statusPromise = null;

export function applyAppStatusToWindow(st) {
	if (!st) {
		return;
	}
	window.isLoggedIn = Boolean(st.isLoggedIn);
	window.userRbacPermissions = st.rbacPermissions || [];
	window.userRbacIsSuperuser = Boolean(st.rbacIsSuperuser);
}

export function invalidateAppStatus() {
	statusPromise = null;
	cachedStatus = null;
	delete window.isLoggedIn;
	window.userRbacPermissions = [];
	window.userRbacIsSuperuser = false;
}

export function getCachedAppStatus() {
	return cachedStatus;
}

/**
 * Fetch GetStatus once per page load (unless force refresh after login/logout).
 * Concurrent callers share the same in-flight request.
 */
export async function fetchAppStatus(options = {}) {
	const { force = false } = options;

	if (force) {
		statusPromise = null;
		cachedStatus = null;
	} else if (cachedStatus) {
		return cachedStatus;
	} else if (statusPromise) {
		return statusPromise;
	}

	statusPromise = (async () => {
		await waitForClient();
		const st = await window.client.getStatus({});
		cachedStatus = st;
		applyAppStatusToWindow(st);
		return st;
	})();

	try {
		return await statusPromise;
	} catch (err) {
		statusPromise = null;
		throw err;
	}
}
