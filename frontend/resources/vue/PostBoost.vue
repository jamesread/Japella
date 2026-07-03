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
			<div v-if="boostedObjectUrl" class="boosted-object-link">
				<a :href="boostedObjectUrl" target="_blank" rel="noopener noreferrer" class="boosted-link">
					<Icon icon="mdi:open-in-new" width="16" height="16" />
					<span>View Boosted Post</span>
				</a>
			</div>
			<div v-else class="boosted-object-info">
				<span class="boosted-object-label">Boosted Object:</span>
				<span class="boosted-object-value">{{ boostedObjectUrl || 'Unknown' }}</span>
			</div>
		</div>

		<div v-if="post.remoteUrl" class="post-actions">
			<a :href="post.remoteUrl" target="_blank" rel="noopener noreferrer" class="original-post-link">
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

	// Check if this is a boost (Announce type)
	const isBoost = computed(() => {
		return activityStreams.value && activityStreams.value.type === 'Announce';
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

	function filterByAuthor() {
		if (props.post.authorId) {
			emit('filter-author', props.post.authorId);
		}
	}

	function onPostRefetched(updatedPost) {
		emit('post-refetched', updatedPost);
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

	// Display remote ID (shortened if it's a URL)
	const displayRemoteId = computed(() => {
		if (!props.post.remoteId) {
			return '';
		}

		// If it's JSON, show a shortened version
		if (activityStreams.value) {
			return `${activityStreams.value.type} activity`;
		}

		// If it's a URL, show just the last part
		if (props.post.remoteId.startsWith('http')) {
			const parts = props.post.remoteId.split('/');
			return parts[parts.length - 1] || props.post.remoteId;
		}

		return props.post.remoteId;
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

.boosted-object-link {
	margin-bottom: 0.5rem;
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
