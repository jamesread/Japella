<template>
	<Section
		title="Campaigns"
		subtitle="Manage your campaigns here. You can create, edit, and delete campaigns."
		classes="campaigns"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshCampaigns()" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button class="good" @click="createCampaign">
				<Icon icon="material-symbols:add-rounded" />
			</button>
		</template>

		<div v-if="!campaignList || campaignList.length === 0">
			<p class="inline-notification note">No campaigns available. Please create a new campaign.</p>
		</div>
		<table class = "data-table" v-else>
			<thead>
				<tr>
					<th class = "larger">Name</th>
					<th>Posts</th>
					<th>Last Post</th>
					<th class="actions" style="text-align: right">Actions</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="campaign in campaignList" :key="campaign.id">
					<td>
					<template v-if="!campaign.editing">
						<router-link :to="{ name: 'campaignDetails', params: { id: campaign.id } }">{{ campaign.name }}</router-link>
					</template>
					<input v-else
						class="editable"
						type="text"
						v-model="campaign.name"
						@keyup.enter="saveCampaign(campaign)"
						@keyup.esc="cancelEditing(campaign)"
						@blur="cancelEditing(campaign)"
					/>
					</td>
					<td>{{ campaign.postCount }}</td>
					<td>{{ campaign.lastPostDate }}</td>
					<td align="right">
					<button @click="usePost(campaign)" class="good">
							<Icon icon="jam:write-f" />
						</button>
					&nbsp;
					<button @click="openAccountsDialog(campaign)" class="neutral">
						<Icon icon="jam:users" /> {{ campaign.accountCount ?? 0 }}
					</button>
					</td>
				</tr>
			</tbody>
		</table>

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

	.larger {
		min-width: 400px;;
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
</style>

<script setup>
	import { waitForClient } from '../javascript/util.js'
	import { Icon } from '@iconify/vue';
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const campaignList = ref([]);
	const clientReady = ref(false)

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
    window.client.getCampaigns().then(res => {
        let campaigns = []

        for (const campaign of res.campaigns) {
            campaigns.push({
                id: campaign.id,
                name: campaign.name,
                postCount: campaign.postCount || 0,
                lastPostDate: campaign.lastPostDate || 'Never',
						accountCount: campaign.accountCount || 0,
                originalName: campaign.name, // Store original name for comparison
            });
        }

        campaignList.value = campaigns;
    }).catch(error => {
        console.error("Error fetching campaigns:", error);
    });
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
