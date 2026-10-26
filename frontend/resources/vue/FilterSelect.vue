<template>
	<div class="filter-field" :class="{ compact }">
		<select v-model="value" class="filter-select">
			<slot />
		</select>
		<button
			v-if="showClear"
			type="button"
			class="inline-icon neutral small"
			:title="clearTitle"
			:aria-label="clearTitle"
			@click="clear"
		>
			<HugeiconsIcon
				:icon="Cancel01Icon"
				width="1em"
				height="1em"
				:strokeWidth="iconStrokeWidth"
				aria-hidden="true"
			/>
		</button>
	</div>
</template>

<script setup>
	import { computed } from 'vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Cancel01Icon } from '@hugeicons/core-free-icons';

	const iconStrokeWidth = 2.5;

	const props = defineProps({
		modelValue: {
			type: [String, Number],
			default: '',
		},
		defaultValue: {
			type: [String, Number],
			default: '',
		},
		clearTitle: {
			type: String,
			default: 'Clear filter',
		},
		// One short line, sized to fit a feed column's detail row.
		compact: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['update:modelValue']);

	const value = computed({
		get: () => props.modelValue,
		set: (nextValue) => emit('update:modelValue', nextValue),
	});

	const showClear = computed(() => props.modelValue !== props.defaultValue);

	function clear() {
		emit('update:modelValue', props.defaultValue);
	}
</script>

<style scoped>
	.filter-field {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	.filter-select {
		padding: 0.5rem;
		border: 1px solid #ddd;
		border-radius: 0.25rem;
		font-size: 0.9rem;
		min-width: 150px;
		background: white;
	}

	.filter-select:focus {
		outline: none;
		border-color: #4CAF50;
		box-shadow: 0 0 0 2px rgba(76, 175, 80, 0.2);
	}

	.filter-field.compact {
		flex: 1 1 0;
		min-width: 0;
		gap: 0.15rem;
	}

	.filter-field.compact .filter-select {
		flex: 1 1 auto;
		min-width: 0;
		width: 100%;
		height: var(--feed-column-meta-height, 1.5rem);
		padding: 0 0.25rem;
		font-size: inherit;
		line-height: 1;
	}

	.filter-field.compact button {
		height: var(--feed-column-meta-height, 1.5rem);
		padding: 0 0.25rem;
		flex-shrink: 0;
	}

	@media (max-width: 768px) {
		.filter-field {
			width: 100%;
		}

		.filter-select {
			min-width: 0;
			flex: 1;
			width: 100%;
		}
	}
</style>
