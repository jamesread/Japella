<template>
	<section class = "oauth2-services social-accounts">
		<h2>Login to account</h2>

		<p>This page shows a list of OAuth services that can be connected to Japella.</p>

		<div v-if="services.length === 0">
			<p class="inline-notification note">No OAuth services available.</p>
		</div>
		<div v-else>
			<div v-for="service in services" :key="service.id">
				<button @click="connectService(service.name)" class = "good" type = "submit">
					<Icon :icon="service.icon" />

					Login with {{ service.name }}
				</button>
				<br />
				<br />
			</div>
		</div>

		<p>If you don't see the service you want, it probably needs initial setup in the settings.</p>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';

	const services = ref([]);

	async function fetchServices() {
		await waitForClient();

		let res = await window.client.getConnectors({
			"onlyWantOauth": true
		})

		// sort connectors by name
		res.connectors.sort((a, b) => a.name.localeCompare(b.name));

		console.log('Fetched services:', res);

		services.value = res.connectors
	}

	function connectService(serviceId) {
		window.client.startOAuth({ connectorId: serviceId })
			.then((res) => {
				console.log('Connected to service with ID:', serviceId);
				console.log('OAuth response:', res);

				window.location.href = res.url;
			})
			.catch((error) => {
				console.error('Error connecting to service:', error);
			});
	}

	onMounted(() => {
		fetchServices();
	});
</script>
