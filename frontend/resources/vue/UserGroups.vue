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
						@click="openGroupDetails(g)"
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
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const router = useRouter();

	const groups = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const newGroupName = ref('');

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
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
			if (!canView.value) {
				errorMessage.value = 'You do not have permission to view user groups (usergroups.view).';
				return;
			}
			const gr = await window.client.listUserGroups({});
			groups.value = gr.groups || [];
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

	async function createGroup() {
		if (!canManage.value || !newGroupName.value.trim()) return;
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.createUserGroup({ name: newGroupName.value.trim() });
			newGroupName.value = '';
			if (res.groupId) {
				router.push({ name: 'userGroupDetails', params: { id: String(res.groupId) } });
				return;
			}
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

	.muted {
		opacity: 0.8;
	}
</style>
