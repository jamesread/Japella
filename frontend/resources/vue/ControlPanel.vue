<template>
	<Section
		title="Control Panel"
		subtitle="System administration and monitoring dashboard for Japella"
		classes="control-panel"
	>
		<template #toolbar>
			<button @click="refreshAll" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage" class="error-section">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<!-- System Status Overview -->
		<div class="status-overview">
			<h3>System Status</h3>
			<div class="status-grid">
				<div class="status-card">
					<div class="status-header">
						<Icon icon="material-symbols:check-circle" />
						<span>Server Status</span>
					</div>
					<div class="status-value" :class="systemStatus.status === 'ok' ? 'good' : 'bad'">
						{{ systemStatus.status || 'Unknown' }}
					</div>
				</div>

				<div class="status-card">
					<div class="status-header">
						<Icon icon="material-symbols:apps" />
						<span>Nanoservices</span>
					</div>
					<div class="status-value">
						{{ systemStatus.nanoservices?.length || 0 }} active
					</div>
				</div>

				<div class="status-card">
					<div class="status-header">
						<Icon icon="material-symbols:info" />
						<span>Version</span>
					</div>
					<div class="status-value">
						{{ systemStatus.version || 'Unknown' }}
					</div>
				</div>

				<div class="status-card">
					<div class="status-header">
						<Icon icon="material-symbols:person" />
						<span>Current User</span>
					</div>
					<div class="status-value">
						{{ systemStatus.username || 'Not logged in' }}
					</div>
				</div>

				<div class="status-card">
					<div class="status-header">
						<Icon icon="material-symbols:storage" />
						<span>Database Host</span>
					</div>
					<div class="status-value">
						{{ systemStatus.databaseHost || 'Unknown' }}
					</div>
				</div>

				<div class="status-card">
					<div class="status-header">
						<Icon icon="material-symbols:database" />
						<span>Database Name</span>
					</div>
					<div class="status-value">
						{{ systemStatus.databaseName || 'Unknown' }}
					</div>
				</div>
			</div>
		</div>

		<!-- Quick Actions -->
		<div class="quick-actions">
			<h3>Quick Actions</h3>
			<div class="action-grid">
				<button @click="goToRoute('/social-accounts')" class="action-button">
					<Icon icon="jam:users" />
					<span>Manage Social Accounts</span>
				</button>

				<button @click="goToRoute('/api-keys')" class="action-button">
					<Icon icon="jam:key" />
					<span>API Keys</span>
				</button>

				<button @click="goToRoute('/users')" class="action-button">
					<Icon icon="material-symbols:people" />
					<span>User Management</span>
				</button>

				<button @click="goToRoute('/settings')" class="action-button">
					<Icon icon="jam:settings-alt" />
					<span>System Settings</span>
				</button>

				<button @click="refreshConnectors" class="action-button" :disabled="!clientReady">
					<Icon icon="material-symbols:refresh" />
					<span>Refresh Connectors</span>
				</button>

				<button @click="showCvars" class="action-button" :disabled="!clientReady">
					<Icon icon="material-symbols:tune" />
					<span>System Variables</span>
				</button>
			</div>
		</div>

		<!-- System Messages -->
		<div v-if="systemStatus.statusMessages && systemStatus.statusMessages.length > 0" class="system-messages">
			<h3>System Messages</h3>
			<div v-for="message in systemStatus.statusMessages" :key="message.id || message.message"
				 class="message-item" :class="message.type">
				<div class="message-content">
					<Icon :icon="getMessageIcon(message.type)" />
					<span>{{ message.message }}</span>
					<a v-if="message.url" :href="message.url" target="_blank" class="message-link">
						<Icon icon="material-symbols:open-in-new" />
					</a>
				</div>
			</div>
		</div>

		<!-- Connectors Status -->
		<div v-if="connectors.length > 0" class="connectors-section">
			<h3>Connectors</h3>
			<div class="connectors-grid">
				<div v-for="connector in connectors" :key="connector.name" class="connector-card">
					<div class="connector-header">
						<Icon :icon="connector.icon" />
						<span>{{ connector.name }}</span>
					</div>
					<div class="connector-status">
						<span :class="connector.isRegistered ? 'good' : 'bad'">
							{{ connector.isRegistered ? 'Registered' : 'Not Registered' }}
						</span>
					</div>
					<div v-if="connector.issues && connector.issues.length > 0" class="connector-issues">
						<small>Issues: {{ connector.issues.length }}</small>
					</div>
				</div>
			</div>
		</div>

		<!-- System Variables Dialog -->
		<dialog ref="cvarsDialog" class="cvars-dialog">
			<h2>System Variables (CVars)</h2>
			<div v-if="cvarsLoading" class="loading">
				<Icon icon="eos-icons:loading" />
				<span>Loading system variables...</span>
			</div>
			<div v-else-if="cvars.length > 0" class="cvars-content">
				<div v-for="category in cvars" :key="category.name" class="cvar-category">
					<h4>{{ category.name }}</h4>
					<div class="cvar-list">
						<div v-for="cvar in category.cvars" :key="cvar.keyName" class="cvar-item">
							<div class="cvar-info">
								<strong>{{ cvar.title || cvar.keyName }}</strong>
								<p v-if="cvar.description">{{ cvar.description }}</p>
								<div class="cvar-meta">
									<span class="cvar-type">{{ cvar.type }}</span>
									<span v-if="cvar.isReadOnly" class="readonly">Read Only</span>
								</div>
							</div>
							<div class="cvar-value">
								<input
									v-if="!cvar.isReadOnly"
									:type="cvar.type === 'int' ? 'number' : 'text'"
									:value="cvar.valueString || cvar.valueInt"
									@change="updateCvar(cvar, $event)"
									:maxlength="cvar.maxLength"
								/>
								<span v-else class="readonly-value">
									{{ cvar.valueString || cvar.valueInt }}
								</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div v-else>
				<p>No system variables found.</p>
			</div>
			<form method="dialog">
				<button type="submit" class="good">Close</button>
			</form>
		</dialog>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const clientReady = ref(false);
	const errorMessage = ref('');
	const systemStatus = ref({});
	const connectors = ref([]);
	const cvars = ref([]);
	const cvarsLoading = ref(false);
	const cvarsDialog = ref(null);

	async function refreshAll() {
		await Promise.all([
			refreshSystemStatus(),
			refreshConnectors(),
		]);
	}

	async function refreshSystemStatus() {
		try {
			const status = await window.client.getStatus();
			systemStatus.value = status;
			errorMessage.value = '';
		} catch (error) {
			errorMessage.value = `Failed to fetch system status: ${error.message}`;
			console.error('Error fetching system status:', error);
		}
	}

	async function refreshConnectors() {
		try {
			const response = await window.client.getConnectors({});
			connectors.value = response.connectors || [];
		} catch (error) {
			console.error('Error fetching connectors:', error);
		}
	}

	async function showCvars() {
		cvarsLoading.value = true;
		cvarsDialog.value.showModal();

		try {
			const response = await window.client.getCvars({});
			cvars.value = response.cvarCategories || [];
		} catch (error) {
			console.error('Error fetching CVars:', error);
			errorMessage.value = `Failed to fetch system variables: ${error.message}`;
		} finally {
			cvarsLoading.value = false;
		}
	}

	async function updateCvar(cvar, event) {
		const newValue = event.target.value;

		try {
			await window.client.setCvar({
				keyName: cvar.keyName,
				valueString: cvar.type !== 'int' ? newValue : '',
				valueInt: cvar.type === 'int' ? parseInt(newValue) : 0,
			});

			// Update the local value
			if (cvar.type === 'int') {
				cvar.valueInt = parseInt(newValue);
			} else {
				cvar.valueString = newValue;
			}
		} catch (error) {
			console.error('Error updating CVar:', error);
			alert(`Failed to update ${cvar.keyName}: ${error.message}`);
		}
	}

	function goToRoute(route) {
		window.router.push(route);
	}

	function getMessageIcon(type) {
		const iconMap = {
			'error': 'material-symbols:error',
			'warning': 'material-symbols:warning',
			'info': 'material-symbols:info',
			'success': 'material-symbols:check-circle',
		};
		return iconMap[type] || 'material-symbols:info';
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await refreshAll();
	});
