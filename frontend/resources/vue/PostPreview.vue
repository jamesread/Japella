<template>
	<div class="post-preview">
		<div class="post-header">
			<div class="social-account-info">
				<template v-if="post.socialAccountId">
					<router-link
						:to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }"
						class="social-account-link avatar-link"
						:title="'View social account ' + (post.socialAccountIdentity || '')"
					>
						<FeedAuthorAvatar
							:avatar-url="post.authorAvatarUrl"
							:fallback-icon="post.socialAccountIcon || 'mdi:account'"
							:alt-text="post.authorName || post.socialAccountIdentity || 'Author'"
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
				<div v-else class="social-account-link canned-post-label">
					<Icon icon="jam:box" />
					<span class="account-name">Canned Post</span>
				</div>
			</div>
			<div class="post-header-actions">
				<SocialAccountChip
					v-if="post.socialAccountId && !timelineMode"
					:social-account-id="post.socialAccountId"
					:identity="post.socialAccountIdentity"
					:icon="post.socialAccountIcon"
					:transparent="transparentSocialAccountChip"
				/>
				<button
					v-if="post.id || post.remoteId"
					type="button"
					@click="openDiagnosticDialog"
					:class="['diagnostic-button', 'small', diagnosticButtonClass]"
					:title="diagnosticButtonTitle"
				>
					<Icon icon="mdi:information-outline" />
				</button>
			</div>
		</div>

		<div class="post-content-section">
			<div class="post-message" v-html="post.content"></div>
		</div>

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

		<div
			v-if="showPostFooter"
			class="post-footer"
			:class="{ 'timeline-footer': timelineMode }"
		>
			<div class="post-footer-start">
				<a
					v-if="externalPostUrl"
					:href="externalPostUrl"
					target="_blank"
					rel="noopener noreferrer"
					class="original-post-link"
				>
					<Icon icon="mdi:open-in-new" width="16" height="16" />
					<span>{{ viewPostOnLabel }}</span>
				</a>
			</div>
			<div v-if="timelineMode" class="timeline-campaign timeline-campaign-footer">
				<span class="timeline-meta-label">Campaign</span>
				<router-link
					v-if="post.campaignId"
					:to="{ name: 'campaignDetails', params: { id: post.campaignId } }"
				>
					{{ post.campaignName }}
				</router-link>
				<div v-else class="no-campaign">
					<button type="button" class="no-campaign-text" title="Assign Campaign" @click="emit('assign-campaign')">
						None
					</button>
					<button type="button" class="neutral small" title="Assign Campaign" @click="emit('assign-campaign')">
						<Icon icon="mdi:folder-edit" width="14" height="14" />
					</button>
				</div>
			</div>
			<div v-if="post.postedDate || post.created" class="post-datetime">
				{{ displayDate }}
			</div>
		</div>

	</div>

	<!-- Diagnostic Info Dialog -->
	<PostDiagnosticDialog
		:open="showDiagnosticDialog"
		:post="post"
		:refetchable="!timelineMode"
		:show-view-details="timelineMode"
		:retryable="timelineMode"
		:retrying="retrying"
		@close="closeDiagnosticDialog"
		@post-refetched="onPostRefetched"
		@view-details="emit('view-details')"
		@retry-post="emit('retry-post')"
	/>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { Icon } from '@iconify/vue';
	import { socialProviderNameFromPost, formatHumanDateTime } from '../javascript/util';
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
		timelineMode: {
			type: Boolean,
			default: false,
		},
		retrying: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits([
		'filter-author',
		'post-refetched',
		'assign-campaign',
		'view-details',
		'retry-post',
	]);

	const displayAuthorName = computed(() => {
		return props.post.authorName || props.post.socialAccountIdentity;
	});

	const externalPostUrl = computed(() => props.post.remoteUrl || props.post.postUrl || '');

	const viewPostOnLabel = computed(() => {
		return `View on ${socialProviderNameFromPost(props.post)}`;
	});

	const showPostFooter = computed(() => {
		return Boolean(
			externalPostUrl.value ||
			props.timelineMode ||
			props.post.postedDate ||
			props.post.created
		);
	});

	const displayDate = computed(() => {
		return formatHumanDateTime(props.post.postedDate || props.post.created);
	});

	const diagnosticButtonClass = computed(() => {
		if (!props.post.state) {
			return 'neutral';
		}
		return statusClass(props.post) || 'neutral';
	});

	const diagnosticButtonTitle = computed(() => {
		if (!props.post.state) {
			return 'Diagnostic Info';
		}
		return `Diagnostic Info — ${statusText(props.post)}`;
	});

	function filterByAuthor() {
		if (props.post.authorId) {
			emit('filter-author', props.post.authorId);
		}
	}

	function onPostRefetched(updatedPost) {
		emit('post-refetched', updatedPost);
	}

	// Diagnostic dialog state
	const showDiagnosticDialog = ref(false);

	function openDiagnosticDialog() {
		showDiagnosticDialog.value = true;
	}

	function closeDiagnosticDialog() {
		showDiagnosticDialog.value = false;
	}

	function getDomainFromUrl(url) {
		if (!url) return '';
		try {
			const urlObj = new URL(url);
			return urlObj.hostname.replace('www.', '');
		} catch (e) {
			return url;
		}
	}

	function statusClass(post) {
		const text = statusText(post);
		if (text === 'Error' || text === 'Rejected') return 'bad';
		if (text === 'Completed') return 'good';
		if (text === 'Unknown') return 'warn';
		if (text === 'Scheduled' || text === 'Pending approval' || text === 'Draft') return 'note';
		return '';
	}

	function statusText(post) {
		if (post.state === 'error') return 'Error';
		if (post.state === 'pending_approval') return 'Pending approval';
		if (post.state === 'rejected') return 'Rejected';
		if (post.state === 'draft') return 'Draft';
		if (post.state === 'pending' || post.state === 'scheduled') return 'Scheduled';
		if (post.state === 'completed') return 'Completed';
		return 'Unknown';
	}
