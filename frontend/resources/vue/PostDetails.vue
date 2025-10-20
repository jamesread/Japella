<template>
	<Section
		title="Post Details"
		subtitle="Detailed view of the selected post."
		classes="post-details"
		:padding="true"
	>
		<template #toolbar>
			<button @click="goBack" class="neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to Timeline
			</button>
		</template>

		<div v-if="loading" class="loading-container">
			<p class="inline-notification note">Loading post details...</p>
		</div>

		<div v-else-if="errorMessage" class="error-container">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-else-if="post" class="post-details-content">
			<PostPreview :post="post" />

			<div class="post-metadata">
				<dl>
					<dt>Created</dt>
					<dd>{{ post.created }}</dd>

					<dt v-if="post.campaignId && post.campaignId !== 0">Campaign</dt>
					<dd v-if="post.campaignId && post.campaignId !== 0" class="campaign-field">
						<div class="campaign-info">
							<router-link :to="{ name: 'campaignDetails', params: { id: post.campaignId } }">
								{{ post.campaignName || 'Unknown Campaign' }}
							</router-link>
						</div>
						<button @click="updateCampaign" class="neutral small campaign-update-btn" title="Update Campaign">
							<Icon icon="mdi:folder-edit" />
						</button>
					</dd>

					<dt v-if="post.postUrl">Post URL</dt>
					<dd v-if="post.postUrl">
						<a :href="post.postUrl" target="_blank" class="post-url">
							{{ post.postUrl }}
							<Icon icon="material-symbols:open-in-new" />
						</a>
					</dd>

					<dt>Post ID</dt>
					<dd>{{ post.id }}</dd>

					<dt v-if="post.state">State</dt>
					<dd v-if="post.state">
						<span :class="['annotation', statusClass(post)]">{{ post.state }}</span>
					</dd>

				</dl>
			</div>

			<div class="post-actions-section">
				<h3>Actions</h3>
				<div class="action-buttons">
					<button v-if="post.state === 'error'" @click="retryPost" class="good" :disabled="retrying">
						<Icon v-if="retrying" icon="eos-icons:loading" />
						<Icon v-else icon="mdi:refresh" />
						{{ retrying ? 'Retrying...' : 'Retry Post' }}
					</button>
					<button @click="forgetPost" class="bad">
						<Icon icon="mdi:delete" />
						Forget Post
					</button>
				</div>
			</div>
		</div>

		<div v-else class="no-post-container">
			<p class="inline-notification note">No post selected or post not found.</p>
		</div>
	</Section>

	<!-- Campaign Update Dialog -->
	<div v-if="showCampaignDialog" class="modal-overlay" @click.self="cancelCampaignDialog">
		<div class="modal">
			<h3>Update Post Campaign</h3>
			<div class="form-group">
				<label for="campaign-select">Select Campaign:</label>
				<select id="campaign-select" v-model="selectedCampaignId" :disabled="campaignsLoading">
					<option value="0">No Campaign</option>
					<option v-for="campaign in campaigns" :key="campaign.id" :value="campaign.id">
						{{ campaign.name }}
					</option>
				</select>
			</div>
			<div class="dialog-actions">
				<button class="neutral" @click="cancelCampaignDialog" :disabled="campaignsSaving">Cancel</button>
				<button class="good" @click="updatePostCampaign" :disabled="campaignsSaving || campaignsLoading">
					<Icon v-if="campaignsSaving" icon="eos-icons:loading" width="16" height="16" />
					<span>{{ campaignsSaving ? 'Updating...' : 'Update Campaign' }}</span>
				</button>
			</div>
			<div v-if="campaignMessage" class="campaign-message" :class="campaignMessageType">
				{{ campaignMessage }}
			</div>
		</div>
	</div>
</template>

