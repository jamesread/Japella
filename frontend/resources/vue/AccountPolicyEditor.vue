<template>
	<Section
		:title="isEdit ? 'Edit account policy' : 'Create account policy'"
		subtitle="Configure ordered approval stages and attach social accounts. Policies can apply to MCP submissions, UI submissions, or both."
		classes="account-policy-editor settings-users"
	>
		<template #toolbar>
			<router-link :to="{ name: 'accountPolicies' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to policies</span>
			</router-link>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-if="loading" class="muted">Loading…</div>

		<FormLayout v-else @submit.prevent="savePolicy">
			<FormField label="Name" for="policy-name" :disabled="saving">
				<input id="policy-name" v-model="form.name" type="text" autocomplete="off" required :disabled="saving" />
			</FormField>

			<FormField label="Description" for="policy-desc" :disabled="saving">
				<input id="policy-desc" v-model="form.description" type="text" autocomplete="off" :disabled="saving" />
			</FormField>

			<FormField label="Apply to MCP submissions" :disabled="saving">
				<input id="policy-apply-mcp" v-model="form.applyToMcp" type="checkbox" :disabled="saving" />
			</FormField>

			<FormField label="Apply to UI submissions" :disabled="saving">
				<input id="policy-apply-ui" v-model="form.applyToUi" type="checkbox" :disabled="saving" />
			</FormField>

			<FormField label="Approval stages (ordered)" fake :disabled="saving">
				<div class="approval-stages">
					<div v-for="(stage, idx) in form.stages" :key="idx" class="stage-row">
						<span class="stage-num">{{ idx + 1 }}.</span>
						<select v-model="stage.kind" :disabled="saving">
							<option value="user">User</option>
							<option value="group">User group</option>
						</select>
						<div v-if="stage.kind === 'user'" class="stage-user-picker">
							<span v-if="stage.userId" class="stage-user-label">{{ stageUserLabel(stage) }}</span>
							<span v-else class="stage-user-label muted">No user selected</span>
							<button
								type="button"
								class="neutral small"
								:disabled="saving"
								@click="openStageUserLookup(idx)"
							>
								{{ stage.userId ? 'Change user' : 'Select user' }}
							</button>
						</div>
						<select v-else v-model.number="stage.userGroupId" :disabled="saving">
							<option :value="0">Select group…</option>
							<option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
						</select>
						<button type="button" class="bad small" :disabled="saving || form.stages.length <= 1" @click="removeStage(idx)">
							Remove
						</button>
					</div>
					<button type="button" class="neutral" :disabled="saving" @click="addStage">Add stage</button>
				</div>
			</FormField>

			<FormField
				v-if="accounts.length"
				label="Social accounts"
				fake
				:disabled="saving"
				description="Policies apply only to posts targeting the selected accounts."
			>
				<CheckGroup
					v-model="form.socialAccountIds"
					name="policy-social-accounts"
					:options="socialAccountOptions"
					:disabled="saving"
				/>
			</FormField>
			<p v-else class="inline-notification note">No social accounts available.</p>

			<template #actions>
				<button type="submit" class="good" :disabled="saving || !form.name.trim()">
					{{ saving ? 'Saving…' : (isEdit ? 'Save changes' : 'Create policy') }}
				</button>
				<router-link :to="{ name: 'accountPolicies' }" class="button neutral">Cancel</router-link>
			</template>
		</FormLayout>

		<UserLookupDialog
			ref="userLookup"
			title="Select approver"
			subtitle="Choose the user who must approve at this stage."
			@picked="onStageUserPicked"
		/>
	</Section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { HugeiconsIcon } from '@hugeicons/vue';
import { ArrowLeft01Icon } from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import FormLayout from 'picocrank/vue/components/FormLayout.vue';
import FormField from 'picocrank/vue/components/FormField.vue';
import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
import UserLookupDialog from './UserLookupDialog.vue';
import { fetchAppStatus } from '../javascript/status.js';

const iconStrokeWidth = 2.5;

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const saving = ref(false);
const errorMessage = ref('');
const users = ref([]);
const groups = ref([]);
const accounts = ref([]);
const form = reactive(emptyForm());
const userLookup = ref(null);
const userLookupStageIdx = ref(null);

const policyId = computed(() => {
	const raw = route.params.id;
	if (!raw || raw === 'new') return 0;
	const n = Number(raw);
	return Number.isFinite(n) && n > 0 ? n : 0;
});
const isEdit = computed(() => policyId.value > 0);

const socialAccountOptions = computed(() =>
	accounts.value.map((a) => ({
		value: a.id,
		label: `${a.identity} (${a.connector})`,
	})),
);

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

function stageUserLabel(stage) {
	if (!stage.userId) return 'No user selected';
	const u = users.value.find((x) => x.id === stage.userId);
	return u?.username || `User #${stage.userId}`;
}

function openStageUserLookup(idx) {
	if (saving.value) return;
	userLookupStageIdx.value = idx;
	userLookup.value?.open();
}

function onStageUserPicked(picked) {
	const idx = userLookupStageIdx.value;
	userLookupStageIdx.value = null;
	if (idx == null || !picked?.length) return;

	const user = picked[0];
	form.stages[idx].userId = user.id;
	if (!users.value.some((u) => u.id === user.id)) {
		users.value.push(user);
	}
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
.stage-user-picker {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-wrap: wrap;
}
.stage-user-label {
	min-width: 6rem;
}
.stage-user-label.muted {
	opacity: 0.75;
}
</style>
