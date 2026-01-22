<template>
	<Section
		title="Campaign Details"
		subtitle="View and manage a single campaign"
		classes="campaign-details"
	>
		<template #toolbar>
			<button @click="goBack" class="neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to Campaigns
			</button>
			<button @click="refresh" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="!clientReady">
			<p>Loading...</p>
		</div>
		<div v-else-if="campaign">
			<div class="campaign-header">
				<div v-if="!editingCampaign" class="campaign-title">
					<h3>{{ campaign?.name || 'Unknown Campaign' }}</h3>
					<button @click="startEditingCampaign" class="neutral small" title="Edit Campaign Name">
						<Icon icon="mdi:pencil" />
						Edit
					</button>
				</div>
				<div v-else class="campaign-edit">
					<input
						v-model="editingName"
						class="campaign-name-input"
						@keyup.enter="saveCampaign"
						@keyup.esc="cancelEditingCampaign"
						@blur="handleBlur"
						ref="campaignNameInput"
					/>
					<button @click="saveCampaign" class="good small" :disabled="!editingName.trim()">
						<Icon icon="mdi:check" />
						Save
					</button>
					<button @click="cancelEditingCampaign" class="neutral small">
						<Icon icon="mdi:close" />
						Cancel
					</button>
				</div>
			</div>

			<p>ID: {{ campaignId }}</p>
			<p>Posts: {{ campaign?.postCount ?? 0 }}</p>
			<p>Last Post: {{ campaign?.lastPostDate || 'Never' }}</p>

			<div class="toolbar">
				<router-link class="button good" :to="{ name: 'postBox', query: { campaignId: campaignId } }">
					<Icon icon="jam:write-f" />
					Create Post
				</router-link>
			</div>

			&nbsp;

		</div>
	</Section>

			<div class="campaign-timeline">

				<Loading v-if="campaignPostsLoading" message="Loading campaign posts..." :centered="false" />

				<div v-else-if="campaignPosts.length === 0">
					<p class="inline-notification note">No posts found for this campaign.</p>
				</div>

				<div v-else class="posts-list">
					<div v-for="post in campaignPosts.slice(0, 5)" :key="post.id">

						<section class = "small">
							<PostPreview :post="post" />

							<div class="post-meta">
								<div class="post-status">
									<span :class="['annotation', getPostStatusClass(post)]">{{ getPostStatusText(post) }}</span>
								</div>
		
								<span class="post-date">{{ post.created }}</span>
								<button @click="viewPost(post)" class="neutral small">
									<Icon icon="mdi:eye" />
									View Details
								</button>
							</div>
						</section>
					</div>
				</div>
			</div>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted, computed, nextTick } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import Section from 'picocrank/vue/components/Section.vue';
	import Loading from './Loading.vue';

	const route = useRoute();
	const router = useRouter();
	const clientReady = ref(false)
	const campaignId = ref(0)
	const campaign = ref(null)
	const campaignPosts = ref([])
	const campaignPostsLoading = ref(true)
	const editingCampaign = ref(false)
	const editingName = ref('')
	const campaignNameInput = ref(null)
	const isSaving = ref(false)

	async function refresh() {
		if (!window.client || !campaignId.value) return

		// Load campaign details
		const res = await window.client.getCampaigns()
		const found = (res.campaigns || []).find(c => c.id === campaignId.value)
		campaign.value = found || null

		// Load posts for this campaign
		loadCampaignPosts()
	}

	async function loadCampaignPosts() {
		if (!window.client || !campaignId.value) return

		campaignPostsLoading.value = true
		try {
			const response = await window.client.getTimeline()
			const allPosts = response.posts || []
			// Filter posts to only include those for this campaign
			campaignPosts.value = allPosts.filter(post => post.campaignId === campaignId.value)
		} catch (error) {
			console.error('Error loading campaign posts:', error)
			campaignPosts.value = []
		} finally {
			campaignPostsLoading.value = false
		}
	}

	function goBack() {
		router.push({ name: 'campaigns' })
	}

	function startEditingCampaign() {
		editingName.value = campaign.value?.name || ''
		editingCampaign.value = true
		// Focus the input field after the next DOM update
		nextTick(() => {
			if (campaignNameInput.value) {
				campaignNameInput.value.focus()
				campaignNameInput.value.select()
			}
		})
	}

	function cancelEditingCampaign() {
		editingCampaign.value = false
		editingName.value = ''
	}

	function handleBlur(event) {
		// Don't cancel if we're in the process of saving
		if (isSaving.value) {
			return
		}
		// Don't cancel if the blur is caused by clicking a button
		// The relatedTarget will be the element receiving focus
		const relatedTarget = event.relatedTarget
		if (relatedTarget && (relatedTarget.tagName === 'BUTTON' || relatedTarget.closest('button'))) {
			return
		}
		cancelEditingCampaign()
	}

	async function saveCampaign() {
		if (!window.client || !campaign.value) return

		isSaving.value = true
		try {
			await window.client.updateCampaign({
				id: campaign.value.id,
				name: editingName.value,
				description: campaign.value.description || ''
			})

			// Update local campaign data
			campaign.value.name = editingName.value
			editingCampaign.value = false
			editingName.value = ''
		} catch (error) {
			console.error('Error updating campaign:', error)
			alert('Failed to update campaign: ' + error.message)
		} finally {
			isSaving.value = false
		}
	}

	function getPostStatusClass(post) {
		if (post.state === 'error') return 'bad';
		if (post.state === 'pending' || post.state === 'scheduled') return 'note';
		if (post.state === 'completed') return 'good';
		return '';
	}

	function getPostStatusText(post) {
		if (post.state === 'error') return 'Error';
		if (post.state === 'pending' || post.state === 'scheduled') return 'Scheduled';
		if (post.state === 'completed') return 'Completed';
		return 'Unknown';
	}

	function viewPost(post) {
		router.push({ name: 'postDetails', params: { id: post.id } });
	}

	onMounted(async () => {
		campaignId.value = parseInt(route.params.id, 10) || 0
		clientReady.value = true
		refresh()
	})
