<template>
	<Section
		title="Recent interactions"
		:subtitle="`Posts, replies and engagement on this account in the last ${days} days`"
		:icon="Notification03Icon"
		classes="social-account-interactions"
	>
		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh recent interactions"
				title="Refresh"
				:disabled="loading"
				@click="load"
			>
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="iconStrokeWidth" aria-hidden="true" />
			</button>
		</template>

		<p v-if="loading && !report">Loading recent interactions...</p>
		<NotificationBlock v-else-if="error" type="error" :message="error" />
		<template v-else-if="report">
			<NotificationBlock
				v-if="!report.supported"
				type="note"
				message="Engagement isn't tracked for this network yet, so only published posts are listed."
			/>
			<NotificationBlock
				v-else-if="!report.trackingEnabled"
				type="note"
				message="Engagement tracking is turned off. Set post_engagement_max_age_days above 0 in Settings to see likes, replies and reposts here."
			/>

			<div class="interaction-stats" data-interaction-totals>
				<div v-for="tile in totalTiles" :key="tile.key" class="stat-display">
					<span class="subtle">{{ tile.label }}</span>
					<span class="stat">{{ tile.value.toLocaleString() }}</span>
				</div>
			</div>

			<ol class="interaction-days" :aria-label="`Activity per day, last ${days} days`">
				<li v-for="day in dailyBars" :key="day.date" :title="day.title">
					<span class="day-track"><span class="day-bar" :style="{ height: day.height }" /></span>
					<span class="day-label subtle">{{ day.label }}</span>
				</li>
			</ol>

			<p v-if="!items.length" class="subtle">Nothing happened on this account in the last {{ days }} days.</p>

			<ol v-else class="interaction-list">
				<li v-for="item in items" :key="item.key" class="interaction" :class="`interaction-${item.kind}`">
					<span class="interaction-icon" aria-hidden="true">
						<FeedAuthorAvatar v-if="item.kind === 'reply'" :avatar-url="item.reply.authorAvatarUrl" />
						<Icon v-else :icon="item.icon" width="20" height="20" />
					</span>

					<div class="interaction-body">
						<div class="interaction-summary">
							<strong>{{ item.headline }}</strong>
							<time class="subtle" :datetime="item.at">{{ formatHumanDateTime(item.at) }}</time>
						</div>
						<p v-if="item.kind === 'reply'" class="interaction-reply">{{ item.reply.content }}</p>
						<router-link :to="item.to" class="interaction-post">
							<span v-if="item.kind !== 'post'" class="subtle">on </span>{{ item.postSnippet || '(no text)' }}
						</router-link>
					</div>

					<a
						v-if="item.externalUrl"
						:href="item.externalUrl"
						target="_blank"
						rel="noopener noreferrer"
						class="interaction-external"
						:title="item.externalTitle"
						:aria-label="item.externalTitle"
					>
						<Icon icon="mdi:open-in-new" width="16" height="16" />
					</a>
				</li>
			</ol>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Notification03Icon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FeedAuthorAvatar from './FeedAuthorAvatar.vue';
	import { formatHumanDateTime, waitForClient } from '../javascript/util';

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

	function interactionsOf(e) {
		if (!e) return 0;
		return (e.likes || 0) + (e.reposts || 0) + (e.replies || 0) + (e.quotes || 0);
	}

	function plural(n, one, many) {
		return `${n.toLocaleString()} ${n === 1 ? one : many}`;
	}

	const totalTiles = computed(() => {
		const t = report.value?.totals || {};
		return [
			{ key: 'posts', label: 'Posts', value: report.value?.posts || 0 },
			{ key: 'interactions', label: 'Interactions', value: interactionsOf(t) },
			{ key: 'likes', label: 'Likes', value: t.likes || 0 },
			{ key: 'reposts', label: 'Reposts', value: t.reposts || 0 },
			{ key: 'replies', label: 'Replies', value: t.replies || 0 },
			{ key: 'quotes', label: 'Quotes', value: t.quotes || 0 },
		];
	});

	const dailyBars = computed(() => {
		const daily = report.value?.daily || [];
		const activity = (d) => interactionsOf(d.engagement) + (d.posts || 0);
		const max = Math.max(1, ...daily.map(activity));
		return daily.map((d) => {
			// Dates are UTC calendar days; parse at noon UTC so the weekday label is stable.
			const date = new Date(`${d.date}T12:00:00Z`);
			return {
				date: d.date,
				height: `${Math.round((activity(d) / max) * 100)}%`,
				label: date.toLocaleDateString(undefined, { weekday: 'short', timeZone: 'UTC' }),
				title: `${date.toLocaleDateString(undefined, { dateStyle: 'medium', timeZone: 'UTC' })}: ${plural(d.posts || 0, 'post', 'posts')}, ${plural(interactionsOf(d.engagement), 'interaction', 'interactions')}`,
			};
		});
	});

	function engagementHeadline(e) {
		const parts = [];
		if (e.likes) parts.push(`+${plural(e.likes, 'like', 'likes')}`);
		if (e.reposts) parts.push(`+${plural(e.reposts, 'repost', 'reposts')}`);
		if (e.replies) parts.push(`+${plural(e.replies, 'reply', 'replies')}`);
		if (e.quotes) parts.push(`+${plural(e.quotes, 'quote', 'quotes')}`);
		return parts.join(', ');
	}

	function engagementIcon(e) {
		const top = [
			['mdi:heart', e.likes || 0],
			['mdi:repeat-variant', e.reposts || 0],
			['mdi:comment-outline', e.replies || 0],
			['mdi:format-quote-close-outline', e.quotes || 0],
		].sort((a, b) => b[1] - a[1]);
		return top[0][0];
	}

	const items = computed(() => {
		return (report.value?.items || []).map((it, i) => {
			const postRoute = { name: 'postDetails', params: { id: it.postId } };
			const base = { key: `${it.kind}-${it.postId}-${i}`, kind: it.kind, at: it.at, postSnippet: it.postSnippet };

			if (it.kind === 'reply' && it.reply) {
				const who = it.reply.authorName || it.reply.authorHandle || 'Someone';
				return {
					...base,
					reply: it.reply,
					headline: `${who} replied`,
					to: { ...postRoute, hash: '#replies' },
					externalUrl: it.reply.remoteUrl,
					externalTitle: 'Open reply',
				};
			}

			if (it.kind === 'engagement' && it.engagement) {
				return {
					...base,
					icon: engagementIcon(it.engagement),
					headline: engagementHeadline(it.engagement),
					to: postRoute,
					externalUrl: it.postUrl,
					externalTitle: 'Open post',
				};
			}

			return {
				...base,
				icon: 'mdi:send-outline',
				headline: 'Published a post',
				to: postRoute,
				externalUrl: it.postUrl,
				externalTitle: 'Open post',
			};
		});
	});

	async function load() {
		loading.value = true;
		error.value = '';
		try {
			await waitForClient();
			report.value = await window.client.getSocialAccountRecentInteractions({
				socialAccountId: props.socialAccountId,
				days: props.days,
			});
		} catch (e) {
			error.value = `Failed to load recent interactions: ${e.message || e}`;
		} finally {
			loading.value = false;
		}
	}

	watch(() => [props.socialAccountId, props.days], load, { immediate: true });
