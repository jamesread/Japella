<template>
	<dialog ref="createDialog" class="dialog" @close="closeCreateDialog">
		<h2>Add LLM tool</h2>
		<p>Tools use a connection and a system prompt. The draft post content is sent as the user message.</p>

		<FormLayout @submit.prevent="submitCreate">
			<FormField label="Name" for="create-llm-tool-name" :disabled="saving">
				<input
					id="create-llm-tool-name"
					ref="createNameInput"
					v-model="createForm.name"
					type="text"
					required
					placeholder="Sentiment Analysis"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Description" for="create-llm-tool-desc" :disabled="saving">
				<input
					id="create-llm-tool-desc"
					v-model="createForm.description"
					type="text"
					placeholder="Check tone and sentiment of the draft"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Connection" for="create-llm-tool-conn" :disabled="saving">
				<select id="create-llm-tool-conn" v-model="createForm.llmConnectionId" required :disabled="saving">
					<option disabled value="0">Select a connection…</option>
					<option v-for="c in connections" :key="c.id" :value="c.id">{{ c.name }} ({{ c.model }})</option>
				</select>
			</FormField>
			<FormField label="System prompt" for="create-llm-tool-prompt" :disabled="saving">
				<textarea
					id="create-llm-tool-prompt"
					v-model="createForm.prompt"
					rows="6"
					required
					placeholder="You are a sentiment analyst. Review the social media post and summarize sentiment…"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Sort order" for="create-llm-tool-sort" :disabled="saving">
				<input id="create-llm-tool-sort" v-model.number="createForm.sortOrder" type="number" :disabled="saving" />
			</FormField>
			<FormField label="Enabled" component-has-label :disabled="saving">
				<RadioGroup
					v-model="createForm.enabled"
					name="create-llm-tool-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="createError" type="error" :message="createError" />

			<template #actions>
				<button type="button" class="neutral" :disabled="saving" @click="closeCreateDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving || !connections.length">
					{{ saving ? 'Creating…' : 'Create tool' }}
				</button>
			</template>
		</FormLayout>
	</dialog>

	<dialog ref="editDialog" class="dialog" @close="closeEditDialog">
		<h2>Edit LLM tool</h2>

		<FormLayout @submit.prevent="saveEdit">
			<FormField label="Name" for="edit-llm-tool-name" :disabled="saving">
				<input id="edit-llm-tool-name" v-model="editForm.name" type="text" required :disabled="saving" />
			</FormField>
			<FormField label="Description" for="edit-llm-tool-desc" :disabled="saving">
				<input id="edit-llm-tool-desc" v-model="editForm.description" type="text" :disabled="saving" />
			</FormField>
			<FormField label="Connection" for="edit-llm-tool-conn" :disabled="saving">
				<select id="edit-llm-tool-conn" v-model="editForm.llmConnectionId" required :disabled="saving">
					<option disabled value="0">Select a connection…</option>
					<option v-for="c in connections" :key="c.id" :value="c.id">{{ c.name }} ({{ c.model }})</option>
				</select>
			</FormField>
			<FormField label="System prompt" for="edit-llm-tool-prompt" :disabled="saving">
				<textarea id="edit-llm-tool-prompt" v-model="editForm.prompt" rows="6" required :disabled="saving" />
			</FormField>
			<FormField label="Sort order" for="edit-llm-tool-sort" :disabled="saving">
				<input id="edit-llm-tool-sort" v-model.number="editForm.sortOrder" type="number" :disabled="saving" />
			</FormField>
			<FormField label="Enabled" component-has-label :disabled="saving">
				<RadioGroup
					v-model="editForm.enabled"
					name="edit-llm-tool-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="editError" type="error" :message="editError" />

			<template #actions>
				<button type="button" class="bad" :disabled="saving" @click="deleteTool">Delete</button>
				<span class="dialog-actions-spacer"></span>
				<button type="button" class="neutral" :disabled="saving" @click="closeEditDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		subtitle="Named prompts that authors can run against draft posts while composing."
		classes="llm-tools-admin settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="AiChat02Icon" width="22" height="22" aria-hidden="true" />
				LLM tools
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
				aria-label="Add tool"
				title="Add tool"
				:disabled="loading || saving || !connections.length"
				@click="openCreateDialog"
			>
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="iconStrokeWidth" aria-hidden="true" />
			</button>
		</template>

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />
		<NotificationBlock
			v-if="!connections.length && !loading"
			type="note"
			message="Create an LLM connection before adding tools."
		/>
		<NotificationBlock
			v-if="!canManage"
			type="note"
			message="You can view LLM tools. Editing requires the system.settings permission."
		/>

		<div class="llm-table-wrap">
			<Table :data="tableRows" :headers="tableHeaders" :loading="loading">
				<template #empty>
					No LLM tools configured yet.
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
	import { Add01Icon, AiChat02Icon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;

	const tools = ref([]);
	const connections = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const createError = ref('');
	const editError = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const editingId = ref(0);
	const createDialog = ref(null);
	const editDialog = ref(null);
	const createNameInput = ref(null);
	const createForm = ref({
		name: '',
		description: '',
		llmConnectionId: 0,
		prompt: '',
		enabled: true,
		sortOrder: 0,
	});
	const editForm = ref({
		name: '',
		description: '',
		llmConnectionId: 0,
		prompt: '',
		enabled: true,
		sortOrder: 0,
	});

	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.settings')),
	);

	const enabledOptions = [
		{ label: 'On', value: true },
		{ label: 'Off', value: false },
	];

	const tableHeaders = computed(() => [
		{ key: 'id', label: 'ID', sortable: true, width: '4rem' },
		{ key: 'name', label: 'Name', sortable: true },
		{ key: 'connectionName', label: 'Connection', sortable: true },
		{ key: 'description', label: 'Description', sortable: true },
		{ key: 'sortOrder', label: 'Order', sortable: true, width: '5rem' },
		{ key: 'status', label: 'Status', sortable: true, width: '6rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: canManage.value ? '6rem' : '1rem' },
	]);

	const tableRows = computed(() =>
		tools.value.map((t) => ({
			id: t.id,
			name: t.name,
			description: t.description || '',
			connectionName: t.connectionName || `Connection #${t.llmConnectionId}`,
			llmConnectionId: t.llmConnectionId,
			prompt: t.prompt,
			sortOrder: t.sortOrder ?? 0,
			status: t.enabled ? 'Enabled' : 'Disabled',
			enabled: t.enabled,
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
			const [toolsRes, connRes] = await Promise.all([
				window.client.listLlmTools({}),
				window.client.listLlmConnections({}),
			]);
			tools.value = toolsRes.tools || [];
			connections.value = connRes.connections || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load LLM tools.';
		} finally {
			loading.value = false;
		}
	}

	function openCreateDialog() {
		createForm.value = {
			name: '',
			description: '',
			llmConnectionId: connections.value[0]?.id || 0,
			prompt: '',
			enabled: true,
			sortOrder: 0,
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
			await window.client.createLlmTool({
				name: createForm.value.name.trim(),
				description: createForm.value.description.trim(),
				llmConnectionId: Number(createForm.value.llmConnectionId),
				prompt: createForm.value.prompt.trim(),
				enabled: createForm.value.enabled,
				sortOrder: Number(createForm.value.sortOrder) || 0,
			});
			closeCreateDialog();
			await loadAll();
		} catch (e) {
			createError.value = e.message || 'Failed to create tool.';
		} finally {
			saving.value = false;
		}
	}

	function openEdit(row) {
		editingId.value = row.id;
		editForm.value = {
			name: row.name,
			description: row.description,
			llmConnectionId: row.llmConnectionId,
			prompt: row.prompt,
			enabled: row.enabled,
			sortOrder: row.sortOrder,
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
			await window.client.updateLlmTool({
				id: editingId.value,
				name: editForm.value.name.trim(),
				description: editForm.value.description.trim(),
				llmConnectionId: Number(editForm.value.llmConnectionId),
				prompt: editForm.value.prompt.trim(),
				enabled: editForm.value.enabled,
				sortOrder: Number(editForm.value.sortOrder) || 0,
			});
			closeEditDialog();
			await loadAll();
		} catch (e) {
			editError.value = e.message || 'Failed to save tool.';
		} finally {
			saving.value = false;
		}
	}

	async function deleteTool() {
		if (!editingId.value) return;
		if (!confirm('Delete this LLM tool? Prior run history is kept.')) return;
		saving.value = true;
		editError.value = '';
		try {
			await waitForClient();
			await window.client.deleteLlmTool({ id: editingId.value });
			closeEditDialog();
			await loadAll();
		} catch (e) {
			editError.value = e.message || 'Failed to delete tool.';
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

	.dialog-actions-spacer {
		flex: 1;
	}

	.muted {
		opacity: 0.7;
	}
</style>
