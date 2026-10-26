// Shared wording for media suitability (see GetMediaSuitability).

export const VERDICT_ICONS = {
	ok: 'mdi:check-circle-outline',
	warning: 'mdi:alert-outline',
	error: 'mdi:close-circle-outline',
};

export const VERDICT_LABELS = {
	ok: 'Suitable',
	warning: 'Warnings',
	error: 'Unsuitable',
};

// femtocrank karma classes: on a .tag they draw a tinted background behind
// normal text, which keeps AA contrast where coloured text wouldn't.
export const VERDICT_KARMA = {
	ok: 'good',
	warning: 'warning',
	error: 'bad',
};

const RANK = { ok: 0, warning: 1, error: 2 };

/** The worst of several verdicts; 'ok' for none. */
export function worstVerdict(verdicts) {
	return verdicts.reduce((worst, v) => ((RANK[v] ?? 0) > (RANK[worst] ?? 0) ? v : worst), 'ok');
}

/** "Bluesky: Unsuitable — 2.5 MB is over the 1.0 MB image limit". */
export function describeNetwork(network) {
	const problems = (network.checks || []).filter((c) => c.level !== 'ok').map((c) => c.message);
	const label = `${network.name}: ${VERDICT_LABELS[network.verdict] || network.verdict}`;
	return problems.length ? `${label} — ${problems.join('; ')}` : label;
}
