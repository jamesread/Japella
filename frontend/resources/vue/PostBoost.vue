<template>
	<div class="post-boost" :class="{ filled }" @contextmenu="contextMenu.onContextMenu">
		<div class="boost-header">
			<div class="boost-info">
				<Icon icon="mdi:repeat" class="boost-icon" />
				<template v-if="post.socialAccountId && post.authorId">
					<!-- The booster opens the author options dialog. -->
					<button
						type="button"
						class="author-avatar-button avatar-link"
						:title="'Author options for ' + displayAuthorName"
						:aria-label="'Author options for ' + displayAuthorName"
						@click="authorOptionsFor = 'booster'"
					>
						<FeedAuthorAvatar
							:avatar-url="post.authorAvatarUrl"
							:fallback-icon="post.socialAccountIcon || 'mdi:account'"
							:alt-text="displayAuthorName || 'Author'"
						/>
					</button>
					<button
						type="button"
						class="author-filter-button account-name"
						:title="'Author options for ' + displayAuthorName"
						@click="authorOptionsFor = 'booster'"
					>
						{{ displayAuthorName }}
					</button>
				</template>
				<template v-else-if="post.socialAccountId">
					<router-link
						:to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }"
						class="social-account-link avatar-link"
						:title="'View social account ' + (post.socialAccountIdentity || '')"
					>
						<FeedAuthorAvatar
							:avatar-url="post.authorAvatarUrl"
							:fallback-icon="post.socialAccountIcon || 'mdi:account'"
							:alt-text="displayAuthorName || 'Author'"
						/>
					</router-link>
					<router-link
						:to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }"
						class="social-account-link"
					>
						<span class="account-name">{{ displayAuthorName }}</span>
					</router-link>
				</template>
				<div v-else class="social-account-link">
					<Icon icon="mdi:account" />
					<span class="account-name">{{ displayAuthorName || 'Unknown' }}</span>
				</div>
				<span class="boost-text">boosted</span>
			</div>
			<div class="post-header-actions">
				<SocialAccountChip
					v-if="post.socialAccountId"
					:social-account-id="post.socialAccountId"
					:identity="post.socialAccountIdentity"
					:icon="post.socialAccountIcon"
					:transparent="transparentSocialAccountChip"
				/>
				<button
					v-if="post.id || post.remoteId"
					@click="openDiagnosticDialog"
					class="diagnostic-button neutral small"
					title="Diagnostic Info"
				>
					<Icon icon="mdi:information-outline" />
				</button>
			</div>
		</div>

		<div class="boosted-content">
			<button
				v-if="originalAuthorName"
				type="button"
				class="boosted-author"
				:title="'Author options for ' + originalAuthorName"
				@click="authorOptionsFor = 'original'"
			>
				<FeedAuthorAvatar
					:avatar-url="post.originalAuthorAvatarUrl"
					fallback-icon="mdi:account"
					:alt-text="originalAuthorName"
				/>
				<span class="boosted-author-name">{{ originalAuthorName }}</span>
			</button>

			<div
				v-if="post.content"
				class="boosted-post-message"
				@click="hashtags.onContentClick"
				v-html="hashtags.contentHtml.value"
			/>

			<PostInteractionBar v-if="interactive" :post="post" />
			<PostTags v-if="interactive" ref="postTags" :post="post" />

			<div v-if="post.previewUrl || post.previewImageUrl" class="preview-card">
				<a
					v-if="post.previewUrl"
					:href="post.previewUrl"
					target="_blank"
					rel="noopener noreferrer"
					class="preview-card-link"
				>
					<div v-if="post.previewImageUrl" class="preview-image">
						<img :src="post.previewImageUrl" :alt="post.previewTitle || 'Preview image'" />
					</div>
					<div class="preview-content">
						<div v-if="post.previewTitle" class="preview-title">{{ post.previewTitle }}</div>
						<div v-if="post.previewDescription" class="preview-description">{{ post.previewDescription }}</div>
						<div v-if="post.previewUrl" class="preview-url">{{ getDomainFromUrl(post.previewUrl) }}</div>
					</div>
				</a>
				<div v-else class="preview-card-static">
					<div v-if="post.previewImageUrl" class="preview-image">
						<img :src="post.previewImageUrl" :alt="post.previewTitle || 'Preview image'" />
					</div>
					<div class="preview-content">
						<div v-if="post.previewTitle" class="preview-title">{{ post.previewTitle }}</div>
						<div v-if="post.previewDescription" class="preview-description">{{ post.previewDescription }}</div>
					</div>
				</div>
			</div>

			<div v-if="!post.content && !post.previewUrl && !post.previewImageUrl" class="boosted-fallback">
				<a
					v-if="boostedObjectUrl"
					:href="boostedObjectUrl"
					target="_blank"
					rel="noopener noreferrer"
					class="boosted-link"
				>
					<Icon icon="mdi:open-in-new" width="16" height="16" />
					<span>View Boosted Post</span>
				</a>
				<div v-else class="boosted-object-info">
					<span class="boosted-object-label">Boosted Object:</span>
					<span class="boosted-object-value">Unknown</span>
				</div>
			</div>
		</div>

		<div v-if="boostedObjectUrl || post.remoteUrl" class="post-actions">
			<a
				v-if="boostedObjectUrl"
				:href="boostedObjectUrl"
				target="_blank"
				rel="noopener noreferrer"
				class="original-post-link"
			>
				<Icon icon="mdi:open-in-new" width="16" height="16" />
				<span>View original post</span>
			</a>
			<a
				v-else-if="post.remoteUrl"
				:href="post.remoteUrl"
				target="_blank"
				rel="noopener noreferrer"
				class="original-post-link"
			>
				<Icon icon="mdi:open-in-new" width="16" height="16" />
				<span>View Boost Activity</span>
			</a>
		</div>

		<div v-if="post.postedDate || post.created" class="post-datetime">
			{{ formatHumanDateTime(post.postedDate || post.created) }}
		</div>
	</div>

	<TagOptionsDialog
		v-if="hashtags.tagOptions.value"
		:tag="hashtags.tagOptions.value.tag"
		:url="hashtags.tagOptions.value.url"
		:post="post"
		@close="hashtags.tagOptions.value = null"
	/>

	<TableRowContextMenu
		v-model:open="contextMenu.open.value"
		:items="contextMenu.items.value"
		:client-x="contextMenu.x.value"
		:client-y="contextMenu.y.value"
		@select="contextMenu.onSelect"
	/>

	<AuthorOptionsDialog
		v-if="authorOptionsFor"
		:author="authorOptionsAuthor"
		:filterable="authorFilterable"
		@filter-author="filterByAuthor"
		@close="authorOptionsFor = null"
	/>

	<!-- Diagnostic Info Dialog -->
	<PostDiagnosticDialog
		:open="showDiagnosticDialog"
		:post="post"
		@close="closeDiagnosticDialog"
		@post-refetched="onPostRefetched"
		@post-hidden="emit('post-hidden', $event)"
	/>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { Icon } from '@iconify/vue';
	import { formatHumanDateTime } from '../javascript/util';
	import FeedAuthorAvatar from './FeedAuthorAvatar.vue';
	import AuthorOptionsDialog from './AuthorOptionsDialog.vue';
	import PostInteractionBar from './PostInteractionBar.vue';
	import PostTags from './PostTags.vue';
	import TagOptionsDialog from './TagOptionsDialog.vue';
	import { useHashtagOptions } from '../javascript/hashtags.js';
	import TableRowContextMenu from 'picocrank/vue/components/TableRowContextMenu.vue';
	import { usePostContextMenu } from '../javascript/postActions.js';
	import PostDiagnosticDialog from './PostDiagnosticDialog.vue';
	import SocialAccountChip from './SocialAccountChip.vue';

	const props = defineProps({
		post: {
			type: Object,
			required: true
		},
		interactive: {
			// Show like and comments buttons (feed and feed columns).
			type: Boolean,
			default: false,
		},
		filled: {
			// Draw the card background itself, for lists not wrapped in a section.
			type: Boolean,
			default: false,
		},
		authorFilterable: {
			type: Boolean,
			default: false,
		},
		transparentSocialAccountChip: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['filter-author', 'post-refetched', 'post-hidden']);

	// Diagnostic dialog state
	const showDiagnosticDialog = ref(false);

	function openDiagnosticDialog() {
		showDiagnosticDialog.value = true;
	}

	function closeDiagnosticDialog() {
		showDiagnosticDialog.value = false;
	}

	// Parse ActivityStreams JSON from remoteId
	const activityStreams = computed(() => {
		if (!props.post.remoteId) {
			return null;
		}

		try {
			// Try to parse as JSON
			const parsed = JSON.parse(props.post.remoteId);
			if (parsed && parsed['@context'] && parsed.type) {
				return parsed;
			}
		} catch (e) {
			// Not JSON, return null
			return null;
		}

		return null;
	});

	// Extract actor from remoteUrl if remoteId doesn't contain JSON
	const actorFromUrl = computed(() => {
		if (activityStreams.value) {
			return null; // Already extracted from JSON
		}

		if (!props.post.remoteUrl) {
			return null;
		}

		// Extract username from URL like "https://fosstodon.org/users/jimsalter/statuses/..."
		const match = props.post.remoteUrl.match(/\/users\/([^\/]+)/);
		if (match) {
			return match[1];
		}

		// Try ActivityPub actor format
		const actorMatch = props.post.remoteUrl.match(/@([^\/@]+)@([^\/]+)/);
		if (actorMatch) {
			return actorMatch[1];
		}

		return null;
	});

	// Extract boost actor (who boosted)
	const boostActor = computed(() => {
		// First try to get from ActivityStreams JSON
		if (activityStreams.value) {
			const actor = activityStreams.value.actor;
			if (typeof actor === 'string') {
				// Extract username from URL like "https://fosstodon.org/users/jimsalter"
				const match = actor.match(/\/users\/([^\/]+)/);
				if (match) {
					return match[1];
				}
				return actor;
			}
		}

		// Fall back to extracting from remoteUrl
		if (actorFromUrl.value) {
			return actorFromUrl.value;
		}

		// Fall back to author name from post
		return props.post.authorName || props.post.socialAccountIdentity;
	});

	const displayAuthorName = computed(() => boostActor.value);

	const originalAuthorName = computed(() => {
		return props.post.originalAuthorName || '';
	});

	// Boosts only appear in feeds.
	const postTags = ref(null);
	const hashtags = useHashtagOptions(() => props.post);
	const contextMenu = usePostContextMenu(() => props.post, () => true, emit, {
		onEditTags: props.interactive ? () => postTags.value?.openEditor() : null,
	});

	// Which author's options are open: 'booster', 'original' or null.
	const authorOptionsFor = ref(null);

	const authorOptionsAuthor = computed(() => {
		const p = props.post;
		const via = {
			socialAccountId: p.socialAccountId,
			socialAccountIdentity: p.socialAccountIdentity,
			socialAccountIcon: p.socialAccountIcon,
		};
		if (authorOptionsFor.value === 'original') {
			return {
				...via,
				id: p.originalAuthorId,
				name: p.originalAuthorName,
				handle: p.originalAuthorHandle,
				avatarUrl: p.originalAuthorAvatarUrl,
				profileUrl: p.originalAuthorProfileUrl,
				external: true,
			};
		}
		return {
			...via,
			id: p.authorId,
			name: displayAuthorName.value,
			handle: p.authorHandle,
			avatarUrl: p.authorAvatarUrl,
			profileUrl: p.authorProfileUrl,
			// The boost's URL is the booster's activity, so it locates the booster.
			postUrl: p.remoteUrl,
			fallbackIcon: p.socialAccountIcon,
			external: Boolean(p.fromExternalAuthor),
		};
	});

	function filterByAuthor() {
		const id = authorOptionsAuthor.value.id;
		if (id) {
			emit('filter-author', id);
		}
	}

	function onPostRefetched(updatedPost) {
		emit('post-refetched', updatedPost);
	}

	function getDomainFromUrl(url) {
		if (!url) {
			return '';
		}
		try {
			const urlObj = new URL(url);
			return urlObj.hostname.replace('www.', '');
		} catch (e) {
			return url;
		}
	}

	// Extract boosted object URL
	const boostedObjectUrl = computed(() => {
		// First try to get from ActivityStreams JSON
		if (activityStreams.value) {
			const obj = activityStreams.value.object;
			if (typeof obj === 'string') {
				return obj;
			}
		}

		// If remoteUrl is an activity URL, try to extract the object URL
		// Activity URLs often look like: .../statuses/123/activity
		// The object would be: .../statuses/123
		if (props.post.remoteUrl && props.post.remoteUrl.endsWith('/activity')) {
			return props.post.remoteUrl.replace('/activity', '');
		}

		return null;
	});

</script>

<style scoped>
.post-boost {
	max-width: 800px;
	margin: 0 auto;
	padding: 1rem;
	border-radius: 0.5rem;
	border-left: 3px solid var(--boost-color, #6366f1);
	background-color: var(--bg-secondary, #f9fafb);
	position: relative;
}

/* Matches femtocrank's section.small card; the left border still marks a boost. */
.post-boost.filled {
	background-color: var(--section-bg-color);
	border-radius: .8em;
	box-shadow: 0 0 .5em var(--shadow-color);
}

.boost-header {
	display: flex;
	align-items: start;
	justify-content: space-between;
	gap: 0.5rem;
	margin-bottom: 1rem;
}

/* Takes only the space the author leaves, so the account chip truncates first. */
.post-header-actions {
	display: flex;
	flex: 1 1 0;
	justify-content: flex-end;
	align-items: center;
	gap: 0.5rem;
	min-width: 4rem;
}

.post-header-actions > * {
	flex-shrink: 0;
}

.post-header-actions > .social-account-chip {
	flex-shrink: 1;
}

.boost-info {
	display: flex;
	min-width: 0;
	align-items: center;
	gap: 0.5rem;
	font-size: 1rem;
	font-weight: 500;
}

.boost-icon {
	color: var(--boost-color, #6366f1);
}

.social-account-link {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	text-decoration: none;
	color: var(--link-color, #007bff);
}

.social-account-link:hover {
	text-decoration: underline;
}

.account-name {
	font-weight: bold;
}

.author-filter-button {
	padding: 0;
	border: none;
	background: none;
	font: inherit;
	font-weight: bold;
	color: var(--link-color, #007bff);
	cursor: pointer;
}

.author-filter-button:hover {
	text-decoration: underline;
}

.avatar-link:hover {
	text-decoration: none;
}

.boost-text {
	color: var(--text-secondary, #666);
	font-weight: normal;
	font-style: italic;
}

.diagnostic-button {
	display: flex;
	align-items: center;
	justify-content: center;
}

.boosted-content {
	margin: 1rem 0;
	padding: 1rem;
	background-color: var(--bg-primary, #ffffff);
	border-radius: 0.25rem;
	border: 1px solid var(--border-color, #e5e7eb);
}

.boosted-author {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin-bottom: 0.75rem;
	padding: 0;
	border: none;
	background: none;
	box-shadow: none;
	font: inherit;
	color: inherit;
	text-align: left;
	cursor: pointer;
}

.boosted-author:hover .boosted-author-name {
	text-decoration: underline;
}

.author-avatar-button {
	display: inline-flex;
	padding: 0;
	border: none;
	background: none;
	box-shadow: none;
	cursor: pointer;
	border-radius: 50%;
}

.boosted-author-name {
	font-weight: bold;
	color: var(--text-primary, #333);
}

.boosted-post-message {
	font-weight: 500;
	word-wrap: break-word;
}

.boosted-post-message :deep(a) {
	color: var(--link-color, #007bff);
	text-decoration: underline;
}

.boosted-post-message :deep(a:hover) {
	color: var(--link-hover-color, #0056b3);
}

.boosted-post-message :deep(p) {
	margin: 0.5em 0;
}

.boosted-post-message :deep(p:first-child) {
	margin-top: 0;
}

.boosted-post-message :deep(p:last-child) {
	margin-bottom: 0;
}

.preview-card {
	margin: 1rem 0 0;
	border: 1px solid var(--border-color, #e5e7eb);
	border-radius: 0.5rem;
	overflow: hidden;
	background-color: var(--bg-secondary, #f9fafb);
}

.preview-card-link {
	display: flex;
	text-decoration: none;
	color: inherit;
	transition: opacity 0.2s ease;
}

.preview-card-link:hover {
	opacity: 0.8;
}

.preview-card-static {
	display: flex;
}

.preview-image {
	flex-shrink: 0;
	width: 200px;
	height: 150px;
	overflow: hidden;
	background-color: var(--bg-secondary, #f5f5f5);
}

.preview-image img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.preview-content {
	flex: 1;
	padding: 1rem;
	min-width: 0;
}

.preview-title {
	font-weight: 600;
	margin-bottom: 0.25rem;
}

.preview-description {
	font-size: 0.9rem;
	color: var(--text-secondary, #666);
	margin-bottom: 0.25rem;
}

.preview-url {
	font-size: 0.8rem;
	color: var(--text-secondary, #888);
}

.boosted-fallback {
	margin-bottom: 0.25rem;
}

.boosted-link {
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	color: var(--link-color, #007bff);
	text-decoration: none;
	font-size: 0.9rem;
	transition: color 0.2s ease;
}

.boosted-link:hover {
	color: var(--link-hover-color, #0056b3);
	text-decoration: underline;
}

.boosted-object-info {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.boosted-object-label {
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	font-weight: 500;
}

.boosted-object-value {
	font-family: monospace;
	font-size: 0.875rem;
	color: var(--text-primary, #333);
	word-break: break-all;
}

.post-actions {
	margin-top: 1rem;
	padding-top: 1rem;
	border-top: 1px solid var(--border-color, #4e5965);
}

.original-post-link {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	color: var(--link-color, #007bff);
	text-decoration: none;
	font-size: 0.9rem;
	transition: color 0.2s ease;
}

.original-post-link:hover {
	color: var(--link-hover-color, #0056b3);
	text-decoration: underline;
}

.post-datetime {
	position: absolute;
	bottom: 1rem;
	right: 1rem;
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	font-style: italic;
}
</style>
