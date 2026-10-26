<template>
	<div v-if="tags.length" class="post-tags" role="group" aria-label="Your tags (private to you)">
		<button
			v-for="tag in tags"
			:key="tag"
			type="button"
			class="neutral small post-tag"
			:title="openTagColumn ? `Show the ${tag} tag column` : `Tagged ${tag}`"
			:disabled="!openTagColumn"
			@click="openTagColumn?.(tag)"
		>
			<Icon icon="mdi:tag-outline" aria-hidden="true" />
			<span>{{ tag }}</span>
		</button>
		<button type="button" class="neutral small post-tag" title="Edit tags" aria-label="Edit tags" @click="openEditor">
			<Icon icon="mdi:pencil-outline" aria-hidden="true" />
		</button>
	</div>

	<dialog ref="dialog" class="dialog post-tags-dialog" :aria-labelledby="`${uid}-title`" @close="onClose">
		<h2 :id="`${uid}-title`">Tags</h2>
		<p class="subtle">
			Tags are private to you in Japella, for keeping track of posts. Nothing is posted to the network.
		</p>

		<ul v-if="tags.length" class="post-tags-list">
			<li v-for="tag in tags" :key="tag">
				<Icon icon="mdi:tag-outline" aria-hidden="true" />
				<span class="post-tags-name">{{ tag }}</span>
				<button
					v-if="openTagColumn"
					type="button"
					class="neutral small"
					:title="`Show the ${tag} tag column`"
					@click="showColumn(tag)"
				>
					<Icon icon="mdi:view-column-outline" aria-hidden="true" />
					<span>Column</span>
				</button>
				<button
					type="button"
					class="neutral small"
					:title="`Remove ${tag}`"
					:aria-label="`Remove tag ${tag}`"
					:disabled="saving"
					@click="removeTag(tag)"
				>
					<Icon icon="mdi:close" aria-hidden="true" />
				</button>
			</li>
		</ul>
		<p v-else class="subtle">This post has no tags.</p>

		<form class="post-tags-add" @submit.prevent="addTag">
			<label :for="`${uid}-input`" class="visually-hidden">Add a tag</label>
			<input
				:id="`${uid}-input`"
				ref="input"
				v-model="draft"
				type="text"
				maxlength="65"
				placeholder="Add a tag, like todo"
				autocomplete="off"
				:list="`${uid}-suggestions`"
				:disabled="saving"
			/>
			<datalist :id="`${uid}-suggestions`">
				<option v-for="s in suggestions" :key="s.tag" :value="s.tag">{{ s.posts }} {{ s.posts === 1 ? 'post' : 'posts' }}</option>
			</datalist>
			<button type="submit" class="good" :disabled="saving || !draft.trim()">Add</button>
		</form>

		<NotificationBlock v-if="error" type="error" :message="error" />

		<div class="post-tags-footer">
			<button type="button" class="neutral" @click="dialog?.close()">Done</button>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, computed, inject, nextTick, useId } from 'vue';
	import { Icon } from '@iconify/vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import { tagsFor, savePostTags, listPostTags, normaliseTag, TAG_PATTERN } from '../javascript/postTags.js';

	const props = defineProps({
		post: {
			type: Object,
			required: true,
		},
	});

	// Provided by the feed page; elsewhere tags can't open a column.
	const openTagColumn = inject('openTagColumn', null);

	const uid = `post-tags-${useId()}`;
	const dialog = ref(null);
	const input = ref(null);
	const draft = ref('');
	const saving = ref(false);
	const error = ref('');
	const allTags = ref([]);

	const tags = computed(() => tagsFor(props.post));
	const suggestions = computed(() => allTags.value.filter((s) => !tags.value.includes(s.tag)));

	async function openEditor() {
		draft.value = '';
		error.value = '';
		dialog.value?.showModal();
		await nextTick();
		input.value?.focus();
		try {
			allTags.value = await listPostTags();
		} catch {
			allTags.value = [];
		}
	}

	function onClose() {
		draft.value = '';
	}

	async function save(next) {
		saving.value = true;
		error.value = '';
		try {
			await savePostTags(props.post, next);
			return true;
		} catch (e) {
			error.value = `Failed to save tags: ${e.rawMessage || e.message || e}`;
			return false;
		} finally {
			saving.value = false;
		}
	}

	async function addTag() {
		const tag = normaliseTag(draft.value);
		if (!TAG_PATTERN.test(tag)) {
			error.value = 'Tags may only contain letters, numbers, - and _ (up to 64 characters).';
			return;
		}
		if (tags.value.includes(tag)) {
			draft.value = '';
			return;
		}
		if (await save([...tags.value, tag])) {
			draft.value = '';
			await nextTick();
			input.value?.focus();
		}
	}

	function removeTag(tag) {
		save(tags.value.filter((t) => t !== tag));
	}

	function showColumn(tag) {
		dialog.value?.close();
		openTagColumn?.(tag);
	}

	defineExpose({ openEditor });
</script>

<style scoped>
	.post-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem;
		margin-top: 0.4rem;
	}

	.post-tag {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		padding: 0.1em 0.45em;
		font-size: 0.8rem;
	}

	.post-tag:disabled {
		cursor: default;
		opacity: 1;
	}

	.post-tags-dialog {
		width: min(28rem, calc(100vw - 2rem));
	}

	.post-tags-list {
		list-style: none;
		margin: 0 0 0.75rem;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}

	.post-tags-list li {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.post-tags-list li button {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
	}

	.post-tags-name {
		flex: 1;
		min-width: 0;
		overflow-wrap: anywhere;
	}

	.post-tags-add {
		display: flex;
		gap: 0.4rem;
	}

	.post-tags-add input {
		flex: 1;
		min-width: 0;
	}

	.post-tags-footer {
		display: flex;
		justify-content: flex-end;
		margin-top: 1rem;
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
