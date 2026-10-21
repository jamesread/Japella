<template>
	<div class="post-boost">
		<div class="boost-header">
			<div class="boost-info">
				<Icon icon="mdi:repeat" class="boost-icon" />
				<template v-if="post.socialAccountId">
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
					<button
						v-if="authorFilterable && post.authorId"
						type="button"
						class="author-filter-button account-name"
						:title="'Show posts by ' + displayAuthorName"
						@click="filterByAuthor"
					>
						{{ displayAuthorName }}
					</button>
					<router-link
						v-else
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
			<div v-if="originalAuthorName" class="boosted-author">
				<FeedAuthorAvatar
					:avatar-url="post.originalAuthorAvatarUrl"
					fallback-icon="mdi:account"
					:alt-text="originalAuthorName"
				/>
				<span class="boosted-author-name">{{ originalAuthorName }}</span>
			</div>

			<div
				v-if="post.content"
				class="boosted-post-message"
				v-html="post.content"
			/>

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

	<!-- Diagnostic Info Dialog -->
	<PostDiagnosticDialog
		:open="showDiagnosticDialog"
		:post="post"
		@close="closeDiagnosticDialog"
		@post-refetched="onPostRefetched"
	/>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { Icon } from '@iconify/vue';
	import { formatHumanDateTime } from '../javascript/util';
	import FeedAuthorAvatar from './FeedAuthorAvatar.vue';
	import PostDiagnosticDialog from './PostDiagnosticDialog.vue';
	import SocialAccountChip from './SocialAccountChip.vue';

	const props = defineProps({
		post: {
			type: Object,
			required: true
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

	const emit = defineEmits(['filter-author', 'post-refetched']);

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

	function filterByAuthor() {
		if (props.post.authorId) {
			emit('filter-author', props.post.authorId);
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

.boost-header {
	display: flex;
	align-items: start;
	justify-content: space-between;
	gap: 0.5rem;
	margin-bottom: 1rem;
}

.post-header-actions {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-shrink: 0;
}

.boost-info {
	display: flex;
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
