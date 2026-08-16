<template>
	<dialog
		ref="dlg"
		@close="onDialogClose"
	>
		<div class="dialog-panel group-lookup-panel">
			<h2 class="dlg-title">{{ title }}</h2>
			<p v-if="subtitle" class="dlg-subtitle">{{ subtitle }}</p>

			<div class="search-row">
				<label class="sr-only" for="group-lookup-search">Search by group name</label>
				<input
					id="group-lookup-search"
					ref="searchInput"
					v-model="searchQuery"
					type="search"
					class="search-input"
					placeholder="Search by group name…"
					autocomplete="off"
					@keydown.enter.prevent="onSearchEnter"
				/>
			</div>

			<div v-if="loading" class="muted">Loading user groups…</div>
			<div v-else-if="loadError" class="inline-notification error">{{ loadError }}</div>
			<div v-else class="list-wrap">
				<p v-if="!filteredGroups.length" class="inline-notification note">
					{{ groups.length === 0 ? 'No user groups in the system.' : 'No matching user groups.' }}
				</p>
				<CheckGroup
					v-else-if="multiple"
					v-model="selectedIds"
					name="group-lookup"
					class="group-check-group"
					aria-label="Select user groups"
					:options="groupOptions"
				/>
				<ul v-else class="group-list" role="listbox">
					<li
						v-for="g in filteredGroups"
						:key="g.id"
						class="group-row"
						role="option"
						@click="pickSingle(g)"
					>
						<span class="group-name">{{ g.name }}</span>
						<span class="meta muted">{{ formatMemberCount(g) }}</span>
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
		</div>
	</dialog>
</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
	import { waitForClient } from '../javascript/util';

	const props = defineProps({
		title: { type: String, default: 'Find user group' },
		subtitle: { type: String, default: '' },
		multiple: { type: Boolean, default: false },
		confirmLabel: { type: String, default: 'Add' },
		/** Group IDs to omit from the list (e.g. already shared). */
		excludeGroupIds: { type: Array, default: () => [] },
	});

	const emit = defineEmits({
		picked(groups) {
			return Array.isArray(groups);
		},
		cancel: () => true,
	});

	const dlg = ref(null);
	const searchInput = ref(null);
	const searchQuery = ref('');
	const groups = ref([]);
	const loading = ref(false);
	const loadError = ref('');
	const selectedIds = ref([]);

	const excludeSet = computed(() => new Set((props.excludeGroupIds || []).map((id) => Number(id))));

	const filteredGroups = computed(() => {
		const q = searchQuery.value.trim().toLowerCase();
		return groups.value.filter((g) => {
			if (excludeSet.value.has(g.id)) return false;
			if (!q) return true;
			return g.name.toLowerCase().includes(q);
		});
	});

	const groupOptions = computed(() =>
		filteredGroups.value.map((g) => ({
			value: g.id,
			label: `${g.name} (${formatMemberCount(g)})`,
		})),
	);

	function memberCount(g) {
		return g.memberCount ?? g.member_count ?? 0;
	}

	function formatMemberCount(g) {
		const n = memberCount(g);
		return n === 1 ? '1 member' : `${n} members`;
	}

	watch(
		() => props.excludeGroupIds,
		() => {
			if (props.multiple) {
				selectedIds.value = selectedIds.value.filter((id) => !excludeSet.value.has(id));
			}
		},
		{ deep: true },
	);

	function resetState() {
		searchQuery.value = '';
		selectedIds.value = [];
		loadError.value = '';
	}

	function onDialogClose() {
		resetState();
	}

	async function loadGroups() {
		loading.value = true;
		loadError.value = '';
		try {
			await waitForClient();
			const res = await window.client.listUserGroups({});
			groups.value = res.groups || [];
		} catch (e) {
			loadError.value = e.message || 'Failed to load user groups';
			groups.value = [];
		} finally {
			loading.value = false;
		}
	}

	function open() {
		resetState();
		loadGroups().then(() => {
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

	function pickSingle(g) {
		emit('picked', [g]);
		close();
	}

	function confirmMultiple() {
		const picked = groups.value.filter((g) => selectedIds.value.includes(g.id));
		emit('picked', picked);
		close();
	}

	function onSearchEnter() {
		if (props.multiple || filteredGroups.value.length !== 1) return;
		pickSingle(filteredGroups.value[0]);
	}

	defineExpose({ open, close });
</script>

<style scoped>
	.group-lookup-panel {
		width: min(32rem, calc(100vw - 2rem));
		max-height: min(70vh, 28rem);
		display: flex;
		flex-direction: column;
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

	.group-check-group {
		overflow-y: auto;
		flex: 1;
		min-height: 0;
		width: 100%;
		max-width: 100%;
	}

	.group-list {
		list-style: none;
		margin: 0;
		padding: 0;
		overflow-y: auto;
		flex: 1;
		border: 1px solid rgba(255, 255, 255, 0.2);
		border-radius: 0.35rem;
	}

	.group-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.45rem 0.65rem;
		cursor: pointer;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.group-row:last-child {
		border-bottom: none;
	}

	.group-row:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.group-name {
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
	}
</style>
