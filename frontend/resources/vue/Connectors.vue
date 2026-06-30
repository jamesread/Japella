<template>
	<Section
		title="Connectors"
		subtitle="Connectors currently started by the server."
		classes="connectors"
	>
		<template #toolbar>
			<router-link :to="{ name: 'controlPanel' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Control Panel
			</router-link>
			<button
				v-if="canAccess"
				type="button"
				class="neutral"
				title="Apply IsPubliclyAccessible setting and refresh connector list"
				:disabled="loading"
				@click="applyRefresh"
			>
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="!loaded" class="muted">Loading…</div>
		<p v-else-if="!canAccess" class="inline-notification error">
			You do not have permission to view connectors.
		</p>

		<template v-else>
			<div v-if="loading && !connectors.length" class="muted">Loading connectors…</div>

			<ConnectorCatalog
				v-else
				:connectors="connectors"
				:unregistered-connectors="unregisteredConnectors"
				started-title="Started Connectors"
				unregistered-title="Unavailable Connectors"
				unregistered-description="These connector types are available but not currently started or configured."
				empty-message="No connectors found."
			/>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import ConnectorCatalog from './ConnectorCatalog.vue';

	const connectors = ref([]);
	const unregisteredConnectors = ref([]);
	const loading = ref(true);
	const loaded = ref(false);
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const canAccess = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.connectors'))
	);

	async function applyRefresh() {
		await waitForClient();
		loading.value = true;
		try {
			await window.client.refreshConnectors({});
			await fetchConnectors();
		} catch (e) {
			console.error('Failed to refresh connectors:', e);
		} finally {
			loading.value = false;
		}
	}

	async function fetchConnectors() {
		await waitForClient();
		try {
			const res = await window.client.getConnectors({ onlyWantOauth: false });
			const list = (res.connectors || []).map((c) => {
				const protocol = c.protocol || c.name;
				return {
					name: c.name,
					protocol,
					icon: c.icon,
					hasOauth: !!c.hasOauth,
					usesYamlConfig: !!c.usesYamlConfig,
					supportsSocialAccounts: !!c.supportsSocialAccounts,
					supportsChatbot: !!c.supportsChatbot,
					issues: c.issues || [],
				};
			});
			list.sort((a, b) => a.name.localeCompare(b.name));
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
			console.error('Failed to fetch connectors:', e);
			connectors.value = [];
			unregisteredConnectors.value = [];
		} finally {
			loading.value = false;
		}
	}

	onMounted(async () => {
		await waitForClient();

		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
		loaded.value = true;

		if (canAccess.value) {
			fetchConnectors();
		}
	});
</script>

<style scoped>
	.muted {
		opacity: 0.7;
	}
</style>
