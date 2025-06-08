<template>
	<section class = "status">
		<h2>Status</h2>
		<p>This shows the current status of the application.</p>

		<div class = "grid-boxed">
			<Stat ref = "health" title = "Health"></stat>
			<Stat ref = "nanoservices" title = "Nanoservices"></stat>
		</div>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue'
	import Stat from './stat.vue'

	const health = ref(null)
	const nanoservices = ref(null)
	const postingServices = ref(null)

	onMounted(() => {
		window.addEventListener('status-updated', (st) => {
			console.log("status-udpated", st.detail)

			health.value.setValue(st.detail.status)
			nanoservices.value.setValue(st.detail.nanoservices.join(', '))
		})
	})
</script>
