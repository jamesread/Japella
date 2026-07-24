<template>
	<Section
		:title="isEdit ? 'Edit account policy' : 'Create account policy'"
		subtitle="Configure ordered approval stages and attach social accounts. Policies can apply to MCP submissions, UI submissions, or both."
		classes="account-policy-editor settings-users"
	>
		<template #toolbar>
			<router-link :to="{ name: 'accountPolicies' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to policies
			</router-link>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-if="loading" class="muted">Loading…</div>

		<form v-else class="policy-form" @submit.prevent="savePolicy">
			<div class="form-row">
				<label for="policy-name">Name</label>
				<input id="policy-name" v-model="form.name" type="text" autocomplete="off" required :disabled="saving" />
			</div>
			<div class="form-row">
				<label for="policy-desc">Description</label>
				<input id="policy-desc" v-model="form.description" type="text" autocomplete="off" :disabled="saving" />
			</div>
			<div class="form-row check-row">
				<label><input v-model="form.applyToMcp" type="checkbox" :disabled="saving" /> Apply to MCP submissions</label>
				<label><input v-model="form.applyToUi" type="checkbox" :disabled="saving" /> Apply to UI submissions</label>
			</div>

			<h3 class="subsection-title">Approval stages (ordered)</h3>
			<div v-for="(stage, idx) in form.stages" :key="idx" class="stage-row">
				<span class="stage-num">{{ idx + 1 }}.</span>
				<select v-model="stage.kind" :disabled="saving">
					<option value="user">User</option>
					<option value="group">User group</option>
				</select>
				<select v-if="stage.kind === 'user'" v-model.number="stage.userId" :disabled="saving">
					<option :value="0">Select user…</option>
					<option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }}</option>
				</select>
				<select v-else v-model.number="stage.userGroupId" :disabled="saving">
					<option :value="0">Select group…</option>
					<option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
				</select>
				<button type="button" class="bad small" :disabled="saving || form.stages.length <= 1" @click="removeStage(idx)">
					Remove
				</button>
			</div>
			<button type="button" class="neutral" :disabled="saving" @click="addStage">Add stage</button>

			<h3 class="subsection-title">Social accounts</h3>
			<fieldset class="member-checks">
				<label v-for="a in accounts" :key="a.id" class="check-label">
					<input v-model="form.socialAccountIds" type="checkbox" :value="a.id" :disabled="saving" />
					{{ a.identity }} ({{ a.connector }})
				</label>
			</fieldset>

			<div class="form-actions">
				<button type="submit" class="good" :disabled="saving || !form.name.trim()">
					{{ saving ? 'Saving…' : (isEdit ? 'Save changes' : 'Create policy') }}
				</button>
				<router-link :to="{ name: 'accountPolicies' }" class="button neutral">Cancel</router-link>
			</div>
		</form>
	</Section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Icon } from '@iconify/vue';
import Section from 'picocrank/vue/components/Section.vue';
import { fetchAppStatus } from '../javascript/status.js';

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const saving = ref(false);
const errorMessage = ref('');
const users = ref([]);
const groups = ref([]);
const accounts = ref([]);
const form = reactive(emptyForm());

const policyId = computed(() => {
	const raw = route.params.id;
	if (!raw || raw === 'new') return 0;
	const n = Number(raw);
	return Number.isFinite(n) && n > 0 ? n : 0;
});
const isEdit = computed(() => policyId.value > 0);

function emptyForm() {
	return {
		name: '',
		description: '',
		applyToMcp: true,
		applyToUi: false,
		stages: [{ kind: 'user', userId: 0, userGroupId: 0 }],
		socialAccountIds: [],
	};
}

function addStage() {
	form.stages.push({ kind: 'user', userId: 0, userGroupId: 0 });
}

function removeStage(idx) {
	if (form.stages.length <= 1) return;
	form.stages.splice(idx, 1);
}

function applyPolicy(p) {
	form.name = p.name || '';
	form.description = p.description || '';
	form.applyToMcp = !!p.applyToMcp;
	form.applyToUi = !!p.applyToUi;
	form.socialAccountIds = [...(p.socialAccountIds || [])];
	form.stages = (p.stages || []).map((s) => ({
		kind: s.userGroupId ? 'group' : 'user',
		userId: s.userId || 0,
		userGroupId: s.userGroupId || 0,
	}));
	if (!form.stages.length) {
		form.stages = [{ kind: 'user', userId: 0, userGroupId: 0 }];
	}
}

function stagesPayload() {
	return form.stages.map((s, i) => {
		const stage = { stageOrder: i };
		if (s.kind === 'user') {
			stage.userId = s.userId;
			stage.userGroupId = 0;
		} else {
			stage.userId = 0;
			stage.userGroupId = s.userGroupId;
		}
		return stage;
	});
}

async function load() {
	loading.value = true;
	errorMessage.value = '';
	try {
		await fetchAppStatus();
		const [usersRes, groupsRes, accountsRes] = await Promise.all([
			window.client.getUsers({}),
			window.client.listUserGroups({}),
			window.client.getSocialAccounts({ onlyActive: false }),
		]);
		users.value = usersRes.users || [];
		groups.value = groupsRes.groups || [];
		accounts.value = accountsRes.accounts || [];

		if (isEdit.value) {
			const res = await window.client.getAccountPolicy({ id: policyId.value });
			if (!res.policy) {
				errorMessage.value = 'Policy not found';
				return;
			}
			applyPolicy(res.policy);
		} else {
			Object.assign(form, emptyForm());
		}
	} catch (e) {
		console.error(e);
		errorMessage.value = e?.message || 'Failed to load form';
	} finally {
		loading.value = false;
	}
}

async function savePolicy() {
	saving.value = true;
	errorMessage.value = '';
	try {
		const body = {
			name: form.name.trim(),
			description: form.description,
			applyToMcp: form.applyToMcp,
			applyToUi: form.applyToUi,
			stages: stagesPayload(),
			socialAccountIds: form.socialAccountIds,
		};
		if (isEdit.value) {
			await window.client.updateAccountPolicy({ id: policyId.value, ...body });
		} else {
			await window.client.createAccountPolicy(body);
		}
		await router.push({ name: 'accountPolicies' });
	} catch (e) {
		console.error(e);
		errorMessage.value = e?.message || 'Failed to save policy';
	} finally {
		saving.value = false;
	}
}

onMounted(load);
</script>

<style scoped>
.policy-form {
	padding: 0.5rem 0;
	max-width: 40rem;
}
.stage-row {
	display: flex;
	gap: 0.5rem;
	align-items: center;
	margin-bottom: 0.5rem;
	flex-wrap: wrap;
}
.stage-num {
	min-width: 1.5rem;
	font-weight: 600;
}
.check-row {
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
}
.form-row {
	margin-bottom: 0.75rem;
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
}
.form-actions {
	display: flex;
	gap: 0.5rem;
	align-items: center;
	margin-top: 1rem;
}
.member-checks {
	border: none;
	padding: 0;
	margin: 0 0 1rem;
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}
.check-label {
	display: flex;
	gap: 0.5rem;
	align-items: center;
}
.subsection-title {
	margin: 1.25rem 0 0.5rem;
}
</style>
