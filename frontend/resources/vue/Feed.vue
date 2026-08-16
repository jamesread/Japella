<template>
	<Section
		title="Feed"
		subtitle="Recent posts from your connected social accounts."
		classes="feed"
		:padding="false"
	>
		<template #toolbar>
			<div class="toolbar-controls">
				<div class="filters">
					<FilterSelect
						v-model="selectedSocialAccountId"
						clear-title="Clear social account filter"
					>
						<option value="">All Social Accounts</option>
						<option v-for="account in socialAccounts" :key="account.id" :value="account.id">
							{{ account.identity }}
						</option>
					</FilterSelect>
					<FilterSelect
						v-model="selectedPostType"
						clear-title="Clear post type filter"
					>
						<option value="">All Post Types</option>
						<option value="regular">Regular</option>
						<option value="boost">Boost</option>
					</FilterSelect>
					<FilterSelect
						v-model="selectedAuthorId"
						clear-title="Clear author filter"
					>
						<option value="">All Authors</option>
						<option v-for="author in authorOptions" :key="author.id" :value="author.id">
							{{ author.name }}
						</option>
					</FilterSelect>
					<button
						v-if="hasActiveFilters"
						type="button"
						class="inline-icon neutral small"
						title="Clear Filters"
						@click="clearFilters"
					>
						<HugeiconsIcon
							:icon="FilterRemoveIcon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>Clear</span>
					</button>
				</div>
				<button
					type="button"
					class="inline-icon neutral"
					aria-label="Refresh"
					:disabled="!clientReady || feedRefreshing"
					@click="refreshFeed()"
				>
					<HugeiconsIcon
						:icon="RefreshIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
				</button>
			</div>
		</template>

		<Loading v-if="feedLoading && feed.length === 0" message="Loading feed..." :centered="true" />
	</Section>

	<Transition name="feed-refresh-fade">
		<p
			v-if="feedRefreshing"
			class="feed-refresh-notice"
			role="status"
			aria-live="polite"
		>
			Checking for new posts…
		</p>
	</Transition>

	<p v-if="!feedLoading && feed.length > 0 && filteredFeed.length === 0" class="feed-empty inline-notification note">
		No posts match the current filters.
	</p>

	<section class="small" v-for="post in visibleFeed" :key="post.id" :data-feed-post-id="post.id">
		<PostBoost
			v-if="isBoost(post)"
			:post="post"
			author-filterable
			:transparent-social-account-chip="true"
			@filter-author="setAuthorFilter"
			@post-refetched="onPostRefetched"
		/>
		<PostPreview
			v-else
			:post="post"
			author-filterable
			:transparent-social-account-chip="true"
			@filter-author="setAuthorFilter"
			@post-refetched="onPostRefetched"
		/>
	</section>

	<div
		v-if="!feedLoading && hasMore"
		ref="loadMoreSentinel"
		class="feed-load-sentinel"
		aria-hidden="true"
	/>

	<p v-if="!feedLoading && filteredFeed.length > 0 && !hasMore" class="feed-end muted">
		End of feed
	</p>
</template>

