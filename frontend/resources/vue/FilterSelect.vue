<template>
	<div class="filter-field">
		<select v-model="value" class="filter-select">
			<slot />
		</select>
		<button
			v-if="showClear"
			type="button"
			class="filter-clear neutral small"
			:title="clearTitle"
			:aria-label="clearTitle"
			@click="clear"
		>
			<Icon icon="mdi:close" width="14" height="14" />
		</button>
	</div>
</template>

<script setup>
	import { computed } from 'vue';
	import { Icon } from '@iconify/vue';

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

	.filter-clear {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		padding: 0.35rem;
		min-width: 0;
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
