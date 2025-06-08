<template>
	<div class = "radio-group">
		<span v-for = "mode in availablePostModes" :key = "mode">
			<label for = "'mode-' + mode">
				<input type = "radio" id = "'mode-' + mode" name = "'mode-' + mode" v-model = "mdlValue" value = "{{ mode }}" />
				{{ mode }}
			</label>
		</span>
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
