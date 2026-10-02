<template>
	<dialog ref="createDialog" class="dialog" @close="closeCreateDialog">
		<h2>Add LLM connection</h2>
		<p>Configure an OpenAI-compatible Chat Completions endpoint (base URL, API key, and model).</p>

		<FormLayout @submit.prevent="submitCreate">
			<FormField label="Preset" for="create-llm-conn-preset" :disabled="saving">
				<select
					id="create-llm-conn-preset"
					v-model="createPresetId"
					:disabled="saving"
					@change="applyCreatePreset"
				>
					<option value="">Custom / blank</option>
					<option v-for="p in presets" :key="p.id" :value="p.id">{{ p.name }}</option>
				</select>
				<p v-if="selectedCreatePreset?.description" class="field-hint">{{ selectedCreatePreset.description }}</p>
			</FormField>
			<FormField label="Name" for="create-llm-conn-name" :disabled="saving">
				<input
					id="create-llm-conn-name"
					ref="createNameInput"
					v-model="createForm.name"
					type="text"
					required
					placeholder="ChatGPT 5.3"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Base URL" for="create-llm-conn-url" :disabled="saving">
				<input
					id="create-llm-conn-url"
					v-model="createForm.baseUrl"
					type="url"
					required
					placeholder="https://openrouter.ai/api/v1"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="API key" for="create-llm-conn-key" :disabled="saving">
				<input
					id="create-llm-conn-key"
					v-model="createForm.apiKey"
					type="password"
					autocomplete="off"
					placeholder="Optional for local providers"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Model" for="create-llm-conn-model" :disabled="saving">
				<div class="model-field">
					<input
						id="create-llm-conn-model"
						v-model="createForm.model"
						type="text"
						list="create-llm-model-options"
						required
						placeholder="openai/gpt-4o-mini"
						:disabled="saving"
					/>
					<button
						type="button"
						class="neutral"
						:disabled="saving || fetchingCreateModels || !createForm.baseUrl"
						@click="fetchCreateModels"
					>
						{{ fetchingCreateModels ? 'Fetching…' : 'Fetch models' }}
					</button>
				</div>
				<datalist id="create-llm-model-options">
					<option v-for="m in createModels" :key="m.id" :value="m.id">{{ m.name }}</option>
				</datalist>
				<p v-if="createModels.length" class="field-hint">{{ createModels.length }} models loaded — pick from the list or type an id.</p>
			</FormField>
			<FormField label="Enabled" component-has-label :disabled="saving">
				<RadioGroup
					v-model="createForm.enabled"
					name="create-llm-conn-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="createError" type="error" :message="createError" />

			<template #actions>
				<button type="button" class="neutral" :disabled="saving" @click="closeCreateDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Creating…' : 'Create connection' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<dialog ref="editDialog" class="dialog" @close="closeEditDialog">
		<h2>Edit LLM connection</h2>
		<p>Leave API key blank to keep the current value.</p>

		<FormLayout @submit.prevent="saveEdit">
			<FormField label="Name" for="edit-llm-conn-name" :disabled="saving">
				<input id="edit-llm-conn-name" v-model="editForm.name" type="text" required :disabled="saving" />
			</FormField>
			<FormField label="Base URL" for="edit-llm-conn-url" :disabled="saving">
				<input id="edit-llm-conn-url" v-model="editForm.baseUrl" type="url" required :disabled="saving" />
			</FormField>
			<FormField label="API key" for="edit-llm-conn-key" :disabled="saving">
				<input
					id="edit-llm-conn-key"
					v-model="editForm.apiKey"
					type="password"
					autocomplete="off"
					:placeholder="editForm.apiKeySet ? 'Leave blank to keep current key' : 'Optional'"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Model" for="edit-llm-conn-model" :disabled="saving">
				<div class="model-field">
					<input
						id="edit-llm-conn-model"
						v-model="editForm.model"
						type="text"
						list="edit-llm-model-options"
						required
						:disabled="saving"
					/>
					<button
						type="button"
						class="neutral"
						:disabled="saving || fetchingEditModels || !editForm.baseUrl"
						@click="fetchEditModels"
					>
						{{ fetchingEditModels ? 'Fetching…' : 'Fetch models' }}
					</button>
				</div>
				<datalist id="edit-llm-model-options">
					<option v-for="m in editModels" :key="m.id" :value="m.id">{{ m.name }}</option>
				</datalist>
				<p v-if="editModels.length" class="field-hint">{{ editModels.length }} models loaded — pick from the list or type an id.</p>
			</FormField>
			<FormField label="Enabled" component-has-label :disabled="saving">
				<RadioGroup
					v-model="editForm.enabled"
					name="edit-llm-conn-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="editError" type="error" :message="editError" />

			<template #actions>
				<button type="button" class="bad" :disabled="saving || testingId === editingId" @click="deleteConnection">Delete</button>
				<button
					type="button"
					class="neutral"
					:disabled="saving || testingId === editingId"
					@click="sendTest(editingId)"
				>
					{{ testingId === editingId ? 'Testing…' : 'Test' }}
				</button>
				<span class="dialog-actions-spacer"></span>
				<button type="button" class="neutral" :disabled="saving" @click="closeEditDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		subtitle="OpenAI-compatible endpoints used by LLM compose tools."
		classes="llm-connections-admin settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="AiBrain01Icon" width="22" height="22" aria-hidden="true" />
				LLM connections
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading || saving"
				@click="loadAll"
			>
				<HugeiconsIcon :icon="RefreshIcon" width="1em" height="1em" :strokeWidth="iconStrokeWidth" aria-hidden="true" />
			</button>
			<button
				v-if="canManage"
				type="button"
				class="inline-icon good"
				aria-label="Add connection"
				title="Add connection"
				:disabled="loading || saving"
				@click="openCreateDialog"
			>
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="iconStrokeWidth" aria-hidden="true" />
			</button>
		</template>

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />
		<NotificationBlock v-if="testMessage" :type="testMessageType" :message="testMessage" />
		<NotificationBlock
			v-if="!canManage"
			type="note"
			message="You can view LLM connections. Editing requires the system.settings permission."
		/>

		<div class="llm-table-wrap">
			<Table :data="tableRows" :headers="tableHeaders" :loading="loading">
				<template #empty>
					No LLM connections configured yet.
				</template>
				<template #cell-status="{ value }">
					<span :class="value === 'Enabled' ? 'fg-good' : 'muted'">{{ value }}</span>
				</template>
				<template #cell-actions="{ row }">
					<div class="actions-cell">
						<button
							v-if="canManage"
							type="button"
							class="small"
							:disabled="saving || testingId === row.id"
							@click="sendTest(row.id)"
						>
							{{ testingId === row.id ? 'Testing…' : 'Test' }}
						</button>
						<button
							v-if="canManage"
							type="button"
							class="small"
							:disabled="saving"
							@click="openEdit(row)"
						>
							Edit
						</button>
					</div>
				</template>
			</Table>
		</div>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Add01Icon, AiBrain01Icon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;

	const connections = ref([]);
	const presets = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const createError = ref('');
	const editError = ref('');
	const testMessage = ref('');
	const testMessageType = ref('good');
	const testingId = ref(0);
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const editingId = ref(0);
	const createDialog = ref(null);
	const editDialog = ref(null);
	const createNameInput = ref(null);
	const createPresetId = ref('');
	const createModels = ref([]);
	const editModels = ref([]);
	const fetchingCreateModels = ref(false);
	const fetchingEditModels = ref(false);
	const createForm = ref({
		name: '',
		baseUrl: '',
		apiKey: '',
		model: '',
		enabled: true,
	});
	const editForm = ref({
		name: '',
		baseUrl: '',
		apiKey: '',
		apiKeySet: false,
		model: '',
		enabled: true,
	});

	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.settings')),
	);

	const selectedCreatePreset = computed(() =>
		presets.value.find((p) => p.id === createPresetId.value) || null,
	);

	const enabledOptions = [
		{ label: 'On', value: true },
		{ label: 'Off', value: false },
	];

	const tableHeaders = computed(() => [
		{ key: 'id', label: 'ID', sortable: true, width: '4rem' },
		{ key: 'name', label: 'Name', sortable: true },
		{ key: 'baseUrl', label: 'Base URL', sortable: true },
		{ key: 'model', label: 'Model', sortable: true },
		{ key: 'status', label: 'Status', sortable: true, width: '6rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: canManage.value ? '10rem' : '1rem' },
	]);

	const tableRows = computed(() =>
		connections.value.map((c) => ({
			id: c.id,
			name: c.name,
			baseUrl: c.baseUrl,
			model: c.model,
			apiKeySet: Boolean(c.apiKeySet),
			status: c.enabled ? 'Enabled' : 'Disabled',
			enabled: c.enabled,
		})),
	);

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function loadAll() {
		errorMessage.value = '';
		loading.value = true;
		try {
			await waitForClient();
			await refreshStatus();
			const [connRes, presetRes] = await Promise.all([
				window.client.listLlmConnections({}),
				window.client.listLlmConnectionPresets({}),
			]);
			connections.value = connRes.connections || [];
			presets.value = presetRes.presets || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load LLM connections.';
		} finally {
			loading.value = false;
		}
	}

	function applyCreatePreset() {
		const preset = selectedCreatePreset.value;
		createModels.value = [];
		if (!preset) {
			return;
		}
		if (!createForm.value.name.trim()) {
			createForm.value.name = preset.name;
		}
		createForm.value.baseUrl = preset.baseUrl || '';
		if (!createForm.value.model.trim()) {
			createForm.value.model = preset.suggestedModel || '';
		}
	}

	async function fetchCreateModels() {
		fetchingCreateModels.value = true;
		createError.value = '';
		try {
			await waitForClient();
			const res = await window.client.listLlmConnectionModels({
				baseUrl: createForm.value.baseUrl.trim(),
				apiKey: createForm.value.apiKey,
			});
			createModels.value = res.models || [];
			if (!createModels.value.length) {
				createError.value = 'No models returned for this endpoint.';
			} else if (!createForm.value.model.trim()) {
				createForm.value.model = createModels.value[0].id;
			}
		} catch (e) {
			createModels.value = [];
			createError.value = e.message || 'Failed to fetch models.';
		} finally {
			fetchingCreateModels.value = false;
		}
	}

	async function fetchEditModels() {
		fetchingEditModels.value = true;
		editError.value = '';
		try {
			await waitForClient();
			const res = await window.client.listLlmConnectionModels({
				connectionId: editingId.value || 0,
				baseUrl: editForm.value.baseUrl.trim(),
				apiKey: editForm.value.apiKey,
			});
			editModels.value = res.models || [];
			if (!editModels.value.length) {
				editError.value = 'No models returned for this endpoint.';
			}
		} catch (e) {
			editModels.value = [];
			editError.value = e.message || 'Failed to fetch models.';
		} finally {
			fetchingEditModels.value = false;
		}
	}

	function openCreateDialog() {
		createPresetId.value = '';
		createModels.value = [];
		createForm.value = {
			name: '',
			baseUrl: '',
			apiKey: '',
			model: '',
			enabled: true,
		};
		createError.value = '';
		createDialog.value?.showModal();
		nextTick(() => createNameInput.value?.focus());
	}

	function closeCreateDialog() {
		createDialog.value?.close();
		createError.value = '';
	}

	async function submitCreate() {
		saving.value = true;
		createError.value = '';
		try {
			await waitForClient();
			await window.client.createLlmConnection({
				name: createForm.value.name.trim(),
				baseUrl: createForm.value.baseUrl.trim(),
				apiKey: createForm.value.apiKey,
				model: createForm.value.model.trim(),
				enabled: createForm.value.enabled,
			});
			closeCreateDialog();
			await loadAll();
		} catch (e) {
			createError.value = e.message || 'Failed to create connection.';
		} finally {
			saving.value = false;
		}
	}

	function openEdit(row) {
		editingId.value = row.id;
		editModels.value = [];
		editForm.value = {
			name: row.name,
			baseUrl: row.baseUrl,
			apiKey: '',
			apiKeySet: row.apiKeySet,
			model: row.model,
			enabled: row.enabled,
		};
		editError.value = '';
		editDialog.value?.showModal();
	}

	function closeEditDialog() {
		editDialog.value?.close();
		editingId.value = 0;
		editError.value = '';
	}

	async function saveEdit() {
		if (!editingId.value) return;
		saving.value = true;
		editError.value = '';
		try {
			await waitForClient();
			await window.client.updateLlmConnection({
				id: editingId.value,
				name: editForm.value.name.trim(),
				baseUrl: editForm.value.baseUrl.trim(),
				apiKey: editForm.value.apiKey,
				model: editForm.value.model.trim(),
				enabled: editForm.value.enabled,
			});
			closeEditDialog();
			await loadAll();
		} catch (e) {
			editError.value = e.message || 'Failed to save connection.';
		} finally {
			saving.value = false;
		}
	}

	async function deleteConnection() {
		if (!editingId.value) return;
		if (!confirm('Delete this LLM connection? Tools that use it must be removed first.')) return;
		saving.value = true;
		editError.value = '';
		try {
			await waitForClient();
			await window.client.deleteLlmConnection({ id: editingId.value });
			closeEditDialog();
			await loadAll();
		} catch (e) {
			editError.value = e.message || 'Failed to delete connection.';
		} finally {
			saving.value = false;
		}
	}

	async function sendTest(id) {
		if (!id) return;
		testingId.value = id;
		testMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.testLlmConnection({ id });
			testMessageType.value = res.success ? 'good' : 'error';
			testMessage.value = res.message || (res.success ? 'Connection OK' : 'Test failed');
		} catch (e) {
			testMessageType.value = 'error';
			testMessage.value = e.message || 'Test failed.';
		} finally {
			testingId.value = 0;
		}
	}

	onMounted(loadAll);
</script>

<style scoped>
	.section-title-with-icon {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
	}

	.llm-table-wrap {
		overflow-x: auto;
	}

	.actions-cell {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
	}

	.model-field {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
	}

	.model-field input {
		flex: 1 1 12rem;
		min-width: 0;
	}

	.field-hint {
		margin: 0.35rem 0 0;
		font-size: 0.9em;
		opacity: 0.75;
	}

	.dialog-actions-spacer {
		flex: 1;
	}

	.muted {
		opacity: 0.7;
	}
</style>
