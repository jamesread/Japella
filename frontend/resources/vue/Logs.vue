<template>
	<Section
		title="Logs"
		subtitle="Application logs and events."
		classes="logs"
		:padding="false"
	>
		<template #toolbar>
			<button v-if="canAccess" @click="refreshLogs" :disabled="!clientReady || loading" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="!clientReady || !loaded">
			<p>Loading...</p>
		</div>
		<div v-else-if="!canAccess">
			<p class="inline-notification error">You do not have permission to view logs.</p>
		</div>
		<div v-else-if="loading">
			<p>Loading logs...</p>
		</div>
		<div v-else-if="error">
			<p class="inline-notification error">{{ error }}</p>
		</div>
		<div v-else-if="logs.length === 0">
			<p class="inline-notification note">No logs found.</p>
		</div>
		<table v-else class="data-table">
			<thead>
				<tr>
					<th>Time</th>
					<th>Level</th>
					<th>Message</th>
					<th>Related Account</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="log in logs" :key="log.id" :class="getLogLevelClass(log.level)">
					<td>{{ formatDate(log.createdAt) }}</td>
					<td>
						<span class="log-level">{{ log.level }}</span>
					</td>
					<td>{{ log.message }}</td>
					<td>
						<SocialAccountChip
							v-if="log.relatedSocialAccountId"
							:social-account-id="log.relatedSocialAccountId"
							:identity="log.relatedSocialAccountIdentity || `Account #${log.relatedSocialAccountId}`"
							:icon="log.relatedSocialAccountIcon"
						/>
						<span v-else class="text-muted">—</span>
					</td>
				</tr>
			</tbody>
		</table>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import SocialAccountChip from './SocialAccountChip.vue';

	const clientReady = ref(false);
	const loading = ref(true);
	const error = ref('');
	const logs = ref([]);
	const loaded = ref(false);
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const canAccess = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.logs'))
	);

	function formatDate(dateString) {
		if (!dateString) {
			return '—';
		}

		try {
			const date = new Date(dateString);
			if (isNaN(date.getTime())) {
				return dateString;
			}
			return date.toLocaleString();
		} catch (e) {
			return dateString;
		}
	}

	function getLogLevelClass(level) {
		const levelLower = (level || '').toLowerCase();
		if (levelLower.includes('error')) {
			return 'log-error';
		} else if (levelLower.includes('warn')) {
			return 'log-warn';
		} else if (levelLower.includes('info')) {
			return 'log-info';
		} else if (levelLower.includes('debug')) {
			return 'log-debug';
		}
		return '';
	}

	async function refreshLogs() {
		if (!window.client) {
			return;
		}

		loading.value = true;
		error.value = '';

		try {
			const response = await window.client.getLogs({ limit: 100 });
			logs.value = response.logs || [];
		} catch (e) {
			error.value = `Failed to load logs: ${e.message || e}`;
			console.error('Error fetching logs:', e);
			logs.value = [];
		} finally {
			loading.value = false;
		}
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;

		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
		loaded.value = true;

		if (canAccess.value) {
			await refreshLogs();
		}
	});
</script>
