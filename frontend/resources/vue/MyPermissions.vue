<template>
	<Section
		title="My Permissions"
		subtitle="Review your group membership and effective permissions"
		classes="my-permissions"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userControlPanel' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to User Control Panel</span>
			</router-link>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading"
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

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-else-if="loading" class="muted">Loading…</div>
		<template v-else>
			<h3 class="subsection-title">Group membership</h3>
			<p v-if="!groupNames.length" class="inline-notification note">You are not a member of any user groups.</p>
			<p v-else>
				<span v-for="name in groupNames" :key="name" class="role-tag">{{ name }}</span>
			</p>

			<h3 class="subsection-title">Effective roles</h3>
			<p class="section-hint">Computed from your group membership.</p>
			<p v-if="!roleNames.length" class="inline-notification note">No roles via group membership.</p>
			<p v-else>
				<span v-for="name in roleNames" :key="name" class="role-tag">{{ name }}</span>
			</p>

			<h3 class="subsection-title">Effective permissions</h3>
			<p v-if="isSuperuser" class="inline-notification note">
				You have the <strong>superuser</strong> role via group membership — all permissions are granted.
			</p>
			<table v-if="permissionAudit.length" class="perm-audit-table">
				<thead>
					<tr>
						<th class="perm-status-col">Status</th>
						<th>Permission</th>
						<th>Granted via groups</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="row in permissionAudit" :key="row.name">
						<td class="perm-status-col">
							<Icon v-if="row.granted" icon="material-symbols:check-circle" class="perm-granted" />
							<Icon v-else icon="material-symbols:cancel" class="perm-denied" />
						</td>
						<td><code>{{ row.name }}</code></td>
						<td>
							<span v-if="isSuperuser && row.grantingGroups.length === 0" class="role-tag superuser-tag">superuser</span>
							<template v-else>
								<span v-for="gn in row.grantingGroups" :key="gn" class="role-tag">{{ gn }}</span>
								<span v-if="row.grantingGroups.length === 0" class="muted">—</span>
							</template>
						</td>
					</tr>
				</tbody>
			</table>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';

	const iconStrokeWidth = 2.5;

	const groupNames = ref([]);
	const roleNames = ref([]);
	const isSuperuser = ref(false);
	const permissions = ref([]);
	const loading = ref(false);
	const errorMessage = ref('');

	const permissionAudit = computed(() =>
		[...permissions.value]
			.sort((a, b) => a.name.localeCompare(b.name))
			.map((p) => ({
				name: p.name,
				granted: p.granted,
				grantingGroups: p.grantingGroups || [],
			})),
	);

	async function load() {
		loading.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.getMyPermissionsAudit({});
			groupNames.value = res.groupNames || [];
			roleNames.value = res.roleNames || [];
			isSuperuser.value = Boolean(res.isSuperuser);
			permissions.value = (res.permissions || []).map((row) => ({
				name: row.permission,
				granted: row.granted,
				grantingGroups: row.grantingGroups || [],
			}));
		} catch (e) {
			errorMessage.value = e.message || 'Failed to load permissions audit.';
		} finally {
			loading.value = false;
		}
	}

	onMounted(() => {
		load();
	});
</script>

<style scoped>
	.subsection-title {
		margin: 1.25rem 0 0.5rem;
		font-size: 1rem;
		font-weight: 600;
	}

	.subsection-title:first-of-type {
		margin-top: 0;
	}

	.section-hint {
		margin: 0 0 0.75rem;
		font-size: 0.88rem;
		opacity: 0.85;
	}

	.perm-audit-table {
		width: 100%;
		max-width: 48rem;
		margin: 0.5rem 0 1.5rem;
	}

	.perm-status-col {
		width: 3.5rem;
		text-align: center;
	}

	.perm-granted {
		color: var(--pico-ins-color, #2a7d2e);
		font-size: 1.25em;
		vertical-align: middle;
	}

	.perm-denied {
		color: var(--pico-del-color, #9e2a2a);
		font-size: 1.25em;
		vertical-align: middle;
		opacity: 0.5;
	}

	.role-tag {
		display: inline-block;
		margin: 0.1rem 0.25rem 0.1rem 0;
		padding: 0.1rem 0.4rem;
		font-size: 0.8em;
		border-radius: 3px;
		background: var(--pico-muted-border-color, rgba(0, 0, 0, 0.08));
	}

	.superuser-tag {
		font-style: italic;
		opacity: 0.75;
	}

	.muted {
		opacity: 0.5;
	}
</style>
