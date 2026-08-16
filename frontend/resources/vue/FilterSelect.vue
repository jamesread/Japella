<template>
	<div class="filter-field">
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