<script setup>
	import { ref, onMounted, computed } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import PostPreview from './PostPreview.vue';

	const route = useRoute();
	const router = useRouter();

	const post = ref(null);
	const loading = ref(true);
	const errorMessage = ref('');
	const retrying = ref(false);

	// Campaign dialog state
	const showCampaignDialog = ref(false);
	const campaigns = ref([]);
	const selectedCampaignId = ref(0);
	const campaignsLoading = ref(false);
	const campaignsSaving = ref(false);
	const campaignMessage = ref('');
	const campaignMessageType = ref('');

	function statusClass(post) {
		if (post.state === 'error') return 'bad';
		if (post.state === 'pending' || post.state === 'scheduled') return 'note';
		if (post.state === 'completed') return 'good';
		return '';
	}

	function statusText(post) {
		if (post.state === 'error') return 'Error';
		if (post.state === 'pending' || post.state === 'scheduled') return 'Scheduled';
		if (post.state === 'completed') return 'Completed';
		return 'Unknown';
	}

	async function loadPostDetails() {
		const postId = route.params.id;
		if (!postId) {
			errorMessage.value = 'No post ID provided';
			loading.value = false;
			return;
		}

		try {
			loading.value = true;
			errorMessage.value = '';

			// Get timeline to find the specific post
			const response = await window.client.getTimeline();
			const posts = response.posts || [];
			const foundPost = posts.find(p => p.id == postId);

			if (foundPost) {
				post.value = foundPost;
			} else {
				errorMessage.value = 'Post not found';
			}
		} catch (error) {
			console.error('Error loading post details:', error);
			errorMessage.value = 'Failed to load post details: ' + error.message;
		} finally {
			loading.value = false;
		}
	}

	function goBack() {
		router.push({ name: 'timeline' });
	}

	// Campaign dialog functions
	async function updateCampaign() {
		if (!post.value) return;
		
		selectedCampaignId.value = post.value.campaignId || 0;
		showCampaignDialog.value = true;
		campaignMessage.value = '';
		campaignMessageType.value = '';
		
		// Load campaigns
		campaignsLoading.value = true;
		try {
			const response = await window.client.getCampaigns();
			campaigns.value = response.campaigns || [];
		} catch (error) {
			console.error('Error loading campaigns:', error);
			campaignMessage.value = 'Failed to load campaigns';
			campaignMessageType.value = 'bad';
		} finally {
			campaignsLoading.value = false;
		}
	}

	function cancelCampaignDialog() {
		showCampaignDialog.value = false;
		selectedCampaignId.value = 0;
		campaignMessage.value = '';
		campaignMessageType.value = '';
	}

	async function updatePostCampaign() {
		if (!post.value) return;
		
		campaignsSaving.value = true;
		campaignMessage.value = '';
		campaignMessageType.value = '';
		
		try {
			await window.client.updatePostCampaign({
				postId: post.value.id,
				campaignId: selectedCampaignId.value
			});
			
			campaignMessage.value = 'Campaign updated successfully';
			campaignMessageType.value = 'good';
			
			// Update the post
			post.value.campaignId = selectedCampaignId.value;
			const campaign = campaigns.value.find(c => c.id === selectedCampaignId.value);
			post.value.campaignName = campaign ? campaign.name : '';
			
			// Close dialog after a short delay
			setTimeout(() => {
				cancelCampaignDialog();
			}, 1500);
			
		} catch (error) {
			console.error('Error updating post campaign:', error);
			campaignMessage.value = 'Failed to update campaign: ' + error.message;
			campaignMessageType.value = 'bad';
		} finally {
			campaignsSaving.value = false;
		}
	}

	// Post forget function
	async function forgetPost() {
		if (!post.value) return;
		
		if (!confirm(`Are you sure you want to forget (delete) this post?\n\n"${post.value.content.substring(0, 50)}${post.value.content.length > 50 ? '...' : ''}"`)) {
			return;
		}
		
		try {
			await window.client.forgetPost({
				postId: post.value.id
			});
			
			console.log('Post forgotten successfully');
			// Navigate back to timeline
			router.push({ name: 'timeline' });
		} catch (error) {
			console.error('Error forgetting post:', error);
			alert('Failed to forget post: ' + error.message);
		}
	}

	// Post retry function
	async function retryPost() {
		if (!post.value) return;
		
		if (!confirm(`Are you sure you want to retry posting this?\n\n"${post.value.content.substring(0, 50)}${post.value.content.length > 50 ? '...' : ''}"`)) {
			return;
		}
		
		retrying.value = true;
		
		try {
			const response = await window.client.retryPost({
				postId: post.value.id
			});
			
			// Update the post with the retry result
			post.value = response.postStatus;
			
			console.log('Post retry completed:', response.standardResponse.message);
			
		} catch (error) {
			console.error('Error retrying post:', error);
			alert('Failed to retry post: ' + error.message);
		} finally {
			retrying.value = false;
		}
	}

	onMounted(async () => {
		await waitForClient();
		loadPostDetails();
	});
