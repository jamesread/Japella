<template>
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
				<button type="button" class="neutral" :disabled="saving" @click="closeEditDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		:title="pageTitle"
		subtitle="Outbound event webhook destination and delivery history"
		:icon="WebhookIcon"
		classes="webhook-details"
		:padding="false"
	>
		<template #toolbar>
			<router-link :to="{ name: 'adminWebhooks' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to Webhooks</span>
			</router-link>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading || testing"
				@click="load"
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
				v-if="canManage && webhook"
				type="button"
				class="inline-icon neutral"
				:disabled="loading || saving || testing"
				@click="openEdit"
			>
				<span>Edit</span>
			</button>
			<button
				v-if="canManage && webhook"
				type="button"
				class="inline-icon good"
				:disabled="loading || saving || testing"
				@click="sendTest"
			>
				<span>{{ testing ? 'Sending…' : 'Send test' }}</span>
			</button>
		</template>

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />
		<NotificationBlock v-if="actionMessage" :type="actionMessageType" :message="actionMessage" />
		<div v-if="loading && !webhook" class="list-banner-pad muted">Loading…</div>
		<div v-else-if="!webhook" class="list-banner-pad">
			<NotificationBlock type="note" message="Webhook not found." />
		</div>
		<dl v-else class="list-banner-pad webhook-meta">
			<dt>ID</dt>
			<dd>{{ webhook.id }}</dd>
			<dt>URL</dt>
			<dd class="url-cell">{{ webhook.url }}</dd>
			<dt>Status</dt>
			<dd>
				<span :class="webhook.enabled ? 'fg-good' : 'muted'">{{ webhook.enabled ? 'Enabled' : 'Disabled' }}</span>
			</dd>
			<dt>Events</dt>
			<dd>{{ (webhook.events || []).join(', ') || '—' }}</dd>
			<dt>Created</dt>
			<dd>{{ formatDate(webhook.created) }}</dd>
			<dt>Updated</dt>
			<dd>{{ formatDate(webhook.updated) }}</dd>
		</dl>
	</Section>

	<Section
		v-if="webhook"
		title="Delivery logs"
		subtitle="Recent outbound delivery attempts for this webhook"
		:icon="ActivityIcon"
		classes="webhook-delivery-logs"
		:padding="false"
	>
		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh logs"
				title="Refresh logs"
				:disabled="loading || testing"
				@click="load"
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

		<div class="logs-table-wrap">
			<Table
				:data="logRows"
				:headers="logHeaders"
			>
				<template #empty>
					No delivery attempts recorded yet. Use Send test or wait for a subscribed event.
				</template>
				<template #cell-status="{ row }">
					<span :class="row.success ? 'fg-good' : 'fg-bad'">{{ row.statusLabel }}</span>
				</template>
				<template #cell-errorMessage="{ value }">
					<span class="error-cell">{{ value || '—' }}</span>
				</template>
			</Table>
		</div>
	</Section>

	<DangerZone
		v-if="canManage && webhook"
		title="Danger zone"
		subtitle="Destructive actions for this webhook"
		description="Show destructive actions"
		warning="Deleting this webhook removes its event subscriptions and delivery history."
	>
		<div role="toolbar" class="danger-zone-actions">
			<button type="button" class="inline-icon bad" :disabled="saving || testing" @click="deleteWebhook">
				<span>Delete webhook</span>
			</button>
		</div>
	</DangerZone>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ActivityIcon, ArrowLeft01Icon, RefreshIcon, WebhookIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import DangerZone from 'picocrank/vue/components/DangerZone.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;
	const route = useRoute();
	const router = useRouter();

	const webhook = ref(null);
	const deliveryLogs = ref([]);
	const catalog = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const testing = ref(false);
	const errorMessage = ref('');
	const actionMessage = ref('');
	const actionMessageType = ref('good');
	const editError = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const editDialog = ref(null);
	const editForm = ref({
		url: '',
		secret: '',
		events: [],
		enabled: true,
	});

	const webhookId = computed(() => parseInt(String(route.params.id || '0'), 10) || 0);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.settings')),
	);
	const pageTitle = computed(() => (webhook.value ? `Webhook #${webhook.value.id}` : 'Webhook'));
	const eventOptions = computed(() => catalog.value.map((e) => ({ label: e, value: e })));
	const enabledOptions = [
		{ label: 'On', value: true },
		{ label: 'Off', value: false },
	];
	const logHeaders = [
		{ key: 'created', label: 'When', sortable: true, width: '11rem' },
		{ key: 'event', label: 'Event', sortable: true, width: '10rem' },
		{ key: 'status', label: 'Status', sortable: false, width: '7rem' },
		{ key: 'httpStatus', label: 'HTTP', sortable: true, width: '5rem' },
		{ key: 'durationMs', label: 'ms', sortable: true, width: '5rem' },
		{ key: 'errorMessage', label: 'Error', sortable: false },
	];
	const logRows = computed(() =>
		deliveryLogs.value.map((l) => ({
			id: l.id,
			created: formatDate(l.created),
			event: l.event || '—',
			success: !!l.success,
			statusLabel: l.success ? 'OK' : 'Failed',
			httpStatus: l.httpStatus || 0,
			durationMs: l.durationMs || 0,
			errorMessage: l.errorMessage || '',
		})),
	);

	function formatDate(value) {
		if (!value) return '—';
		try {
			const d = new Date(value);
			if (Number.isNaN(d.getTime())) return value;
			return d.toLocaleString();
		} catch {
			return value;
		}
	}

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
		catalog.value = st.webhookEvents || [];
	}

	async function load() {
		if (!webhookId.value) {
			webhook.value = null;
			deliveryLogs.value = [];
			loading.value = false;
			errorMessage.value = 'Invalid webhook id.';
			return;
		}
		loading.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await refreshStatus();
			const res = await window.client.getWebhook({ id: webhookId.value, logLimit: 50 });
			webhook.value = res.webhook || null;
			deliveryLogs.value = res.deliveryLogs || [];
			if (!catalog.value.length) {
				const list = await window.client.listWebhooks({});
				catalog.value = list.events || [];
			}
		} catch (e) {
			console.error(e);
			webhook.value = null;
			deliveryLogs.value = [];
			errorMessage.value = e.message || 'Failed to load webhook.';
		} finally {
			loading.value = false;
		}
	}

	function openEdit() {
		if (!webhook.value) return;
		editForm.value = {
			url: webhook.value.url,
			secret: '',
			events: [...(webhook.value.events || [])],
			enabled: !!webhook.value.enabled,
		};
		editError.value = '';
		editDialog.value?.showModal();
	}

	function closeEditDialog() {
		editDialog.value?.close();
		editError.value = '';
	}

	async function saveEdit() {
		if (!webhook.value) return;
		saving.value = true;
		editError.value = '';
		try {
			await waitForClient();
			await window.client.updateWebhook({
				id: webhook.value.id,
				url: editForm.value.url.trim(),
				secret: editForm.value.secret.trim(),
				events: editForm.value.events,
				enabled: editForm.value.enabled,
			});
			closeEditDialog();
			actionMessage.value = 'Webhook updated.';
			actionMessageType.value = 'good';
			await load();
		} catch (e) {
			editError.value = e.message || 'Failed to update webhook.';
		} finally {
			saving.value = false;
		}
	}

	async function sendTest() {
		if (!webhook.value || testing.value) return;
		testing.value = true;
		actionMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.testWebhook({ id: webhook.value.id });
			actionMessage.value = res.message || (res.success ? 'Test webhook delivered.' : 'Test webhook failed.');
			actionMessageType.value = res.success ? 'good' : 'error';
			await load();
		} catch (e) {
			actionMessage.value = e.message || 'Failed to send test webhook.';
			actionMessageType.value = 'error';
		} finally {
			testing.value = false;
		}
	}

	async function deleteWebhook() {
		if (!webhook.value || !confirm('Delete this webhook target?')) return;
		saving.value = true;
		actionMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteWebhook({ id: webhook.value.id });
			router.push({ name: 'adminWebhooks' });
		} catch (e) {
			actionMessage.value = e.message || 'Failed to delete webhook.';
			actionMessageType.value = 'error';
		} finally {
			saving.value = false;
		}
	}

	watch(webhookId, () => {
		load();
	}, { immediate: true });
</script>

<style scoped>
	.list-banner-pad {
		padding-left: 1em;
		padding-right: 1em;
	}

	.webhook-meta {
		display: grid;
		grid-template-columns: 8rem 1fr;
		gap: 0.5rem 1rem;
		margin: 0;
		padding-top: 1rem;
		padding-bottom: 1rem;
	}

	.webhook-meta dt {
		font-weight: 600;
		opacity: 0.85;
	}

	.webhook-meta dd {
		margin: 0;
	}

	.url-cell {
		word-break: break-all;
	}

	.logs-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1rem;
	}

	.error-cell {
		word-break: break-word;
	}

	.danger-zone-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.muted {
		opacity: 0.7;
	}
</style>
