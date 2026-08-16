<template>
	<Section
		v-if="canAccessControlPanel"
		title="Control Panel"
		subtitle="System administration and monitoring dashboard for Japella"
		classes="control-panel"
	>
		<template #toolbar>
			<button @click="refreshAll" :disabled="!clientReady" class="neutral">
				<HugeiconsIcon :icon="RefreshIcon" />
			</button>
		</template>

		<div v-if="errorMessage" class="error-section">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<!-- Quick Actions -->
		<div class="quick-actions">
			<h3>Administration Actions</h3>
			<Navigation ref="localNavigation">
				<NavigationGrid />
			</Navigation>
		</div>

		<!-- System Messages -->
		<div v-if="systemStatus.statusMessages && systemStatus.statusMessages.length > 0" class="system-messages">
			<h3>System Messages</h3>
			<div v-for="message in systemStatus.statusMessages" :key="message.id || message.message"
				 class="message-item" :class="message.type">
				<div class="message-content">
					<HugeiconsIcon :icon="getMessageIcon(message.type)" />
					<span>{{ message.message }}</span>
					<a v-if="message.url" :href="message.url" target="_blank" class="message-link">
						<HugeiconsIcon :icon="LinkSquare01Icon" />
					</a>
				</div>
			</div>
		</div>

	</Section>

	<Section
		v-if="canViewSystemDiagnostics"
		title="System Diagnostics"
		subtitle="System status and diagnostic information"
		classes="system-diagnostics"
	>
		<div class="status-overview">
			<div class="status-grid">
				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="CheckmarkCircle01Icon" />
						<span>Server Status</span>
					</div>
					<div class="stat" :class="systemStatus.status === 'OK!' ? 'fg-success' : 'fg-warning'">
						{{ systemStatus.status || 'Unknown' }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="GridViewIcon" />
						<span>Nanoservices</span>
					</div>
					<div class="stat">
						{{ systemStatus.nanoservices?.length || 0 }} active
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="InformationCircleIcon" />
						<span>Version</span>
					</div>
					<div class="stat">
						{{ systemStatus.version || 'Unknown' }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="UserMultiple02Icon" />
						<span>Current User</span>
					</div>
					<div class="stat">
						{{ systemStatus.username || 'Not logged in' }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="HardDriveIcon" />
						<span>Database Host</span>
					</div>
					<div class="stat">
						{{ systemStatus.databaseHost || 'Unknown' }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="Database01Icon" />
						<span>Database Name</span>
					</div>
					<div class="stat">
						{{ systemStatus.databaseName || 'Unknown' }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="Settings01Icon" />
						<span>Listen Address</span>
					</div>
					<div class="stat">
						{{ systemStatus.listenAddress || 'Unknown' }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="LeftToRightListNumberIcon" />
						<span>Schema Version</span>
					</div>
					<div class="stat">
						{{ systemStatus.databaseSchemaVersion || 0 }}
					</div>
				</div>

				<div class="stat-display">
					<div class="status-header">
						<HugeiconsIcon :icon="AlertCircleIcon" />
						<span>Schema Dirty</span>
					</div>
					<div class="stat" :class="systemStatus.databaseSchemaDirty ? 'fg-error' : 'fg-success'">
						{{ systemStatus.databaseSchemaDirty ? 'Yes' : 'No' }}
					</div>
				</div>
			</div>
		</div>
	</Section>

	<Section
		v-if="canLogs"
		title="Background Jobs"
		subtitle="Status and last run times for background jobs"
		classes="background-jobs"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshJobs" :disabled="!clientReady || jobsLoading" class="neutral">
				<HugeiconsIcon :icon="RefreshIcon" />
			</button>
		</template>

		<div v-if="jobsLoading" class="loading">
			<HugeiconsIcon :icon="Loading01Icon" />
			<span>Loading job status...</span>
		</div>
		<div v-else-if="jobs.length === 0" class="inline-notification note">
			No background jobs found.
		</div>
		<table v-else class="data-table row-hover">
			<thead>
				<tr>
					<th>Job Name</th>
					<th>Schedule</th>
					<th>Last Run</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="job in jobs" :key="job.name">
					<td>{{ job.displayName }}</td>
					<td>{{ job.schedule }}</td>
					<td>
						<span v-if="job.lastRun" :title="formatAbsoluteDate(job.lastRun)">
							{{ formatRelativeDate(job.lastRun) }}
						</span>
						<span v-else class="text-muted">Never</span>
					</td>
				</tr>
			</tbody>
		</table>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import {
		canAccessControlPanelFromStatus,
		canViewSystemDiagnosticsFromStatus,
		canAccessIamFromStatus,
	} from '../javascript/rbacAccess.js';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import { HugeiconsIcon} from '@hugeicons/vue';
	import {
		RefreshIcon,
		CheckmarkCircle01Icon,
		GridViewIcon,
		InformationCircleIcon,
		UserMultiple02Icon,
		HardDriveIcon,
		Database01Icon,
		LeftToRightListNumberIcon,
		Settings01Icon,
		LinkSquare01Icon,
		Loading01Icon,
		CancelCircleIcon,
		ApproximatelyEqualCircleIcon,
		ActivityIcon,
		Flowchart01Icon,
		AlertCircleIcon,
		UserShield01Icon,
		WebhookIcon,
	} from '@hugeicons/core-free-icons';

	const clientReady = ref(false);
	const errorMessage = ref('');
	const systemStatus = ref({});
	const localNavigation = ref(null);
	const jobs = ref([]);
	const jobsLoading = ref(false);

	const canLogs = computed(() =>
		systemStatus.value?.rbacIsSuperuser ||
		(Array.isArray(systemStatus.value?.rbacPermissions) &&
			systemStatus.value.rbacPermissions.includes('system.logs'))
	);

	const canAccessControlPanel = computed(() =>
		canAccessControlPanelFromStatus(systemStatus.value)
	);

	const canViewSystemDiagnostics = computed(() =>
		canViewSystemDiagnosticsFromStatus(systemStatus.value)
	);

	async function refreshAll() {
		await refreshSystemStatus();
		if (canLogs.value) {
			await refreshJobs();
		}
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

	function goToRoute(route) {
		window.router.push(route);
	}

	function getMessageIcon(type) {
		const iconMap = {
			'error': CancelCircleIcon,
			'warning': AlertCircleIcon,
			'info': ApproximatelyEqualCircleIcon,
			'success': CheckmarkCircle01Icon,
		};
		return iconMap[type] || InformationCircleIcon;
	}

	async function refreshJobs() {
		if (!clientReady.value) return;

		jobsLoading.value = true;
		try {
			const response = await window.client.getJobsStatus({});
			jobs.value = response.jobs || [];
		} catch (error) {
			console.error('Error fetching jobs status:', error);
			errorMessage.value = `Failed to fetch jobs status: ${error.message}`;
			jobs.value = [];
		} finally {
			jobsLoading.value = false;
		}
	}

	function formatRelativeDate(dateString) {
		if (!dateString) {
			return 'Never';
		}

		try {
			const date = new Date(dateString);
			if (isNaN(date.getTime())) {
				return dateString;
			}

			const now = new Date();
			const diffMs = now - date;
			const diffSeconds = Math.floor(Math.abs(diffMs) / 1000);
			const diffMinutes = Math.floor(diffSeconds / 60);
			const diffHours = Math.floor(diffMinutes / 60);
			const diffDays = Math.floor(diffHours / 24);

			if (diffSeconds < 60) {
				return 'Just now';
			} else if (diffMinutes < 60) {
				return `${diffMinutes} minute${diffMinutes !== 1 ? 's' : ''} ago`;
			} else if (diffHours < 24) {
				return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
			} else if (diffDays < 7) {
				return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
			} else {
				return date.toLocaleDateString();
			}
		} catch (e) {
			return dateString;
		}
	}

	function formatAbsoluteDate(dateString) {
		if (!dateString) {
			return '';
		}

		try {
			const date = new Date(dateString);
			if (isNaN(date.getTime())) {
				return '';
			}
			return date.toLocaleString();
		} catch (e) {
			return '';
		}
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await refreshAll();

		if (!canAccessControlPanelFromStatus(systemStatus.value)) {
			window.router.push('/');
			return;
		}

		// Setup administration actions navigation
		if (localNavigation.value) {
			const st = systemStatus.value;
			const canConnectors =
				st?.rbacIsSuperuser ||
				(Array.isArray(st?.rbacPermissions) && st.rbacPermissions.includes('system.connectors'));
			const canSettings =
				st?.rbacIsSuperuser ||
				(Array.isArray(st?.rbacPermissions) && st.rbacPermissions.includes('system.settings'));
			const canLogsNav =
				st?.rbacIsSuperuser ||
				(Array.isArray(st?.rbacPermissions) && st.rbacPermissions.includes('system.logs'));
			const canDiagnosticsNav = canViewSystemDiagnosticsFromStatus(st);

			if (canAccessIamFromStatus(st)) {
				localNavigation.value.addCallback('IAM', () => goToRoute('/control-panel/iam'), {
					icon: UserShield01Icon,
					name: 'iam',
					description: 'Users, groups, account policies, and roles',
				});
			}

			if (canConnectors) {
				localNavigation.value.addCallback('Connectors', () => goToRoute('/connectors'), {
					icon: GridViewIcon,
					name: 'connectors',
					description: 'View currently started connectors',
				});
			}

			if (canSettings) {
				localNavigation.value.addCallback('System Settings', () => goToRoute('/settings'), {
					icon: Settings01Icon,
					name: 'system-settings',
					description: 'Configure system settings',
				});
				localNavigation.value.addCallback('Webhooks', () => goToRoute('/admin/webhooks'), {
					icon: WebhookIcon,
					name: 'webhooks',
					description: 'Configure event webhook endpoints',
				});
			}

			localNavigation.value.addCallback('System Architecture', () => goToRoute('/control-panel/system-architecture'), {
				icon: Flowchart01Icon,
				name: 'system-architecture',
				description: 'Diagram of services, database, and message broker',
			});

			if (canDiagnosticsNav) {
				localNavigation.value.addCallback('System Diagnostics', () => goToRoute('/control-panel/system-diagnostics'), {
					icon: HardDriveIcon,
					name: 'system-diagnostics',
					description: 'YAML diagnostics for server, database, and runtime',
				});
			}

			if (canLogsNav) {
				localNavigation.value.addCallback('System Logs', () => goToRoute('/logs'), {
					icon: ActivityIcon,
					name: 'system-logs',
					description: 'View application logs and events',
				});
			}
		}
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

	.stat-display {
		min-width: 0;
		overflow: hidden;
	}

	.stat-display .stat {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.status-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
		font-weight: bold;
		min-width: 0;
	}

	.status-header span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
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
		height: 5em;
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

	.text-muted {
		color: #999;
		font-style: italic;
	}
</style>