</script>

<style scoped>
.loading-container,
.error-container,
.no-post-container {
	text-align: center;
	padding: 2rem;
}

.post-metadata {
	margin-bottom: 2rem;
}

.campaign-field {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	justify-content: space-between;
}

.campaign-info {
	flex: 1;
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

.post-url {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	color: var(--link-color, #007bff);
	text-decoration: none;
}

.post-url:hover {
	text-decoration: underline;
}

.post-actions-section h3 {
	margin: 0 0 1rem 0;
	color: var(--text-primary, #333);
	font-size: 1.2rem;
}

.action-buttons {
	display: flex;
	gap: 1rem;
	flex-wrap: wrap;
}

.action-buttons button {
	padding: 0.75rem 1rem;
	border-radius: 0.5rem;
	border: none;
	cursor: pointer;
	font-size: 0.9rem;
	display: flex;
	align-items: center;
	gap: 0.5rem;
	transition: all 0.2s ease;
	font-weight: 500;
}

.action-buttons button:hover:not(:disabled) {
	transform: translateY(-1px);
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.action-buttons button:disabled {
	opacity: 0.6;
	cursor: not-allowed;
	transform: none;
}

.action-buttons button.neutral {
	background-color: #6c757d;
	color: white;
}

.action-buttons button.neutral:hover:not(:disabled) {
	background-color: #5a6268;
}

.action-buttons button.good {
	background-color: #28a745;
	color: white;
}

.action-buttons button.good:hover:not(:disabled) {
	background-color: #218838;
}

.action-buttons button.bad {
	background-color: #dc3545;
	color: white;
}

.action-buttons button.bad:hover:not(:disabled) {
	background-color: #c82333;
}

/* Modal styles */
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background-color: rgba(0, 0, 0, 0.5);
	display: flex;
	justify-content: center;
	align-items: center;
	z-index: 1000;
}

.modal {
	background: var(--background-primary, white);
	padding: 2rem;
	border-radius: 0.5rem;
	box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	min-width: 400px;
	max-width: 500px;
}

.modal h3 {
	margin: 0 0 1rem 0;
	color: var(--text-primary, #333);
}

.form-group {
	margin-bottom: 1rem;
}

.form-group label {
	display: block;
	margin-bottom: 0.5rem;
	font-weight: 500;
	color: var(--text-primary, #333);
}

.form-group select {
	width: 100%;
	padding: 0.5rem;
	border: 1px solid var(--border-color, #ddd);
	border-radius: 0.25rem;
	font-size: 1rem;
	background-color: var(--background-primary, white);
	color: var(--text-primary, #333);
}

.dialog-actions {
	display: flex;
	gap: 0.5rem;
	justify-content: flex-end;
	margin-top: 1.5rem;
}

.dialog-actions button {
	padding: 0.5rem 1rem;
	border-radius: 0.25rem;
	border: none;
	cursor: pointer;
	font-size: 0.9rem;
	display: flex;
	align-items: center;
	gap: 0.5rem;
	transition: all 0.2s ease;
}

.dialog-actions button:disabled {
	opacity: 0.6;
	cursor: not-allowed;
}

.dialog-actions button.neutral {
	background-color: var(--button-neutral-bg, #f0f0f0);
	color: var(--text-primary, #333);
}

.dialog-actions button.neutral:hover:not(:disabled) {
	background-color: var(--button-neutral-hover, #e0e0e0);
}

.dialog-actions button.good {
	background-color: #4CAF50;
	color: white;
}

.dialog-actions button.good:hover:not(:disabled) {
	background-color: #45a049;
}

.campaign-message {
	padding: 0.75rem;
	border-radius: 0.25rem;
	margin-top: 1rem;
	font-size: 0.9rem;
}

.campaign-message.good {
	background-color: #d4edda;
	color: #155724;
	border: 1px solid #c3e6cb;
}

.campaign-message.bad {
	background-color: #f8d7da;
	color: #721c24;
	border: 1px solid #f5c6cb;
}
</style>
