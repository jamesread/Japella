<template>
	<Section
		title="Health"
		:icon="Activity01Icon"
		:padding="false"
		subtitle="Problems found by background checks. They run every 15 minutes, or on demand."
	>
		<template #toolbar>
			<button
				type="button"
				class="neutral"
				:disabled="loading"
				title="Refresh"
				aria-label="Refresh"
				@click="load"
			>
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" aria-hidden="true" />
			</button>
			<button
				type="button"
				class="good inline-icon"
				:disabled="running !== null"
				@click="runChecks()"
			>
				<HugeiconsIcon :icon="Activity01Icon" width="1em" height="1em" aria-hidden="true" />
				<span>{{ running === 'all' ? 'Running checks…' : 'Run checks now' }}</span>
			</button>
		</template>

		<div v-if="error" class="section-content padding">
			<NotificationBlock type="error" :message="error" />
		</div>
		<div v-if="loading" class="section-content padding">Loading…</div>
		<div v-else-if="openIssues.length === 0" class="section-content padding">
			<p class="inline-icon">
				<HugeiconsIcon :icon="CheckmarkCircle02Icon" width="1.25em" height="1.25em" aria-hidden="true" />
				<span>No open issues.</span>
			</p>
		</div>
		<template v-else>
			<p class="section-content padding">
				<template v-for="(severity, index) in openCounts" :key="severity.id">
					<span v-if="index > 0"> · </span>
					<span :class="['tag', severity.id]">
						{{ severity.count }} {{ severity.count === 1 ? severity.one : severity.many }}
					</span>
				</template>
			</p>
			<Table
				:data="tableRows(openIssues)"
				:headers="issueHeaders"
				row-key="id"
				:filterable="false"
				:column-options="false"
			>
				<template #cell-severity="{ row }">
					<span :class="['tag', row.severity]">{{ row.severity }}</span>
				</template>
				<template #cell-title="{ row }">
					<strong>{{ row.title }}</strong>
					<p class="health-detail">{{ row.detail }}</p>
					<p v-if="row.occurrences > 1" class="subtle">
						Seen {{ row.occurrences }} times · first seen {{ formatTime(row.firstSeenUnix) }}
					</p>
				</template>
				<template #cell-lastSeen="{ row }">{{ formatTime(row.lastSeenUnix) }}</template>
				<template #cell-actions="{ row }">
					<div class="flex-row g1">
						<router-link v-if="row.link" :to="row.link" class="button good">Fix</router-link>
						<button
							type="button"
							class="neutral inline-icon"
							title="Hide from the badge until it recurs"
							@click="setAcknowledged(row, true)"
						>
							<HugeiconsIcon :icon="ViewOffIcon" width="1em" height="1em" aria-hidden="true" />
							<span>Acknowledge</span>
						</button>
					</div>
				</template>
			</Table>
		</template>
	</Section>

	<Section
		v-if="acknowledgedIssues.length > 0"
		title="Acknowledged"
		:icon="Tick02Icon"
		:padding="false"
		subtitle="Still present, but not counted in the navigation badge."
	>
		<Table
			:data="tableRows(acknowledgedIssues)"
			:headers="issueHeaders"
			row-key="id"
			:filterable="false"
			:column-options="false"
		>
			<template #cell-severity="{ row }">
				<span :class="['tag', row.severity]">{{ row.severity }}</span>
			</template>
			<template #cell-title="{ row }">
				<strong>{{ row.title }}</strong>
				<p class="health-detail">{{ row.detail }}</p>
			</template>
			<template #cell-lastSeen="{ row }">{{ formatTime(row.lastSeenUnix) }}</template>
			<template #cell-actions="{ row }">
				<div class="flex-row g1">
					<router-link v-if="row.link" :to="row.link" class="button good">Fix</router-link>
					<button type="button" class="neutral" @click="setAcknowledged(row, false)">Restore</button>
				</div>
			</template>
		</Table>
	</Section>

	<Section
		v-if="resolvedIssues.length > 0"
		title="Recently resolved"
		:icon="CheckmarkCircle02Icon"
		:padding="false"
		subtitle="Issues no longer found in the last 30 days."
	>
		<Table
			:data="tableRows(resolvedIssues)"
			:headers="resolvedHeaders"
			row-key="id"
			:filterable="false"
			:column-options="false"
		>
			<template #cell-severity="{ row }">
				<span :class="['tag', row.severity]">{{ row.severity }}</span>
			</template>
			<template #cell-resolved="{ row }">{{ formatTime(row.resolvedAtUnix) }}</template>
		</Table>
	</Section>

	<Section v-if="checks.length > 0" title="Checks" :icon="Activity01Icon" :padding="false">
		<Table
			:data="checks"
			:headers="checkHeaders"
			row-key="checkId"
			:filterable="false"
			:column-options="false"
		>
			<template #cell-title="{ row }">
				{{ row.title }}
				<p v-if="row.error" class="health-detail">{{ row.error }}</p>
			</template>
			<template #cell-status="{ row }">
				<span :class="['tag', row.status === 'ok' ? 'good' : row.status === 'failed' ? 'error' : 'note']">
					{{ row.status }}
				</span>
			</template>
			<template #cell-lastRun="{ row }">{{ formatTime(row.lastRunUnix) }}</template>
			<template #cell-durationMs="{ row }">
				{{ Number(row.lastRunUnix || 0) ? `${row.durationMs} ms` : '—' }}
			</template>
			<template #cell-actions="{ row }">
				<button
					type="button"
					class="neutral"
					:disabled="running !== null"
					@click="runChecks(row.checkId)"
				>
					{{ running === row.checkId ? 'Running…' : 'Run' }}
				</button>
			</template>
		</Table>
	</Section>
