<template>
	<Section
		subtitle="Configure ordered approval stages for groups of social accounts. Policies can apply to MCP submissions, UI submissions, or both."
		classes="account-policies settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="SecurityValidationIcon" width="22" height="22" aria-hidden="true" />
				Account Policies
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading"
				@click="loadAll"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
			<button
				v-if="canManage"
				type="button"
				class="inline-icon good"
				aria-label="Create policy"
				title="Create policy"
				:disabled="loading"
				@click="goCreate"
			>
				<HugeiconsIcon
					:icon="Add01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error list-banner-pad">{{ errorMessage }}</div>
		<div v-if="loading && !policies.length" class="list-banner-pad muted">Loading…</div>

		<template v-else>
			<p v-if="!policies.length" class="inline-notification note list-banner-pad">No account policies yet.</p>

			<Table
				v-else
				class="policies-table-wrap"
				row-clickable
				:data="tableRows"
				:headers="tableHeaders"
				@row-click="openPolicy"
			>
				<template #cell-name="{ value }">
					<strong>{{ value }}</strong>
				</template>
				<template #cell-actions="{ row }">
					<div v-if="canManage" class="actions-cell">
						<button type="button" class="bad small" :disabled="saving" @click="deletePolicy(row)">
							Delete
						</button>
					</div>
				</template>
			</Table>
		</template>
	</Section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { waitForClient } from '../javascript/util';
import { HugeiconsIcon } from '@hugeicons/vue';
import { Add01Icon, RefreshIcon, SecurityValidationIcon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import Table from './picocrank/TableWithRowClick.vue';

const iconStrokeWidth = 2.5;

const router = useRouter();
const loading = ref(true);
const saving = ref(false);
const errorMessage = ref('');
const policies = ref([]);
const statusPerms = ref([]);
const statusSuper = ref(false);

const canManage = computed(
	() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('account-policies.manage')),
);

const tableHeaders = computed(() => {
	const headers = [
		{ key: 'name', label: 'Name', sortable: true },
		{ key: 'mcpLabel', label: 'MCP', sortable: true, width: '6rem' },
		{ key: 'uiLabel', label: 'UI', sortable: true, width: '6rem' },
		{ key: 'stageCount', label: 'Stages', sortable: true, width: '8rem' },
		{ key: 'accountCount', label: 'Accounts', sortable: true, width: '8rem' },
	];
	if (canManage.value) {
		headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
	}
	return headers;
});

const tableRows = computed(() =>
	policies.value.map((p) => ({
		...p,
		mcpLabel: p.applyToMcp ? 'Yes' : 'No',
		uiLabel: p.applyToUi ? 'Yes' : 'No',
		stageCount: (p.stages || []).length,
		accountCount: (p.socialAccountIds || []).length,
	})),
);

function goCreate() {
	router.push({ name: 'createAccountPolicy' });
}

function openPolicy(p) {
	router.push({ name: 'editAccountPolicy', params: { id: String(p.id) } });
}

async function refreshStatus() {
	await waitForClient();
	const st = await window.client.getStatus({});
	statusPerms.value = st.rbacPermissions || [];
	statusSuper.value = Boolean(st.rbacIsSuperuser);
}

async function loadAll() {
	loading.value = true;
	errorMessage.value = '';
	try {
		await waitForClient();
		await refreshStatus();
		const polRes = await window.client.listAccountPolicies({});
		policies.value = polRes.policies || [];
	} catch (e) {
		console.error(e);
		errorMessage.value = e?.message || 'Failed to load account policies';
	} finally {
		loading.value = false;
	}
}

async function deletePolicy(p) {
	if (!canManage.value || !confirm(`Delete policy “${p.name}”?`)) return;
	saving.value = true;
	errorMessage.value = '';
	try {
		await window.client.deleteAccountPolicy({ id: p.id });
		await loadAll();
	} catch (e) {
		errorMessage.value = e?.message || 'Failed to delete policy';
	} finally {
		saving.value = false;
	}
}

onMounted(loadAll);
</script>

<style scoped>
.section-title-with-icon {
	display: inline-flex;
	align-items: center;
	gap: 0.45em;
	vertical-align: middle;
}

.list-banner-pad {
	padding-left: 1em;
	padding-right: 1em;
}

.policies-table-wrap {
	margin-top: 0.5rem;
	margin-bottom: 1.5rem;
}

.actions-cell {
	text-align: right;
}
</style>
