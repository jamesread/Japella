<template>
	<dialog
		ref="dlg"
		@close="onDialogClose"
		@click="onBackdropClick"
	>
		<h2>Diagnostic Info</h2>

		<dl>
			<template v-if="post.socialAccountId">
				<dt>Social account</dt>
				<dd>
					<FieldValue>
						<SocialAccountChip
							:social-account-id="post.socialAccountId"
							:identity="post.socialAccountIdentity || `Account #${post.socialAccountId}`"
							:icon="post.socialAccountIcon"
						/>
					</FieldValue>
				</dd>
			</template>
			<template v-if="post.state">
				<dt>Status</dt>
				<dd>
					<FieldValue>
						<span class="tag" :class="postStatusClass">{{ postStatusText }}</span>
					</FieldValue>
				</dd>
			</template>
			<template v-if="post.id">
				<dt>ID</dt>
				<dd>
					<FieldValue>
						<code>{{ post.id }}</code>
					</FieldValue>
				</dd>
			</template>
			<template v-if="post.remoteId">
				<dt>Remote ID</dt>
				<dd>
					<FieldValue>
						<code>{{ post.remoteId }}</code>
					</FieldValue>
				</dd>
			</template>
		</dl>

		<NotificationBlock
			v-if="refetchMessage"
			:type="refetchMessageType"
			:message="refetchMessage"
		/>

		<FormLayout @submit.prevent>
			<template #actions>
				<button
					v-if="canRepost"
					type="button"
					class="inline-icon neutral"
					@click="repost"
				>
					<Icon icon="mdi:content-copy" />
					<span>Repost</span>
				</button>
				<button
					v-if="showViewDetails && post.id"
					type="button"
					class="inline-icon neutral"
					@click="viewDetails"
				>
					<Icon icon="mdi:eye" />
					<span>View Post Details</span>
				</button>
				<button
					v-if="canRetry"
					type="button"
					class="inline-icon good"
					:disabled="retrying"
					@click="retryPost"
				>
					<Icon v-if="retrying" icon="eos-icons:loading" />
					<Icon v-else icon="mdi:refresh" />
					<span>{{ retrying ? 'Retrying…' : 'Retry Post' }}</span>
				</button>
				<button
					v-if="canRefetch"
					type="button"
					class="inline-icon good"
					:disabled="refetching"
					@click="refetchPost"
				>
					<Icon v-if="refetching" icon="eos-icons:loading" />
					<Icon v-else icon="mdi:refresh" />
					<span>{{ refetching ? 'Refetching…' : 'Refetch' }}</span>
				</button>
				<button type="button" class="neutral" @click="close">Close</button>
			</template>
		</FormLayout>
	</dialog>
</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import { useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import FieldValue from 'picocrank/vue/components/FieldValue.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import { stashComposePrefill } from '../javascript/composePrefill.js';
	import SocialAccountChip from './SocialAccountChip.vue';

	const props = defineProps({
		open: {
			type: Boolean,
			default: false,
		},
		post: {
			type: Object,
			required: true,
		},
		refetchable: {
			type: Boolean,
			default: true,
		},
		showViewDetails: {
			type: Boolean,
			default: false,
		},
		retryable: {
			type: Boolean,
			default: false,
		},
		retrying: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['close', 'post-refetched', 'view-details', 'retry-post']);
	const router = useRouter();

	const dlg = ref(null);
	const refetching = ref(false);
	const refetchMessage = ref('');
	const refetchMessageType = ref('note');
	const closingFromParent = ref(false);

	const canRefetch = computed(() => props.refetchable && Boolean(props.post.id));
	const canRetry = computed(() => props.retryable && props.post.state === 'error');
	const canRepost = computed(() => Boolean((props.post.content || '').trim()));

	const postStatusText = computed(() => {
		if (props.post.state === 'error') return 'Error';
		if (props.post.state === 'pending_approval') return 'Pending approval';
		if (props.post.state === 'rejected') return 'Rejected';
		if (props.post.state === 'draft') return 'Draft';
		if (props.post.state === 'pending' || props.post.state === 'scheduled') return 'Scheduled';
		if (props.post.state === 'completed') return 'Completed';
		return 'Unknown';
	});

	const postStatusClass = computed(() => {
		const text = postStatusText.value;
		if (text === 'Error' || text === 'Rejected') return 'bad';
		if (text === 'Completed') return 'good';
		if (text === 'Unknown') return 'warning';
		if (text === 'Scheduled' || text === 'Draft' || text === 'Pending approval') return 'note';
		return '';
	});

	watch(
		() => props.open,
		async (isOpen) => {
			await nextTick();
			if (isOpen) {
				refetchMessage.value = '';
				refetchMessageType.value = 'note';
				if (dlg.value && !dlg.value.open) {
					dlg.value.showModal();
				}
				return;
			}
			if (dlg.value?.open) {
				closingFromParent.value = true;
				dlg.value.close();
			}
		},
	);

	function close() {
		dlg.value?.close();
	}

	function onDialogClose() {
		refetchMessage.value = '';
		if (closingFromParent.value) {
			closingFromParent.value = false;
			return;
		}
		emit('close');
	}

	function onBackdropClick(event) {
		if (event.target === dlg.value) {
			close();
		}
	}

	function viewDetails() {
		emit('view-details');
		close();
	}

	function retryPost() {
		emit('retry-post');
	}

	function plainPostContent(content) {
		const raw = String(content || '');
		if (!raw.includes('<')) {
			return raw;
		}
		const el = document.createElement('div');
		el.innerHTML = raw;
		return (el.textContent || el.innerText || '').trim();
	}

	function repost() {
		const content = plainPostContent(props.post.content);
		if (!content) {
			return;
		}
		const socialAccountIds = [];
		const accountId = Number(props.post.socialAccountId);
		if (Number.isFinite(accountId) && accountId > 0) {
			socialAccountIds.push(accountId);
		}
		stashComposePrefill({ content, socialAccountIds });
		close();
		router.push({ name: 'postBox', query: { prefill: '1' } });
	}

	async function refetchPost() {
		if (!props.post.id || !window.client || refetching.value) {
			return;
		}

		refetching.value = true;
		refetchMessage.value = '';
		refetchMessageType.value = 'note';

		try {
			const response = await window.client.refetchFeedPost({ id: props.post.id });
			if (!response.standardResponse?.success || !response.post) {
				refetchMessageType.value = 'error';
				refetchMessage.value = response.standardResponse?.message || 'Failed to refetch post.';
				return;
			}

			refetchMessageType.value = 'good';
			refetchMessage.value = 'Post refreshed from social account.';
			emit('post-refetched', response.post);
		} catch (error) {
			console.error('Error refetching feed post:', error);
			refetchMessageType.value = 'error';
			refetchMessage.value = 'Failed to refetch post from social account.';
		} finally {
			refetching.value = false;
		}
	}
</script>