</template>

<script setup>
	import { computed, onMounted, ref } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		Activity01Icon,
		CheckmarkCircle02Icon,
		RefreshIcon,
		Tick02Icon,
		ViewOffIcon,
	} from '@hugeicons/core-free-icons';
	import { waitForClient } from '../javascript/util';
	import { HEALTH_SEVERITIES, notifyHealthChanged } from '../javascript/health.js';

	const issues = ref([]);
	const checks = ref([]);
	const loading = ref(true);
	const running = ref(null);
	const error = ref(null);

	const severityRank = Object.fromEntries(HEALTH_SEVERITIES.map((s, index) => [s.id, index]));
	const bySeverity = (a, b) =>
		(severityRank[a.severity] ?? 9) - (severityRank[b.severity] ?? 9) ||
		Number(b.lastSeenUnix || 0) - Number(a.lastSeenUnix || 0);

	const openIssues = computed(() =>
		issues.value
			.filter((issue) => !Number(issue.resolvedAtUnix || 0) && !Number(issue.acknowledgedAtUnix || 0))
			.sort(bySeverity),
	);
	const acknowledgedIssues = computed(() =>
		issues.value
			.filter((issue) => !Number(issue.resolvedAtUnix || 0) && Number(issue.acknowledgedAtUnix || 0))
			.sort(bySeverity),
	);
	const resolvedIssues = computed(() =>
		issues.value
			.filter((issue) => Number(issue.resolvedAtUnix || 0))
			.sort((a, b) => Number(b.resolvedAtUnix || 0) - Number(a.resolvedAtUnix || 0)),
	);

	const openCounts = computed(() =>
		HEALTH_SEVERITIES.map((severity) => ({
			...severity,
			count: openIssues.value.filter((issue) => issue.severity === severity.id).length,
		})).filter((severity) => severity.count > 0),
	);

	const issueHeaders = [
		{ key: 'severity', label: 'Severity', width: '7rem' },
		{ key: 'title', label: 'Issue' },
		{ key: 'checkTitle', label: 'Check', width: '14rem' },
		{ key: 'lastSeen', label: 'Last seen', width: '11rem' },
		{ key: 'actions', label: 'Actions', width: '13rem' },
	];
	const resolvedHeaders = [
		{ key: 'severity', label: 'Severity', width: '7rem' },
		{ key: 'title', label: 'Issue' },
		{ key: 'checkTitle', label: 'Check', width: '14rem' },
		{ key: 'resolved', label: 'Resolved', width: '11rem' },
	];
	const checkHeaders = [
		{ key: 'title', label: 'Check' },
		{ key: 'status', label: 'Status', width: '7rem' },
		{ key: 'lastRun', label: 'Last run', width: '11rem' },
		{ key: 'durationMs', label: 'Took', width: '6rem' },
		{ key: 'actions', label: 'Actions', width: '6rem' },
	];

	const tableRows = (list) => list.map((issue) => ({ ...issue, rawId: issue.id, id: String(issue.id) }));

	function formatTime(unix) {
		const secs = Number(unix || 0);
		if (!secs) {
			return '—';
		}
		return new Date(secs * 1000).toLocaleString();
	}

	async function load() {
		error.value = null;
		try {
			await waitForClient();
			const res = await window.client.listHealthIssues({
				includeResolved: true,
				includeAcknowledged: true,
			});
			issues.value = res.issues || [];
			checks.value = res.checks || [];
		} catch (e) {
			error.value = e instanceof Error ? e.message : String(e);
		} finally {
			loading.value = false;
		}
	}

	async function runChecks(checkId = '') {
		running.value = checkId || 'all';
		try {
			await waitForClient();
			await window.client.runHealthChecks({ checkId });
			await load();
			notifyHealthChanged();
		} catch (e) {
			error.value = e instanceof Error ? e.message : String(e);
		} finally {
			running.value = null;
		}
	}

	async function setAcknowledged(issue, acknowledged) {
		try {
			await waitForClient();
			await window.client.acknowledgeHealthIssue({
				id: issue.rawId ?? issue.id,
				acknowledged,
			});
			await load();
			notifyHealthChanged();
		} catch (e) {
			error.value = e instanceof Error ? e.message : String(e);
		}
	}

	onMounted(load);
</script>

<style scoped>
	.health-detail {
		margin: 0.25rem 0 0;
		white-space: pre-wrap;
	}
</style>
