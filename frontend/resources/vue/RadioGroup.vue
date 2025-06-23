<template>
	<div class = "radio-group">
		<label v-for = "mode in availablePostModes" :key = "mode" :disabled = "mode.disabled">
			<input type = "radio" name = "'mode-' + mode" v-model = "mdlValue" value = "{{ mode.title }}" disabled = "{{ mode.disabled }}" />
			{{ mode.title }}
		</label>
	</div>
</template>

<script setup>
	import { ref, watch } from 'vue'


	defineProps({
		"availablePostModes": {
			type: Array,
			required: true
		},
		"defaultValue": {
			type: String,
			default: "canned"
		}
	})

	const mdlValue = ref('canned')
</script>
