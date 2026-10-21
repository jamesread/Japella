<template>
	<Section
		title="Campaigns"
		subtitle="Manage your campaigns here. You can create, edit, and delete campaigns."
		:icon="VolumeUpIcon"
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

		<Table
			v-else
			class="campaigns-table"
			:data="campaignList || []"
			:headers="tableHeaders"
			:show-pagination="true"
		>
			<template #empty>
				No campaigns available. Please create a new campaign.
			</template>
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

		<SocialAccountsSelectorDialog
			ref="accountsDialog"
			title="Select social accounts"
			subtitle="Choose which social accounts belong to this campaign."
			@confirmed="onAccountsConfirmed"
		/>
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
		font-size: 1em;
	}

	input[readonly] {
		background-color: transparent;
		border: 1px solid transparent;
	}
</style>

<script setup>
	import { waitForClient } from '../javascript/util.js'
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { EditIcon, UserMultiple02Icon, VolumeUpIcon } from '@hugeicons/core-free-icons';
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import Loading from './Loading.vue';
	import SocialAccountsSelectorDialog from './SocialAccountsSelectorDialog.vue';

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
	const accountsDialog = ref(null);
	const dialogCampaignId = ref(0);

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
	dialogCampaignId.value = campaign.id
	try {
		const memRes = await window.client.getCampaignSocialAccounts({ campaignId: campaign.id })
		accountsDialog.value?.open({ selectedIds: memRes.socialAccountIds || [] })
	} catch (e) {
		console.error('Failed to load campaign accounts', e)
		accountsDialog.value?.open({ selectedIds: [] })
	}
}

async function onAccountsConfirmed(selectedIds) {
	const campaignId = dialogCampaignId.value
	if (!window.client || !campaignId) return
	try {
		const memRes = await window.client.getCampaignSocialAccounts({ campaignId })
		const before = new Set(memRes.socialAccountIds || [])
		const after = new Set((selectedIds || []).map(Number))

		const adds = [...after].filter((id) => !before.has(id))
		const removes = [...before].filter((id) => !after.has(id))

		await Promise.all([
			...adds.map((id) => window.client.addSocialAccountToCampaign({ campaignId, socialAccountId: id })),
			...removes.map((id) => window.client.removeSocialAccountFromCampaign({ campaignId, socialAccountId: id })),
		])

		const row = campaignList.value.find((c) => c.id === campaignId)
		if (row) {
			row.accountCount = after.size
		}
	} catch (e) {
		console.error('Failed to save campaign accounts', e)
	} finally {
		dialogCampaignId.value = 0
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
