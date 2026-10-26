<template>
	<FeedColumnFrame
		:title="column.title"
		:icon="column.tag ? 'mdi:tag-outline' : (column.isHashtag ? 'mdi:pound' : 'mdi:magnify')"
		:tooltip="column.tag ? `Posts you tagged ${column.tag} (private to you)` : column.query"
		:label="`Feed column: ${column.title}`"
		:data-feed-column-id="column.id"
		:width-key="`column-${column.id}`"
	>
		<template #actions>
			<button type="button" class="neutral small" title="Refresh" aria-label="Refresh column" :disabled="loading" @click="load">
				<Icon icon="mdi:refresh" />
			</button>
			<button type="button" class="neutral small" title="Edit" aria-label="Edit column" @click="emit('edit', column)">
				<Icon icon="mdi:pencil-outline" />
			</button>
			<button type="button" class="neutral small" title="Move left" aria-label="Move column left" :disabled="first" @click="emit('move', column, -1)">
				<Icon icon="mdi:chevron-left" />
			</button>
			<button type="button" class="neutral small" title="Move right" aria-label="Move column right" :disabled="last" @click="emit('move', column, 1)">
				<Icon icon="mdi:chevron-right" />
			</button>
			<button type="button" class="neutral small" title="Remove" aria-label="Remove column" @click="emit('remove', column)">
				<Icon icon="mdi:close" />
			</button>
		</template>

		<template #meta>
			<span v-if="column.tag" class="feed-column-account">
				<Icon icon="mdi:lock-outline" width="14" height="14" aria-hidden="true" />
				Your tagged posts
			</span>
			<span
				v-for="account in column.accounts"
				:key="account.id"
				class="feed-column-account"
				:class="{ unsupported: !account.supported }"
				:title="account.supported ? account.identity : `${account.identity}: search isn't available for this network`"
			>
				<Icon :icon="account.icon" width="14" height="14" aria-hidden="true" />
				{{ account.identity }}
			</span>
			<span v-if="!column.tag && !column.accounts.length">No accounts</span>
			<span v-if="fetchedAt">· updated {{ formatHumanDateTime(fetchedAt) }}</span>
		</template>

		<NotificationBlock v-if="!supported" type="note" message="Search isn't available for any of this column's accounts." />
		<template v-else>
			<NotificationBlock v-if="error" type="warning" :message="error" />
			<Loading v-if="loading && !posts.length" message="Loading…" />
			<p v-else-if="!posts.length && !error && column.tag" class="subtle feed-column-empty">
				No posts tagged {{ column.tag }} yet. Right-click a post and choose Tags… to add one.
			</p>
			<p v-else-if="!posts.length && !error" class="subtle feed-column-empty">No recent posts found.</p>
			<article v-for="post in posts" :key="post.remoteId || post.remoteUrl" class="feed-column-post">
				<PostPreview :post="post" filled interactive :transparent-social-account-chip="true" @post-hidden="onPostHidden" @post-refetched="onPostRefetched" />
			</article>
		</template>
	</FeedColumnFrame>
</template>

<script setup>
	import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Loading from './Loading.vue';
	import FeedColumnFrame from './FeedColumnFrame.vue';
	import PostPreview from './PostPreview.vue';
	import { formatHumanDateTime, waitForClient } from '../javascript/util';
	import { tagsVersion } from '../javascript/postTags.js';

	const props = defineProps({
		column: {
			type: Object,
			required: true,
		},
		first: {
			type: Boolean,
			default: false,
		},
		last: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['edit', 'move', 'remove']);

	// The server caches results for a minute; refreshing more often only re-reads that cache.
	const REFRESH_MS = 2 * 60 * 1000;

	// Tag columns read Japella's own records, so they need no searchable account.
	const supported = computed(() => Boolean(props.column.tag) || (props.column.accounts || []).some((a) => a.supported));

	const posts = ref([]);
	const loading = ref(false);
	const error = ref('');
	const fetchedAt = ref('');
	let timer = null;

	async function load() {
		if (!supported.value || loading.value) {
			return;
		}
		loading.value = true;
		try {
			await waitForClient();
			const res = await window.client.getFeedColumnPosts({ id: props.column.id });
			posts.value = res.posts || [];
			error.value = res.error || '';
			fetchedAt.value = res.fetchedAt || '';
		} catch (e) {
			error.value = `Failed to load column: ${e.rawMessage || e.message || e}`;
		} finally {
			loading.value = false;
		}
	}

	function postKey(post) {
		return post.remoteUrl || post.remoteId || '';
	}

	function onPostHidden(hidden) {
		const key = postKey(hidden);
		posts.value = posts.value.filter((post) => postKey(post) !== key);
	}

	function onPostRefetched(updated) {
		const key = postKey(updated);
		posts.value = posts.value.map((post) => (postKey(post) === key ? { ...post, ...updated } : post));
	}

	watch(() => [props.column.query, (props.column.accounts || []).map((a) => a.id).join(',')], () => {
		posts.value = [];
		load();
	});

	// A tag column changes whenever the user tags or untags a post anywhere.
	watch(tagsVersion, () => {
		if (props.column.tag) {
			load();
		}
	});

	onMounted(() => {
		load();
		timer = window.setInterval(load, REFRESH_MS);
	});

	onUnmounted(() => {
		if (timer !== null) {
			window.clearInterval(timer);
		}
	});
</script>

<style scoped>
	.feed-column-account {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		flex-shrink: 0;
	}

	.feed-column-account.unsupported {
		text-decoration: line-through;
		opacity: 0.6;
	}

	.feed-column-empty {
		margin: 0.5rem 0;
	}
</style>
