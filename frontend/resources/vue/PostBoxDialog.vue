<template>
	<dialog ref="dialog" class="post-box-dialog" aria-labelledby="post-box-dialog-title" @close="emit('close')">
		<h2 id="post-box-dialog-title">{{ t('section.postbox.title') }}</h2>

		<PostBox embedded />

		<div class="post-box-dialog-actions">
			<router-link :to="{ name: 'postBox' }" class="button neutral" @click="dialog?.close()">Open full page</router-link>
			<button type="button" class="neutral" @click="dialog?.close()">Close</button>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useI18n } from 'vue-i18n';
	import PostBox from './PostBox.vue';

	const { t } = useI18n();

	const emit = defineEmits(['close']);

	const dialog = ref(null);

	onMounted(() => dialog.value?.showModal());
</script>

<style scoped>
	/* Standard femtocrank dialog; only allow tall composers to scroll. */
	.post-box-dialog {
		max-height: calc(100vh - 2rem);
		overflow-y: auto;
	}

	.post-box-dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: 1em;
		margin-top: 1em;
	}
</style>
