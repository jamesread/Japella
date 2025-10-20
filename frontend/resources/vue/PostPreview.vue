<template>
	<div class="post-preview">
		<div class="post-header">
			<div class="social-account-info">
				<Icon :icon="post.socialAccountIcon" />
				<span class="account-name">{{ post.socialAccountIdentity }}</span>
			</div>
		</div>


		<div class="post-content-section">
			<div class="post-message">
				{{ post.content }}
			</div>
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
	border: 2px solid var(--border-color, #4e5965);
    background-color: var(--background-secondary, #242424);
    line-height: 2;;
}

.social-account-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 1.1rem;
	font-weight: 500;
}

.account-name {
    font-weight: bold;
}

.post-message {
	font-weight: 500;
}

.post-content-section {
	margin-bottom: 2rem;
}

.post-content-section h3 {
	margin: 0 0 1rem 0;
	color: var(--text-primary, #333);
	font-size: 1.2rem;
}

</style>
