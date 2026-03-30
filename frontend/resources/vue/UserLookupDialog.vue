<template>
	<dialog
		ref="dlg"
		class="user-lookup-dialog dialog"
		@close="onDialogClose"
	>
		<h2 class="dlg-title">{{ title }}</h2>
		<p v-if="subtitle" class="dlg-subtitle">{{ subtitle }}</p>

		<div class="search-row">
			<label class="sr-only" for="user-lookup-search">Search by username</label>
			<input
				id="user-lookup-search"
				ref="searchInput"
				v-model="searchQuery"
				type="search"
				class="search-input"
				placeholder="Search by username…"
				autocomplete="off"
				@keydown.enter.prevent="onSearchEnter"
			/>
		</div>

		<div v-if="loading" class="muted">Loading users…</div>
		<div v-else-if="loadError" class="inline-notification error">{{ loadError }}</div>
		<div v-else class="list-wrap">
			<p v-if="!filteredUsers.length" class="inline-notification note">
				{{ users.length === 0 ? 'No users in the system.' : 'No matching users.' }}
			</p>
			<ul v-else class="user-list" role="listbox" :aria-multiselectable="multiple ? 'true' : undefined">
				<li
					v-for="u in filteredUsers"
					:key="u.id"
					class="user-row"
					:class="{
						selected: multiple && selectedIds.includes(u.id),
					}"
					role="option"
					:aria-selected="multiple ? selectedIds.includes(u.id) : undefined"
					@click="multiple ? toggleSelect(u.id) : pickSingle(u)"
				>
					<span v-if="multiple" class="check" aria-hidden="true">
						<Icon
							:icon="selectedIds.includes(u.id) ? 'material-symbols:check-box' : 'material-symbols:check-box-outline-blank'"
							width="20"
							height="20"
						/>
					</span>
					<span class="username">{{ u.username }}</span>
					<span class="meta muted">{{ u.createdAt }}</span>
				</li>
			</ul>
		</div>

		<div class="dialog-actions">
			<button type="button" class="neutral" @click="onCancel">Cancel</button>
			<button
				v-if="multiple"
				type="button"
				class="good"
				:disabled="!selectedIds.length"
				@click="confirmMultiple"
			>
				{{ confirmLabel }}{{ selectedIds.length ? ` (${selectedIds.length})` : '' }}
			</button>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';

	const props = defineProps({
		title: { type: String, default: 'Find user' },
		subtitle: { type: String, default: '' },
		multiple: { type: Boolean, default: false },
		confirmLabel: { type: String, default: 'Add' },
		/** User IDs to omit from the list (e.g. already shared, owner, already in group). */
		excludeUserIds: { type: Array, default: () => [] },
	});

	const emit = defineEmits({
		picked(users) {
			return Array.isArray(users);
		},
		cancel: () => true,
	});

	const dlg = ref(null);
	const searchInput = ref(null);
	const searchQuery = ref('');
	const users = ref([]);
	const loading = ref(false);
	const loadError = ref('');
	const selectedIds = ref([]);

	const excludeSet = computed(() => new Set((props.excludeUserIds || []).map((id) => Number(id))));

	const filteredUsers = computed(() => {
		const q = searchQuery.value.trim().toLowerCase();
		return users.value.filter((u) => {
			if (excludeSet.value.has(u.id)) return false;
			if (!q) return true;
			return u.username.toLowerCase().includes(q);
		});
	});

	watch(
		() => props.excludeUserIds,
		() => {
			if (props.multiple) {
				selectedIds.value = selectedIds.value.filter((id) => !excludeSet.value.has(id));
			}
		},
		{ deep: true }
	);

	function resetState() {
		searchQuery.value = '';
		selectedIds.value = [];
		loadError.value = '';
	}

	function onDialogClose() {
		resetState();
	}

	async function loadUsers() {
		loading.value = true;
		loadError.value = '';
		try {
			await waitForClient();
			const res = await window.client.getUsers({});
			users.value = res.users || [];
		} catch (e) {
			loadError.value = e.message || 'Failed to load users';
			users.value = [];
		} finally {
			loading.value = false;
		}
	}

	function open() {
		resetState();
		loadUsers().then(() => {
			dlg.value?.showModal();
			nextTick(() => searchInput.value?.focus());
		});
	}

	function close() {
		dlg.value?.close();
	}

	function onCancel() {
		emit('cancel');
		close();
	}

	function pickSingle(u) {
		emit('picked', [u]);
		close();
	}

	function toggleSelect(id) {
		const i = selectedIds.value.indexOf(id);
		if (i >= 0) {
			selectedIds.value = selectedIds.value.filter((x) => x !== id);
		} else {
			selectedIds.value = [...selectedIds.value, id];
		}
	}

	function confirmMultiple() {
		const picked = users.value.filter((u) => selectedIds.value.includes(u.id));
		emit('picked', picked);
		close();
	}

	/** In single mode, Enter picks first filtered row if exactly one match. */
	function onSearchEnter() {
		if (props.multiple || filteredUsers.value.length !== 1) return;
		pickSingle(filteredUsers.value[0]);
	}

	defineExpose({ open, close });
</script>

<style scoped>
	.user-lookup-dialog {
		width: min(32rem, calc(100vw - 2rem));
		max-height: min(70vh, 28rem);
		display: flex;
		flex-direction: column;
		padding: 1rem 1.25rem;
	}

	.dlg-title {
		margin: 0 0 0.35rem;
		font-size: 1.15rem;
	}

	.dlg-subtitle {
		margin: 0 0 0.75rem;
		font-size: 0.88rem;
		opacity: 0.85;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	.search-row {
		margin-bottom: 0.75rem;
	}

	.search-input {
		width: 100%;
		box-sizing: border-box;
	}

	.list-wrap {
		flex: 1;
		min-height: 8rem;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.user-list {
		list-style: none;
		margin: 0;
		padding: 0;
		overflow-y: auto;
		flex: 1;
		border: 1px solid rgba(255, 255, 255, 0.2);
		border-radius: 0.35rem;
	}

	.user-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.45rem 0.65rem;
		cursor: pointer;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.user-row:last-child {
		border-bottom: none;
	}

	.user-row:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.user-row.selected {
		background: rgba(255, 255, 255, 0.1);
	}

	.check {
		display: flex;
		flex-shrink: 0;
		opacity: 0.9;
	}

	.username {
		font-weight: 600;
		flex: 1;
		min-width: 0;
	}

	.meta {
		font-size: 0.75rem;
		flex-shrink: 0;
	}

	.muted {
		opacity: 0.8;
	}

	.dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 1rem;
		padding-top: 0.75rem;
		border-top: 1px solid rgba(255, 255, 255, 0.15);
	}
</style>
