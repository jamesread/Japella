<template>
	<NotificationBlock v-if="columnsError" type="error" :message="columnsError" class="feed-empty" />

	<div class="feed-layout">
		<FeedColumnFrame ref="mainColumn" title="Feed" icon="mdi:rss" class="feed-main" label="Feed" width-key="feed">
			<template #actions>
				<button
					type="button"
					class="neutral small"
					title="Add a column for a hashtag, search or tag"
					aria-label="Add column"
					:disabled="!clientReady || columns.length >= MAX_COLUMNS"
					@click="openColumnDialog()"
				>
					<Icon icon="mdi:table-column-plus-after" />
				</button>
				<button
					type="button"
					class="neutral small"
					title="Refresh"
					aria-label="Refresh feed"
					:disabled="!clientReady || feedRefreshing"
					@click="refreshFeed()"
				>
					<Icon icon="mdi:refresh" />
				</button>
			</template>

			<template #meta>
				<div class="filters" role="group" aria-label="Feed filters">
					<FilterSelect
						compact
						v-model="selectedSocialAccountId"
						clear-title="Clear social account filter"
					>
						<option value="">All Social Accounts</option>
						<option v-for="account in socialAccounts" :key="account.id" :value="account.id">
							{{ account.identity }}
						</option>
					</FilterSelect>
					<FilterSelect
						compact
						v-model="selectedPostType"
						clear-title="Clear post type filter"
					>
						<option value="">All Post Types</option>
						<option value="regular">Regular</option>
						<option value="boost">Boost</option>
					</FilterSelect>
					<FilterSelect
						compact
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
						class="neutral small filters-clear"
						title="Clear all filters"
						aria-label="Clear all filters"
						@click="clearFilters"
					>
						<Icon icon="mdi:filter-remove-outline" />
					</button>
				</div>
			</template>

			<p v-if="feedRefreshing" class="subtle feed-refresh-notice" role="status" aria-live="polite">
				Checking for new posts…
			</p>

			<Loading v-if="feedLoading && feed.length === 0" message="Loading feed..." :centered="true" />

			<NotificationBlock
				v-if="!feedLoading && feed.length > 0 && filteredFeed.length === 0"
				type="note"
				message="No posts match the current filters."
			/>

			<article v-for="post in visibleFeed" :key="post.id" class="feed-column-post" :data-feed-post-id="post.id">
				<PostBoost
					v-if="isBoost(post)"
					:post="post"
					filled
					interactive
					author-filterable
					:transparent-social-account-chip="true"
					@filter-author="setAuthorFilter"
					@post-refetched="onPostRefetched"
					@post-hidden="onPostHidden"
				/>
				<PostPreview
					v-else
					:post="post"
					filled
					interactive
					author-filterable
					:transparent-social-account-chip="true"
					@filter-author="setAuthorFilter"
					@post-refetched="onPostRefetched"
					@post-hidden="onPostHidden"
				/>
			</article>

			<div
				v-if="!feedLoading && hasMore"
				ref="loadMoreSentinel"
				class="feed-load-sentinel"
				aria-hidden="true"
			/>

			<p v-if="!feedLoading && filteredFeed.length > 0 && !hasMore" class="feed-end muted">
				End of feed
			</p>
		</FeedColumnFrame>

		<FeedColumn
			v-for="(column, index) in columns"
			:key="column.id"
			:column="column"
			:first="index === 0"
			:last="index === columns.length - 1"
			@edit="openColumnDialog"
			@move="moveColumn"
			@remove="removeColumn"
		/>

		<button
			v-if="clientReady && columns.length < MAX_COLUMNS"
			type="button"
			class="feed-add-column"
			title="Add a column for a hashtag, search or tag"
			:style="{ width: `${DEFAULT_COLUMN_WIDTH}px` }"
			@click="openColumnDialog()"
		>
			<Icon icon="mdi:plus" width="20" height="20" aria-hidden="true" />
			<span>Add column</span>
		</button>
	</div>

	<FeedColumnDialog ref="columnDialog" :accounts="socialAccounts" @saved="onColumnSaved" />
</template>

