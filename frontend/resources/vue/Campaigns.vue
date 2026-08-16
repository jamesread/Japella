<template>
	<Section
		title="Campaigns"
		subtitle="Manage your campaigns here. You can create, edit, and delete campaigns."
		classes="campaigns"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshCampaigns()" :disabled="!clientReady" class="">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button class="good" @click="createCampaign">
				<Icon icon="material-symbols:add-rounded" />
			</button>
		</template>

		<Loading v-if="campaignsLoading" message="Loading campaigns..." :centered="true" />

		<div v-else-if="!campaignList || campaignList.length === 0">
			<p class="inline-notification note">No campaigns available. Please create a new campaign.</p>
		</div>
		<Table
			v-else
			class="campaigns-table"
			:data="campaignList"
			:headers="tableHeaders"
			:show-pagination="true"
		>
			<template #cell-name="{ row }">
				<template v-if="!row.editing">
					<router-link :to="{ name: 'campaignDetails', params: { id: row.id } }">
						{{ getCampaignDisplayName(row) }}
					</router-link>
				</template>
				<input
					v-else
					class="editable"
					type="text"
					v-model="row.name"
					@keyup.enter="saveCampaign(row)"
					@keyup.esc="cancelEditing(row)"
					@blur="cancelEditing(row)"
				/>
			</template>
			<template #cell-lastPostDate="{ row }">
				<span :title="getFullDate(row.lastPostDate)">{{ formatRelativeDate(row.lastPostDate) }}</span>
			</template>
			<template #cell-actions="{ row }">
				<div class="actions-cell">
					<button type="button" class="inline-icon neutral" @click="openAccountsDialog(row)">
						<HugeiconsIcon
							:icon="UserMultiple02Icon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>{{ row.accountCount ?? 0 }}</span>
					</button>
					<button
						type="button"
						class="inline-icon good"
						aria-label="Use campaign for post"
						@click="usePost(row)"
					>
						<HugeiconsIcon
							:icon="EditIcon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
					</button>
				</div>
			</template>
		</Table>

	<!-- Campaign Accounts Dialog -->
	<div v-if="showAccountsDialog" class="modal-overlay" @click.self="cancelAccountsDialog">
		<div class="modal">
			<h3>Select Social Accounts</h3>
			<div v-if="accountsLoading">
				<p>Loading accounts...</p>
			</div>
			<div v-else>
				<div v-if="allAccounts.length === 0">
					<p class="inline-notification note">No social accounts found.</p>
				</div>
				<div v-else class="account-list">
					<label v-for="acc in allAccounts" :key="acc.id" class="check-list">
						<input type="checkbox" v-model="acc.selected" />
						<span>
							<Icon :icon="acc.icon" />
							{{ acc.identity }}
						</span>
					</label>
				</div>
			</div>
			<div class="dialog-actions">
				<button class="neutral" @click="cancelAccountsDialog">Cancel</button>
				<button class="good" @click="saveAccountsDialog" :disabled="accountsSaving">Save</button>
			</div>
		</div>
	</div>
	</Section>
</template>