</script>

<style scoped>
	.status-overview {
		margin-bottom: 2rem;
	}

	.status-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		margin-top: 1rem;
	}

	.status-card {
		background: #242424;
		border-radius: 0.5rem;
		padding: 1rem;
		border: 1px solid #444;
	}

	.status-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
		font-weight: bold;
	}

	.status-value {
		font-size: 1.2em;
		font-weight: bold;
	}

	.status-value.good {
		color: #4caf50;
	}

	.status-value.bad {
		color: #f44336;
	}

	.quick-actions {
		margin-bottom: 2rem;
	}

	.action-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		margin-top: 1rem;
	}

	.action-button {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		padding: 1rem;
		background: #242424;
		border: 1px solid #444;
		border-radius: 0.5rem;
		color: white;
		cursor: pointer;
		transition: background-color 0.2s;
	}

	.action-button:hover:not(:disabled) {
		background: #333;
	}

	.action-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.system-messages {
		margin-bottom: 2rem;
	}

	.message-item {
		display: flex;
		align-items: center;
		padding: 0.75rem;
		margin: 0.5rem 0;
		border-radius: 0.25rem;
		border-left: 4px solid;
	}

	.message-item.error {
		background: #ffebee;
		border-left-color: #f44336;
		color: #c62828;
	}

	.message-item.warning {
		background: #fff3e0;
		border-left-color: #ff9800;
		color: #e65100;
	}

	.message-item.info {
		background: #e3f2fd;
		border-left-color: #2196f3;
		color: #1565c0;
	}

	.message-content {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex: 1;
	}

	.message-link {
		margin-left: auto;
		color: inherit;
	}

	.connectors-section {
		margin-bottom: 2rem;
	}

	.connectors-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 1rem;
		margin-top: 1rem;
	}

	.connector-card {
		background: #242424;
		border-radius: 0.5rem;
		padding: 1rem;
		border: 1px solid #444;
	}

	.connector-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
		font-weight: bold;
	}

	.connector-status {
		margin-bottom: 0.5rem;
	}

	.connector-issues {
		font-size: 0.9em;
		color: #ff9800;
	}

	.cvars-dialog {
		max-width: 800px;
		max-height: 80vh;
		overflow-y: auto;
	}

	.cvars-content {
		max-height: 60vh;
		overflow-y: auto;
		margin-bottom: 1rem;
	}

	.cvar-category {
		margin-bottom: 1.5rem;
	}

	.cvar-category h4 {
		margin-bottom: 0.5rem;
		padding-bottom: 0.25rem;
		border-bottom: 1px solid #444;
	}

	.cvar-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem;
		margin: 0.5rem 0;
		background: #242424;
		border-radius: 0.25rem;
		border: 1px solid #444;
	}

	.cvar-info {
		flex: 1;
		margin-right: 1rem;
	}

	.cvar-info p {
		margin: 0.25rem 0;
		font-size: 0.9em;
		color: #ccc;
	}

	.cvar-meta {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.25rem;
	}

	.cvar-type {
		font-size: 0.8em;
		background: #444;
		padding: 0.125rem 0.25rem;
		border-radius: 0.125rem;
	}

	.readonly {
		font-size: 0.8em;
		color: #ff9800;
	}

	.cvar-value {
		min-width: 150px;
	}

	.cvar-value input {
		width: 100%;
		padding: 0.25rem;
		background: #333;
		border: 1px solid #555;
		border-radius: 0.25rem;
		color: white;
	}

	.readonly-value {
		color: #ccc;
		font-style: italic;
	}

	.loading {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		justify-content: center;
		padding: 2rem;
	}

	h3 {
		margin-bottom: 0.5rem;
		color: #fff;
	}

	h4 {
		color: #fff;
	}
</style>
