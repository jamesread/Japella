<template>
	<div class="post-preview">
		<div class="post-header">
			<div class="social-account-info">
				<router-link 
					v-if="post.socialAccountId" 
					:to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }" 
					class="social-account-link"
				>
					<Icon :icon="post.socialAccountIcon || 'mdi:account'" />
					<span class="account-name">{{ post.authorName || post.socialAccountIdentity }}</span>
				</router-link>
				<div v-else class="social-account-link canned-post-label">
					<Icon icon="jam:box" />
					<span class="account-name">Canned Post</span>
				</div>
			</div>
		</div>


		<div class="post-content-section">
			<div class="post-message" v-html="post.content"></div>
		</div>

		<div v-if="post.remoteUrl" class="post-actions">
			<a :href="post.remoteUrl" target="_blank" rel="noopener noreferrer" class="original-post-link">
				<Icon icon="mdi:open-in-new" width="16" height="16" />
				<span>View Original Post</span>
			</a>
		</div>

	</div>
</template>

<script setup>
	import { Icon } from '@iconify/vue';

	// Props
	const props = defineProps({
		post: {
			type: Object,
			required: true
		}
	});

	function statusClass(post) {
		if (post.state === 'error') return 'bad';
		if (post.state === 'pending' || post.state === 'scheduled') return 'note';
		if (post.state === 'completed') return 'good';
		return '';
	}

	function statusText(post) {
		if (post.state === 'error') return 'Error';
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
    line-height: 2;;
}

.social-account-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 1.1rem;
	font-weight: 500;
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

</style>
