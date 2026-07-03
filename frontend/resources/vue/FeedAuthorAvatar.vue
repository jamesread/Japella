<template>
	<span class="feed-author-avatar" aria-hidden="true">
		<img
			v-if="avatarUrl && imageVisible"
			:src="avatarUrl"
			:alt="altText"
			class="feed-author-avatar-img"
			loading="lazy"
			referrerpolicy="no-referrer"
			@error="onImageError"
		/>
		<Icon v-else :icon="fallbackIcon" class="feed-author-avatar-fallback" />
	</span>
</template>

<script setup>
	import { ref, watch } from 'vue';
	import { Icon } from '@iconify/vue';

	const props = defineProps({
		avatarUrl: {
			type: String,
			default: '',
		},
		fallbackIcon: {
			type: String,
			default: 'mdi:account',
		},
		altText: {
			type: String,
			default: 'Author avatar',
		},
	});

	const imageVisible = ref(Boolean(props.avatarUrl));

	watch(() => props.avatarUrl, (url) => {
		imageVisible.value = Boolean(url);
	});

	function onImageError() {
		imageVisible.value = false;
	}
</script>

<style scoped>
	.feed-author-avatar {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 1.75rem;
		height: 1.75rem;
		border-radius: 50%;
		overflow: hidden;
		background: var(--bg-secondary, #eef1f4);
	}

	.feed-author-avatar-img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}

	.feed-author-avatar-fallback {
		width: 1.1rem;
		height: 1.1rem;
		opacity: 0.85;
	}
</style>
