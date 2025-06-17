<template>
	<div :class = "'inline-notification ' + type">
		<span v-if="message">{{ message }}</span>
	</div>
</template>

<script setup>

	defineProps({
		type: {
			type: String,
			default: 'info'
		},
		message: {
			type: String,
			default: ''
		}
	});
</script>
