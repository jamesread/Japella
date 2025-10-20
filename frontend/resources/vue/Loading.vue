<template>
	<div class="loading-container" :class="{ 'inline': inline, 'centered': centered }">
		<div class="loading-spinner">
			<Icon icon="eos-icons:loading" />
		</div>
		<div v-if="message" class="loading-message">
			{{ message }}
		</div>
	</div>
</template>

<script setup>
	import { Icon } from '@iconify/vue';

	// Props
	const props = defineProps({
		message: {
			type: String,
			default: 'Loading...'
		},
		inline: {
			type: Boolean,
			default: false
		},
		centered: {
			type: Boolean,
			default: true
		}
	});
</script>

<style scoped>
.loading-container {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 1rem;
	padding: 2rem;
}

.loading-container.centered {
	min-height: 200px;
}

.loading-container.inline {
	flex-direction: row;
	padding: 1rem;
	gap: 0.5rem;
}

.loading-spinner {
	display: flex;
	align-items: center;
	justify-content: center;
}

.loading-spinner svg {
	width: 24px;
	height: 24px;
	animation: spin 1s linear infinite;
}

.loading-message {
	color: var(--text-secondary, #666);
	font-size: 0.9rem;
	text-align: center;
}

@keyframes spin {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(360deg);
	}
}

/* Dark theme support */
@media (prefers-color-scheme: dark) {
	.loading-message {
		color: var(--text-secondary, #ccc);
	}
}
</style>
