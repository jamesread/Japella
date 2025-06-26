<template>
	<section class = "oauth2-services social-accounts">
		<h2>Login to account</h2>

		<p>This page shows a list of OAuth services that can be connected to Japella.</p>

		<div v-if="services.length === 0">
			<p class="inline-notification note">No OAuth services available.</p>
		</div>
		<div v-else class = "service-list">
			<div v-for="service in services" :key="service.id"  class = "oauth-service">
				<h3>{{ service.name }}</h3>
				<div v-if = "service.issues.length > 0">
					<p class = "inline-notification error">There are issues with this service, please check the details below.</p>
					<details>
						<p class = "inline-notification note" v-for="issue in service.issues">
							{{ issue }}
						</p>
						<summary>
							{{ service.issues.length > 0 ? 'Open issues' : 'No issues' }}
						</summary>
					</details>

				</div>
				<div v-else>
					<p><a href = "#">Docs</a></p>
					<button @click="connectService(service.name)" :class = "service.issues.length == 0 ? 'good' : 'bad'" type = "submit">
						<Icon :icon="service.icon" />

						Login
					</button>
				</div>
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

<style scoped>
	.service-list {
		display: flex;
		flex-direction: row;
		gap: 1em;
	}

	.oauth-service {
		border: 1px solid #555;
		border-radius: 1em;
		padding: 1em;
		text-align: center;
	}

	h3 {
		margin: 0;
	}
</style>
