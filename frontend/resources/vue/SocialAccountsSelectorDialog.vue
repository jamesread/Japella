<template>
	<dialog
		ref="dlg"
		@close="onDialogClose"
	>
		<div class="dialog-panel accounts-selector-panel">
			<h2 class="dlg-title">{{ title }}</h2>
			<p v-if="subtitle" class="dlg-subtitle">{{ subtitle }}</p>

			<SocialAccountsSelector
				v-model="selectedIds"
				:only-active="onlyActive"
				:only-can-post="onlyCanPost"
				:show-connector="showConnector"
				:empty-message="emptyMessage"
				:auto-load="false"
				ref="selector"
			/>

			<div class="dialog-actions">
				<button type="button" class="neutral" :disabled="saving" @click="onCancel">Cancel</button>
				<button
					type="button"
					class="good"
					:disabled="saving || (requireSelection && !selectedIds.length)"
					@click="onConfirm"
				>
					{{ saving ? savingLabel : confirmLabel }}{{ selectedIds.length ? ` (${selectedIds.length})` : '' }}
				</button>
			</div>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, nextTick } from 'vue';
	import SocialAccountsSelector from './SocialAccountsSelector.vue';

	const props = defineProps({
		title: { type: String, default: 'Select social accounts' },
		subtitle: { type: String, default: '' },
		confirmLabel: { type: String, default: 'Save' },
		savingLabel: { type: String, default: 'Saving…' },
		onlyActive: { type: Boolean, default: false },
		onlyCanPost: { type: Boolean, default: false },
		showConnector: { type: Boolean, default: false },
		emptyMessage: { type: String, default: 'No social accounts found.' },
		/** When true, confirm stays disabled until at least one account is selected. */
		requireSelection: { type: Boolean, default: false },
	});

	const emit = defineEmits({
		confirmed(ids) {
			return Array.isArray(ids);
		},
		cancel: () => true,
	});

	const dlg = ref(null);
	const selector = ref(null);
	const selectedIds = ref([]);
	const saving = ref(false);

	function resetState() {
		selectedIds.value = [];
		saving.value = false;
	}

	function onDialogClose() {
		resetState();
	}

	/**
	 * @param {{ selectedIds?: number[] }} [opts]
	 */
	async function open(opts = {}) {
		resetState();
		selectedIds.value = [...(opts.selectedIds || [])].map(Number).filter((id) => id > 0);
		dlg.value?.showModal();
		await nextTick();
		await selector.value?.loadAccounts();
	}

	function close() {
		dlg.value?.close();
	}

	function onCancel() {
		emit('cancel');
		close();
	}

	function onConfirm() {
		emit('confirmed', [...selectedIds.value]);
		close();
	}

	defineExpose({ open, close });
</script>

<style scoped>
	.accounts-selector-panel {
		width: min(36rem, calc(100vw - 2rem));
		max-height: min(70vh, 32rem);
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.dlg-title {
		margin: 0;
		font-size: 1.15rem;
	}

	.dlg-subtitle {
		margin: 0;
		font-size: 0.88rem;
		opacity: 0.85;
	}

	.dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 0.25rem;
	}
</style>
