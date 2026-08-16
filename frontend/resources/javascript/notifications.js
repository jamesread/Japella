import { showNotificationPopup } from 'picocrank/vue/composables/useNotificationPopups.js';

export {
	useNotificationPopups,
	showNotificationPopup,
	notificationPopups,
	dismissNotificationPopup,
	dismissAllNotificationPopups,
	clearNotificationPopupTimers,
} from 'picocrank/vue/composables/useNotificationPopups.js';

/**
 * Show a stacked corner notification (Picocrank).
 *
 * @param {string} cssClass - Karma class, e.g. good, success, bad, warning, note, info
 * @param {string} label - Short title shown before the message
 * @param {string} message - Body text
 * @param {string | import('vue-router').RouteLocationRaw | null} [linkTo] - Optional link target
 * @param {string} [linkLabel='View'] - Link button label when linkTo is set
 */
export function showNotification(cssClass, label, message, linkTo = null, linkLabel = 'View') {
	const normalizedClass = cssClass === 'good' ? 'success' : cssClass;

	return showNotificationPopup({
		class: normalizedClass,
		label,
		message,
		linkTo: linkTo || null,
		linkLabel: linkTo ? (linkLabel || 'View') : null,
	});
}
