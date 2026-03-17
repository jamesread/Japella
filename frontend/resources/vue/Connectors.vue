<template>
	<section class = "connectors">
		<h2>Connectors</h2>
		<p>These connectors are currently started by the server.</p>

		<div v-if = "loading" class = "icon-and-text" style = "margin-top: 1em;">
			<Icon icon = "eos-icons:loading" width = "24" height = "24" />
			<span style = "margin-left: .5em;">Loading connectors...</span>
		</div>

		<div v-else>
			<!-- Started Connectors -->
			<div v-if = "connectors.length > 0">
				<h3 style = "margin-top: 1.5em; margin-bottom: 0.5em;">Started Connectors</h3>
				<div class = "connector-list">
					<div v-for = "c in connectors" :key = "c.name" class = "connector-card">
						<div class = "header">
							<Icon :icon = "c.icon || 'mdi:help-circle-outline'" width = "20" height = "20" />
							<h3>
								<span v-if = "c.customName">{{ c.protocol }} ({{ c.customName }})</span>
								<span v-else>{{ c.name }}</span>
							</h3>
						</div>
						<div class = "meta">
							<div class = "tag fg-good" v-if = "c.hasOauth">
								OAuth
							</div>
							<div class = "tag fg-good" v-else-if = "c.usesYamlConfig">
								YAML Config
							</div>
							<div class = "tag fg-good" v-if = "c.supportsSocialAccounts">
								Social Account
							</div>
							<div class = "tag fg-good" v-if = "c.supportsChatbot">
								Chat Bot
							</div>
						</div>
						<div v-if = "c.issues && c.issues.length > 0" class = "issues">
							<strong>Issues:</strong>
							<ul>
								<li v-for = "issue in c.issues" :key = "issue" class = "inline-notification bad">{{ issue }}</li>
							</ul>
						</div>
					</div>
				</div>
			</div>

			<!-- Unregistered Connectors -->
			<div v-if = "unregisteredConnectors.length > 0">
				<h3 style = "margin-top: 1.5em; margin-bottom: 0.5em;">Unregistered Connectors</h3>
				<p class = "inline-notification note" style = "margin-bottom: 0.5em;">These connector types are available but not currently started or configured.</p>
				<div class = "connector-list">
					<div v-for = "c in unregisteredConnectors" :key = "c.protocol" class = "connector-card unregistered">
						<div class = "header">
							<Icon :icon = "c.icon || 'mdi:help-circle-outline'" width = "20" height = "20" />
							<h3>{{ c.name }}</h3>
						</div>
						<div class = "meta">
							<div class = "tag fg-neutral">Not Started</div>
						</div>
					</div>
				</div>
			</div>

			<p v-if = "connectors.length === 0 && unregisteredConnectors.length === 0" class = "inline-notification note">No connectors found.</p>
		</div>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';

	const connectors = ref([]);
	const unregisteredConnectors = ref([]);
	const loading = ref(true);

	async function fetchConnectors() {
		await waitForClient();
		try {
			const res = await window.client.getConnectors({ onlyWantOauth: false });
			// Normalize and sort connectors by name
			const list = (res.connectors || []).map(c => {
				const protocol = c.protocol || c.name;
				const displayName = c.name;
				const hasCustomName = displayName !== protocol;

				return {
					name: displayName,
					protocol: protocol,
					customName: hasCustomName ? displayName : null,
					icon: c.icon,
					hasOauth: !!c.hasOauth,
					usesYamlConfig: !!c.usesYamlConfig,
					isRegistered: c.isRegistered !== false, // default true if not provided
					supportsSocialAccounts: !!c.supportsSocialAccounts,
					supportsChatbot: !!c.supportsChatbot,
					issues: c.issues || []
				};
			});
			list.sort((a, b) => a.name.localeCompare(b.name));
			connectors.value = list;

			// Normalize and sort unregistered connectors
			const unregisteredList = (res.unregisteredConnectors || []).map(c => ({
				protocol: c.protocol,
				name: c.name || c.protocol,
				icon: c.icon
			}));
			unregisteredList.sort((a, b) => a.name.localeCompare(b.name));
			unregisteredConnectors.value = unregisteredList;
		} catch (e) {
			console.error('Failed to fetch connectors:', e);
			connectors.value = [];
			unregisteredConnectors.value = [];
		} finally {
			loading.value = false;
		}
	}

	onMounted(() => {
		fetchConnectors();
	});
</script>

<style scoped>
	.connector-list {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 1em;
		margin-top: 1em;
	}

	.connector-card {
		border: 1px solid #555;
		border-radius: .5em;
		padding: .75em;
	}

	.header {
		display: flex;
		align-items: center;
		gap: .5em;
	}

	h3 {
		margin: 0;
	}

	.meta {
		display: flex;
		gap: .5em;
		margin-top: .5em;
	}

	.issues {
		margin-top: .75em;
	}

	.connector-card.unregistered {
		opacity: 0.7;
	}

</style>
