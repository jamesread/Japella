<template>
	<Section
		title="Add Social Account"
		subtitle="Connect a new social account by choosing a connector below."
		classes="add-social-account"
	>
		<template #toolbar>
			<router-link :to="{ name: 'socialAccounts' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to Accounts
			</router-link>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="fetchConnectors">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-if="loading && !connectors.length" class="muted">Loading connectors…</div>

		<ConnectorCatalog
			v-else
			:connectors="connectors"
			:unregistered-connectors="unregisteredConnectors"
			selection-mode="preAdd"
			empty-message="No connectors are available. Ask an administrator to configure connectors in the system settings."
		/>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import ConnectorCatalog from './ConnectorCatalog.vue';

	const connectors = ref([]);
	const unregisteredConnectors = ref([]);
	const loading = ref(true);
	const errorMessage = ref('');

	async function fetchConnectors() {
		loading.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.getConnectors({ onlyWantOauth: false });

			const list = (res.connectors || []).map((c) => ({
				name: c.name,
				protocol: c.protocol || c.name,
				icon: c.icon,
				hasOauth: !!c.hasOauth,
				usesYamlConfig: !!c.usesYamlConfig,
				supportsSocialAccounts: !!c.supportsSocialAccounts,
				supportsChatbot: !!c.supportsChatbot,
				issues: c.issues || [],
			}));
			list.sort((a, b) => {
				if (a.hasOauth !== b.hasOauth) return a.hasOauth ? -1 : 1;
				return a.name.localeCompare(b.name);
			});
			connectors.value = list;

			const unregisteredList = (res.unregisteredConnectors || []).map((c) => ({
				protocol: c.protocol,
				name: c.name || c.protocol,
				icon: c.icon,
				notStartedReason: c.notStartedReason || '',
			}));
			unregisteredList.sort((a, b) => a.name.localeCompare(b.name));
			unregisteredConnectors.value = unregisteredList;
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load connectors.';
		} finally {
			loading.value = false;
		}
	}

	onMounted(() => {
		fetchConnectors();
	});
</script>

<style scoped>
	.muted {
		opacity: 0.7;
	}
</style>
