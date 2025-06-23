<template>
	<div class = "stat-display">
		<span class = "subtle">{{ title }}</span>
		<span class = "stat">{{ value }}</span>
	</div>
</template>

<script setup>
	import { ref } from 'vue'

	const props = defineProps({
		title: String,
		value: [String, Number]
	})

	const value = ref('???')

	function setValue(v) {
		value.value = v
	}

	defineExpose({
		setValue
	})
</script>
