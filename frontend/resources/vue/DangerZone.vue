<template>
	<details class="danger-zone">
		<summary class="danger-zone-summary">
			<Icon class="danger-zone-chevron" icon="mdi:chevron-right" aria-hidden="true" />
			<span class="danger-zone-title">{{ title }}</span>
		</summary>
		<div class="danger-zone-content">
			<p v-if="description" class="danger-zone-description">{{ description }}</p>
			<slot />
		</div>
	</details>
</template>

<script setup>
	import { Icon } from '@iconify/vue';

	defineProps({
		title: {
			type: String,
			default: 'Danger zone',
		},
		description: {
			type: String,
			default: '',
		},
	});
</script>

<style scoped>
	.danger-zone {
		--danger-zone-border: #f5c6cb;
		--danger-zone-bg: rgba(248, 215, 218, 0.15);
		--danger-zone-fg: #721c24;
		--danger-zone-description: var(--text-secondary, #666);
		margin-top: 1.5rem;
		border: 1px solid var(--danger-zone-border);
		border-radius: 0.5rem;
		background: var(--danger-zone-bg);
		overflow: hidden;
	}

	.danger-zone-summary {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.75rem 1rem;
		cursor: pointer;
		font-weight: 600;
		color: var(--danger-zone-fg);
		list-style: none;
		user-select: none;
	}

	.danger-zone-summary::-webkit-details-marker {
		display: none;
	}

	.danger-zone-chevron {
		flex-shrink: 0;
		transition: transform 0.15s ease;
	}

	.danger-zone[open] .danger-zone-chevron {
		transform: rotate(90deg);
	}

	.danger-zone-content {
		padding: 0 1rem 1rem;
		border-top: 1px solid var(--danger-zone-border);
	}

	.danger-zone-description {
		margin: 0.75rem 0 1rem;
		color: var(--danger-zone-description);
		font-size: 0.9rem;
		line-height: 1.5;
	}

	html[data-theme="dark"] .danger-zone {
		--danger-zone-border: var(--karma-bad-border, #8b1a1a);
		--danger-zone-bg: rgba(139, 26, 26, 0.22);
		--danger-zone-fg: #ffb4b4;
		--danger-zone-description: #bbb;
	}
</style>
