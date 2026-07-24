<template>
	<Section
		title="Account Policies"
		subtitle="Configure ordered approval stages for groups of social accounts. Policies can apply to MCP submissions, UI submissions, or both."
		classes="account-policies settings-users"
		:padding="false"
	>
		<template #toolbar>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="loadAll">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button type="button" class="good" title="Create policy" @click="goCreate">
				<Icon icon="material-symbols:add-rounded" />
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error groups-banner-pad">{{ errorMessage }}</div>
		<div v-if="loading && !policies.length" class="groups-banner-pad muted">Loading…</div>

		<template v-else>
			<table v-if="policies.length" class="groups-table user-table-wrap">
				<thead>
					<tr>
						<th>Name</th>
						<th>MCP</th>
						<th>UI</th>
						<th>Stages</th>
						<th>Accounts</th>
						<th class="actions-col">Actions</th>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="p in policies"
						:key="p.id"
						class="group-row"
						@click="openPolicy(p)"
					>
						<td><strong>{{ p.name }}</strong></td>
						<td>{{ p.applyToMcp ? 'Yes' : 'No' }}</td>
						<td>{{ p.applyToUi ? 'Yes' : 'No' }}</td>
						<td>{{ (p.stages || []).length }}</td>
						<td>{{ (p.socialAccountIds || []).length }}</td>
						<td align="right">
							<button type="button" class="bad small" @click.stop="deletePolicy(p)">Delete</button>
						</td>
					</tr>
				</tbody>
			</table>
			<p v-else class="inline-notification note groups-banner-pad">No account policies yet.</p>
		</template>
	</Section>
</template>

<script setup>
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Icon } from '@iconify/vue';
import Section from 'picocrank/vue/components/Section.vue';

const router = useRouter();
const loading = ref(true);
const errorMessage = ref('');
const policies = ref([]);

function goCreate() {
	router.push({ name: 'createAccountPolicy' });
}

function openPolicy(p) {
	router.push({ name: 'editAccountPolicy', params: { id: String(p.id) } });
}

async function loadAll() {
	loading.value = true;
	errorMessage.value = '';
	try {
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
	if (!confirm(`Delete policy “${p.name}”?`)) return;
	try {
		await window.client.deleteAccountPolicy({ id: p.id });
		await loadAll();
	} catch (e) {
		errorMessage.value = e?.message || 'Failed to delete policy';
	}
}

onMounted(loadAll);
</script>

<style scoped>
.groups-banner-pad {
	padding: 0.75rem 1rem;
}
.group-row {
	cursor: pointer;
}
</style>