<script setup>
	import { ref, onMounted, onUnmounted, computed, watch, nextTick, provide } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { DEFAULT_COLUMN_WIDTH, forgetColumnWidth } from '../javascript/feedColumnWidths';
	import { Icon } from '@iconify/vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Loading from './Loading.vue';
	import PostPreview from './PostPreview.vue';
	import PostBoost from './PostBoost.vue';
	import FilterSelect from './FilterSelect.vue';
	import FeedColumn from './FeedColumn.vue';
	import FeedColumnFrame from './FeedColumnFrame.vue';
	import FeedColumnDialog from './FeedColumnDialog.vue';
	import { TAG_QUERY_PREFIX } from '../javascript/postTags.js';


	const PAGE_SIZE = 10;
	// Matches the server's per-user limit.
	const MAX_COLUMNS = 10;

	const columns = ref([]);
	const columnsError = ref('');
	const columnDialog = ref(null);

	async function loadColumns() {
		try {
			const res = await window.client.listFeedColumns({});
			columns.value = res.columns || [];
			columnsError.value = '';
		} catch (e) {
			columnsError.value = `Failed to load feed columns: ${e.rawMessage || e.message || e}`;
		}
	}

	function openColumnDialog(column = null) {
		const defaults = selectedSocialAccountId.value ? [Number(selectedSocialAccountId.value)] : [];
		columnDialog.value?.open(column, defaults);
	}

	function onColumnSaved(column, updated) {
		if (updated) {
			columns.value = columns.value.map((c) => (c.id === column.id ? column : c));
		} else {
			columns.value = [...columns.value, column];
		}
		scrollToColumn(column.id);
	}

	function scrollToColumn(id) {
		nextTick(() => {
			document.querySelector(`[data-feed-column-id="${id}"]`)
				?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
		});
	}

	// Shows the column for an internal tag, adding it if there isn't one yet.
	async function openTagColumn(tag) {
		const existing = columns.value.find((c) => c.tag === tag);
		if (existing) {
			scrollToColumn(existing.id);
			return;
		}
		if (columns.value.length >= MAX_COLUMNS) {
			window.alert(`You already have ${MAX_COLUMNS} columns. Remove one to add the ${tag} tag column.`);
			return;
		}
		try {
			const res = await window.client.createFeedColumn({ query: `${TAG_QUERY_PREFIX}${tag}`, title: '', socialAccountIds: [] });
			columns.value = [...columns.value, res.column];
			columnsError.value = '';
			scrollToColumn(res.column.id);
		} catch (e) {
			columnsError.value = `Failed to add the ${tag} tag column: ${e.rawMessage || e.message || e}`;
		}
	}

	provide('openTagColumn', openTagColumn);

	// Adds a search column for a hashtag, searched with the given account.
	async function openHashtagColumn(tag, socialAccountId) {
		if (columns.value.length >= MAX_COLUMNS) {
			window.alert(`You already have ${MAX_COLUMNS} columns. Remove one to add a column for #${tag}.`);
			return;
		}
		try {
			const res = await window.client.createFeedColumn({ query: `#${tag}`, title: '', socialAccountIds: [socialAccountId] });
			columns.value = [...columns.value, res.column];
			columnsError.value = '';
			scrollToColumn(res.column.id);
		} catch (e) {
			columnsError.value = `Failed to add a column for #${tag}: ${e.rawMessage || e.message || e}`;
		}
	}

	provide('openHashtagColumn', openHashtagColumn);
	provide('feedColumns', columns);

	async function moveColumn(column, direction) {
		const from = columns.value.findIndex((c) => c.id === column.id);
		const to = from + direction;
		if (from < 0 || to < 0 || to >= columns.value.length) {
			return;
		}
		const previous = columns.value;
		const next = [...previous];
		[next[from], next[to]] = [next[to], next[from]];
		columns.value = next;
		try {
			await window.client.reorderFeedColumns({ ids: next.map((c) => c.id) });
			columnsError.value = '';
		} catch (e) {
			columns.value = previous;
			columnsError.value = `Failed to move column: ${e.rawMessage || e.message || e}`;
		}
	}

	async function removeColumn(column) {
		if (!window.confirm(`Remove the "${column.title}" column?`)) {
			return;
		}
		try {
			await window.client.deleteFeedColumn({ id: column.id });
			columns.value = columns.value.filter((c) => c.id !== column.id);
			forgetColumnWidth(`column-${column.id}`);
			columnsError.value = '';
		} catch (e) {
			columnsError.value = `Failed to remove column: ${e.rawMessage || e.message || e}`;
		}
	}
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
	const mainColumn = ref(null);

	// The main feed scrolls inside its column rather than the page.
	function feedScroller() {
		return mainColumn.value?.body || null;
	}

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
		feedScroller()?.scrollTo({ top: 0, behavior: 'smooth' });
	}

	function onPostRefetched(updatedPost) {
		const index = feed.value.findIndex((post) => post.id === updatedPost.id);
		if (index === -1) {
			return;
		}

		feed.value[index] = { ...feed.value[index], ...updatedPost };
	}

	function onPostHidden(hidden) {
		feed.value = feed.value.filter((post) => post.id !== hidden.id);
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
			{ root: feedScroller(), rootMargin: '240px' },
		);
		loadMoreObserver.observe(loadMoreSentinel.value);
	}

	function isNearTop() {
		const scroller = feedScroller();
		return !scroller || scroller.scrollTop < SCROLL_TOP_THRESHOLD;
	}

	// Anchors are measured relative to the column, so a background refresh that
	// adds posts above keeps the reader's place.
	function getScrollAnchor() {
		const scroller = feedScroller();
		if (!scroller) {
			return null;
		}
		const top = scroller.getBoundingClientRect().top;
		for (const element of scroller.querySelectorAll('[data-feed-post-id]')) {
			const rect = element.getBoundingClientRect();
			if (rect.bottom > top) {
				return {
					id: element.dataset.feedPostId,
					top: rect.top - top,
				};
			}
		}
		return null;
	}

	async function restoreScrollAnchor(anchor) {
		const scroller = feedScroller();
		if (!anchor || !scroller) {
			return;
		}

		await nextTick();
		const element = scroller.querySelector(`[data-feed-post-id="${anchor.id}"]`);
		if (!element) {
			return;
		}

		const delta = element.getBoundingClientRect().top - scroller.getBoundingClientRect().top - anchor.top;
		if (delta !== 0) {
			scroller.scrollBy(0, delta);
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
		loadColumns();
	});

	onUnmounted(() => {
		stopAutoRefresh();
		loadMoreObserver?.disconnect();
	});
