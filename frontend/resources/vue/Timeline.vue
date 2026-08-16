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
					<FilterSelect
						v-model="selectedSocialAccountId"
						clear-title="Clear social account filter"
					>
						<option value="">All Social Accounts</option>
						<option v-for="account in socialAccounts" :key="account.id" :value="account.id">
							{{ account.identity }}
						</option>
					</FilterSelect>
					<FilterSelect
						v-model="selectedCampaignFilterId"
						clear-title="Clear campaign filter"
					>
						<option value="">All Campaigns</option>
						<option v-for="campaign in campaigns" :key="campaign.id" :value="campaign.id">
							{{ campaign.name }}
						</option>
					</FilterSelect>
					<button
						v-if="hasActiveFilters"
						type="button"
						class="inline-icon neutral small"
						title="Clear Filters"
						@click="clearFilters"
					>
						<HugeiconsIcon
							:icon="FilterRemoveIcon"
							width="1em"
							height="1em"
							:strokeWidth="iconStrokeWidth"
							aria-hidden="true"
						/>
						<span>Clear</span>
					</button>
				</div>
				<button
					type="button"
					class="inline-icon neutral"
					aria-label="Refresh"
					:disabled="!clientReady"
					@click="refreshTimeline"
				>
					<HugeiconsIcon
						:icon="RefreshIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
				</button>
			</div>
		</template>

		<Loading v-if="timelineLoading && timeline.length === 0" message="Loading timeline..." :centered="true" />
	</Section>

	<p v-if="!timelineLoading && timeline.length > 0 && filteredTimeline.length === 0" class="timeline-empty inline-notification note">
		No posts match the current filters.
	</p>

	<p v-if="!timelineLoading && timeline.length === 0" class="timeline-empty inline-notification note">
		No posts available.
	</p>

	<section
		class="small"
		v-for="post in visibleTimeline"
		:key="post.id"
		:data-timeline-post-id="post.id"
	>
		<PostPreview
			:post="post"
			timeline-mode
			:retrying="retryingPosts.has(post.id)"
			@assign-campaign="openCampaignDialog(post)"
			@view-details="openPostDetails(post)"
			@retry-post="retryPost(post)"
		/>
	</section>

	<div
		v-if="!timelineLoading && hasMore"
		ref="loadMoreSentinel"
		class="timeline-load-sentinel"
		aria-hidden="true"
	/>

	<p v-if="!timelineLoading && filteredTimeline.length > 0 && !hasMore" class="timeline-end muted">
		End of timeline
	</p>

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
				<button type="button" class="neutral" :disabled="campaignsSaving" @click="cancelCampaignDialog">Cancel</button>
				<button
					type="button"
					class="inline-icon good"
					:disabled="campaignsSaving || campaignsLoading"
					@click="updatePostCampaign"
				>
					<HugeiconsIcon
						v-if="campaignsSaving"
						:icon="Loading01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
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
	import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		FilterRemoveIcon,
		Loading01Icon,
		RefreshIcon,
	} from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Loading from './Loading.vue';
	import PostPreview from './PostPreview.vue';
	import FilterSelect from './FilterSelect.vue';

	const iconStrokeWidth = 2.5;

	const PAGE_SIZE = 10;

	const router = useRouter();

	const timeline = ref([]);
	const clientReady = ref(false);
	const visibleCount = ref(PAGE_SIZE);
	const timelineLoading = ref(true);
	const loadMoreSentinel = ref(null);

	let loadMoreObserver = null;

	const selectedSocialAccountId = ref('');
	const selectedCampaignFilterId = ref('');
	const socialAccounts = ref([]);
	const campaigns = ref([]);

	const showCampaignDialog = ref(false);
	const selectedPost = ref(null);
	const selectedCampaignId = ref(0);
	const campaignsLoading = ref(false);
	const campaignsSaving = ref(false);
	const campaignMessage = ref('');
	const campaignMessageType = ref('');

	const retryingPosts = ref(new Set());

	const filteredTimeline = computed(() => {
		let filtered = timeline.value;

		if (selectedSocialAccountId.value) {
			filtered = filtered.filter((post) => post.socialAccountId == selectedSocialAccountId.value);
		}

		if (selectedCampaignFilterId.value) {
			filtered = filtered.filter((post) => post.campaignId == selectedCampaignFilterId.value);
		}

		return filtered;
	});

	const hasActiveFilters = computed(() => {
		return selectedSocialAccountId.value !== '' || selectedCampaignFilterId.value !== '';
	});

	const visibleTimeline = computed(() => filteredTimeline.value.slice(0, visibleCount.value));

	const hasMore = computed(() => visibleCount.value < filteredTimeline.value.length);

	async function getTimeline() {
		if (!window.client) {
			return [];
		}

		try {
			const response = await window.client.getTimeline({});
			return response.posts || [];
		} catch (error) {
			console.error('Error fetching timeline:', error);
			return [];
		}
	}

	function loadMore() {
		if (!hasMore.value) {
			return;
		}
		visibleCount.value = Math.min(visibleCount.value + PAGE_SIZE, filteredTimeline.value.length);
	}

	function setupInfiniteScroll() {
		if (loadMoreObserver) {
			loadMoreObserver.disconnect();
			loadMoreObserver = null;
		}

		if (!loadMoreSentinel.value || !hasMore.value) {
			return;
		}

		loadMoreObserver = new IntersectionObserver(
			(entries) => {
				if (entries[0]?.isIntersecting) {
					loadMore();
				}
			},
			{ rootMargin: '240px' },
		);
		loadMoreObserver.observe(loadMoreSentinel.value);
	}

	function refreshTimeline() {
		timelineLoading.value = true;
		getTimeline()
			.then((posts) => {
				timeline.value = posts;
				visibleCount.value = PAGE_SIZE;
			})
			.catch((error) => {
				console.error('Error fetching timeline:', error);
				timeline.value = [];
			})
			.finally(() => {
				timelineLoading.value = false;
			});
	}

	function clearFilters() {
		selectedSocialAccountId.value = '';
		selectedCampaignFilterId.value = '';
		visibleCount.value = PAGE_SIZE;
	}

	async function loadFilterData() {
		try {
			const socialAccountsResponse = await window.client.getSocialAccounts({ onlyActive: true });
			socialAccounts.value = socialAccountsResponse.accounts || [];

			const campaignsResponse = await window.client.getCampaigns();
			campaigns.value = campaignsResponse.campaigns || [];
		} catch (error) {
			console.error('Error loading filter data:', error);
		}
	}

	function openPostDetails(post) {
		router.push({ name: 'postDetails', params: { id: post.id } });
	}

	async function openCampaignDialog(post) {
		selectedPost.value = post;
		selectedCampaignId.value = post.campaignId || 0;
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
		selectedPost.value = null;
		selectedCampaignId.value = 0;
		campaignMessage.value = '';
		campaignMessageType.value = '';
	}

	async function updatePostCampaign() {
		if (!selectedPost.value) {
			return;
		}

		campaignsSaving.value = true;
		campaignMessage.value = '';
		campaignMessageType.value = '';

		try {
			await window.client.updatePostCampaign({
				postId: selectedPost.value.id,
				campaignId: selectedCampaignId.value,
			});

			campaignMessage.value = 'Campaign updated successfully';
			campaignMessageType.value = 'good';

			const postIndex = timeline.value.findIndex((p) => p.id === selectedPost.value.id);
			if (postIndex !== -1) {
				timeline.value[postIndex].campaignId = selectedCampaignId.value;
				const campaign = campaigns.value.find((c) => c.id === selectedCampaignId.value);
				timeline.value[postIndex].campaignName = campaign ? campaign.name : '';
			}

			setTimeout(() => {
				cancelCampaignDialog();
			}, 1500);
		} catch (error) {
			console.error('Error updating post campaign:', error);
			campaignMessage.value = `Failed to update campaign: ${error.message}`;
			campaignMessageType.value = 'bad';
		} finally {
			campaignsSaving.value = false;
		}
	}

	async function retryPost(post) {
		if (!confirm(`Are you sure you want to retry posting this?\n\n"${post.content.substring(0, 50)}${post.content.length > 50 ? '...' : ''}"`)) {
			return;
		}

		retryingPosts.value.add(post.id);

		try {
			const response = await window.client.retryPost({
				postId: post.id,
			});

			const postIndex = timeline.value.findIndex((p) => p.id === post.id);
			if (postIndex !== -1 && response.postStatus) {
				timeline.value[postIndex] = response.postStatus;
			}
		} catch (error) {
			console.error('Error retrying post:', error);
			alert(`Failed to retry post: ${error.message}`);
		} finally {
			retryingPosts.value.delete(post.id);
		}
	}

	watch([selectedSocialAccountId, selectedCampaignFilterId], () => {
		visibleCount.value = PAGE_SIZE;
	});

	watch([loadMoreSentinel, hasMore, () => filteredTimeline.value.length], () => {
		nextTick(() => setupInfiniteScroll());
	});

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await loadFilterData();
		refreshTimeline();
	});

	onUnmounted(() => {
		loadMoreObserver?.disconnect();
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
		flex-wrap: wrap;
	}

	section {
		margin-bottom: 1rem;
		padding: 0;
	}

	.timeline-empty {
		margin: 0 0 1rem;
	}

	.timeline-load-sentinel {
		height: 1px;
		margin: 0;
	}

	.timeline-end {
		text-align: center;
		padding: 1rem 0 2rem;
		font-size: 0.9em;
		opacity: 0.7;
	}

	.muted {
		opacity: 0.7;
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
	}
</style>
