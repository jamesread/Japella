<template>
	<Section
		title="Post Details"
		:subtitle="pageSubtitle"
		classes="post-details"
		:padding="true"
	>
		<template #toolbar>
			<button @click="goBack" class="neutral">
				<Icon icon="material-symbols:arrow-back" />
				{{ fromApprovals ? 'Back to Approvals' : 'Back to Timeline' }}
			</button>
		</template>

		<div v-if="loading" class="loading-container">
			<p class="inline-notification note">Loading post details...</p>
		</div>

		<div v-else-if="errorMessage" class="error-container">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-else-if="post" class="post-details-content">
			<div v-if="actionMessage" class="inline-notification" :class="actionMessageType">{{ actionMessage }}</div>

			<div v-if="isPendingApproval && canEdit" class="edit-section">
				<h3>Edit content</h3>
				<p class="edit-hint">
					Save changes without advancing approval. Approve separately when ready for the next stage.
				</p>
				<textarea
					v-model="editContent"
					rows="10"
					class="edit-content"
					:disabled="saving"
					placeholder="Post content"
				/>
				<div class="edit-actions">
					<button
						type="button"
						class="inline-icon good"
						:disabled="saving || !canSaveEdit"
						@click="savePendingContent"
					>
						<HugeiconsIcon
							v-if="saving"
							:icon="Loading01Icon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<HugeiconsIcon
							v-else
							:icon="SaveIcon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>{{ saving ? 'Saving…' : 'Save' }}</span>
					</button>
					<button
						type="button"
						class="neutral"
						:disabled="saving || editContent === (post.content || '')"
						@click="resetEditContent"
					>
						Reset
					</button>
				</div>
			</div>
			<PostPreview v-else :post="post" />

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
						<span :class="['annotation', statusClass(post)]">{{ statusText(post) }}</span>
					</dd>

					<template v-if="isPendingApproval">
						<dt>Approval stage</dt>
						<dd>{{ (approvalStage ?? 0) + 1 }}</dd>
						<dt v-if="accountPolicyName">Policy</dt>
						<dd v-if="accountPolicyName">{{ accountPolicyName }}</dd>
						<dt v-if="waitingOn">Waiting on</dt>
						<dd v-if="waitingOn">{{ waitingOn }}</dd>
						<dt v-if="submittedByUsername">Submitted by</dt>
						<dd v-if="submittedByUsername">{{ submittedByUsername }}</dd>
					</template>
				</dl>
			</div>

			<div class="post-actions-section">
				<h3>Actions</h3>
				<div role="toolbar" class="action-buttons">
					<button
						v-if="isPendingApproval && canApprove"
						type="button"
						class="inline-icon good"
						:disabled="acting"
						@click="approvePost"
					>
						<HugeiconsIcon
							v-if="acting"
							:icon="Loading01Icon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<HugeiconsIcon
							v-else
							:icon="CheckmarkCircle01Icon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>{{ acting ? 'Approving…' : 'Approve' }}</span>
					</button>
					<button
						v-if="isPendingApproval && canReject"
						type="button"
						class="inline-icon bad"
						:disabled="acting"
						@click="rejectPost"
					>
						<HugeiconsIcon
							:icon="CancelCircleIcon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>Reject</span>
					</button>
					<button v-if="post.state === 'error'" @click="retryPost" class="good" :disabled="retrying">
						<Icon v-if="retrying" icon="eos-icons:loading" />
						<Icon v-else icon="mdi:refresh" />
						{{ retrying ? 'Retrying...' : 'Retry Post' }}
					</button>
					<button
						type="button"
						class="inline-icon bad"
						:disabled="acting || saving"
						@click="forgetPost"
					>
						<HugeiconsIcon
							:icon="Delete02Icon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>Forget</span>
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
	import { ref, onMounted, computed, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		CancelCircleIcon,
		CheckmarkCircle01Icon,
		Delete02Icon,
		Loading01Icon,
		SaveIcon,
	} from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import PostPreview from './PostPreview.vue';

	const iconStrokeWidth = 2.5;

	const route = useRoute();
	const router = useRouter();

	const post = ref(null);
	const loading = ref(true);
	const errorMessage = ref('');
	const retrying = ref(false);
	const saving = ref(false);
	const acting = ref(false);
	const editContent = ref('');
	const actionMessage = ref('');
	const actionMessageType = ref('');

	const canApprove = ref(false);
	const canReject = ref(false);
	const canEdit = ref(false);
	const approvalStage = ref(0);
	const accountPolicyName = ref('');
	const waitingOn = ref('');
	const submittedByUsername = ref('');

	const showCampaignDialog = ref(false);
	const campaigns = ref([]);
	const selectedCampaignId = ref(0);
	const campaignsLoading = ref(false);
	const campaignsSaving = ref(false);
	const campaignMessage = ref('');
	const campaignMessageType = ref('');

	const fromApprovals = computed(() => route.query.from === 'approvals');
	const isPendingApproval = computed(() => post.value?.state === 'pending_approval');
	const canSaveEdit = computed(() => {
		const next = editContent.value.trim();
		return next.length > 0 && next !== (post.value?.content || '').trim();
	});
	const pageSubtitle = computed(() => {
		if (isPendingApproval.value && canEdit.value) {
			return 'Edit and save this held post before approving the current stage.';
		}
		return 'Detailed view of the selected post.';
	});

	function statusClass(p) {
		if (p.state === 'error') return 'bad';
		if (p.state === 'pending_approval') return 'note';
		if (p.state === 'rejected') return 'bad';
		if (p.state === 'pending' || p.state === 'scheduled') return 'note';
		if (p.state === 'completed') return 'good';
		return '';
	}

	function statusText(p) {
		if (p.state === 'error') return 'Error';
		if (p.state === 'pending_approval') return 'Pending approval';
		if (p.state === 'rejected') return 'Rejected';
		if (p.state === 'pending' || p.state === 'scheduled') return 'Scheduled';
		if (p.state === 'completed') return 'Completed';
		return p.state || 'Unknown';
	}

	function applyGetPostResponse(res) {
		post.value = res.post || null;
		canApprove.value = Boolean(res.canApprove);
		canReject.value = Boolean(res.canReject);
		canEdit.value = Boolean(res.canEdit);
		approvalStage.value = res.approvalStage ?? 0;
		accountPolicyName.value = res.accountPolicyName || '';
		waitingOn.value = res.waitingOn || '';
		submittedByUsername.value = res.submittedByUsername || '';
		editContent.value = res.post?.content || '';
	}

	async function loadPostDetails() {
		const postId = parseInt(String(route.params.id), 10);
		if (!Number.isFinite(postId) || postId <= 0) {
			errorMessage.value = 'No post ID provided';
			loading.value = false;
			return;
		}

		try {
			loading.value = true;
			errorMessage.value = '';
			actionMessage.value = '';

			const response = await window.client.getPost({ postId });
			if (response.post) {
				applyGetPostResponse(response);
			} else {
				errorMessage.value = 'Post not found';
				post.value = null;
			}
		} catch (error) {
			console.error('Error loading post details:', error);
			errorMessage.value = 'Failed to load post details: ' + error.message;
			post.value = null;
		} finally {
			loading.value = false;
		}
	}

	function resetEditContent() {
		editContent.value = post.value?.content || '';
	}

	async function savePendingContent() {
		if (!post.value || !canSaveEdit.value) return;
		saving.value = true;
		actionMessage.value = '';
		actionMessageType.value = '';
		try {
			const res = await window.client.updatePendingPost({
				postId: post.value.id,
				content: editContent.value.trim(),
			});
			if (res.post) {
				post.value = res.post;
				editContent.value = res.post.content || '';
			}
			actionMessage.value = res.standardResponse?.message || 'Post content saved.';
			actionMessageType.value = 'good';
		} catch (error) {
			console.error('Error saving pending post:', error);
			actionMessage.value = error.message || 'Failed to save post content.';
			actionMessageType.value = 'error';
		} finally {
			saving.value = false;
		}
	}

	async function approvePost() {
		if (!post.value || !canApprove.value) return;
		if (!confirm('Approve this post for the current stage?')) return;
		acting.value = true;
		actionMessage.value = '';
		try {
			const res = await window.client.approvePost({ postId: post.value.id });
			actionMessage.value = res.standardResponse?.message || 'Approved.';
			actionMessageType.value = 'good';
			if (fromApprovals.value) {
				router.push({ name: 'approvals' });
				return;
			}
			await loadPostDetails();
		} catch (error) {
			actionMessage.value = error.message || 'Failed to approve post.';
			actionMessageType.value = 'error';
		} finally {
			acting.value = false;
		}
	}

	async function rejectPost() {
		if (!post.value || !canReject.value) return;
		const reason = window.prompt('Optional rejection reason:', '') ?? null;
		if (reason === null) return;
		acting.value = true;
		actionMessage.value = '';
		try {
			const res = await window.client.rejectPost({ postId: post.value.id, reason: reason.trim() });
			actionMessage.value = res.standardResponse?.message || 'Rejected.';
			actionMessageType.value = 'good';
			if (fromApprovals.value) {
				router.push({ name: 'approvals' });
				return;
			}
			await loadPostDetails();
		} catch (error) {
			actionMessage.value = error.message || 'Failed to reject post.';
			actionMessageType.value = 'error';
		} finally {
			acting.value = false;
		}
	}

	function goBack() {
		if (fromApprovals.value) {
			router.push({ name: 'approvals' });
			return;
		}
		router.push({ name: 'timeline' });
	}

	async function updateCampaign() {
		if (!post.value) return;

		selectedCampaignId.value = post.value.campaignId || 0;
		showCampaignDialog.value = true;
		campaignMessage.value = '';
		campaignMessageType.value = '';

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

			post.value.campaignId = selectedCampaignId.value;
			const campaign = campaigns.value.find(c => c.id === selectedCampaignId.value);
			post.value.campaignName = campaign ? campaign.name : '';

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

	async function forgetPost() {
		if (!post.value) return;

		if (!confirm(`Are you sure you want to forget (delete) this post?\n\n"${post.value.content.substring(0, 50)}${post.value.content.length > 50 ? '...' : ''}"`)) {
			return;
		}

		try {
			await window.client.forgetPost({
				postId: post.value.id
			});
			goBack();
		} catch (error) {
			console.error('Error forgetting post:', error);
			alert('Failed to forget post: ' + error.message);
		}
	}

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
			post.value = response.postStatus;
		} catch (error) {
			console.error('Error retrying post:', error);
			alert('Failed to retry post: ' + error.message);
		} finally {
			retrying.value = false;
		}
	}

	watch(() => route.params.id, () => {
		loadPostDetails();
	});

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

.edit-section {
	margin-bottom: 1.5rem;
	max-width: 48rem;
}

.edit-section h3 {
	margin: 0 0 0.35rem;
	font-size: 1.1rem;
}

.edit-hint {
	margin: 0 0 0.75rem;
	font-size: 0.9rem;
	opacity: 0.85;
}

.edit-content {
	width: 100%;
	box-sizing: border-box;
	min-height: 12rem;
	padding: 0.75rem;
	font: inherit;
	line-height: 1.45;
	resize: vertical;
}

.edit-actions {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
	margin-top: 0.75rem;
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
