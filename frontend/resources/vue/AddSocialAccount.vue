<template>
	<Section
		title="Add Social Account"
		subtitle="Connect a new social account by choosing a connector below."
		classes="add-social-account"
	>
		<template #toolbar>
			<router-link :to="{ name: 'socialAccounts' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to Accounts</span>
			</router-link>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				:disabled="loading"
				@click="fetchConnectors"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
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
			empty-message="No social account connectors are available. Ask an administrator to configure OAuth connectors in the system settings."
		/>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, RefreshIcon } from '@hugeicons/core-free-icons';
	import { waitForClient, normalizeConnectorIssues, connectorUsesOauth } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import ConnectorCatalog from './ConnectorCatalog.vue';

	const iconStrokeWidth = 2.5;

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
				issues: normalizeConnectorIssues(c.issues),
			}));
			list.sort((a, b) => {
				if (a.hasOauth !== b.hasOauth) return a.hasOauth ? -1 : 1;
				return a.name.localeCompare(b.name);
			});
			connectors.value = list.filter((c) => c.supportsSocialAccounts);

			const unregisteredList = (res.unregisteredConnectors || [])
				.map((c) => ({
					protocol: c.protocol,
					name: c.name || c.protocol,
					icon: c.icon,
					notStartedReason: c.notStartedReason || '',
				}))
				.filter((c) => connectorUsesOauth(c.protocol));
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
