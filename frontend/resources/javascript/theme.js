// Color scheme (auto/light/dark): PicoCrank useTheme side effects apply data-theme on load.
import 'picocrank/vue/composables/useTheme.js';

export { theme, useTheme } from 'picocrank/vue/composables/useTheme.js';
import { initCustomTheme, useCustomTheme } from 'picocrank/vue/composables/useCustomTheme.js';

export { useCustomTheme, initCustomTheme };

const DEFAULT_CUSTOM_THEME_OPTIONS = {
	storageKey: 'japella-custom-theme',
	includeSupplementalThemes: true,
};

/** Boot named CSS themes (useCustomTheme). Color scheme is handled by useTheme above. */
export function initAppThemes(options = {}) {
	return initCustomTheme({ ...DEFAULT_CUSTOM_THEME_OPTIONS, ...options });
}
