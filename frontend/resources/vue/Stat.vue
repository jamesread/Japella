<template>
	<div class = "stat-display">
		<span class = "subtle">{{ title }}</span>
		<span class = "stat">{{ value }}</span>
	</div>
</template>

<script setup>
	import { ref } from 'vue'

	defineProps({
		title: String,
		value: [String, Number]
	})

	const valueRef = ref('???')

	function setValue(v) {
		valueRef.value = v
	}

	defineExpose({
		setValue
	})
</script>