<script setup>
	import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { FilterRemoveIcon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Loading from './Loading.vue';
	import PostPreview from './PostPreview.vue';
	import PostBoost from './PostBoost.vue';
	import FilterSelect from './FilterSelect.vue';

	const iconStrokeWidth = 2.5;

	const PAGE_SIZE = 10;
	const AUTO_REFRESH_MS = 30_000;
	const SCROLL_TOP_THRESHOLD = 50;
	const FILTER_STORAGE_KEY = 'feed.filters';
	const POST_TYPES = ['', 'regular', 'boost'];

	function loadFilterPreferences() {
		try {
			const raw = localStorage.getItem(FILTER_STORAGE_KEY);
			if (!raw) {
				return { socialAccountId: '', postType: '', authorId: '' };
			}

			const parsed = JSON.parse(raw);
			const postType = POST_TYPES.includes(parsed.postType) ? parsed.postType : '';
			const socialAccountId = parsed.socialAccountId;
			const authorId = parsed.authorId;
			const normalizedAuthorId = typeof authorId === 'string' && authorId !== '' ? authorId : '';

			if (socialAccountId === '' || socialAccountId === null || socialAccountId === undefined) {
				return { socialAccountId: '', postType, authorId: normalizedAuthorId };
			}

			const id = Number(socialAccountId);
			return {
				socialAccountId: Number.isFinite(id) && id > 0 ? id : '',
				postType,
				authorId: normalizedAuthorId,
			};
		} catch {
			return { socialAccountId: '', postType: '', authorId: '' };
		}
	}

	function saveFilterPreferences() {
		localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify({
			socialAccountId: selectedSocialAccountId.value,
			postType: selectedPostType.value,
			authorId: selectedAuthorId.value,
		}));
	}

	const savedFilters = loadFilterPreferences();

	const feed = ref([]);
	const socialAccounts = ref([]);
	const selectedSocialAccountId = ref(savedFilters.socialAccountId);
	const selectedPostType = ref(savedFilters.postType);
	const selectedAuthorId = ref(savedFilters.authorId);
	const clientReady = ref(false);
	const visibleCount = ref(PAGE_SIZE);
	const feedLoading = ref(true);
	const feedRefreshing = ref(false);
	const loadMoreSentinel = ref(null);

	let loadMoreObserver = null;
	let autoRefreshTimer = null;
	let refreshInFlight = false;

	const authorOptions = computed(() => {
		const authors = new Map();

		for (const post of feed.value) {
			if (!post.authorId) {
				continue;
			}

			authors.set(post.authorId, {
				id: post.authorId,
				name: post.authorName || post.socialAccountIdentity || `Author ${post.authorId}`,
			});
		}

		return [...authors.values()].sort((a, b) => a.name.localeCompare(b.name));
	});

	const filteredFeed = computed(() => {
		let filtered = feed.value;

		if (selectedSocialAccountId.value) {
			filtered = filtered.filter((post) => post.socialAccountId == selectedSocialAccountId.value);
		}

		if (selectedAuthorId.value) {
			filtered = filtered.filter((post) => post.authorId == selectedAuthorId.value);
		}

		if (selectedPostType.value === 'regular') {
			filtered = filtered.filter((post) => !isBoost(post));
		} else if (selectedPostType.value === 'boost') {
			filtered = filtered.filter((post) => isBoost(post));
		}

		return filtered;
	});

	const hasActiveFilters = computed(() => {
		return selectedSocialAccountId.value !== ''
			|| selectedPostType.value !== ''
			|| selectedAuthorId.value !== '';
	});

	const visibleFeed = computed(() => filteredFeed.value.slice(0, visibleCount.value));

	const hasMore = computed(() => visibleCount.value < filteredFeed.value.length);

	async function getFeed() {
		if (!window.client) {
			return [];
		}

		try {
			const response = await window.client.getFeed();
			return response.posts || [];
		} catch (error) {
			console.error('Error fetching feed:', error);
			return [];
		}
	}

	function loadMore() {
		if (!hasMore.value) {
			return;
		}
		visibleCount.value = Math.min(visibleCount.value + PAGE_SIZE, filteredFeed.value.length);
	}

	function clearFilters() {
		selectedSocialAccountId.value = '';
		selectedPostType.value = '';
		selectedAuthorId.value = '';
		visibleCount.value = PAGE_SIZE;
	}

	function setAuthorFilter(authorId) {
		if (!authorId) {
			return;
		}

		selectedAuthorId.value = authorId;
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	function onPostRefetched(updatedPost) {
		const index = feed.value.findIndex((post) => post.id === updatedPost.id);
		if (index === -1) {
			return;
		}

		feed.value[index] = { ...feed.value[index], ...updatedPost };
	}

	function validateSavedSocialAccountFilter() {
		if (selectedSocialAccountId.value === '') {
			return;
		}

		const accountExists = socialAccounts.value.some(
			(account) => account.id == selectedSocialAccountId.value,
		);
		if (!accountExists) {
			selectedSocialAccountId.value = '';
			saveFilterPreferences();
		}
	}

	function validateSavedAuthorFilter() {
		if (selectedAuthorId.value === '') {
			return;
		}

		const authorExists = feed.value.some(
			(post) => post.authorId == selectedAuthorId.value,
		);
		if (!authorExists) {
			selectedAuthorId.value = '';
			saveFilterPreferences();
		}
	}

	async function loadFilterData() {
		if (!window.client) {
			return;
		}

		try {
			const response = await window.client.getSocialAccounts({ onlyActive: true });
			socialAccounts.value = response.accounts || [];
			validateSavedSocialAccountFilter();
		} catch (error) {
			console.error('Error loading social accounts for feed filters:', error);
		}
	}

	function setupInfiniteScroll() {
		if (loadMoreObserver) {
			loadMoreObserver.disconnect();
			loadMoreObserver = null;
		}

		if (!loadMoreSentinel.value || !hasMore.value) {
			return;
		}

		loadMoreObserver = new IntersectionObserver(
			(entries) => {
				if (entries[0]?.isIntersecting) {
					loadMore();
				}
			},
			{ rootMargin: '240px' },
		);
		loadMoreObserver.observe(loadMoreSentinel.value);
	}

	function isNearTop() {
		return window.scrollY < SCROLL_TOP_THRESHOLD;
	}

	function getScrollAnchor() {
		const sections = document.querySelectorAll('section.small[data-feed-post-id]');
		for (const element of sections) {
			const rect = element.getBoundingClientRect();
			if (rect.bottom > 0) {
				return {
					id: element.dataset.feedPostId,
					top: rect.top,
				};
			}
		}
		return null;
	}

	async function restoreScrollAnchor(anchor) {
		if (!anchor) {
			return;
		}

		await nextTick();
		const element = document.querySelector(`[data-feed-post-id="${anchor.id}"]`);
		if (!element) {
			return;
		}

		const delta = element.getBoundingClientRect().top - anchor.top;
		if (delta !== 0) {
			window.scrollBy(0, delta);
		}
	}

	async function applyFeedPosts(posts, { background }) {
		const preserveScroll = background && !isNearTop();
		const anchor = preserveScroll ? getScrollAnchor() : null;
		const previousVisibleCount = visibleCount.value;

		feed.value = posts;
		validateSavedAuthorFilter();

		if (background) {
			visibleCount.value = Math.max(previousVisibleCount, PAGE_SIZE);
		} else {
			visibleCount.value = PAGE_SIZE;
		}

		await restoreScrollAnchor(anchor);
	}

	function refreshFeed({ background = false } = {}) {
		if (refreshInFlight) {
			return;
		}

		refreshInFlight = true;
		const isInitialLoad = feed.value.length === 0;

		if (isInitialLoad) {
			feedLoading.value = true;
		} else {
			feedRefreshing.value = true;
		}

		getFeed()
			.then(async (posts) => {
				await applyFeedPosts(posts, { background: background || !isInitialLoad });
			})
			.catch((error) => {
				console.error('Error fetching feed:', error);
				if (!background && isInitialLoad) {
					feed.value = [];
				}
			})
			.finally(() => {
				feedLoading.value = false;
				feedRefreshing.value = false;
				refreshInFlight = false;
			});
	}

	function startAutoRefresh() {
		stopAutoRefresh();
		autoRefreshTimer = window.setInterval(() => {
			if (!clientReady.value || refreshInFlight) {
				return;
			}
			refreshFeed({ background: true });
		}, AUTO_REFRESH_MS);
	}

	function stopAutoRefresh() {
		if (autoRefreshTimer !== null) {
			window.clearInterval(autoRefreshTimer);
			autoRefreshTimer = null;
		}
	}

	function isBoost(post) {
		if (post.remoteId) {
			try {
				const parsed = JSON.parse(post.remoteId);
				if (parsed && parsed['@context'] && parsed.type === 'Announce') {
					return true;
				}
			} catch (e) {
				// Not JSON, continue checking other indicators
			}
		}

		if (post.remoteUrl && (
			post.remoteUrl.includes('/activity') ||
			post.remoteUrl.match(/\/statuses\/\d+\/activity/)
		)) {
			const content = (post.content || '').trim();
			if (content === '' || content.length < 50) {
				return true;
			}
		}

		const content = (post.content || '').trim();
		if (content === '' && post.remoteUrl) {
			if (post.remoteUrl.includes('mastodon') ||
			    post.remoteUrl.includes('fosstodon') ||
			    post.remoteUrl.match(/https?:\/\/[^\/]+\/(users|@)[^\/]+\/statuses\/\d+/)) {
				return true;
			}
		}

		return false;
	}

	watch([selectedSocialAccountId, selectedPostType, selectedAuthorId], () => {
		visibleCount.value = PAGE_SIZE;
		saveFilterPreferences();
	});

	watch([loadMoreSentinel, hasMore, () => filteredFeed.value.length], () => {
		nextTick(() => setupInfiniteScroll());
	});

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await loadFilterData();
		refreshFeed({ background: false });
		startAutoRefresh();
	});

	onUnmounted(() => {
		stopAutoRefresh();
		loadMoreObserver?.disconnect();
	});
