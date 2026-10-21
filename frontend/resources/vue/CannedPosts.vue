<template>
	<Section
		title="Canned Posts"
		subtitle="Manage your canned posts here. You can create, edit, and delete canned posts."
		:icon="PackageIcon"
		classes="canned-posts"
		:padding="false"
	>
		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="!clientReady || loading"
				@click="refreshPosts"
			>
				<Icon icon="material-symbols:refresh" />
			</button>
			<button
				type="button"
				class="inline-icon good"
				aria-label="Add canned post"
				title="Add canned post"
				:disabled="!clientReady || loading"
				@click="createCannedPost"
			>
				<Icon icon="material-symbols:add-rounded" />
			</button>
		</template>

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />

		<Table
			class="canned-posts-table-wrap"
			:data="posts"
			:headers="tableHeaders"
			:loading="loading"
		>
			<template #empty>
				No canned posts available. Please create a new canned post.
			</template>
			<template #cell-content="{ row }">
				<PostPreview v-if="!row.editing" :post="formatPostForPreview(row)" />
				<textarea
					v-else
					:id="'canned-post-' + row.id"
					v-model="row.content"
					class="edit-textarea"
					rows="6"
					@keyup.enter="saveCannedPost(row)"
					@keyup.esc="cancelEditing(row)"
				></textarea>
			</template>
			<template #cell-createdAt="{ row }">
				{{ formatFuzzyDate(row.createdAt) }}
			</template>
			<template #cell-actions="{ row }">
				<div class="actions-cell">
					<button
						v-if="!row.editing"
						type="button"
						class="inline-icon good"
						title="Use this post"
						@click="usePost(row)"
					>
						<span>Use</span>
					</button>
					<button
						v-if="!row.editing"
						type="button"
						class="inline-icon neutral"
						title="Edit post"
						@click="beginEditing(row)"
					>
						<span>Edit</span>
					</button>
					<button
						v-if="row.editing"
						type="button"
						class="inline-icon good"
						title="Save changes"
						@click="saveCannedPost(row)"
					>
						<span>Save</span>
					</button>
					<button
						v-if="row.editing"
						type="button"
						class="inline-icon neutral"
						title="Cancel editing"
						@click="cancelEditing(row)"
					>
						<span>Cancel</span>
					</button>
					<button
						type="button"
						class="inline-icon bad"
						title="Delete post"
						@click="deleteCannedPost(row.id)"
					>
						<span>Delete</span>
					</button>
				</div>
			</template>
		</Table>
	</Section>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { PackageIcon } from '@hugeicons/core-free-icons';
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import PostPreview from './PostPreview.vue';

	const posts = ref([]);
	const clientReady = ref(false);
	const loading = ref(true);
	const errorMessage = ref('');

	const tableHeaders = [
		{ key: 'content', label: 'Content', sortable: false },
		{ key: 'createdAt', label: 'Created', sortable: true, width: '11rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: '14rem' },
	];

	function usePost(p) {
		window.router.push({
			path: '/post',
			query: { cannedPostId: p.id },
		});
	}

	function beginEditing(post) {
		post.originalContent = post.content;
		post.editing = true;
	}

	function cancelEditing(post) {
		post.content = post.originalContent;
		post.editing = false;
	}

	function formatPostForPreview(post) {
		return {
			id: post.id,
			content: post.content,
			socialAccountId: null,
			socialAccountIcon: null,
			socialAccountIdentity: null,
		};
	}

	function formatFuzzyDate(dateString) {
		if (!dateString) {
			return '';
		}

		try {
			let date = new Date(dateString.replace(' ', 'T'));
			if (isNaN(date.getTime())) {
				date = new Date(dateString);
			}
			if (isNaN(date.getTime())) {
				return dateString;
			}

			const now = new Date();
			const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
			const targetDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
			const diffDays = Math.round((targetDate - today) / (1000 * 60 * 60 * 24));

			const hours = date.getHours();
			const minutes = date.getMinutes();
			const ampm = hours >= 12 ? 'pm' : 'am';
			const displayHours = hours % 12 || 12;
			const timeStr = minutes > 0
				? `${displayHours}:${minutes.toString().padStart(2, '0')}${ampm}`
				: `${displayHours}${ampm}`;

			if (diffDays === 0) {
				return `Today at ${timeStr}`;
			}
			if (diffDays === 1) {
				return `Tomorrow at ${timeStr}`;
			}
			if (diffDays === -1) {
				return `Yesterday at ${timeStr}`;
			}
			if (diffDays > 1 && diffDays <= 7) {
				const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
				return `${dayNames[date.getDay()]} at ${timeStr}`;
			}
			if (diffDays < -1 && diffDays >= -7) {
				const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
				return `Last ${dayNames[date.getDay()]} at ${timeStr}`;
			}

			const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
			const month = monthNames[date.getMonth()];
			const day = date.getDate();
			if (date.getFullYear() === now.getFullYear()) {
				return `${month} ${day} at ${timeStr}`;
			}
			return `${month} ${day}, ${date.getFullYear()} at ${timeStr}`;
		} catch (error) {
			console.error('Error formatting fuzzy date:', error);
			return dateString;
		}
	}

	function saveCannedPost(post) {
		if (!window.client) {
			errorMessage.value = 'Client is not ready.';
			return;
		}

		window.client.updateCannedPost({
			id: post.id,
			content: post.content,
		})
			.then(() => {
				post.editing = false;
				refreshPosts();
			})
			.catch((error) => {
				errorMessage.value = 'Failed to update canned post: ' + error.message;
				console.error('Error updating canned post:', error);
			});
	}

	function deleteCannedPost(id) {
		if (!window.client) {
			errorMessage.value = 'Client is not ready.';
			return;
		}

		window.client.deleteCannedPost({ id })
			.then(() => {
				refreshPosts();
			})
			.catch((error) => {
				errorMessage.value = 'Failed to delete canned post: ' + error.message;
				console.error('Error deleting canned post:', error);
			});
	}

	async function getCannedPosts() {
		return await window.client.getCannedPosts()
			.then((ret) => {
				(ret.posts || []).forEach((post) => {
					post.editing = false;
					post.originalContent = post.content;
				});
				return ret.posts || [];
			})
			.catch((error) => {
				errorMessage.value = 'Failed to fetch canned posts: ' + error.message;
				console.error('Error fetching canned posts:', error);
				return [];
			});
	}

	function refreshPosts() {
		loading.value = true;
		errorMessage.value = '';
		getCannedPosts().then((fetchedPosts) => {
			posts.value = fetchedPosts;
		}).finally(() => {
			loading.value = false;
		});
	}

	function createCannedPost() {
		if (!window.client) {
			errorMessage.value = 'Client is not ready.';
			return;
		}

		window.client.createCannedPost({
			content: 'This is a new canned post.',
		})
			.then(() => {
				refreshPosts();
			})
			.catch((error) => {
				errorMessage.value = 'Failed to create canned post: ' + error.message;
				console.error('Error creating canned post:', error);
			});
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		refreshPosts();
	});
</script>

<style scoped>
	.canned-posts-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1rem;
	}

	.edit-textarea {
		width: 100%;
		box-sizing: border-box;
		padding: 0.75rem;
		border: 1px solid var(--border-color, #ddd);
		border-radius: 0.25rem;
		font-size: 1rem;
		font-family: inherit;
		resize: vertical;
		min-height: 120px;
	}

	.actions-cell {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		justify-content: flex-end;
	}

	html[data-theme="dark"] {
		.edit-textarea {
			background-color: #1a1a1a;
			border-color: #505050;
			color: #e0e0e0;
		}
	}
</style>
