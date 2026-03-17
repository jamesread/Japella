<template>
	<Section
		title="Timeline"
		subtitle="This shows the latest posts from your social accounts."
		classes="timeline"
		:padding="false"
	>
		<template #toolbar>
			<div class="toolbar-controls">
				<div class="filters">
					<select v-model="selectedSocialAccountId" @change="applyFilters" class="filter-select">
						<option value="">All Social Accounts</option>
						<option v-for="account in socialAccounts" :key="account.id" :value="account.id">
							{{ account.identity }}
						</option>
					</select>
					<select v-model="selectedCampaignFilterId" @change="applyFilters" class="filter-select">
						<option value="">All Campaigns</option>
						<option v-for="campaign in campaigns" :key="campaign.id" :value="campaign.id">
							{{ campaign.name }}
						</option>
					</select>
					<button v-if="hasActiveFilters" @click="clearFilters" class="neutral small" title="Clear Filters">
						<Icon icon="mdi:filter-remove" width="16" height="16" />
						Clear
					</button>
				</div>
				<button @click="refreshTimeline" :disabled="!clientReady" class="neutral">
					<Icon icon="material-symbols:refresh" />
				</button>
			</div>
		</template>

		<Loading v-if="timelineLoading" message="Loading timeline..." :centered="true" />

		<template v-else>
			<div v-if="filteredTimeline.length === 0">
				<p class="inline-notification note">No posts available.</p>
			</div>
			<template v-else>
				<table class = "data-table">
					<thead>
						<tr>
							<th>Social Account</th>
							<th>Campaign</th>
							<th>Content</th>
							<th>Date</th>
							<th>Status</th>
							<th class="actions" style="text-align: right">Actions</th>
						</tr>
					</thead>
					<tbody>
						<tr v-if="!clientReady">
							<td colspan="6">Loading timeline...</td>
						</tr>
						<tr v-else v-for="post in pagedTimeline" :key="post.id">
							<td>
								<router-link :to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }" class="social-account">
									<Icon :icon="post.socialAccountIcon" />
									{{ post.socialAccountIdentity }}
								</router-link>
							</td>
							<td>
								<div v-if="post.campaignId != 0">
									<router-link :to="{ name: 'campaignDetails', params: { id: post.campaignId } }">
										{{ post.campaignName }}
									</router-link>
								</div>
								<div v-else class="no-campaign">
									<span @click="openCampaignDialog(post)" class="no-campaign-text" title="Assign Campaign">None</span>
									<button @click="openCampaignDialog(post)" class="neutral small" title="Assign Campaign">
										<Icon icon="mdi:folder-edit" width="14" height="14" />
									</button>
								</div>
							</td>
							<td>{{ post.content }}</td>
							<td>{{ formatFuzzyDate(post.postedDate || post.created) }}</td>
							<td>
								<span :class="['annotation', statusClass(post)]">{{ statusText(post) }}</span>
							</td>
							<td align="right">
								<div class="post-actions">
									<button v-if="post.postUrl" @click="openPostUrl(post.postUrl)" class="button neutral" title="Open Post URL">
										<Icon icon="mdi:open-in-new" width="16" height="16" />
									</button>
									<button @click="openPostDetails(post)" class="button neutral" title="View Post Details">
										<Icon icon="mdi:eye" width="16" height="16" />
									</button>
									<button v-if="post.state === 'error'" @click="retryPost(post)" class="button good" title="Retry Post" :disabled="retryingPosts.has(post.id)">
										<Icon v-if="retryingPosts.has(post.id)" icon="eos-icons:loading" width="16" height="16" />
										<Icon v-else icon="mdi:refresh" width="16" height="16" />
									</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
				<Pagination
					v-if="totalPosts > 0"
					:total="totalPosts"
					v-model:page="currentPage"
					v-model:pageSize="pageSize"
				/>
			</template>
		</template>
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
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Pagination from 'picocrank/vue/components/Pagination.vue';
	import Loading from './Loading.vue';

	const router = useRouter();

	const timeline = ref([]);
	const clientReady = ref(false);
	const currentPage = ref(1);
	const pageSize = ref(10);
	const timelineLoading = ref(true);
	const totalPosts = ref(0);

	// Filter state
	const selectedSocialAccountId = ref('');
	const selectedCampaignFilterId = ref('');
	const socialAccounts = ref([]);
	const campaigns = ref([]);

	// Campaign dialog state
	const showCampaignDialog = ref(false);
	const selectedPost = ref(null);
	const selectedCampaignId = ref(0);
	const campaignsLoading = ref(false);
	const campaignsSaving = ref(false);
	const campaignMessage = ref('');
	const campaignMessageType = ref('');

	// Retry state
	const retryingPosts = ref(new Set());

	const filteredTimeline = computed(() => {
		let filtered = timeline.value;

		// Filter by social account
		if (selectedSocialAccountId.value) {
			filtered = filtered.filter(post => post.socialAccountId == selectedSocialAccountId.value);
		}

		// Filter by campaign
		if (selectedCampaignFilterId.value) {
			filtered = filtered.filter(post => post.campaignId == selectedCampaignFilterId.value);
		}

		return filtered;
	});

	const hasActiveFilters = computed(() => {
		return selectedSocialAccountId.value !== '' || selectedCampaignFilterId.value !== '';
	});

	const pagedTimeline = computed(() => {
		// With server-side pagination, the timeline already contains only the
		// posts for the current page. We still apply client-side filters.
		return filteredTimeline.value;
	});

	function statusClass(post) {
		const text = statusText(post);
		if (text === 'Error') return 'bad';
		if (text === 'Completed') return 'good';
		if (text === 'Unknown') return 'warn';
		if (text === 'Scheduled') return 'note';
		return '';
	}

	function statusText(post) {
		if (post.state === 'error') return 'Error';
		if (post.state === 'pending' || post.state === 'scheduled') return 'Scheduled';
		if (post.state === 'completed') return 'Completed';
		return 'Unknown';
	}

	function formatFuzzyDate(dateString) {
		if (!dateString) {
			return '';
		}

		try {
			// Parse the date string (format: "2006-01-02 15:04:05")
			// Try replacing space with T for ISO format, or parse directly
			let date = new Date(dateString.replace(' ', 'T'));
			if (isNaN(date.getTime())) {
				date = new Date(dateString);
			}
			if (isNaN(date.getTime())) {
				return dateString; // Return original if can't parse
			}

			const now = new Date();
			const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
			const targetDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
			const diffDays = Math.round((targetDate - today) / (1000 * 60 * 60 * 24));

			// Format time (e.g., "4pm", "2:30pm", "11:15am")
			const hours = date.getHours();
			const minutes = date.getMinutes();
			const ampm = hours >= 12 ? 'pm' : 'am';
			const displayHours = hours % 12 || 12;
			const timeStr = minutes > 0
				? `${displayHours}:${minutes.toString().padStart(2, '0')}${ampm}`
				: `${displayHours}${ampm}`;

			if (diffDays === 0) {
				return `Today at ${timeStr}`;
			} else if (diffDays === 1) {
				return `Tomorrow at ${timeStr}`;
			} else if (diffDays === -1) {
				return `Yesterday at ${timeStr}`;
			} else if (diffDays > 1 && diffDays <= 7) {
				// Within next week
				const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
				return `${dayNames[date.getDay()]} at ${timeStr}`;
			} else if (diffDays < -1 && diffDays >= -7) {
				// Within past week
				const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
				return `Last ${dayNames[date.getDay()]} at ${timeStr}`;
			} else {
				// Further away - show date
				const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
				const month = monthNames[date.getMonth()];
				const day = date.getDate();

				// If same year, don't show year
				if (date.getFullYear() === now.getFullYear()) {
					return `${month} ${day} at ${timeStr}`;
				} else {
					return `${month} ${day}, ${date.getFullYear()} at ${timeStr}`;
				}
			}
		} catch (error) {
			console.error('Error formatting fuzzy date:', error);
			return dateString;
		}
	}

	async function getTimeline(page = currentPage.value, pageSizeVal = pageSize.value) {
		if (!window.client) {
			return { posts: [], total: 0 };
		}

		return await window.client.getTimeline({ page, pageSize: pageSizeVal })
			.then((ret) => {
				return {
					posts: ret.posts || [],
					total: ret.total ?? (ret.posts ? ret.posts.length : 0),
				};
			})
			.catch((error) => {
				console.error('Error fetching timeline:', error);
				return { posts: [], total: 0 };
			});
	}

	function refreshTimeline() {
		timelineLoading.value = true;
		const firstPage = 1;
		getTimeline(firstPage, pageSize.value).then(({ posts, total }) => {
			timeline.value = posts;
			totalPosts.value = total;
			currentPage.value = firstPage;
		}).catch(error => {
			console.error("Error fetching timeline:", error);
			timeline.value = [];
			totalPosts.value = 0;
		}).finally(() => {
			timelineLoading.value = false;
		});
	}

	// When the current page or page size changes via the Pagination component,
	// fetch the corresponding page from the backend.
	watch([currentPage, pageSize], ([newPage, newPageSize]) => {
		if (!clientReady.value) {
			return;
		}
		timelineLoading.value = true;
		getTimeline(newPage, newPageSize).then(({ posts, total }) => {
			timeline.value = posts;
			totalPosts.value = total;
		}).catch(error => {
			console.error("Error fetching timeline page:", error);
			timeline.value = [];
		}).finally(() => {
			timelineLoading.value = false;
		});
	});

	// Filter functions
	function applyFilters() {
		currentPage.value = 1; // Reset to first page when filters change
	}

	function clearFilters() {
		selectedSocialAccountId.value = '';
		selectedCampaignFilterId.value = '';
		currentPage.value = 1;
	}

	async function loadFilterData() {
		try {
			// Load social accounts
			const socialAccountsResponse = await window.client.getSocialAccounts({ onlyActive: true });
			socialAccounts.value = socialAccountsResponse.accounts || [];

			// Load campaigns
			const campaignsResponse = await window.client.getCampaigns();
			campaigns.value = campaignsResponse.campaigns || [];
		} catch (error) {
			console.error('Error loading filter data:', error);
		}
	}

	// Navigation functions
	function openPostDetails(post) {
		router.push({ name: 'postDetails', params: { id: post.id } });
	}

	function openPostUrl(url) {
		if (url) {
			window.open(url, '_blank', 'noopener,noreferrer');
		}
	}

	// Campaign dialog functions
	async function openCampaignDialog(post) {
		selectedPost.value = post;
		selectedCampaignId.value = post.campaignId || 0;
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
		selectedPost.value = null;
		selectedCampaignId.value = 0;
		campaignMessage.value = '';
		campaignMessageType.value = '';
	}

	async function updatePostCampaign() {
		if (!selectedPost.value) return;

		campaignsSaving.value = true;
		campaignMessage.value = '';
		campaignMessageType.value = '';

		try {
			await window.client.updatePostCampaign({
				postId: selectedPost.value.id,
				campaignId: selectedCampaignId.value
			});

			campaignMessage.value = 'Campaign updated successfully';
			campaignMessageType.value = 'good';

			// Update the post in the timeline
			const postIndex = timeline.value.findIndex(p => p.id === selectedPost.value.id);
			if (postIndex !== -1) {
				timeline.value[postIndex].campaignId = selectedCampaignId.value;
				// Find campaign name
				const campaign = campaigns.value.find(c => c.id === selectedCampaignId.value);
				timeline.value[postIndex].campaignName = campaign ? campaign.name : '';
			}

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
	async function forgetPost(post) {
		if (!confirm(`Are you sure you want to forget (delete) this post?\n\n"${post.content.substring(0, 50)}${post.content.length > 50 ? '...' : ''}"`)) {
			return;
		}

		try {
			await window.client.forgetPost({
				postId: post.id
			});

			// Remove the post from the timeline
			const postIndex = timeline.value.findIndex(p => p.id === post.id);
			if (postIndex !== -1) {
				timeline.value.splice(postIndex, 1);
			}

			console.log('Post forgotten successfully');
		} catch (error) {
			console.error('Error forgetting post:', error);
			alert('Failed to forget post: ' + error.message);
		}
	}

	// Post retry function
	async function retryPost(post) {
		if (!confirm(`Are you sure you want to retry posting this?\n\n"${post.content.substring(0, 50)}${post.content.length > 50 ? '...' : ''}"`)) {
			return;
		}

		// Add to retrying set
		retryingPosts.value.add(post.id);

		try {
			const response = await window.client.retryPost({
				postId: post.id
			});

			// Update the post in the timeline with the retry result
			const postIndex = timeline.value.findIndex(p => p.id === post.id);
			if (postIndex !== -1) {
				// Update the post with the retry result
				const updatedPost = response.postStatus;
				timeline.value[postIndex] = updatedPost;
			}

			console.log('Post retry completed:', response.standardResponse.message);

			// Show success/error message
			if (response.postStatus.success) {
				console.log('Post retry succeeded!');
			} else {
				console.log('Post retry failed, but post status updated');
			}

		} catch (error) {
			console.error('Error retrying post:', error);
			alert('Failed to retry post: ' + error.message);
		} finally {
			// Remove from retrying set
			retryingPosts.value.delete(post.id);
		}
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;

		await loadFilterData();
		refreshTimeline();
	});
</script>

<style scoped>
	.toolbar-controls {
		display: flex;
		align-items: center;
		gap: 1rem;
		width: 100%;
	}

	.filters {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex: 1;
	}

	.filter-select {
		padding: 0.5rem;
		border: 1px solid #ddd;
		border-radius: 0.25rem;
		font-size: 0.9rem;
		min-width: 150px;
		background: white;
	}

	.filter-select:focus {
		outline: none;
		border-color: #4CAF50;
		box-shadow: 0 0 0 2px rgba(76, 175, 80, 0.2);
	}

	.data-table {
		width: 100%;
		table-layout: fixed;
	}

	th:nth-child(1) {
		width: 220px;
		min-width: 200px;
	}

	th:nth-child(2) {
		width: 120px;
	}

	th:nth-child(3) {
		width: 100%;
		min-width: 200px;
	}

	th:nth-child(4) {
		width: 100px;
	}

	th:nth-child(5) {
		width: 80px;
	}

	th.actions {
		width: 140px;
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

	.annotation.warn {
		background-color: #fff3cd;
		color: #856404;
		border: 1px solid #ffeaa7;
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

	.post-actions {
	display: flex;
	gap: 0.5rem;
	justify-content: flex-end;
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
	background: white;
	padding: 2rem;
	border-radius: 0.5rem;
	box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	min-width: 400px;
	max-width: 500px;
}

.modal h3 {
	margin: 0 0 1rem 0;
	color: #333;
}

.form-group {
	margin-bottom: 1rem;
}

.form-group label {
	display: block;
	margin-bottom: 0.5rem;
	font-weight: 500;
	color: #333;
}

.form-group select {
	width: 100%;
	padding: 0.5rem;
	border: 1px solid #ddd;
	border-radius: 0.25rem;
	font-size: 1rem;
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
	background-color: #f0f0f0;
	color: #333;
}

.dialog-actions button.neutral:hover:not(:disabled) {
	background-color: #e0e0e0;
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

/* Use global .social-account styling from main.css */
/* No overrides needed - using global styles */

.no-campaign {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.no-campaign-text {
	cursor: pointer;
	color: var(--link-color);
}

.no-campaign-text:hover {
	color: var(--link-hover-color);
}

.no-campaign button.small {
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
}

	.no-campaign button.small:hover {
		background-color: #e0e0e0;
		transform: translateY(-1px);
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
	}

	/* Responsive design */
	@media (max-width: 768px) {
		.toolbar-controls {
			flex-direction: column;
			align-items: stretch;
			gap: 0.5rem;
		}

		.filters {
			flex-direction: column;
			gap: 0.5rem;
		}

		.filter-select {
			min-width: auto;
			width: 100%;
		}
	}
</style>
