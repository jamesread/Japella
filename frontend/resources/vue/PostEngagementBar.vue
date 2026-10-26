<template>
	<div
		v-if="engagement && engagement.hasData"
		class="post-engagement"
		:title="updatedTitle"
		data-post-engagement
	>
		<span
			v-for="metric in visibleMetrics"
			:key="metric.key"
			class="engagement-metric"
			:class="{ zero: metric.value === 0 }"
			:aria-label="`${metric.value} ${metric.label}`"
			:title="`${metric.value.toLocaleString()} ${metric.label}`"
		>
			<Icon :icon="metric.icon" width="16" height="16" aria-hidden="true" />
			<span class="engagement-value">{{ formatCount(metric.value) }}</span>
		</span>
	</div>
</template>

<script setup>
	import { computed } from 'vue';
	import { Icon } from '@iconify/vue';
	import { formatHumanDateTime } from '../javascript/util';

	const props = defineProps({
		engagement: {
			type: Object,
			default: null,
		},
	});

	const compactFormatter = new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 });

	function formatCount(n) {
		return n < 1000 ? String(n) : compactFormatter.format(n);
	}

	const visibleMetrics = computed(() => {
		const e = props.engagement || {};
		const metrics = [
			{ key: 'likes', label: 'likes', icon: 'mdi:heart-outline', value: e.likes || 0 },
			{ key: 'reposts', label: 'reposts', icon: 'mdi:repeat-variant', value: e.reposts || 0 },
			{ key: 'replies', label: 'replies', icon: 'mdi:comment-outline', value: e.replies || 0 },
			{ key: 'quotes', label: 'quotes', icon: 'mdi:format-quote-close-outline', value: e.quotes || 0 },
		];
		// Only X reports impressions; hide the metric rather than show a misleading 0.
		if (e.impressions) {
			metrics.push({ key: 'impressions', label: 'impressions', icon: 'mdi:eye-outline', value: e.impressions });
		}
		return metrics;
	});

	const updatedTitle = computed(() => {
		const at = props.engagement?.lastFetchedAt;
		return at ? `Engagement updated ${formatHumanDateTime(at)}` : 'Engagement';
	});
</script>

<style scoped>
	.post-engagement {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 1.25rem;
		margin-top: 0.75rem;
		font-size: 0.9rem;
		color: var(--text-secondary, #666);
	}

	.engagement-metric {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-variant-numeric: tabular-nums;
	}

	.engagement-metric.zero {
		opacity: 0.55;
	}

	.engagement-value {
		font-weight: 600;
	}
</style>