<style scoped>
    input[type="text"] {
		width: 95%;
	}

	.campaigns-table {
		width: 100%;
	}

	.actions-cell {
		display: flex;
		justify-content: flex-end;
		gap: 0.35rem;
	}

	input.editable {
		cursor: pointer;
		font-size: 1em;;
	}

	input[readonly] {
		background-color: transparent;
		border: 1px solid transparent;
	}

	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0,0,0,0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: #111;
		border: 1px solid #555;
		border-radius: 8px;
		padding: 1rem;
		width: 520px;
		max-width: 90vw;
	}

	.account-list {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: .5rem 1rem;
		margin: 1rem 0;
	}

	.dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: .5rem;
	}

	.check-list {
		display: inline-block;
		margin: 0.25em 0.5em 0.25em 0;
	}

	.check-list label,
	.check-list {
		cursor: pointer;
		display: inline-block;
	}

	.check-list label > span,
	.check-list > span {
		white-space: nowrap;
		border-radius: 0.5em;
		background-color: #f5f5f5;
		color: #333;
		border: 1px solid #e0e0e0;
		font-weight: 500;
		padding: 0.4em 0.6em;
		display: inline-flex;
		align-items: center;
		gap: .4em;
		user-select: none;
		transition: all 0.2s ease;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.check-list input[type="checkbox"]:checked + span {
		background-color: #e3f2fd;
		border-color: #2196f3;
		color: #1565c0;
		box-shadow: 0 2px 4px rgba(33, 150, 243, 0.2);
	}

	.check-list label:hover > span,
	.check-list:hover > span {
		background-color: #e8e8e8;
		border-color: #d0d0d0;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		transform: translateY(-1px);
	}

	.check-list input[type="checkbox"]:checked + span:hover {
		background-color: #bbdefb;
		border-color: #1976d2;
		box-shadow: 0 2px 4px rgba(33, 150, 243, 0.3);
	}

	@media (prefers-color-scheme: dark) {
		.check-list label > span,
		.check-list > span {
			background-color: #2a2a2a;
			color: #e0e0e0;
			border-color: #404040;
			box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
		}

		.check-list input[type="checkbox"]:checked + span {
			background-color: #1e3a5f;
			border-color: #2196f3;
			color: #90caf9;
			box-shadow: 0 2px 4px rgba(33, 150, 243, 0.3);
		}

		.check-list label:hover > span,
		.check-list:hover > span {
			background-color: #353535;
			border-color: #505050;
			box-shadow: 0 2px 4px rgba(0, 0, 0, 0.4);
		}

		.check-list input[type="checkbox"]:checked + span:hover {
			background-color: #2a4a6f;
			border-color: #42a5f5;
			box-shadow: 0 2px 4px rgba(33, 150, 243, 0.4);
		}
	}
</style>

<script setup>
	import { waitForClient } from '../javascript/util.js'
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { EditIcon, UserMultiple02Icon } from '@hugeicons/core-free-icons';
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import Loading from './Loading.vue';

	const iconStrokeWidth = 2.5;

	const tableHeaders = [
		{ key: 'name', label: 'Name', sortable: true },
		{ key: 'postCount', label: 'Posts', sortable: true, width: '5rem' },
		{ key: 'lastPostDate', label: 'Last Post', sortable: true, width: '8rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: '9rem' },
	];

	const campaignList = ref([]);
	const clientReady = ref(false);
	const campaignsLoading = ref(true);

// Accounts dialog state
const showAccountsDialog = ref(false)
const dialogCampaignId = ref(0)
const allAccounts = ref([])
const accountsLoading = ref(false)
const accountsSaving = ref(false)

	function usePost(campaign) {
		// Navigate to post box with preselected campaign
		window.router.push({ path: '/post', query: { campaignId: campaign.id } });
	}

	function startEditing(campaign) {
		campaign.editing = true;
	}

	function cancelEditing(campaign) {
		campaign.editing = false;
	}

	function saveCampaign(campaign) {
		window.client.updateCampaign({
			id: campaign.id,
			name: campaign.name,
		}).then(() => {
			refreshCampaigns();
		}).catch(error => {
			console.error("Error editing campaign:", error);
		});
	}

	function createCampaign() {
		window.client.createCampaign().then(() => {
			refreshCampaigns();
		}).catch(error => {
			console.error("Error creating campaign:", error);
		});
	}

function refreshCampaigns() {
    campaignsLoading.value = true;
    window.client.getCampaigns().then(res => {
        let campaigns = []

        for (const campaign of res.campaigns) {
            campaigns.push({
                id: campaign.id,
                name: campaign.name,
                postCount: campaign.postCount || 0,
                lastPostDate: campaign.lastPostDate || null,
						accountCount: campaign.accountCount || 0,
                originalName: campaign.name, // Store original name for comparison
            });
        }

        campaignList.value = campaigns;
    }).catch(error => {
        console.error("Error fetching campaigns:", error);
        campaignList.value = [];
    }).finally(() => {
        campaignsLoading.value = false;
    });
}

function formatRelativeDate(dateString) {
    if (!dateString || dateString === 'Never') {
        return 'Never';
    }

    try {
        const date = new Date(dateString);
        if (isNaN(date.getTime())) {
            return 'Invalid date';
        }

        const now = new Date();
        const diffMs = now - date;
        const diffSeconds = Math.floor(diffMs / 1000);
        const diffMinutes = Math.floor(diffSeconds / 60);
        const diffHours = Math.floor(diffMinutes / 60);
        const diffDays = Math.floor(diffHours / 24);
        const diffWeeks = Math.floor(diffDays / 7);
        const diffMonths = Math.floor(diffDays / 30);
        const diffYears = Math.floor(diffDays / 365);

        if (diffSeconds < 60) {
            return 'Just now';
        } else if (diffMinutes < 60) {
            return `${diffMinutes} minute${diffMinutes !== 1 ? 's' : ''} ago`;
        } else if (diffHours < 24) {
            return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
        } else if (diffDays < 7) {
            return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
        } else if (diffWeeks < 4) {
            return `${diffWeeks} week${diffWeeks !== 1 ? 's' : ''} ago`;
        } else if (diffMonths < 12) {
            return `${diffMonths} month${diffMonths !== 1 ? 's' : ''} ago`;
        } else {
            return `${diffYears} year${diffYears !== 1 ? 's' : ''} ago`;
        }
    } catch (error) {
        console.error('Error formatting date:', error);
        return 'Invalid date';
    }
}

function getFullDate(dateString) {
    if (!dateString || dateString === 'Never') {
        return 'No posts yet';
    }

    try {
        const date = new Date(dateString);
        if (isNaN(date.getTime())) {
            return 'Invalid date';
        }

        return date.toLocaleString('en-US', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            timeZoneName: 'short'
        });
    } catch (error) {
        console.error('Error formatting full date:', error);
        return 'Invalid date';
    }
}

function getCampaignDisplayName(campaign) {
    if (!campaign.name || campaign.name.trim() === '') {
        return `untitled campaign ${campaign.id}`;
    }
    return campaign.name;
}

async function openAccountsDialog(campaign) {
    if (!window.client) return
    showAccountsDialog.value = true
    dialogCampaignId.value = campaign.id
    accountsLoading.value = true

    try {
        // Load all social accounts
        const accRes = await window.client.getSocialAccounts({})
        const accounts = accRes.accounts || []

        // Load campaign memberships
        const memRes = await window.client.getCampaignSocialAccounts({ campaignId: campaign.id })
        const memberIds = new Set(memRes.socialAccountIds || [])

        allAccounts.value = accounts.map(a => ({ id: a.id, identity: a.identity, icon: a.icon, selected: memberIds.has(a.id) }))
    } catch (e) {
        console.error('Failed to load campaign accounts', e)
    } finally {
        accountsLoading.value = false
    }
}

function cancelAccountsDialog() {
    showAccountsDialog.value = false
    allAccounts.value = []
    dialogCampaignId.value = 0
}

async function saveAccountsDialog() {
    if (!window.client) return
    accountsSaving.value = true
    try {
        const memRes = await window.client.getCampaignSocialAccounts({ campaignId: dialogCampaignId.value })
        const before = new Set(memRes.socialAccountIds || [])
        const after = new Set(allAccounts.value.filter(a => a.selected).map(a => a.id))

        // Determine adds and removes
        const adds = [...after].filter(id => !before.has(id))
        const removes = [...before].filter(id => !after.has(id))

        // Apply changes
        await Promise.all([
            ...adds.map(id => window.client.addSocialAccountToCampaign({ campaignId: dialogCampaignId.value, socialAccountId: id })),
            ...removes.map(id => window.client.removeSocialAccountFromCampaign({ campaignId: dialogCampaignId.value, socialAccountId: id }))
        ])

        // Update count in table
        const row = campaignList.value.find(c => c.id === dialogCampaignId.value)
        if (row) {
            row.accountCount = allAccounts.value.filter(a => a.selected).length
        }

        cancelAccountsDialog()
    } catch (e) {
        console.error('Failed to save campaign accounts', e)
    } finally {
        accountsSaving.value = false
    }
}

	function deleteCampaign(campaignId) {
		window.client.deleteCampaign({
			id: campaignId
		}).then(() => {
			refreshCampaigns();
		}).catch(error => {
			console.error("Error deleting campaign:", error);
		});
	}

function viewCampaign(campaign) {
	window.router.push({ name: 'campaignDetails', params: { id: campaign.id } })
}

    onMounted(async () => {
		await waitForClient()
		clientReady.value = true;
	    refreshCampaigns();
	});

</script>
