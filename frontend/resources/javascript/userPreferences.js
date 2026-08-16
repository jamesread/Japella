/**
 * Apply a saved user language preference to vue-i18n.
 * Empty string means browser default (fallback locale).
 */
export function applyUserLanguage(localeRef, fallbackLocale, language) {
	const next = (language || '').trim();
	localeRef.value = next || fallbackLocale || 'en';
}

let sidebarApplier = null;
let themeToggleApplier = null;

/** @param {(enabled: boolean) => void} fn */
export function registerSidebarApplier(fn) {
	sidebarApplier = fn;
}

/** @param {(enabled: boolean) => void} fn */
export function registerThemeToggleApplier(fn) {
	themeToggleApplier = fn;
}

export function applyUserSidebar(enabled) {
	sidebarApplier?.(Boolean(enabled));
}

export function applyUserThemeToggle(enabled) {
	themeToggleApplier?.(Boolean(enabled));
}

/**
 * Load preferences for the logged-in user and apply language and sidebar when set.
 */
export async function loadAndApplyUserPreferences({ localeRef, fallbackLocale } = {}) {
	if (!window.client) {
		return;
	}
	try {
		const res = await window.client.getUserPreferences({});
		if (localeRef && res.language) {
			applyUserLanguage(localeRef, fallbackLocale, res.language);
		}
		applyUserSidebar(res.sidebarEnabled !== false);
		applyUserThemeToggle(res.themeToggleEnabled === true);
		return res;
	} catch (e) {
		console.warn('Failed to load user preferences', e);
		return null;
	}
}

/** @deprecated Use loadAndApplyUserPreferences */
export async function loadAndApplyUserLanguage(localeRef, fallbackLocale) {
	return loadAndApplyUserPreferences({ localeRef, fallbackLocale });
}
