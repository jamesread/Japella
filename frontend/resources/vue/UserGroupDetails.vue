<template>
	<Section
		:title="group?.name || 'User group'"
		subtitle="View members and manage who belongs to this group."
		classes="user-group-details"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userGroups' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to User Groups
			</router-link>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="load">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button
				v-if="canManage"
				type="button"
				class="good"
				:disabled="saving || loading"
				@click="openMemberLookup"
			>
				<Icon icon="material-symbols:person-add" />
				Add users
			</button>
			<button
				v-if="canManage && group"
				type="button"
				class="bad"
				:disabled="saving"
				@click="deleteGroup"
			>
				<Icon icon="material-symbols:delete" />
				Delete group
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-if="actionMessage" class="inline-notification" :class="actionMessageType">{{ actionMessage }}</div>

		<div v-if="loading" class="muted">Loading…</div>
		<div v-else-if="!group" class="inline-notification note">User group not found.</div>
		<template v-else>
			<dl class="group-meta">
				<dt>Group ID</dt>
				<dd>{{ group.id }}</dd>
				<dt>Name</dt>
				<dd>{{ group.name }}</dd>
				<dt>Members</dt>
				<dd>{{ members.length }}</dd>
				<dt>Shared accounts</dt>
				<dd>{{ sharedAccounts.length }}</dd>
			</dl>

			<h3 class="subsection-title">Members</h3>
			<p v-if="!members.length" class="inline-notification note">No members in this group yet.</p>
			<table v-else class="members-table">
				<thead>
					<tr>
						<th>Username</th>
						<th v-if="canManage" class="actions-col"></th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="m in members" :key="m.id">
						<td>
							<router-link
								:to="{ name: 'userDetails', params: { id: String(m.id) } }"
								class="username-link"
							>
								{{ m.username }}
							</router-link>
						</td>
						<td v-if="canManage" align="right">
							<button
								type="button"
								class="bad small"
								:disabled="saving"
								@click="removeMember(m)"
							>
								Remove
							</button>
						</td>
					</tr>
				</tbody>
			</table>

			<h3 class="subsection-title">Shared social accounts</h3>
			<p class="section-hint">Social accounts shared with this group. Members inherit the listed permissions.</p>
			<p v-if="!sharedAccounts.length" class="inline-notification note">No social accounts are shared with this group.</p>
			<table v-else class="shared-accounts-table">
				<thead>
					<tr>
						<th>Account</th>
						<th>Connector</th>
						<th class="perm-col">Read</th>
						<th class="perm-col">Post</th>
						<th class="perm-col">Manage</th>
						<th>Status</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="a in sharedAccounts" :key="a.socialAccountId">
						<td>
							<router-link
								:to="{ name: 'socialAccountDetails', params: { id: String(a.socialAccountId) } }"
								class="username-link account-link"
							>
								<Icon v-if="a.icon" :icon="a.icon" width="18" height="18" />
								{{ a.identity }}
							</router-link>
						</td>
						<td>{{ a.connector }}</td>
						<td class="perm-col">
							<Icon
								:icon="a.canRead ? 'material-symbols:check' : 'material-symbols:close'"
								width="18"
								height="18"
								:class="a.canRead ? 'perm-yes' : 'perm-no'"
							/>
						</td>
						<td class="perm-col">
							<Icon
								:icon="a.canPost ? 'material-symbols:check' : 'material-symbols:close'"
								width="18"
								height="18"
								:class="a.canPost ? 'perm-yes' : 'perm-no'"
							/>
						</td>
						<td class="perm-col">
							<Icon
								:icon="a.canManage ? 'material-symbols:check' : 'material-symbols:close'"
								width="18"
								height="18"
								:class="a.canManage ? 'perm-yes' : 'perm-no'"
							/>
						</td>
						<td>{{ a.active ? 'Active' : 'Inactive' }}</td>
					</tr>
				</tbody>
			</table>
		</template>

		<UserLookupDialog
			ref="memberLookup"
			title="Add users to group"
			:subtitle="group ? `Select users to add to ${group.name}. Existing members are hidden.` : ''"
			multiple
			confirm-label="Add to group"
			:exclude-user-ids="memberIds"
			@picked="onMembersPicked"
		/>
	</Section>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import UserLookupDialog from './UserLookupDialog.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const router = useRouter();

	const group = ref(null);
	const members = ref([]);
	const sharedAccounts = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const actionMessage = ref('');
	const actionMessageType = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const memberLookup = ref(null);

	const groupId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);

	const memberIds = computed(() => members.value.map((m) => m.id));

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function load() {
		errorMessage.value = '';
		actionMessage.value = '';
		loading.value = true;
		group.value = null;
		members.value = [];
		sharedAccounts.value = [];

		if (!groupId.value) {
			errorMessage.value = 'Invalid group ID.';
			loading.value = false;
			return;
		}

		try {
			await waitForClient();
			await refreshStatus();
			if (!canView.value) {
				errorMessage.value = 'You do not have permission to view user groups (usergroups.view).';
				return;
			}

			const [gr, ur, mr, sr] = await Promise.all([
				window.client.listUserGroups({}),
				window.client.getUsers({}),
				window.client.getUserGroupMembers({ groupId: groupId.value }),
				window.client.getUserGroupSharedAccounts({ groupId: groupId.value }),
			]);

			const found = (gr.groups || []).find((g) => g.id === groupId.value) || null;
			group.value = found;

			const map = new Map();
			for (const u of ur.users || []) {
				map.set(u.id, u);
			}

			const ids = mr.userIds || [];
			members.value = ids.map((id) => {
				const u = map.get(id);
				return u || { id, username: `User #${id}` };
			}).sort((a, b) => a.username.localeCompare(b.username));

			sharedAccounts.value = sr.accounts || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load user group.';
		} finally {
			loading.value = false;
		}
	}

	function openMemberLookup() {
		memberLookup.value?.open();
	}

	async function persistMembers(nextIds, successMsg) {
		if (!canManage.value || !groupId.value) return;
		saving.value = true;
		actionMessage.value = '';
		actionMessageType.value = '';
		try {
			await waitForClient();
			const res = await window.client.setUserGroupMembers({
				groupId: groupId.value,
				userIds: nextIds,
			});
			if (res.standardResponse?.success) {
				actionMessage.value = successMsg || res.standardResponse.message || 'Members updated.';
				actionMessageType.value = 'success';
				await load();
			} else {
				actionMessage.value = res.standardResponse?.message || 'Failed to update members.';
				actionMessageType.value = 'error';
			}
		} catch (e) {
			console.error(e);
			actionMessage.value = e.message || 'Failed to update members.';
			actionMessageType.value = 'error';
		} finally {
			saving.value = false;
		}
	}

	async function onMembersPicked(picked) {
		const set = new Set(memberIds.value);
		for (const u of picked) {
			set.add(u.id);
		}
		const nextIds = Array.from(set);
		if (nextIds.length === memberIds.value.length) {
			actionMessage.value = 'No new users selected.';
			actionMessageType.value = 'note';
			return;
		}
		const added = nextIds.length - memberIds.value.length;
		await persistMembers(
			nextIds,
			added === 1 ? 'Added 1 user to the group.' : `Added ${added} users to the group.`
		);
	}

	async function removeMember(user) {
		if (!canManage.value || !confirm(`Remove "${user.username}" from this group?`)) return;
		const nextIds = memberIds.value.filter((id) => id !== user.id);
		await persistMembers(nextIds, `Removed ${user.username} from the group.`);
	}

	async function deleteGroup() {
		if (!canManage.value || !group.value) return;
		if (!confirm(`Delete group "${group.value.name}"? Members will be removed.`)) return;
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteUserGroup({ groupId: group.value.id });
			router.push({ name: 'userGroups' });
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to delete group.';
			saving.value = false;
		}
	}

	watch(groupId, () => {
		load();
	}, { immediate: true });
</script>

<style scoped>
	.group-meta {
		display: grid;
		grid-template-columns: minmax(6rem, 10rem) 1fr;
		gap: 0.35rem 1rem;
		margin: 0 0 1.25rem;
		max-width: 28rem;
	}

	.group-meta dt {
		font-weight: 600;
		opacity: 0.85;
	}

	.group-meta dd {
		margin: 0;
	}

	.subsection-title {
		margin: 1.25rem 0 0.5rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.members-table {
		width: 100%;
		max-width: 36rem;
	}

	.shared-accounts-table {
		width: 100%;
		max-width: 48rem;
	}

	.section-hint {
		margin: 0 0 0.75rem;
		font-size: 0.88rem;
		opacity: 0.85;
	}

	.account-link {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.perm-col {
		width: 4.5rem;
		text-align: center;
	}

	.perm-yes {
		color: var(--pico-color-green-500, #2f9e44);
	}

	.perm-no {
		opacity: 0.35;
	}

	.actions-col {
		width: 6rem;
	}

	.username-link {
		font-weight: 600;
		text-decoration: none;
	}

	.username-link:hover {
		text-decoration: underline;
	}

	.muted {
		opacity: 0.8;
	}
</style>
