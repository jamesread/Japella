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
		<table v-else>
			<thead>
				<tr>
					<th>Name</th>
					<th>Posts</th>
					<th>Last Post</th>
					<th class="actions" style="text-align: right">Actions</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="campaign in campaignList" :key="campaign.id">
					<td>
						<input
							class="editable"
							type="text"
							v-model="campaign.name"
							:readonly="!campaign.editing"
							@click="startEditing(campaign)"
							@keyup.enter="saveCampaign(campaign)"
							@keyup.esc="cancelEditing(campaign)"
							@blur="cancelEditing(campaign)"
						/>
					</td>
					<td>{{ campaign.postCount }}</td>
					<td>{{ campaign.lastPostDate }}</td>
					<td align="right">
						<button @click="goToCannedPosts()" class="neutral">
							<Icon icon="jam:box" />
						</button>
						&nbsp;
						<button @click="usePost(p)" class="good">
							<Icon icon="jam:write-f" />
						</button>
						&nbsp;
						<button @click="deleteCampaign(campaign.id)" class="bad">
							<Icon icon="material-symbols:delete" />
						</button>
					</td>
				</tr>
			</tbody>
		</table>
	</Section>
</template>

<style scoped>
    input[type="text"] {
		width: 95%;
	}

	input.editable {
		cursor: pointer;
		font-size: 1em;;
	}

	input[readonly] {
		background-color: transparent;
		border: 1px solid transparent;
	}
</style>

<script setup>
	import { waitForClient } from '../javascript/util.js'
	import { Icon } from '@iconify/vue';
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const campaignList = ref([]);
	const clientReady = ref(false)

	function goToCannedPosts() {
		// Navigate to canned posts using router
		window.router.push('/canned-posts');
	}

	function usePost(p) {
		// Navigate to post box using router
		window.router.push('/post');
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
					originalName: campaign.name, // Store original name for comparison
				});
			}

			campaignList.value = campaigns;
		}).catch(error => {
			console.error("Error fetching campaigns:", error);
		});
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

    onMounted(async () => {
		await waitForClient()
		clientReady.value = true;
	    refreshCampaigns();
	});

</script>
