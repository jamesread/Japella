// Per-browser feed column widths, keyed by column ("feed" for the main feed,
// the column id for custom columns). Widths are a viewing preference, so they
// live in localStorage; every access tolerates storage being unavailable.

const STORAGE_KEY = 'feed.columnWidths';

export const DEFAULT_COLUMN_WIDTH = 442;
export const MIN_COLUMN_WIDTH = 256;
export const MAX_COLUMN_WIDTH = 960;

export function clampColumnWidth(px) {
	return Math.round(Math.min(MAX_COLUMN_WIDTH, Math.max(MIN_COLUMN_WIDTH, px)));
}

function readAll() {
	try {
		const parsed = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
		return parsed && typeof parsed === 'object' ? parsed : {};
	} catch {
		return {};
	}
}

function writeAll(widths) {
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(widths));
	} catch {
		// Storage unavailable (private mode, blocked); widths just won't persist.
	}
}

export function loadColumnWidth(key) {
	const w = Number(readAll()[key]);
	return Number.isFinite(w) && w > 0 ? clampColumnWidth(w) : DEFAULT_COLUMN_WIDTH;
}

export function saveColumnWidth(key, px) {
	const widths = readAll();
	widths[key] = clampColumnWidth(px);
	writeAll(widths);
}

export function forgetColumnWidth(key) {
	const widths = readAll();
	delete widths[key];
	writeAll(widths);
}
