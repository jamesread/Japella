<template>
	<Section
		title="System Diagnostics"
		subtitle="System status and diagnostic information"
		classes="system-diagnostics"
	>
		<template #toolbar>
			<router-link :to="{ name: 'controlPanel' }" class="button neutral">
				<Icon icon="mdi:arrow-left" />
				Control Panel
			</router-link>
			<button type="button" class="neutral" :disabled="!clientReady" @click="refreshDiagnostics">
				<HugeiconsIcon :icon="RefreshIcon" />
			</button>
			<button
				v-if="canStopService"
				type="button"
				class="bad"
				title="Stop the service so the container can restart"
				:disabled="!clientReady || stoppingService"
				@click="confirmStopService"
			>
				{{ stoppingService ? 'Stopping…' : 'Restart Service' }}
			</button>
		</template>

		<p v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</p>

		<ReadOnlyTextArea
			ref="diagnosticsRef"
			v-model="diagnosticsYaml"
			label="Diagnostics (YAML)"
			:rows="22"
			markdown-ticks
			markdown-lang="yaml"
		/>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import ReadOnlyTextArea from 'picocrank/vue/components/ReadOnlyTextArea.vue';
	import { waitForClient } from '../javascript/util';
	import { canViewSystemDiagnosticsFromStatus } from '../javascript/rbacAccess.js';

	const clientReady = ref(false);
	const errorMessage = ref('');
	const systemStatus = ref({});
	const jobs = ref([]);
	const diagnosticsYaml = ref('');
	const diagnosticsRef = ref(null);
	const stoppingService = ref(false);

	const canStopService = computed(() =>
		systemStatus.value?.rbacIsSuperuser ||
		(Array.isArray(systemStatus.value?.rbacPermissions) &&
			systemStatus.value.rbacPermissions.includes('system.settings'))
	);

	const canLogs = computed(() =>
		systemStatus.value?.rbacIsSuperuser ||
		(Array.isArray(systemStatus.value?.rbacPermissions) &&
			systemStatus.value.rbacPermissions.includes('system.logs'))
	);

	function yamlValue(value) {
		if (value === null || value === undefined || value === '') {
			return 'unknown';
		}
		if (typeof value === 'boolean') {
			return value ? 'true' : 'false';
		}
		return String(value);
	}

	function yamlList(items) {
		if (!Array.isArray(items) || items.length === 0) {
			return 'none';
		}
		return items.join(', ');
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
		} catch {
			return '';
		}
	}

	async function buildDiagnosticsYaml() {
		diagnosticsYaml.value = '';
		await nextTick();

		const area = diagnosticsRef.value;
		if (!area) {
			return;
		}

		const st = systemStatus.value;

		area.appendSection('Server');
		area.appendYamlProperty('status', yamlValue(st.status));
		area.appendYamlProperty('version', yamlValue(st.version));
		area.appendYamlProperty('listenAddress', yamlValue(st.listenAddress));
		area.appendYamlProperty('usesSecureCookies', yamlValue(st.usesSecureCookies));
		area.appendYamlProperty('nanoservices', yamlList(st.nanoservices));

		area.appendSection('Session');
		area.appendYamlProperty('username', yamlValue(st.username));
		area.appendYamlProperty('isLoggedIn', yamlValue(st.isLoggedIn));
		area.appendYamlProperty('isImpersonating', yamlValue(st.isImpersonating));
		area.appendYamlProperty('impersonatorUsername', yamlValue(st.impersonatorUsername));
		area.appendYamlProperty('rbacIsSuperuser', yamlValue(st.rbacIsSuperuser));

		area.appendSection('Database');
		area.appendYamlProperty('host', yamlValue(st.databaseHost));
		area.appendYamlProperty('name', yamlValue(st.databaseName));
		area.appendYamlProperty('connected', yamlValue(st.databaseConnected));
		area.appendYamlProperty('schemaVersion', yamlValue(st.databaseSchemaVersion));
		area.appendYamlProperty('schemaDirty', yamlValue(st.databaseSchemaDirty));

		area.appendSection('Message broker');
		area.appendYamlProperty('amqpEnabled', yamlValue(st.amqpEnabled));
		area.appendYamlProperty('amqpConnected', yamlValue(st.amqpConnected));

		area.appendSection('Configuration');
		area.appendYamlProperty('configFileAbsolutePath', yamlValue(st.configFileAbsolutePath));

		if (st.statusMessages?.length) {
			area.appendSection('Status messages');
			for (const msg of st.statusMessages) {
				const key = msg.type || 'message';
				area.appendYamlProperty(key, yamlValue(msg.message));
			}
		}

		if (canLogs.value && jobs.value.length > 0) {
			area.appendSection('Background jobs');
			for (const job of jobs.value) {
				const lastRun = job.lastRun ? formatAbsoluteDate(job.lastRun) : 'never';
				area.appendYamlProperty(job.displayName || job.name, `${job.schedule} (last run: ${lastRun})`);
			}
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

	async function refreshJobs() {
		if (!clientReady.value || !canLogs.value) {
			return;
		}

		try {
			const response = await window.client.getJobsStatus({});
			jobs.value = response.jobs || [];
		} catch (error) {
			console.error('Error fetching jobs status:', error);
			errorMessage.value = `Failed to fetch jobs status: ${error.message}`;
			jobs.value = [];
		}
	}

	async function refreshDiagnostics() {
		await refreshSystemStatus();
		await refreshJobs();
		await buildDiagnosticsYaml();
	}

	async function confirmStopService() {
		if (
			!confirm(
				'Stop the Japella service now?\n\nThe process will exit and your container runtime should restart it automatically if a restart policy is configured (for example Docker restart: unless-stopped).'
			)
		) {
			return;
		}

		stoppingService.value = true;
		errorMessage.value = '';

		try {
			await window.client.stopService({});
		} catch (error) {
			console.error('Error stopping service:', error);
			errorMessage.value = `Failed to stop service: ${error.message}`;
			stoppingService.value = false;
		}
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await refreshDiagnostics();

		if (!canViewSystemDiagnosticsFromStatus(systemStatus.value)) {
			window.router.push('/');
		}
	});
</script>
