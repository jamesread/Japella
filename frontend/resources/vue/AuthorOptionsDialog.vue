<template>
	<dialog ref="dialog" class="dialog author-options" aria-labelledby="author-options-title" @close="emit('close')">
		<div class="author-card">
			<FeedAuthorAvatar
				class="author-card-avatar"
				:avatar-url="author.avatarUrl"
				:fallback-icon="author.fallbackIcon || 'mdi:account'"
				:alt-text="author.name || author.handle || 'Author'"
			/>
			<div class="author-card-identity">
				<h2 id="author-options-title">{{ author.name || author.handle || 'Unknown author' }}</h2>
				<p v-if="author.handle && author.handle !== author.name" class="subtle">{{ author.handle }}</p>
				<p v-if="author.socialAccountIdentity" class="subtle author-card-via">
					<Icon v-if="author.socialAccountIcon" :icon="author.socialAccountIcon" width="14" height="14" aria-hidden="true" />
					Seen via {{ author.socialAccountIdentity }}
				</p>
			</div>
		</div>

		<div class="author-actions" role="group" aria-label="Author options">
			<button
				v-if="follow.supported"
				type="button"
				:class="follow.following || follow.requested ? 'good' : 'neutral'"
				:aria-pressed="follow.following || follow.requested"
				:disabled="followBusy || !follow.canChange"
				:title="followTitle"
				@click="toggleFollow"
			>
				<Icon :icon="followIcon" aria-hidden="true" />
				<span>{{ followLabel }}</span>
			</button>
			<NotificationBlock v-if="followError" type="error" :message="followError" />
			<a
				v-if="profileUrl"
				:href="profileUrl"
				target="_blank"
				rel="noopener noreferrer"
				class="button neutral"
				@click="close"
			>
				<Icon icon="mdi:open-in-new" aria-hidden="true" />
				<span>Open {{ displayName }}'s profile</span>
			</a>
			<button
				v-else
				type="button"
				class="neutral"
				disabled
				title="This post didn't include a link to the author's profile"
			>
				<Icon icon="mdi:open-in-new" aria-hidden="true" />
				<span>Open {{ displayName }}'s profile</span>
			</button>
			<button v-if="filterable && author.id" type="button" class="neutral" @click="filterByAuthor">
				<Icon icon="mdi:filter-outline" aria-hidden="true" />
				<span>Filter feed by this author</span>
			</button>
			<button v-if="author.handle" type="button" class="neutral" @click="copyHandle">
				<Icon :icon="copied ? 'mdi:check' : 'mdi:content-copy'" aria-hidden="true" />
				<span>{{ copied ? 'Copied' : 'Copy handle' }}</span>
			</button>
			<router-link
				v-if="author.socialAccountId"
				:to="{ name: 'socialAccountDetails', params: { id: author.socialAccountId } }"
				class="button neutral"
				@click="close"
			>
				<Icon icon="mdi:account-cog-outline" aria-hidden="true" />
				<span>View {{ author.socialAccountIdentity || 'social account' }}</span>
			</router-link>
		</div>

		<div class="author-options-footer">
			<button type="button" class="neutral" @click="close">Close</button>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FeedAuthorAvatar from './FeedAuthorAvatar.vue';
	import { profileUrlFromPostUrl } from '../javascript/util';

	const props = defineProps({
		// { id, name, handle, avatarUrl, profileUrl, postUrl, fallbackIcon,
		//   socialAccountId, socialAccountIdentity, socialAccountIcon, external }
		// external is true when the author isn't the social account itself.
		// postUrl is one of the author's posts, used to find their profile
		// when profileUrl is missing.
		author: {
			type: Object,
			required: true,
		},
		// Offer "Filter feed by this author" (only where the feed can filter).
		filterable: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['filter-author', 'close']);

	const dialog = ref(null);

	const displayName = computed(() => props.author.name || props.author.handle || 'the author');

	const profileUrl = computed(() => props.author.profileUrl || profileUrlFromPostUrl(props.author.postUrl || ''));
	const copied = ref(false);

	function close() {
		dialog.value?.close();
	}

	function filterByAuthor() {
		emit('filter-author');
		close();
	}

	async function copyHandle() {
		try {
			await navigator.clipboard.writeText(props.author.handle);
			copied.value = true;
		} catch {
			copied.value = false;
		}
	}

	const follow = ref({ supported: false, following: false, requested: false, canChange: false });
	const followBusy = ref(false);
	const followError = ref('');

	const followLabel = computed(() => {
		if (follow.value.requested) {
			return 'Follow requested';
		}
		return follow.value.following ? 'Following' : 'Follow';
	});

	const followIcon = computed(() => {
		if (follow.value.requested) {
			return 'mdi:account-clock-outline';
		}
		return follow.value.following ? 'mdi:account-check' : 'mdi:account-plus-outline';
	});

	const followTitle = computed(() => {
		const as = props.author.socialAccountIdentity || 'this social account';
		if (!follow.value.canChange) {
			return `You need post access to ${as} to change who it follows`;
		}
		if (follow.value.following || follow.value.requested) {
			return `Unfollow ${displayName.value} as ${as}`;
		}
		return `Follow ${displayName.value} as ${as}`;
	});

	// Only offered for other people, through the account the post was seen by.
	async function loadFollow() {
		if (!props.author.external || !props.author.id || !props.author.socialAccountId || !window.client) {
			return;
		}
		try {
			const res = await window.client.getAuthorFollow({
				socialAccountId: props.author.socialAccountId,
				authorId: props.author.id,
			});
			follow.value = { supported: res.supported, following: res.following, requested: res.requested, canChange: res.canChange };
		} catch (e) {
			// Unsupported networks answer normally; anything else is worth showing.
			follow.value = { ...follow.value, supported: true };
			followError.value = `Couldn't check whether you follow ${displayName.value}: ${e.rawMessage || e.message || e}`;
		}
	}

	async function toggleFollow() {
		followBusy.value = true;
		followError.value = '';
		const wanted = !(follow.value.following || follow.value.requested);
		try {
			const res = await window.client.setAuthorFollow({
				socialAccountId: props.author.socialAccountId,
				authorId: props.author.id,
				follow: wanted,
			});
			follow.value = { ...follow.value, following: res.following, requested: res.requested, canChange: true };
		} catch (e) {
			followError.value = `Failed to ${wanted ? 'follow' : 'unfollow'}: ${e.rawMessage || e.message || e}`;
		} finally {
			followBusy.value = false;
		}
	}

	onMounted(() => {
		dialog.value?.showModal();
		loadFollow();
	});
</script>

<style scoped>
	.author-options {
		min-width: min(22rem, calc(100vw - 2rem));
	}

	.author-card {
		display: flex;
		align-items: center;
		gap: 0.85rem;
		margin-bottom: 1rem;
	}

	.author-card-avatar {
		width: 3.5rem;
		height: 3.5rem;
		flex-shrink: 0;
	}

	.author-card-avatar :deep(img),
	.author-card-avatar :deep(svg) {
		width: 100%;
		height: 100%;
	}

	.author-card-identity {
		min-width: 0;
	}

	.author-card-identity h2 {
		margin: 0;
		font-size: 1.15rem;
		overflow-wrap: anywhere;
	}

	.author-card-identity p {
		margin: 0.15rem 0 0;
		overflow-wrap: anywhere;
	}

	.author-card-via {
		display: flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.85rem;
	}

	.author-actions {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.author-actions > * {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		justify-content: flex-start;
		text-decoration: none;
	}

	.author-options-footer {
		display: flex;
		justify-content: flex-end;
		margin-top: 1rem;
	}
</style>
