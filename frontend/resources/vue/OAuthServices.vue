<template>
	<section class = "oauth2-services social-accounts">
		<h2>Login to account</h2>

		<p>This is a list of OAuth services that can be connected to Japella.</p>

	</section>

		<div v-if="services.length === 0">
			<p class="inline-notification note">No OAuth services available.</p>
		</div>
		<div v-else class = "service-list">
			<div v-for="service in services" :key="service.id"  class = "oauth-service">
				<h3>
					<Icon :icon="service.icon" />
					<br />
					{{ service.name }}
				</h3>

				<div v-if = "service.issues.length > 0">
					<button @click="showIssues(service)" class = "bad">Show issues</button>
				</div>
				<div v-else>
					<button @click="connectService(service.name)" :class = "service.issues.length == 0 ? 'good' : 'bad'" type = "submit">

						Login
					</button>
				</div>
			</div>
		</div>

	<dialog ref = "serviceIssuesDialog" v-if="selectedService" class = "dialog">
		<h3>Service Issues for {{ selectedService.name }}</h3>
		<a :href = "selectedService.name" target="_blank">Documentation</a>
		<p>There are issues with this service, please check the details below.</p>
		<div>
			<ul>
				<li class = "inline-notification bad" v-for="issue in selectedService.issues" :key="issue">
					{{ issue }}
				</li>
			</ul>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';

	const services = ref([]);
	const serviceIssuesDialog = ref(null);
	const selectedService = ref(null);

	function showIssues(service) {
		selectedService.value = service;
		serviceIssuesDialog.value.showModal();
	}

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
    button {
		margin-top: 1em;
	}

	.service-list {
		display: flex;
		flex-direction: row;
		gap: 1em;
	}

	.oauth-service {
		border: 1px solid #555;
		border-radius: .5em;
		padding: 1em;
		text-align: center;
		width: 160px;
	}

	h3 {
		margin: 0;
	}

	li {
		margin-bottom: 1em;
	}
</style>