</script>

<style scoped>
	/* Columns keep their own widths and sit in one row; the row scrolls sideways
	   when they don't all fit. */
	.feed-layout {
		display: flex;
		align-items: stretch;
		gap: 1rem;
		overflow-x: auto;
		overscroll-behavior-x: contain;
		padding-bottom: 0.5rem;
	}

	.feed-layout > :deep(.feed-column),
	.feed-add-column {
		height: calc(100vh - var(--picocrank-fixed-header-offset, 0px) - 3.5rem);
	}

	/* A quiet full-height placeholder after the last column, at the default column width. */
	.feed-add-column {
		flex: 0 0 auto;
		max-width: calc(100vw - 2rem);
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.4rem;
		padding: 0;
		font-size: 0.95rem;
		margin: 0;
		border: 1px dashed var(--border-color, rgba(127, 127, 127, 0.35));
		border-radius: .4em;
		background: transparent;
		color: var(--text-secondary, #888);
		box-shadow: none;
		opacity: 0.6;
		cursor: pointer;
		transition: opacity 0.15s ease, border-color 0.15s ease;
	}

	.feed-add-column:hover,
	.feed-add-column:focus-visible {
		opacity: 1;
		border-color: var(--link-color, #007bff);
		color: var(--link-color, #007bff);
	}

	/* The filters share the column's single detail line equally. */
	.filters {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		flex: 1;
		min-width: 0;
		height: 100%;
	}

	.filters-clear {
		height: 100%;
		padding: 0 0.25rem;
		flex-shrink: 0;
	}

	.feed-empty {
		margin: 0 0 1rem;
	}

	.feed-refresh-notice {
		margin: 0 0 0.5rem;
		font-size: 0.85rem;
		text-align: center;
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
