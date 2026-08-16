<template>
	<dialog ref="createGroupDialog" class="dialog" @close="closeCreateGroupDialog">
		<h2>Create group</h2>
		<p>Give the new group a name. You can add members after it is created.</p>

		<FormLayout @submit.prevent="createGroup">
			<FormField label="Name" for="new-group-name" :disabled="saving">
				<input
					id="new-group-name"
					ref="createGroupNameInput"
					v-model="newGroupName"
					type="text"
					autocomplete="off"
					:disabled="saving"
					required
				/>
			</FormField>

			<p v-if="createGroupError" class="inline-notification error">{{ createGroupError }}</p>

			<template #actions>
				<button type="button" class="neutral" :disabled="saving" @click="closeCreateGroupDialog">Cancel</button>
				<button type="submit" class="good" :disabled="saving || !newGroupName.trim()">
					{{ saving ? 'Creating…' : 'Create group' }}
				</button>
			</template>
		</FormLayout>
	</dialog>

	<Section
		subtitle="Manage user groups and their membership. Groups let you organise users for easier administration."
		classes="user-groups settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserGroupIcon" width="22" height="22" aria-hidden="true" />
				User Groups
			</span>
		</template>
		<template #toolbar>
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
				v-if="canManage"
				type="button"
				class="inline-icon good"
				aria-label="Create group"
				title="Create group"
				:disabled="loading"
				@click="openCreateGroupDialog"
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

		<div v-if="errorMessage" class="inline-notification error groups-banner-pad">{{ errorMessage }}</div>

		<div v-if="loading && !groups.length" class="groups-banner-pad muted">Loading…</div>

		<template v-else>
			<p v-if="!groups.length" class="inline-notification note groups-banner-pad">No user groups yet.</p>

			<Table
				v-else
				class="user-table-wrap"
				row-clickable
				:data="groups"
				:headers="tableHeaders"
				@row-click="openGroupDetails"
			>
				<template #cell-name="{ row, value }">
					<span class="group-name-cell">
						<strong>{{ value }}</strong>
						<span v-if="isSystemGroup(row.name)" class="tag fg-note">system group</span>
					</span>
				</template>
				<template #cell-actions="{ row }">
					<div v-if="canDeleteGroup(row)" class="actions-cell">
						<button
							type="button"
							class="bad small"
							:disabled="saving"
							@click="deleteGroup(row)"
						>
							Delete
						</button>
					</div>
				</template>
			</Table>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Add01Icon, RefreshIcon, UserGroupIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import Table from './picocrank/TableWithRowClick.vue';

	const router = useRouter();
	const iconStrokeWidth = 2.5;

	const groups = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const createGroupError = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const newGroupName = ref('');
	const createGroupDialog = ref(null);
	const createGroupNameInput = ref(null);

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);

	const canViewRoles = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);

	const tableHeaders = computed(() => {
		const headers = [
			{ key: 'name', label: 'Name', sortable: true },
			{ key: 'memberCount', label: 'Members', sortable: true, width: '8rem' },
		];
		if (canViewRoles.value) {
			headers.push({ key: 'roleCount', label: 'Roles', sortable: true, width: '8rem' });
		}
		headers.push({ key: 'sharedAccountCount', label: 'Shared accounts', sortable: true, width: '10rem' });
		if (canManage.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	function isSystemGroup(name) {
		return name === 'Everyone' || name === 'Administrators';
	}

	function canDeleteGroup(group) {
		return canManage.value && !isSystemGroup(group.name);
	}

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
			if (!canView.value) {
				errorMessage.value = 'You do not have permission to view user groups (usergroups.view).';
				return;
			}
			const gr = await window.client.listUserGroups({});
			const rawGroups = gr.groups || [];
			if (rawGroups.length) {
				const [roleResults, sharedResults] = await Promise.all([
					canViewRoles.value
						? Promise.all(rawGroups.map((g) => window.client.getUserGroupRbacRoles({ groupId: g.id })))
						: Promise.resolve([]),
					Promise.all(rawGroups.map((g) => window.client.getUserGroupSharedAccounts({ groupId: g.id }))),
				]);
				groups.value = rawGroups.map((g, i) => ({
					...g,
					roleCount: canViewRoles.value ? (roleResults[i].roleIds || []).length : 0,
					sharedAccountCount: (sharedResults[i].accounts || []).length,
				}));
			} else {
				groups.value = [];
			}
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load user groups.';
		} finally {
			loading.value = false;
		}
	}

	function openGroupDetails(g) {
		router.push({ name: 'userGroupDetails', params: { id: String(g.id) } });
	}

	function openCreateGroupDialog() {
		newGroupName.value = '';
		createGroupError.value = '';
		createGroupDialog.value?.showModal();
		nextTick(() => createGroupNameInput.value?.focus());
	}

	function closeCreateGroupDialog() {
		createGroupDialog.value?.close();
		newGroupName.value = '';
		createGroupError.value = '';
	}

	async function createGroup() {
		if (!canManage.value || !newGroupName.value.trim()) return;
		saving.value = true;
		createGroupError.value = '';
		try {
			await waitForClient();
			const res = await window.client.createUserGroup({ name: newGroupName.value.trim() });
			closeCreateGroupDialog();
			if (res.groupId) {
				router.push({ name: 'userGroupDetails', params: { id: String(res.groupId) } });
				return;
			}
			await loadAll();
		} catch (e) {
			console.error(e);
			createGroupError.value = e.message || 'Failed to create group.';
		} finally {
			saving.value = false;
		}
	}

	async function deleteGroup(g) {
		if (!canDeleteGroup(g) || !confirm(`Delete group "${g.name}"? Members will be removed.`)) return;
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteUserGroup({ groupId: g.id });
			await loadAll();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to delete group.';
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

	.groups-banner-pad {
		padding-left: 1em;
		padding-right: 1em;
	}

	.user-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.actions-cell {
		text-align: right;
	}

	.group-name-cell {
		display: inline-flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.45rem;
	}

	.muted {
		opacity: 0.8;
	}
</style>
