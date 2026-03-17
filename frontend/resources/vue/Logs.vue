<template>
	<Section
		title="Logs"
		subtitle="Application logs and events."
		classes="logs"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshLogs" :disabled="!clientReady || loading" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="!clientReady || loading">
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
						<router-link
							v-if="log.relatedSocialAccountId"
							:to="{ name: 'socialAccountDetails', params: { id: log.relatedSocialAccountId } }"
							class="social-account-link"
						>
							{{ log.relatedSocialAccountIdentity || `Account #${log.relatedSocialAccountId}` }}
						</router-link>
						<span v-else class="text-muted">—</span>
					</td>
				</tr>
			</tbody>
		</table>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const clientReady = ref(false);
	const loading = ref(true);
	const error = ref('');
	const logs = ref([]);

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
		await refreshLogs();
	});
</script>

<style scoped>
	.log-level {
		display: inline-block;
		padding: 0.25em 0.5em;
		border-radius: 3px;
		font-size: 0.85em;
		font-weight: 600;
		text-transform: uppercase;
	}

	.log-error .log-level {
		background-color: #fee;
		color: #c33;
	}

	.log-warn .log-level {
		background-color: #ffe;
		color: #c90;
	}

	.log-info .log-level {
		background-color: #eef;
		color: #369;
	}

	.log-debug .log-level {
		background-color: #f5f5f5;
		color: #666;
	}

	.social-account-link {
		color: #369;
		text-decoration: none;
	}

	.social-account-link:hover {
		text-decoration: underline;
	}

	.text-muted {
		color: #999;
	}

	.log-error {
		background-color: #fff5f5;
	}

	.log-warn {
		background-color: #fffbf0;
	}
</style>
