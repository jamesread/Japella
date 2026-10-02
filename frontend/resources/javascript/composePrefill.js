const COMPOSE_PREFILL_KEY = 'japella.compose.prefill';

/**
 * Stash content + account IDs for PostBox to apply once (does not submit).
 * @param {{ content?: string, socialAccountIds?: number[] }} prefill
 */
export function stashComposePrefill(prefill) {
	const payload = {
		content: typeof prefill?.content === 'string' ? prefill.content : '',
		socialAccountIds: normalizeAccountIds(prefill?.socialAccountIds),
	};
	try {
		sessionStorage.setItem(COMPOSE_PREFILL_KEY, JSON.stringify(payload));
	} catch (e) {
		console.error('Failed to stash compose prefill', e);
	}
}

/**
 * Read and clear a pending compose prefill.
 * @returns {{ content: string, socialAccountIds: number[] } | null}
 */
export function takeComposePrefill() {
	let raw;
	try {
		raw = sessionStorage.getItem(COMPOSE_PREFILL_KEY);
		sessionStorage.removeItem(COMPOSE_PREFILL_KEY);
	} catch {
		return null;
	}
	if (!raw) {
		return null;
	}
	try {
		const parsed = JSON.parse(raw);
		return {
			content: typeof parsed?.content === 'string' ? parsed.content : '',
			socialAccountIds: normalizeAccountIds(parsed?.socialAccountIds),
		};
	} catch {
		return null;
	}
}

function normalizeAccountIds(ids) {
	if (!Array.isArray(ids)) {
		return [];
	}
	return [...new Set(
		ids.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0),
	)];
}
