<template>
	<div v-if="open" ref="panelEl" class="emoji-picker-panel">
		<emoji-picker v-if="ready" class="emoji-picker-widget" @emoji-click="onEmojiClick" />
		<p v-else class="emoji-picker-loading">Loading…</p>
	</div>
</template>

<script setup>
	import { ref, watch, onMounted, onUnmounted } from 'vue';

	const props = defineProps({
		open: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['select', 'close']);

	const panelEl = ref(null);
	const ready = ref(false);
	let pickerModuleLoaded = false;

	async function loadPicker() {
		if (!pickerModuleLoaded) {
			await import('emoji-picker-element');
			pickerModuleLoaded = true;
		}
		ready.value = true;
	}

	function onEmojiClick(event) {
		emit('select', event.detail.unicode);
	}

	function onDocumentPointerDown(event) {
		if (!props.open) {
			return;
		}
		const panel = panelEl.value;
		if (!panel) {
			return;
		}
		if (panel.contains(event.target) || event.target.closest('.emoji-picker-trigger')) {
			return;
		}
		emit('close');
	}

	function onKeydown(event) {
		if (event.key === 'Escape' && props.open) {
			emit('close');
		}
	}

	watch(
		() => props.open,
		(isOpen) => {
			if (isOpen) {
				ready.value = false;
				loadPicker();
			}
		},
	);

	onMounted(() => {
		document.addEventListener('pointerdown', onDocumentPointerDown, true);
		document.addEventListener('keydown', onKeydown);
	});

	onUnmounted(() => {
		document.removeEventListener('pointerdown', onDocumentPointerDown, true);
		document.removeEventListener('keydown', onKeydown);
	});
</script>

<style scoped>
	.emoji-picker-panel {
		position: absolute;
		top: calc(100% + 0.35rem);
		left: 0;
		z-index: 1000;
		border-radius: 0.5rem;
		overflow: hidden;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
		border: 1px solid var(--border-color, #ccc);
		background: var(--background-primary, #fff);
	}

	.emoji-picker-widget {
		--emoji-size: 1.375rem;
		--num-columns: 8;
		--background: var(--background-primary, #fff);
		--border-color: var(--border-color, #ccc);
		--button-active-background: var(--hover-background-color, #e8e8e8);
		--button-hover-background: var(--hover-background-color, #f0f0f0);
		--input-border-color: var(--border-color, #ccc);
		--outline-color: var(--accent-color, #5a9fd4);
	}

	.emoji-picker-loading {
		margin: 0;
		padding: 1rem 1.5rem;
		font-size: 0.9rem;
		opacity: 0.75;
	}

	html[data-theme="dark"] {
		.emoji-picker-widget {
			--background: #343434;
			--border-color: #595959;
			--button-active-background: #1d345c;
			--button-hover-background: #404040;
			--input-border-color: #595959;
			--category-font-color: #ddd;
			--indicator-color: #90caf9;
			--input-font-color: #ddd;
			--input-placeholder-color: #999;
		}

		.emoji-picker-panel {
			background: #343434;
		}
	}
</style>
