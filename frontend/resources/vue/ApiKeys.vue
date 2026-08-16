<template>
	<dialog ref="createKeyDialog" class="dialog" @close="closeCreateKeyDialog">
		<h2>{{ createKeyDialogTitle }}</h2>
		<p>Give this key a descriptive name so you can tell it apart later (for example, “Cursor MCP” or “CI deploy”).</p>

		<FormLayout @submit.prevent="submitCreateKey">
			<FormField label="Name" for="create-api-key-name" :disabled="creatingKey">
				<input
					id="create-api-key-name"
					ref="createKeyNameInput"
					v-model="createKeyName"
					type="text"
					maxlength="128"
					placeholder="e.g. Cursor MCP"
					:disabled="creatingKey"
					required
				/>
			</FormField>

			<p v-if="createKeyError" class="inline-notification error">{{ createKeyError }}</p>

			<template #actions>
				<button type="button" class="neutral" :disabled="creatingKey" @click="closeCreateKeyDialog">Cancel</button>
				<button type="submit" class="good" :disabled="creatingKey">
					{{ creatingKey ? 'Creating…' : 'Create key' }}
				</button>
			</template>
		</FormLayout>
	</dialog>

	<dialog ref="newKeyDialog" class="dialog">
		<h2>{{ newKeyDialogTitle }}</h2>
		<p>Here is the new API key. Please copy it and store it securely, as you will not be able to see it again.</p>

		<FormLayout @submit.prevent="closeNewKeyDialog">
			<FormField label="API key" for="new-api-key-value">
				<input id="new-api-key-value" type="text" readonly class="key-input" :value="newKeyValue" />
			</FormField>
			<template #actions>
				<button type="submit" class="good">OK</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		:subtitle="sectionSubtitle"
		:padding="false"
		classes="api-keys settings-users"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="KeyIcon" width="22" height="22" aria-hidden="true" />
				{{ pageTitle }}
			</span>
		</template>

		<template #toolbar>
			<router-link
				v-if="filterUserId"
				:to="{ name: 'userDetails', params: { id: String(filterUserId) } }"
				class="button inline-icon neutral"
			>
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to User</span>
			</router-link>
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
				type="button"
				class="inline-icon good"
				aria-label="Create API key"
				title="Create API key"
				:disabled="loading"
				@click="openCreateKeyDialog"
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
		<div v-if="loading && !apiKeys.length" class="list-banner-pad muted">Loading…</div>

		<template v-else>
			<p v-if="!apiKeys.length" class="inline-notification note list-banner-pad">No API keys available.</p>

			<Table
				v-else
				class="api-keys-table-wrap"
				:data="tableRows"
				:headers="tableHeaders"
			>
				<template #cell-name="{ value }">
					<strong>{{ value }}</strong>
				</template>
				<template #cell-keyValue="{ value }">
					<code>{{ value }}</code>
				</template>
				<template #cell-actions="{ row }">
					<div class="actions-cell">
						<button
							type="button"
							class="bad small"
							:disabled="revoking"
							@click="revokeKey(row)"
						>
							Revoke
						</button>
					</div>
				</template>
			</Table>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Add01Icon, ArrowLeft01Icon, KeyIcon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;
	const route = useRoute();

	const allKeys = ref([]);
	const loading = ref(true);
	const revoking = ref(false);
	const errorMessage = ref('');
	const createKeyDialog = ref(null);
	const createKeyNameInput = ref(null);
	const newKeyDialog = ref(null);
	const newKeyValue = ref('');
	const createKeyName = ref('');
	const displayedKeyName = ref('');
	const createKeyError = ref('');
	const creatingKey = ref(false);
	const targetUser = ref(null);

	const filterUserId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const apiKeys = computed(() => {
		if (!filterUserId.value) return allKeys.value;
		return allKeys.value.filter((k) => k.userId === filterUserId.value);
	});

	const pageTitle = computed(() => {
		if (targetUser.value) return `API Keys — ${targetUser.value.username}`;
		if (filterUserId.value) return `API Keys — User #${filterUserId.value}`;
		return 'API Keys';
	});

	const sectionSubtitle = computed(() => {
		if (filterUserId.value) {
			return 'View, revoke, and create API keys for this user. Use Bearer token in Authorization header.';
		}
		return 'View, revoke, and create API keys. Use Bearer token in Authorization header.';
	});

	const createKeyDialogTitle = computed(() => {
		if (filterUserId.value && targetUser.value) {
			return `Create API key for ${targetUser.value.username}`;
		}
		if (filterUserId.value) {
			return 'Create API key for user';
		}
		return 'Create API key';
	});

	const newKeyDialogTitle = computed(() => {
		const name = displayedKeyName.value.trim();
		if (name) {
			return `New key: ${name}`;
		}
		return 'Your new key';
	});

	const tableHeaders = computed(() => {
		const headers = [
			{ key: 'name', label: 'Name', sortable: true },
			{ key: 'keyValue', label: 'Key', sortable: true },
		];
		if (!filterUserId.value) {
			headers.push({ key: 'username', label: 'User Account', sortable: true });
		}
		headers.push(
			{ key: 'createdAtLabel', label: 'Created', sortable: true, width: '11rem' },
			{ key: 'lastUsedLabel', label: 'Last Used', sortable: true, width: '11rem' },
			{ key: 'actions', label: 'Actions', sortable: false, width: '6rem' },
		);
		return headers;
	});

	function formatLastUsed(value) {
		if (!value) return 'Never';
		const date = new Date(value);
		return Number.isNaN(date.getTime()) ? 'Never' : date.toLocaleString();
	}

	function formatKeyName(value) {
		const name = String(value || '').trim();
		return name || '—';
	}

	function formatCreatedAt(value) {
		if (!value) return '—';
		const date = new Date(value);
		return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
	}

	const tableRows = computed(() =>
		apiKeys.value.map((key) => ({
			...key,
			name: formatKeyName(key.name),
			createdAtLabel: formatCreatedAt(key.createdAt),
			lastUsedLabel: formatLastUsed(key.lastUsedAt),
		})),
	);

	async function loadTargetUser() {
		if (!filterUserId.value) {
			targetUser.value = null;
			return;
		}
		try {
			await waitForClient();
			const res = await window.client.getUser({ userId: filterUserId.value });
			targetUser.value = res.user ?? null;
		} catch {
			targetUser.value = null;
		}
	}

	async function loadAll() {
		loading.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await loadTargetUser();
			const res = await window.client.getApiKeys();
			allKeys.value = res.keys || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e?.message || 'Failed to load API keys';
		} finally {
			loading.value = false;
		}
	}

	async function revokeKey(key) {
		if (!confirm(`Are you sure you want to revoke the API key “${key.name}”?`)) {
			return;
		}
		revoking.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.revokeApiKey({ id: key.id });
			await loadAll();
		} catch (e) {
			console.error('Error revoking API key:', e);
			errorMessage.value = e?.message || 'Failed to revoke API key.';
		} finally {
			revoking.value = false;
		}
	}

	function openCreateKeyDialog() {
		createKeyName.value = '';
		createKeyError.value = '';
		createKeyDialog.value?.showModal();
		nextTick(() => createKeyNameInput.value?.focus());
	}

	function closeCreateKeyDialog() {
		createKeyDialog.value?.close();
		createKeyName.value = '';
		createKeyError.value = '';
	}

	function showNewKeyDialog(keyValue, name) {
		newKeyValue.value = keyValue;
		displayedKeyName.value = name;
		newKeyDialog.value.showModal();
	}

	function closeNewKeyDialog() {
		newKeyDialog.value?.close();
	}

	async function submitCreateKey() {
		const name = createKeyName.value.trim();
		if (!name) {
			createKeyError.value = 'Enter a descriptive name for this key.';
			return;
		}

		creatingKey.value = true;
		createKeyError.value = '';

		try {
			await waitForClient();
			const req = { name };
			if (filterUserId.value) {
				req.userId = filterUserId.value;
			}
			const res = await window.client.createApiKey(req);
			closeCreateKeyDialog();
			showNewKeyDialog(res.newKeyValue, name);
			await loadAll();
		} catch (e) {
			console.error('Error creating API key:', e);
			createKeyError.value = e?.message || 'Failed to create API key.';
		} finally {
			creatingKey.value = false;
		}
	}

	watch(filterUserId, () => {
		loadAll();
	}, { immediate: true });
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

	.api-keys-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.actions-cell {
		text-align: right;
	}

	.key-input {
		text-align: center;
		width: 100%;
		box-sizing: border-box;
	}
</style>
