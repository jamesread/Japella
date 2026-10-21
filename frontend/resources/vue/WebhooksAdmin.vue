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
				component-has-label
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
			<FormField label="Enabled" component-has-label :disabled="saving">
				<RadioGroup
					v-model="createForm.enabled"
					name="create-webhook-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="createError" type="error" :message="createError" />

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
				component-has-label
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
			<FormField label="Enabled" component-has-label :disabled="saving">
				<RadioGroup
					v-model="editForm.enabled"
					name="edit-webhook-enabled"
					variant="boolean"
					:options="enabledOptions"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="editError" type="error" :message="editError" />

			<template #actions>
				<button type="button" class="bad" :disabled="saving || testingId === editingId" @click="deleteWebhook">Delete</button>
				<button
					type="button"
					class="neutral"
					:disabled="saving || testingId === editingId"
					@click="sendTest({ id: editingId })"
				>
					{{ testingId === editingId ? 'Sending…' : 'Send test' }}
				</button>
				<span class="dialog-actions-spacer"></span>
				<button type="button" class="neutral" :disabled="saving" @click="closeEditDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		title="Event webhooks"
		subtitle="Configure HTTP endpoints that receive signed JSON when app events occur."
		:icon="WebhookIcon"
		classes="webhooks-admin settings-users"
		:padding="false"
	>

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

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />
		<NotificationBlock v-if="testMessage" :type="testMessageType" :message="testMessage" />
		<NotificationBlock
			v-if="!canManage"
			type="note"
			message="You can view webhooks. Editing requires the system.settings permission."
		/>

		<div class="webhooks-table-wrap">
			<Table
				:data="tableRows"
				:headers="tableHeaders"
				:loading="loading"
			>
				<template #empty>
					No webhooks configured yet.
				</template>
				<template #cell-url="{ row, value }">
					<router-link
						:to="{ name: 'webhookDetails', params: { id: row.id } }"
						class="url-cell"
					>
						{{ value }}
					</router-link>
				</template>
				<template #cell-eventsLabel="{ value }">
					{{ value || '—' }}
				</template>
				<template #cell-status="{ value }">
					<span :class="value === 'Enabled' ? 'fg-good' : 'muted'">{{ value }}</span>
				</template>
				<template #cell-actions="{ row }">
					<div class="actions-cell">
						<router-link
							:to="{ name: 'webhookDetails', params: { id: row.id } }"
							class="small"
						>
							View
						</router-link>
						<button
							v-if="canManage"
							type="button"
							class="small"
							:disabled="saving || testingId === row.id"
							@click="sendTest(row)"
						>
							{{ testingId === row.id ? 'Sending…' : 'Send test' }}
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

	<Section
		title="Bot webhooks"
		subtitle="Incoming message webhooks configured per chat bot. Use Manage to edit hooks on a bot's details page."
		:icon="Robot01Icon"
		classes="webhooks-admin settings-users"
		:padding="botWebhooks.length === 0"
	>

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

		<NotificationBlock v-if="botErrorMessage" type="error" :message="botErrorMessage" />

		<div class="webhooks-table-wrap">
			<Table
				:data="botTableRows"
				:headers="botTableHeaders"
				:loading="loading"
			>
				<template #empty>
					No bot message hooks configured yet.
				</template>
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
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Add01Icon, RefreshIcon, Robot01Icon, WebhookIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;
	const router = useRouter();

	const webhooks = ref([]);
	const botWebhooks = ref([]);
	const catalog = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const botErrorMessage = ref('');
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
			{
				key: 'actions',
				label: 'Actions',
				sortable: false,
				width: canManage.value ? '14rem' : '5rem',
			},
		];
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
			const created = await window.client.createWebhook({
				url: createForm.value.url.trim(),
				secret: createForm.value.secret,
				events: createForm.value.events,
				enabled: createForm.value.enabled,
			});
			closeCreateDialog();
			if (created?.id) {
				router.push({ name: 'webhookDetails', params: { id: created.id } });
				return;
			}
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

	async function sendTest(row) {
		const id = row?.id;
		if (!id || testingId.value) return;
		testingId.value = id;
		testMessage.value = '';
		editError.value = '';
		try {
			await waitForClient();
			const res = await window.client.testWebhook({ id });
			const msg = res.message || (res.success ? 'Test webhook delivered.' : 'Test webhook failed.');
			testMessage.value = `Webhook #${id}: ${msg}`;
			testMessageType.value = res.success ? 'good' : 'error';
			if (editingId.value === id && !res.success) {
				editError.value = msg;
			}
		} catch (e) {
			const msg = e.message || 'Failed to send test webhook.';
			testMessage.value = `Webhook #${id}: ${msg}`;
			testMessageType.value = 'error';
			if (editingId.value === id) {
				editError.value = msg;
			}
		} finally {
			testingId.value = 0;
		}
	}

	onMounted(() => {
		loadAll();
	});
</script>

<style scoped>


	.webhooks-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.url-cell {
		word-break: break-all;
	}

	.actions-cell {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 0.35rem;
		text-align: right;
	}

	.dialog-actions-spacer {
		flex: 1;
	}

	.muted {
		opacity: 0.7;
	}
</style>
