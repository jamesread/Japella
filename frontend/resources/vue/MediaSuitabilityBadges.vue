<template>
	<span v-if="shown.length" class="suitability-badges" role="list" :aria-label="summary">
		<span
			v-for="network in shown"
			:key="network.protocol"
			class="tag suitability-badge"
			:class="VERDICT_KARMA[network.verdict]"
			role="listitem"
			:title="describeNetwork(network)"
		>
			<Icon :icon="network.icon" aria-hidden="true" />
			<Icon :icon="VERDICT_ICONS[network.verdict]" aria-hidden="true" />
			<span class="visually-hidden">{{ describeNetwork(network) }}</span>
		</span>
	</span>
</template>

<script setup>
	import { computed } from 'vue';
	import { Icon } from '@iconify/vue';
	import { VERDICT_ICONS, VERDICT_KARMA, describeNetwork } from '../javascript/mediaSuitability.js';

	const props = defineProps({
		// MediaSuitability entries for one file.
		networks: {
			type: Array,
			default: () => [],
		},
		// Only show these protocols (e.g. the accounts chosen for a post); null shows all.
		protocols: {
			type: Array,
			default: null,
		},
	});

	const shown = computed(() => props.networks.filter((n) => !props.protocols || props.protocols.includes(n.protocol)));

	const summary = computed(() => `Suitability: ${shown.value.map(describeNetwork).join('. ')}`);
</script>

<style scoped>
	.suitability-badges {
		display: inline-flex;
		flex-wrap: wrap;
		gap: 0.2rem;
	}

	.suitability-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		padding: 0.1rem 0.35rem;
		font-size: 0.75rem;
	}

	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
</style>
