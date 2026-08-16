<template>
	<dialog ref="createDialog" class="dialog" @close="closeCreateDialog">
		<h2>Add webhook</h2>
		<p>Add an HTTP endpoint that receives signed JSON when subscribed events occur.</p>

		<FormLayout @submit.prevent="submitCreate">
			<FormField label="URL" for="create-webhook-url" :disabled="saving">
				<input
					id="create-webhook-url"
					ref="createUrlInput"
					v-model="createForm.url"
					type="url"
					required
					placeholder="https://example.com/webhooks/japella"
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Secret" for="create-webhook-secret" :disabled="saving">
				<input
					id="create-webhook-secret"
					v-model="createForm.secret"
					type="password"
					required
					autocomplete="off"
					:disabled="saving"
				/>
			</FormField>
			<FormField
				label="Events"
				fake
				description="Select one or more events that should POST to this URL."
				:disabled="saving"
			>
				<CheckGroup
					v-model="createForm.events"
					name="create-webhook-events"
					:options="eventOptions"
					:disabled="saving"
					aria-label="Webhook events"
				/>
			</FormField>
			<FormField label="Enabled" fake :disabled="saving">
				<RadioGroup
					v-model="createForm.enabled"
					name="create-webhook-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<p v-if="createError" class="inline-notification error">{{ createError }}</p>

			<template #actions>
				<button type="button" class="neutral" :disabled="saving" @click="closeCreateDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Creating…' : 'Create webhook' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<dialog ref="editDialog" class="dialog" @close="closeEditDialog">
		<h2>Edit webhook</h2>
		<p>Update the destination URL, subscribed events, and enabled state. Leave secret blank to keep the current value.</p>

		<FormLayout @submit.prevent="saveEdit">
			<FormField label="URL" for="edit-webhook-url" :disabled="saving">
				<input
					id="edit-webhook-url"
					v-model="editForm.url"
					type="url"
					required
					:disabled="saving"
				/>
			</FormField>
			<FormField label="Secret" for="edit-webhook-secret" :disabled="saving">
				<input
					id="edit-webhook-secret"
					v-model="editForm.secret"
					type="password"
					autocomplete="off"
					placeholder="Leave blank to keep current secret"
					:disabled="saving"
				/>
			</FormField>
			<FormField
				label="Events"
				fake
				description="Select one or more events that should POST to this URL."
				:disabled="saving"
			>
				<CheckGroup
					v-model="editForm.events"
					name="edit-webhook-events"
					:options="eventOptions"
					:disabled="saving"
					aria-label="Webhook events"
				/>
			</FormField>
			<FormField label="Enabled" fake :disabled="saving">
				<RadioGroup
					v-model="editForm.enabled"
					name="edit-webhook-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<p v-if="editError" class="inline-notification error">{{ editError }}</p>

			<template #actions>
				<button type="button" class="bad" :disabled="saving" @click="deleteWebhook">Delete</button>
				<span class="dialog-actions-spacer"></span>
				<button type="button" class="neutral" :disabled="saving" @click="closeEditDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		subtitle="Configure HTTP endpoints that receive signed JSON when app events occur."
		classes="webhooks-admin settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="WebhookIcon" width="22" height="22" aria-hidden="true" />
				Event webhooks
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
				aria-label="Add webhook"
				title="Add webhook"
				:disabled="loading || saving"
				@click="openCreateDialog"
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
		<div v-if="loading && !webhooks.length" class="list-banner-pad muted">Loading…</div>

		<template v-else>
			<p v-if="!canManage" class="inline-notification note list-banner-pad">
				You can view webhooks. Editing requires the system.settings permission.
			</p>
			<p v-if="!webhooks.length" class="inline-notification note list-banner-pad">No webhooks configured yet.</p>

			<div v-else class="webhooks-table-wrap">
				<Table
					:data="tableRows"
					:headers="tableHeaders"
				>
					<template #cell-url="{ value }">
						<span class="url-cell">{{ value }}</span>
					</template>
					<template #cell-eventsLabel="{ value }">
						{{ value || '—' }}
					</template>
					<template #cell-status="{ value }">
						<span :class="value === 'Enabled' ? 'fg-good' : 'muted'">{{ value }}</span>
					</template>
					<template #cell-actions="{ row }">
						<div class="actions-cell">
							<button type="button" class="small" :disabled="saving" @click="openEdit(row)">Edit</button>
						</div>
					</template>
				</Table>
			</div>
		</template>
	</Section>

	<Section
		subtitle="Incoming message webhooks configured per chat bot. Use Manage to edit hooks on a bot's details page."
		classes="webhooks-admin settings-users"
		:padding="botWebhooks.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="Robot01Icon" width="22" height="22" aria-hidden="true" />
				Bot webhooks
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
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<div v-if="botErrorMessage" class="inline-notification error" :class="{ 'list-banner-pad': botWebhooks.length > 0 }">{{ botErrorMessage }}</div>
		<div v-if="loading && !botWebhooks.length" class="muted">Loading…</div>

		<template v-else>
			<p v-if="!botWebhooks.length" class="inline-notification note">
				No bot message hooks configured yet.
			</p>

			<div v-else class="webhooks-table-wrap">
				<Table
					:data="botTableRows"
					:headers="botTableHeaders"
				>
					<template #cell-botName="{ value }">
						{{ value || '—' }}
					</template>
					<template #cell-protocol="{ value }">
						{{ value || '—' }}
					</template>
					<template #cell-url="{ value }">
						<span class="url-cell">{{ value }}</span>
					</template>
					<template #cell-status="{ value }">
						<span :class="value === 'Enabled' ? 'fg-good' : 'muted'">{{ value }}</span>
					</template>
					<template #cell-actions="{ row }">
						<div class="actions-cell">
							<router-link
								v-if="row.manageTo"
								:to="row.manageTo"
								class="small"
							>
								Manage
							</router-link>
							<span v-else class="muted">—</span>
						</div>
					</template>
				</Table>
			</div>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Add01Icon, RefreshIcon, Robot01Icon, WebhookIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;

	const webhooks = ref([]);
	const botWebhooks = ref([]);
	const catalog = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const botErrorMessage = ref('');
	const createError = ref('');
	const editError = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const editingId = ref(0);
	const createDialog = ref(null);
	const editDialog = ref(null);
	const createUrlInput = ref(null);
	const createForm = ref({
		url: '',
		secret: '',
		events: [],
		enabled: true,
	});
	const editForm = ref({
		url: '',
		secret: '',
		events: [],
		enabled: true,
	});

	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.settings')),
	);

	const enabledOptions = [
		{ label: 'On', value: true },
		{ label: 'Off', value: false },
	];

	const tableHeaders = computed(() => {
		const headers = [
			{ key: 'id', label: 'ID', sortable: true, width: '4rem' },
			{ key: 'url', label: 'URL', sortable: true },
			{ key: 'eventsLabel', label: 'Events', sortable: true },
			{ key: 'status', label: 'Status', sortable: true, width: '6rem' },
		];
		if (canManage.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	const botTableHeaders = [
		{ key: 'botName', label: 'Bot', sortable: true },
		{ key: 'protocol', label: 'Protocol', sortable: true, width: '8rem' },
		{ key: 'url', label: 'URL', sortable: true },
		{ key: 'status', label: 'Status', sortable: true, width: '6rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: '6rem' },
	];

	const eventOptions = computed(() =>
		catalog.value.map((e) => ({ label: e, value: e })),
	);

	const tableRows = computed(() =>
		webhooks.value.map((wh) => ({
			id: wh.id,
			url: wh.url,
			eventsLabel: (wh.events || []).join(', '),
			status: wh.enabled ? 'Enabled' : 'Disabled',
			events: wh.events || [],
			enabled: wh.enabled,
		})),
	);

	const botTableRows = computed(() =>
		botWebhooks.value.map((wh) => {
			const protocol = String(wh.protocol || wh.connector || '').trim();
			const botId = String(wh.botId || wh.bot_id || wh.identity || '').trim();
			return {
				id: wh.id,
				botName: wh.botName || wh.bot_name || botId,
				protocol,
				botId,
				url: wh.url,
				status: wh.enabled ? 'Enabled' : 'Disabled',
				enabled: wh.enabled,
				manageTo: protocol && botId
					? { name: 'chatBotHooks', params: { protocol, botId } }
					: null,
			};
		}),
	);

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
		catalog.value = st.webhookEvents || [];
	}

	async function loadAll() {
		errorMessage.value = '';
		botErrorMessage.value = '';
		loading.value = true;
		try {
			await waitForClient();
			await refreshStatus();
			const [webRes, botRes] = await Promise.allSettled([
				window.client.listWebhooks({}),
				window.client.listBotWebhooks({}),
			]);
			if (webRes.status === 'fulfilled') {
				webhooks.value = webRes.value.webhooks || [];
				if (!catalog.value.length && (webRes.value.events || []).length) {
					catalog.value = webRes.value.events;
				}
			} else {
				console.error(webRes.reason);
				errorMessage.value = webRes.reason?.message || 'Failed to load event webhooks.';
			}
			if (botRes.status === 'fulfilled') {
				botWebhooks.value = botRes.value.webhooks || [];
			} else {
				console.error(botRes.reason);
				botErrorMessage.value = botRes.reason?.message || 'Failed to load bot webhooks.';
			}
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load webhooks.';
		} finally {
			loading.value = false;
		}
	}

	function openCreateDialog() {
		createForm.value = {
			url: '',
			secret: '',
			events: [],
			enabled: true,
		};
		createError.value = '';
		createDialog.value?.showModal();
		nextTick(() => createUrlInput.value?.focus());
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
			await window.client.createWebhook({
				url: createForm.value.url.trim(),
				secret: createForm.value.secret,
				events: createForm.value.events,
				enabled: createForm.value.enabled,
			});
			closeCreateDialog();
			await loadAll();
		} catch (e) {
			createError.value = e.message || 'Failed to create webhook.';
		} finally {
			saving.value = false;
		}
	}

	function openEdit(row) {
		editingId.value = row.id;
		editForm.value = {
			url: row.url,
			secret: '',
			events: [...(row.events || [])],
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
			await window.client.updateWebhook({
				id: editingId.value,
				url: editForm.value.url.trim(),
				secret: editForm.value.secret.trim(),
				events: editForm.value.events,
				enabled: editForm.value.enabled,
			});
			closeEditDialog();
			await loadAll();
		} catch (e) {
			editError.value = e.message || 'Failed to update webhook.';
		} finally {
			saving.value = false;
		}
	}

	async function deleteWebhook() {
		if (!editingId.value || !confirm('Delete this webhook target?')) return;
		saving.value = true;
		editError.value = '';
		try {
			await waitForClient();
			await window.client.deleteWebhook({ id: editingId.value });
			closeEditDialog();
			await loadAll();
		} catch (e) {
			editError.value = e.message || 'Failed to delete webhook.';
		} finally {
			saving.value = false;
		}
	}

	onMounted(() => {
		loadAll();
	});
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

	.webhooks-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.url-cell {
		word-break: break-all;
	}

	.actions-cell {
		text-align: right;
	}

	.dialog-actions-spacer {
		flex: 1;
	}

	.muted {
		opacity: 0.7;
	}
</style>