</script>

<style scoped>
	.interaction-stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(7rem, 1fr));
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.interaction-days {
		list-style: none;
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: 1fr;
		gap: 0.35rem;
		margin: 0 0 1.25rem;
		padding: 0;
	}

	.interaction-days li {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.2rem;
		min-width: 0;
	}

	.day-track {
		width: 100%;
		max-width: 2.5rem;
		height: 2.5rem;
		display: flex;
		align-items: flex-end;
		border-bottom: 1px solid var(--border-color, #ccc);
	}

	.day-bar {
		width: 100%;
		min-height: 2px;
		border-radius: 3px 3px 0 0;
		background: var(--link-color, #007bff);
	}

	.day-label {
		font-size: 0.75rem;
	}

	.interaction-list {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.interaction {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 0.65rem 0;
		border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.2));
	}

	.interaction:first-child {
		border-top: none;
	}

	.interaction-icon {
		flex-shrink: 0;
		display: inline-flex;
		justify-content: center;
		width: 2rem;
		color: var(--text-secondary, #666);
	}

	.interaction-engagement .interaction-icon {
		color: var(--link-color, #007bff);
	}

	.interaction-body {
		flex: 1;
		min-width: 0;
	}

	.interaction-summary {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem 0.6rem;
		align-items: baseline;
	}

	.interaction-summary time {
		font-size: 0.85rem;
	}

	.interaction-reply {
		margin: 0.25rem 0 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.interaction-post {
		display: block;
		margin-top: 0.2rem;
		font-size: 0.9rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.interaction-external {
		flex-shrink: 0;
		color: var(--text-secondary, #666);
	}
</style>
