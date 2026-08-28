import { useCustomTheme } from 'picocrank/vue/composables/useCustomTheme.js';

export const ThemeControlSystem = 'system';
export const ThemeControlUser = 'user';

export function themeControlFromFeatures(features = {}) {
	const raw = (features.themeControl || features.theme_control || ThemeControlUser).trim();
	return raw === ThemeControlSystem ? ThemeControlSystem : ThemeControlUser;
}

export function themeFeaturesFromStatus(st = {}) {
	return {
		themeColorSchemeSwitcherEnabled: st.themeColorSchemeSwitcherEnabled === true,
		themeName: (st.themeName || '').trim(),
		themeControl: themeControlFromFeatures({
			themeControl: st.themeControl || ThemeControlUser,
		}),
		availableThemes: Array.isArray(st.availableThemes) ? st.availableThemes : [],
	};
}

export function themeColorSchemeSwitcherEnabledFromFeatures(features = {}) {
	return features.themeColorSchemeSwitcherEnabled === true
		|| features.theme_color_scheme_switcher_enabled === true;
}

export function applyAppTheming(features = {}) {
	const { themePreference, setTheme, clearTheme } = useCustomTheme();

	const control = themeControlFromFeatures(features);
	const systemTheme = (features.themeName || features.theme_name || '').trim();

	if (control === ThemeControlSystem) {
		if (systemTheme) {
			setTheme(systemTheme);
		} else {
			clearTheme();
		}
		return { control, appliedTheme: systemTheme };
	}

	const stored = (themePreference.value || '').trim();
	if (stored) {
		setTheme(stored);
		return { control, appliedTheme: stored };
	}
	if (systemTheme) {
		setTheme(systemTheme);
		return { control, appliedTheme: systemTheme };
	}
	clearTheme();
	return { control, appliedTheme: '' };
}
