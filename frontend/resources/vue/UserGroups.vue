<template>
	<Section
		title="User Groups"
		subtitle="Manage user groups and their membership. Groups let you organise users for easier administration."
		classes="user-groups settings-users"
		:padding="false"
	>
		<template #toolbar>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="loadAll">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error groups-banner-pad">{{ errorMessage }}</div>

		<div v-if="loading && !groups.length" class="groups-banner-pad muted">Loading…</div>

		<template v-else>
			<h3 class="subsection-title groups-banner-pad">Groups</h3>
			<p v-if="!groups.length" class="inline-notification note groups-banner-pad">No user groups yet.</p>

			<table v-if="groups.length" class="groups-table user-table-wrap">
				<thead>
					<tr>
						<th>Name</th>
						<th>Members</th>
						<th v-if="canManage" class="actions-col">Actions</th>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="g in groups"
						:key="g.id"
						class="group-row"
						:class="{ selected: selectedGroupId === g.id }"
						@click="selectGroup(g)"
					>
						<td><strong>{{ g.name }}</strong></td>
						<td>{{ g.memberCount }}</td>
						<td v-if="canManage" align="right">
							<button
								type="button"
								class="bad small"
								@click.stop="deleteGroup(g)"
							>
								Delete
							</button>
						</td>
					</tr>
				</tbody>
			</table>

			<div v-if="canManage" class="create-group-panel">
				<h4 class="subsection-title">Create group</h4>
				<div class="form-row">
					<label for="new-group-name">Name</label>
					<input id="new-group-name" v-model="newGroupName" type="text" autocomplete="off" @keydown.enter="createGroup" />
					<button type="button" class="good" :disabled="saving || !newGroupName.trim()" @click="createGroup">
						Create
					</button>
				</div>
			</div>

			<div v-if="selectedGroupId" class="members-section">
				<h3 class="subsection-title">
					Members of <em>{{ selectedGroupName }}</em>
				</h3>
				<p v-if="membersLoading" class="muted">Loading members…</p>
				<template v-else>
					<div v-if="users.length && canManage" class="member-toolbar">
						<button type="button" class="good" @click="openMemberLookup">
							<Icon icon="material-symbols:person-search" />
							Add users…
						</button>
					</div>
					<UserLookupDialog
						ref="memberLookup"
						title="Add users to group"
						:subtitle="`Select users to add to ${selectedGroupName}. Already in the group are hidden.`"
						multiple
						confirm-label="Add to selection"
						:exclude-user-ids="selectedUserIds"
						@picked="onMembersPicked"
					/>
					<fieldset v-if="users.length" class="member-checks">
						<legend>Select users to include in this group</legend>
						<label v-for="u in users" :key="u.id" class="check-label">
							<input v-model="selectedUserIds" type="checkbox" :value="u.id" :disabled="!canManage" />
							{{ u.username }}
						</label>
						<button
							v-if="canManage"
							type="button"
							class="good"
							:disabled="saving"
							@click="saveMembers"
						>
							Save members
						</button>
					</fieldset>
					<p v-else class="inline-notification note">No users in the system.</p>
					<div v-if="membersSaveMessage" class="inline-notification" :class="membersSaveType">
						{{ membersSaveMessage }}
					</div>
				</template>
			</div>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import UserLookupDialog from './UserLookupDialog.vue';

	const groups = ref([]);
	const users = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const newGroupName = ref('');
	const selectedGroupId = ref(0);
	const selectedUserIds = ref([]);
	const membersLoading = ref(false);
	const membersSaveMessage = ref('');
	const membersSaveType = ref('');
	const memberLookup = ref(null);

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);

	const selectedGroupName = computed(() => {
		const g = groups.value.find((g) => g.id === selectedGroupId.value);
		return g ? g.name : '';
	});

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
			const [gr, ur] = await Promise.all([
				window.client.listUserGroups({}),
				window.client.getUsers({})
			]);
			groups.value = gr.groups || [];
			users.value = ur.users || [];

			if (selectedGroupId.value && !groups.value.find((g) => g.id === selectedGroupId.value)) {
				selectedGroupId.value = 0;
				selectedUserIds.value = [];
			}
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load user groups.';
		} finally {
			loading.value = false;
		}
	}

	function openMemberLookup() {
		memberLookup.value?.open();
	}

	function onMembersPicked(picked) {
		const set = new Set(selectedUserIds.value);
		for (const u of picked) {
			set.add(u.id);
		}
		selectedUserIds.value = Array.from(set);
	}

	async function selectGroup(g) {
		selectedGroupId.value = g.id;
		membersSaveMessage.value = '';
		membersSaveType.value = '';
		await loadMembers();
	}

	async function loadMembers() {
		if (!selectedGroupId.value) return;
		membersLoading.value = true;
		try {
			await waitForClient();
			const res = await window.client.getUserGroupMembers({ groupId: selectedGroupId.value });
			selectedUserIds.value = [...(res.userIds || [])];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load group members.';
		} finally {
			membersLoading.value = false;
		}
	}

	async function saveMembers() {
		if (!canManage.value || !selectedGroupId.value) return;
		saving.value = true;
		membersSaveMessage.value = '';
		membersSaveType.value = '';
		try {
			await waitForClient();
			const res = await window.client.setUserGroupMembers({
				groupId: selectedGroupId.value,
				userIds: selectedUserIds.value,
			});
			if (res.standardResponse?.success) {
				membersSaveMessage.value = res.standardResponse.message || 'Members updated.';
				membersSaveType.value = 'success';
			} else {
				membersSaveMessage.value = res.standardResponse?.message || 'Failed to update members.';
				membersSaveType.value = 'error';
			}
			await loadAll();
			await loadMembers();
		} catch (e) {
			console.error(e);
			membersSaveMessage.value = e.message || 'Failed to save members.';
			membersSaveType.value = 'error';
		} finally {
			saving.value = false;
		}
	}

	async function createGroup() {
		if (!canManage.value || !newGroupName.value.trim()) return;
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.createUserGroup({ name: newGroupName.value.trim() });
			newGroupName.value = '';
			await loadAll();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to create group.';
		} finally {
			saving.value = false;
		}
	}

	async function deleteGroup(g) {
		if (!canManage.value || !confirm(`Delete group "${g.name}"? Members will be removed.`)) return;
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteUserGroup({ groupId: g.id });
			if (selectedGroupId.value === g.id) {
				selectedGroupId.value = 0;
				selectedUserIds.value = [];
			}
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
	.groups-banner-pad {
		padding-left: 1em;
		padding-right: 1em;
	}

	.subsection-title {
		margin: 1.25rem 0 0.5rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.user-table-wrap {
		margin-top: 0.5rem;
	}

	.groups-table {
		width: 100%;
		margin-bottom: 1.5rem;
	}

	.group-row {
		cursor: pointer;
	}

	.group-row:hover {
		background: var(--pico-muted-border-color, rgba(0, 0, 0, 0.04));
	}

	.group-row.selected {
		background: var(--pico-primary-focus, rgba(26, 115, 232, 0.1));
	}

	.actions-col {
		width: 6rem;
	}

	.create-group-panel {
		margin-bottom: 2rem;
		padding: 1rem 1em;
		border-top: 1px solid var(--pico-muted-border-color, rgba(0, 0, 0, 0.12));
	}

	.form-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.75rem;
	}

	.form-row label {
		font-weight: 600;
	}

	.form-row input {
		flex: 1;
		min-width: 12rem;
	}

	.member-checks {
		margin: 0.75rem 0;
		border: 1px solid var(--pico-muted-border-color, rgba(0, 0, 0, 0.12));
		padding: 0.75rem 1rem;
		border-radius: 4px;
	}

	.check-label {
		display: block;
		margin: 0.25rem 0;
		font-size: 0.9em;
	}

	.muted {
		opacity: 0.8;
	}

	.members-section {
		padding-left: 1em;
		padding-right: 1em;
	}

	.member-toolbar {
		margin: 0.5rem 0 0.75rem;
	}
</style>
