<template>
	<Section
		title="Post performance"
		:subtitle="`Engagement on posts published from this account in the last ${days} days`"
		:icon="ActivityIcon"
		classes="social-account-performance"
	>
		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh post performance"
				title="Refresh"
				:disabled="loading"
				@click="load"
			>
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="iconStrokeWidth" aria-hidden="true" />
			</button>
		</template>

		<p v-if="loading && !report">Loading post performance...</p>
		<NotificationBlock v-else-if="error" type="error" :message="error" />
		<template v-else-if="report">
			<NotificationBlock
				v-if="!report.supported"
				type="note"
				message="Engagement stats aren't available for this network yet."
			/>
			<NotificationBlock
				v-else-if="!report.trackingEnabled"
				type="note"
				message="Engagement tracking is turned off. Set post_engagement_max_age_days above 0 in Settings to enable it."
			/>

			<template v-if="report.supported">
				<div class="performance-stats" data-performance-totals>
					<div v-for="tile in totalTiles" :key="tile.key" class="stat-display">
						<span class="subtle">{{ tile.label }}</span>
						<span class="stat">{{ tile.value.toLocaleString() }}</span>
					</div>
				</div>

				<div class="performance-chart" role="img" :aria-label="chartLabel">
					<div
						v-for="day in dailyBars"
						:key="day.date"
						class="chart-day"
						:title="day.title"
					>
						<span class="chart-value">{{ day.interactions || '' }}</span>
						<div class="chart-track">
							<div class="chart-bar" :style="{ height: day.height }" />
						</div>
						<span class="chart-label">{{ day.label }}</span>
						<span class="chart-posts subtle">{{ day.posts }} {{ day.posts === 1 ? 'post' : 'posts' }}</span>
					</div>
				</div>

				<NotificationBlock
					v-if="report.postCount === 0"
					type="note"
					:message="`No posts published from this account in the last ${days} days.`"
				/>
				<Table
					v-else
					class="performance-table-wrap"
					:data="rows"
					:headers="tableHeaders"
				>
					<template #cell-snippet="{ row }">
						<router-link :to="{ name: 'postDetails', params: { id: row.id } }" class="post-snippet">
							{{ row.snippet }}
						</router-link>
						<span v-if="!row.hasData" class="subtle not-fetched"> (not fetched yet)</span>
					</template>
					<template #cell-postedDate="{ row }">
						<span class="nowrap">{{ formatHumanDateTime(row.postedDate) }}</span>
					</template>
					<template #cell-postUrl="{ row }">
						<a v-if="row.postUrl" :href="row.postUrl" target="_blank" rel="noopener noreferrer" title="Open post">
							<Icon icon="mdi:open-in-new" width="16" height="16" />
						</a>
					</template>
				</Table>

				<p v-if="lastUpdated" class="subtle performance-updated">Counts last updated {{ lastUpdated }}.</p>
			</template>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, watch } from 'vue';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ActivityIcon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import { decodeHtmlEntities, formatHumanDateTime, waitForClient } from '../javascript/util';

	const props = defineProps({
		socialAccountId: {
			type: Number,
			required: true,
		},
		days: {
			type: Number,
			default: 7,
		},
	});

	const iconStrokeWidth = 2.5;

	const report = ref(null);
	const loading = ref(false);
	const error = ref('');

	const tableHeaders = [
		{ key: 'snippet', label: 'Post', sortable: false },
		{ key: 'postedDate', label: 'Posted', sortable: true, width: '10rem' },
		{ key: 'likes', label: 'Likes', sortable: true, width: '5rem' },
		{ key: 'reposts', label: 'Reposts', sortable: true, width: '5rem' },
		{ key: 'replies', label: 'Replies', sortable: true, width: '5rem' },
		{ key: 'quotes', label: 'Quotes', sortable: true, width: '5rem' },
		{ key: 'interactions', label: 'Total', sortable: true, width: '5rem' },
		{ key: 'postUrl', label: '', sortable: false, width: '3rem' },
	];

	function interactionsOf(e) {
		if (!e) return 0;
		return (e.likes || 0) + (e.reposts || 0) + (e.replies || 0) + (e.quotes || 0);
	}

	function snippetOf(content) {
		const text = decodeHtmlEntities(String(content || '').replace(/<[^>]*>/g, ' ')).replace(/\s+/g, ' ').trim();
		return text.length > 90 ? `${text.slice(0, 87)}...` : text || '(no text)';
	}

	const totals = computed(() => report.value?.totals || {});

	const totalTiles = computed(() => {
		const t = totals.value;
		const tiles = [
			{ key: 'posts', label: 'Posts', value: report.value?.postCount || 0 },
			{ key: 'interactions', label: 'Interactions', value: interactionsOf(t) },
			{ key: 'likes', label: 'Likes', value: t.likes || 0 },
			{ key: 'reposts', label: 'Reposts', value: t.reposts || 0 },
			{ key: 'replies', label: 'Replies', value: t.replies || 0 },
			{ key: 'quotes', label: 'Quotes', value: t.quotes || 0 },
		];
		if (t.impressions) {
			tiles.push({ key: 'impressions', label: 'Impressions', value: t.impressions });
		}
		return tiles;
	});

	const dailyBars = computed(() => {
		const daily = report.value?.daily || [];
		const max = Math.max(1, ...daily.map((d) => interactionsOf(d.engagement)));
		return daily.map((d) => {
			const interactions = interactionsOf(d.engagement);
			// Dates are UTC calendar days; parse at noon UTC so the weekday label is stable.
			const date = new Date(`${d.date}T12:00:00Z`);
			const e = d.engagement || {};
			return {
				date: d.date,
				posts: d.posts || 0,
				interactions,
				height: `${Math.round((interactions / max) * 100)}%`,
				label: date.toLocaleDateString(undefined, { weekday: 'short', timeZone: 'UTC' }),
				title: `${date.toLocaleDateString(undefined, { dateStyle: 'medium', timeZone: 'UTC' })}: ${d.posts || 0} posts, ${e.likes || 0} likes, ${e.reposts || 0} reposts, ${e.replies || 0} replies, ${e.quotes || 0} quotes`,
			};
		});
	});

	const chartLabel = computed(() => {
		return `Interactions per day for posts published in the last ${props.days} days`;
	});

	const rows = computed(() => {
		return (report.value?.posts || []).map((p) => {
			const e = p.engagement || {};
			return {
				id: p.id,
				snippet: snippetOf(p.content),
				postedDate: p.postedDate || p.created,
				postUrl: p.postUrl,
				hasData: Boolean(e.hasData),
				likes: e.likes || 0,
				reposts: e.reposts || 0,
				replies: e.replies || 0,
				quotes: e.quotes || 0,
				interactions: interactionsOf(e),
			};
		});
	});

	const lastUpdated = computed(() => {
		let latest = '';
		for (const p of report.value?.posts || []) {
			const at = p.engagement?.hasData ? p.engagement.lastFetchedAt : '';
			if (at && at > latest) latest = at;
		}
		return latest ? formatHumanDateTime(latest) : '';
	});

	async function load() {
		if (!props.socialAccountId) return;
		loading.value = true;
		error.value = '';
		try {
			report.value = await window.client.getSocialAccountPostPerformance({
				socialAccountId: props.socialAccountId,
				days: props.days,
			});
		} catch (e) {
			error.value = `Failed to load post performance: ${e.message || e}`;
		} finally {
			loading.value = false;
		}
	}

	watch(() => props.socialAccountId, load);

	onMounted(async () => {
		await waitForClient();
		await load();
	});