</script>

<style scoped>

.campaign-actions .good {
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.75rem 1rem;
	border-radius: 0.5rem;
	text-decoration: none;
	font-weight: 500;
	transition: all 0.2s ease;
}

.campaign-actions .good:hover {
	transform: translateY(-1px);
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

/* Campaign Editing Styles */
.campaign-header {
	margin-bottom: 1.5rem;
}

.campaign-title {
	display: flex;
	align-items: center;
	gap: 1rem;
	justify-content: space-between;
}

.campaign-title h3 {
	margin: 0;
	color: var(--text-primary, #333);
	font-size: 1.5rem;
}

.campaign-edit {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-wrap: wrap;
}

.campaign-name-input {
	padding: 0.5rem;
	border: 1px solid var(--border-color, #ddd);
	border-radius: 0.25rem;
	font-size: 1.5rem;
	font-weight: 500;
	min-width: 200px;
	background-color: var(--background-primary, white);
	color: var(--text-primary, #333);
}

.campaign-name-input:focus {
	outline: none;
	border-color: var(--primary-color, #007bff);
	box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}


.campaign-timeline h3 {
	margin: 0 0 1.5rem 0;
	color: var(--text-primary, #333);
	font-size: 1.3rem;
}

.posts-list {
	display: flex;
	flex-direction: column;
	gap: 1.5rem;
}

.post-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1rem;
}

.social-account {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-weight: 500;
	color: var(--text-primary, #333);
}

.post-status {
	font-weight: 500;
}

.post-content {
	margin-bottom: 1rem;
	line-height: 1.6;
	color: var(--text-primary, #333);
	white-space: pre-wrap;
}

.post-meta {
	display: flex;
	flex-grow: 0;
	justify-content: space-between;
	align-items: center;
	font-size: 0.9rem;
	color: var(--text-secondary, #666);
}

.post-date {
	font-style: italic;
}

.annotation {
	display: inline-block;
	padding: 0.25rem 0.5rem;
	border-radius: 0.25rem;
	font-size: 0.875rem;
	font-weight: 500;
	text-align: center;
	min-width: 60px;
}

.annotation.bad {
	background-color: #f8d7da;
	color: #721c24;
	border: 1px solid #f5c6cb;
}

.annotation.good {
	background-color: #d4edda;
	color: #155724;
	border: 1px solid #c3e6cb;
}

.annotation.note {
	background-color: #fff3cd;
	color: #856404;
	border: 1px solid #ffeaa7;
}

.campaign-update-btn {
	padding: 0.2rem 0.4rem;
	border-radius: 0.25rem;
	border: none;
	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: all 0.2s ease;
	background-color: #f0f0f0;
	color: #333;
	font-size: 0.8rem;
	min-width: 28px;
	height: 28px;
}

.campaign-update-btn:hover {
	background-color: #e0e0e0;
	transform: translateY(-1px);
	box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}
</style>
