<template>
	<div :class = "'inline-notification ' + type">
		<p v-if="message">{{ message }}</p>
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