</script>

<style scoped>
	.performance-stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr));
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.performance-chart {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: 1fr;
		gap: 0.5rem;
		align-items: end;
		margin-bottom: 1.5rem;
	}

	.chart-day {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		min-width: 0;
	}

	.chart-track {
		position: relative;
		width: 100%;
		max-width: 3rem;
		height: 7rem;
		display: flex;
		align-items: flex-end;
		border-bottom: 1px solid var(--border-color, #ccc);
	}

	.chart-bar {
		width: 100%;
		min-height: 2px;
		border-radius: 4px 4px 0 0;
		background: var(--link-color, #007bff);
		transition: height 0.2s ease;
	}

	.chart-value {
		font-size: 0.8rem;
		font-weight: 600;
		min-height: 1.2em;
		font-variant-numeric: tabular-nums;
	}

	.chart-label {
		font-size: 0.85rem;
		font-weight: 500;
	}

	.chart-posts {
		font-size: 0.75rem;
		white-space: nowrap;
	}

	.performance-table-wrap {
		margin-bottom: 0.5rem;
	}

	.post-snippet {
		color: var(--link-color, #007bff);
	}

	.not-fetched,
	.performance-updated {
		font-size: 0.85rem;
	}

	.nowrap {
		white-space: nowrap;
	}

	.subtle {
		opacity: 0.7;
	}
</style>
