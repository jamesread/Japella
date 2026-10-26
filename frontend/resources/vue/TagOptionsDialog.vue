<template>
	<dialog ref="dialog" class="dialog tag-options" aria-labelledby="tag-options-title" @close="emit('close')">
		<h2 id="tag-options-title">
			<Icon icon="mdi:pound" aria-hidden="true" />
			<span>{{ tag }}</span>
		</h2>

		<div class="tag-actions" role="group" aria-label="Tag options">
			<a
				v-if="safeUrl"
				:href="safeUrl"
				target="_blank"
				rel="noopener noreferrer"
				class="button neutral"
				@click="close"
			>
				<Icon icon="mdi:open-in-new" aria-hidden="true" />
				<span>Open #{{ tag }} on {{ providerName }}</span>
			</a>
			<button v-else type="button" class="neutral" disabled :title="`There's no link to #${tag} on ${providerName}`">
				<Icon icon="mdi:open-in-new" aria-hidden="true" />
				<span>Open #{{ tag }} on {{ providerName }}</span>
			</button>

			<button
				type="button"
				class="neutral"
				:disabled="!canOpenColumn || opening"
				:title="columnTitle"
				@click="openColumn"
			>
				<Icon icon="mdi:table-column-plus-after" aria-hidden="true" />
				<span>Open #{{ tag }} in search column</span>
			</button>
		</div>

		<div class="tag-options-footer">
			<button type="button" class="neutral" @click="close">Close</button>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, computed, inject, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { socialProviderNameFromPost } from '../javascript/util';

	const props = defineProps({
		// Without '#'.
		tag: {
			type: String,
			required: true,
		},
		// The tag's page on the post's network, if known.
		url: {
			type: String,
			default: '',
		},
		// The post the tag was clicked in; its account searches the new column.
		post: {
			type: Object,
			required: true,
		},
	});

	const emit = defineEmits(['close']);

	// Provided by the feed page.
	const feedColumns = inject('feedColumns', null);
	const openHashtagColumn = inject('openHashtagColumn', null);

	const dialog = ref(null);
	const opening = ref(false);

	const providerName = computed(() => socialProviderNameFromPost(props.post));

	const safeUrl = computed(() => (/^https?:\/\//i.test(props.url) ? props.url : ''));

	const hasColumn = computed(() => {
		const query = `#${props.tag}`.toLowerCase();
		return (feedColumns?.value || []).some((c) => c.isHashtag && String(c.query).toLowerCase() === query);
	});

	const canOpenColumn = computed(() => Boolean(openHashtagColumn && props.post.socialAccountId && !hasColumn.value));

	const columnTitle = computed(() => {
		if (!openHashtagColumn) {
			return 'Search columns are on the feed page';
		}
		if (hasColumn.value) {
			return `There's already a search column for #${props.tag}`;
		}
		if (!props.post.socialAccountId) {
			return 'This post has no social account to search with';
		}
		return `Add a column searching #${props.tag} with ${props.post.socialAccountIdentity || 'this account'}`;
	});

	function close() {
		dialog.value?.close();
	}

	async function openColumn() {
		opening.value = true;
		try {
			await openHashtagColumn(props.tag, props.post.socialAccountId);
			close();
		} finally {
			opening.value = false;
		}
	}

	onMounted(() => dialog.value?.showModal());
</script>

<style scoped>
	.tag-options {
		min-width: min(22rem, calc(100vw - 2rem));
	}

	.tag-options h2 {
		display: flex;
		align-items: center;
		gap: 0.2rem;
		margin-top: 0;
		overflow-wrap: anywhere;
	}

	.tag-actions {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.tag-actions > * {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		justify-content: flex-start;
		text-decoration: none;
	}

	.tag-options-footer {
		display: flex;
		justify-content: flex-end;
		margin-top: 1rem;
	}
</style>
