<template>
	<dialog ref="dialog" class="dialog" @close="onClose">
		<h2>{{ column ? 'Edit column' : 'Add column' }}</h2>
		<p v-if="kind === 'search'">
			Show recent posts for a hashtag (like <code>#golang</code>) or any search, next to your feed.
			Posts are searched through each account you pick and merged, newest first.
		</p>
		<p v-else>
			Show the posts you've tagged, most recently tagged first. Tags are private to you in Japella;
			add them by right-clicking a post and choosing Tags….
		</p>

		<FormLayout @submit.prevent="submit">
			<FormField label="Column type" :disabled="saving">
				<fieldset class="feed-column-kind" :disabled="saving">
					<legend class="visually-hidden">Column type</legend>
					<label class="checkbox-label">
						<input v-model="kind" type="radio" value="search" />
						Hashtag or search
					</label>
					<label class="checkbox-label">
						<input v-model="kind" type="radio" value="tag" />
						Japella tag
					</label>
				</fieldset>
			</FormField>

			<FormField v-if="kind === 'tag'" label="Tag" for="feed-column-tag" :disabled="saving">
				<input
					id="feed-column-tag"
					ref="tagInput"
					v-model="tag"
					type="text"
					maxlength="65"
					placeholder="todo"
					list="feed-column-tag-suggestions"
					autocomplete="off"
					:disabled="saving"
					required
				/>
				<datalist id="feed-column-tag-suggestions">
					<option v-for="t in knownTags" :key="t.tag" :value="t.tag">{{ t.posts }} {{ t.posts === 1 ? 'post' : 'posts' }}</option>
				</datalist>
			</FormField>

			<FormField v-if="kind === 'search'" label="Hashtag or search" for="feed-column-query" :disabled="saving">
				<input
					id="feed-column-query"
					ref="queryInput"
					v-model="query"
					type="text"
					maxlength="255"
					placeholder="#golang or japella release"
					:disabled="saving"
					required
				/>
			</FormField>

			<FormField v-if="kind === 'search'" label="Search with accounts" :disabled="saving">
				<fieldset class="feed-column-accounts" :disabled="saving">
					<legend class="visually-hidden">Search with accounts</legend>
					<label v-for="account in accounts" :key="account.id" class="checkbox-label">
						<input v-model="socialAccountIds" type="checkbox" :value="account.id" />
						{{ account.identity }} <span class="subtle">({{ account.connector }})</span>
					</label>
					<p v-if="!accounts.length" class="subtle">No active social accounts.</p>
				</fieldset>
			</FormField>

			<FormField label="Title (optional)" for="feed-column-title" :disabled="saving">
				<input
					id="feed-column-title"
					v-model="title"
					type="text"
					maxlength="255"
					:placeholder="(kind === 'tag' ? tag.trim() : query.trim()) || 'Defaults to the search'"
					:disabled="saving"
				/>
			</FormField>

			<NotificationBlock v-if="error" type="error" :message="error" />

			<template #actions>
				<button type="button" class="neutral" :disabled="saving" @click="close">Cancel</button>
				<button type="submit" class="good" :disabled="saving || !canSubmit">
					{{ saving ? 'Saving…' : (column ? 'Save' : 'Add column') }}
				</button>
			</template>
		</FormLayout>
	</dialog>
</template>

<script setup>
	import { ref, computed, nextTick, watch } from 'vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import { TAG_QUERY_PREFIX, listPostTags, normaliseTag } from '../javascript/postTags.js';

	defineProps({
		accounts: {
			type: Array,
			default: () => [],
		},
	});

	const emit = defineEmits(['saved']);

	const dialog = ref(null);
	const queryInput = ref(null);
	const tagInput = ref(null);
	const column = ref(null);
	// 'search' (hashtag or search) or 'tag' (internal Japella tag).
	const kind = ref('search');
	const query = ref('');
	const tag = ref('');
	const knownTags = ref([]);
	const title = ref('');
	const socialAccountIds = ref([]);
	const saving = ref(false);
	const error = ref('');

	const canSubmit = computed(() => (kind.value === 'tag'
		? normaliseTag(tag.value) !== ''
		: query.value.trim() !== '' && socialAccountIds.value.length > 0));

	function focusFirstInput() {
		nextTick(() => (kind.value === 'tag' ? tagInput : queryInput).value?.focus());
	}

	watch(kind, focusFirstInput);

	async function open(existing = null, defaultAccountIds = []) {
		column.value = existing;
		kind.value = existing?.tag ? 'tag' : 'search';
		query.value = existing && !existing.tag ? existing.query : '';
		tag.value = existing?.tag || '';
		// Only keep a custom title; the server fills in the query as a default.
		title.value = existing && existing.title !== (existing.tag || existing.query) ? existing.title : '';
		socialAccountIds.value = existing ? (existing.accounts || []).map((a) => a.id) : [...defaultAccountIds];
		error.value = '';
		dialog.value?.showModal();
		focusFirstInput();
		try {
			knownTags.value = await listPostTags();
		} catch {
			knownTags.value = [];
		}
	}

	function close() {
		dialog.value?.close();
	}

	function onClose() {
		saving.value = false;
	}

	async function submit() {
		saving.value = true;
		error.value = '';
		const payload = kind.value === 'tag'
			? { query: `${TAG_QUERY_PREFIX}${normaliseTag(tag.value)}`, title: title.value, socialAccountIds: [] }
			: { query: query.value, title: title.value, socialAccountIds: socialAccountIds.value.map(Number) };
		try {
			const res = column.value
				? await window.client.updateFeedColumn({ id: column.value.id, ...payload })
				: await window.client.createFeedColumn(payload);
			emit('saved', res.column, Boolean(column.value));
			close();
		} catch (e) {
			error.value = e.rawMessage || e.message || String(e);
		} finally {
			saving.value = false;
		}
	}

	defineExpose({ open });
</script>

<style scoped>
	.feed-column-kind {
		display: flex;
		gap: 1rem;
		margin: 0;
		padding: 0;
		border: none;
	}

	.feed-column-accounts {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		margin: 0;
		padding: 0;
		border: none;
		max-height: 14rem;
		overflow-y: auto;
	}

	.feed-column-accounts p {
		margin: 0;
	}

	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
</style>