</script>

<style scoped>
	.toolbar-controls {
		display: flex;
		align-items: center;
		gap: 1rem;
		width: 100%;
	}

	.filters {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex: 1;
		flex-wrap: wrap;
	}

	section {
		margin-bottom: 1rem;
		padding: 0;
	}

	.feed-empty {
		margin: 0 0 1rem;
	}

	.feed-refresh-notice {
		position: fixed;
		top: 0.75rem;
		left: 50%;
		transform: translateX(-50%);
		z-index: 100;
		margin: 0;
		padding: 0.4rem 0.85rem;
		border-radius: 999px;
		font-size: 0.85rem;
		color: var(--text-secondary, #555);
		background: rgba(255, 255, 255, 0.92);
		border: 1px solid rgba(0, 0, 0, 0.08);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
		pointer-events: none;
	}

	.feed-refresh-fade-enter-active,
	.feed-refresh-fade-leave-active {
		transition: opacity 0.2s ease, transform 0.2s ease;
	}

	.feed-refresh-fade-enter-from,
	.feed-refresh-fade-leave-to {
		opacity: 0;
		transform: translateX(-50%) translateY(-0.35rem);
	}

	.feed-load-sentinel {
		height: 1px;
		margin: 0;
	}

	.feed-end {
		text-align: center;
		padding: 1rem 0 2rem;
		font-size: 0.9em;
		opacity: 0.7;
	}

	.muted {
		opacity: 0.7;
	}
</style>