</script>

<style scoped>
.post-preview {
	max-width: 800px;
	margin: 0 auto;
	padding: 1rem;
	border-radius: 0.5rem;
    line-height: 2;
}

.post-header {
	display: flex;
	align-items: start;
	justify-content: space-between;
	gap: 0.5rem;
	margin-bottom: 0.5rem;
}

.post-header-actions {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-shrink: 0;
}

.social-account-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 1.1rem;
	font-weight: 500;
}

.diagnostic-button {
	display: flex;
	align-items: center;
	justify-content: center;
	border: 1px solid transparent;
}

.diagnostic-button.neutral {
	background-color: #f0f0f0;
	color: #333;
	border-color: #e0e0e0;
}

.diagnostic-button.warn {
	background-color: #fff3cd;
	color: #856404;
	border-color: #ffeaa7;
}

.diagnostic-button.bad {
	background-color: #f8d7da;
	color: #721c24;
	border-color: #f5c6cb;
}

.diagnostic-button.good {
	background-color: #d4edda;
	color: #155724;
	border-color: #c3e6cb;
}

.diagnostic-button.note {
	background-color: #fff3cd;
	color: #856404;
	border-color: #ffeaa7;
}

.timeline-campaign {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 0.9rem;
}

.timeline-campaign-footer {
	flex-wrap: wrap;
	justify-content: center;
	text-align: center;
}

.timeline-meta-label {
	color: var(--text-secondary, #666);
	font-weight: 500;
}

.no-campaign {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.no-campaign-text {
	padding: 0;
	border: none;
	background: none;
	font: inherit;
	cursor: pointer;
	color: var(--link-color, #007bff);
}

.no-campaign-text:hover {
	color: var(--link-hover-color, #0056b3);
	text-decoration: underline;
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

.canned-post-label {
	cursor: default;
}

.canned-post-label:hover {
	text-decoration: none;
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

.post-message {
	font-weight: 500;
	word-wrap: break-word;
}

.post-message :deep(a) {
	color: var(--link-color, #007bff);
	text-decoration: underline;
}

.post-message :deep(a:hover) {
	color: var(--link-hover-color, #0056b3);
}

.post-message :deep(p) {
	margin: 0.5em 0;
}

.post-message :deep(p:first-child) {
	margin-top: 0;
}

.post-message :deep(p:last-child) {
	margin-bottom: 0;
}

.post-content-section {
	margin-bottom: 2rem;
}

.post-content-section h3 {
	margin: 0 0 1rem 0;
	color: var(--text-primary, #333);
	font-size: 1.2rem;
}

.post-footer {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin-top: 1rem;
	padding-top: 1rem;
	border-top: 1px solid var(--border-color, #4e5965);
}

.post-footer.timeline-footer {
	display: grid;
	grid-template-columns: 1fr auto 1fr;
	align-items: center;
}

.post-footer-start {
	min-width: 0;
}

.post-footer.timeline-footer .post-footer-start {
	justify-self: start;
}

.post-footer.timeline-footer .timeline-campaign-footer {
	justify-self: center;
}

.post-footer.timeline-footer .post-datetime {
	justify-self: end;
	margin-left: 0;
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
	margin-left: auto;
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	font-style: italic;
	white-space: nowrap;
}

.preview-card {
	margin: 1rem 0;
	border: 1px solid var(--border-color, #e5e7eb);
	border-radius: 0.5rem;
	overflow: hidden;
	background-color: var(--bg-primary, #ffffff);
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
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.preview-title {
	font-weight: 600;
	font-size: 1rem;
	color: var(--text-primary, #333);
	line-height: 1.4;
}

.preview-description {
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	line-height: 1.5;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

.preview-url {
	font-size: 0.75rem;
	color: var(--text-secondary, #999);
	text-transform: uppercase;
	letter-spacing: 0.5px;
	margin-top: auto;
}

@media (max-width: 600px) {
	.preview-image {
		width: 120px;
		height: 90px;
	}

	.preview-content {
		padding: 0.75rem;
	}
}

</style>
