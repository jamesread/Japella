<template>
	<section
		class="feed-column transparent"
		:class="{ resizing }"
		:aria-label="label || title"
		:style="{ width: `${width}px` }"
	>
		<div class="feed-column-header">
			<div class="feed-column-title">
				<Icon :icon="icon" width="18" height="18" aria-hidden="true" />
				<h3 :title="tooltip || title">{{ title }}</h3>
			</div>
			<div class="feed-column-actions" role="toolbar" :aria-label="`${title} column actions`">
				<slot name="actions" />
			</div>
			<div v-if="$slots.meta" class="feed-column-meta subtle">
				<slot name="meta" />
			</div>
		</div>

		<div ref="body" class="feed-column-body">
			<slot />
		</div>

		<div
			class="feed-column-resize"
			role="separator"
			aria-orientation="vertical"
			tabindex="0"
			:aria-label="`Resize ${title} column`"
			:aria-valuenow="width"
			:aria-valuemin="MIN_COLUMN_WIDTH"
			:aria-valuemax="MAX_COLUMN_WIDTH"
			title="Drag to resize; double-click to reset"
			@pointerdown="startResize"
			@dblclick="setWidth(DEFAULT_COLUMN_WIDTH)"
			@keydown="onResizeKey"
		/>
	</section>
</template>

<script setup>
	import { ref, watch, onUnmounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import {
		DEFAULT_COLUMN_WIDTH,
		MIN_COLUMN_WIDTH,
		MAX_COLUMN_WIDTH,
		clampColumnWidth,
		loadColumnWidth,
		saveColumnWidth,
	} from '../javascript/feedColumnWidths';

	const props = defineProps({
		title: {
			type: String,
			required: true,
		},
		icon: {
			type: String,
			default: 'mdi:view-column-outline',
		},
		tooltip: {
			type: String,
			default: '',
		},
		label: {
			type: String,
			default: '',
		},
		// Identifies the column for remembering its width.
		widthKey: {
			type: String,
			required: true,
		},
	});

	const KEY_STEP = 32;

	const width = ref(loadColumnWidth(props.widthKey));
	const resizing = ref(false);
	let dragStartX = 0;
	let dragStartWidth = 0;

	watch(() => props.widthKey, (key) => {
		width.value = loadColumnWidth(key);
	});

	function setWidth(px) {
		width.value = clampColumnWidth(px);
		saveColumnWidth(props.widthKey, width.value);
	}

	function onPointerMove(e) {
		width.value = clampColumnWidth(dragStartWidth + e.clientX - dragStartX);
	}

	function stopResize() {
		if (!resizing.value) {
			return;
		}
		resizing.value = false;
		window.removeEventListener('pointermove', onPointerMove);
		window.removeEventListener('pointerup', stopResize);
		window.removeEventListener('pointercancel', stopResize);
		saveColumnWidth(props.widthKey, width.value);
	}

	function startResize(e) {
		if (e.button !== 0) {
			return;
		}
		e.preventDefault();
		dragStartX = e.clientX;
		dragStartWidth = width.value;
		resizing.value = true;
		window.addEventListener('pointermove', onPointerMove);
		window.addEventListener('pointerup', stopResize);
		window.addEventListener('pointercancel', stopResize);
	}

	function onResizeKey(e) {
		if (e.key === 'ArrowLeft') {
			setWidth(width.value - KEY_STEP);
		} else if (e.key === 'ArrowRight') {
			setWidth(width.value + KEY_STEP);
		} else if (e.key === 'Home') {
			setWidth(MIN_COLUMN_WIDTH);
		} else if (e.key === 'End') {
			setWidth(MAX_COLUMN_WIDTH);
		} else {
			return;
		}
		e.preventDefault();
	}

	onUnmounted(stopResize);

	// The scrolling element, for infinite scroll and scroll anchoring.
	const body = ref(null);

	defineExpose({ body });
</script>

<style scoped>
	/* femtocrank's section.transparent drops the filled background and shadow;
	   a thin border keeps side-by-side columns apart. */
	.feed-column {
		position: relative;
		flex: 0 0 auto;
		/* Never wider than the screen, so a column is always fully viewable. */
		max-width: calc(100vw - 2rem);
		display: flex;
		flex-direction: column;
		padding: 0;
		margin: 0;
		overflow: hidden;
		border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
	}

	.feed-column.resizing {
		user-select: none;
	}

	.feed-column-resize {
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		width: 8px;
		cursor: col-resize;
		touch-action: none;
		z-index: 1;
	}

	.feed-column-resize:hover,
	.feed-column-resize:focus-visible,
	.feed-column.resizing .feed-column-resize {
		background: color-mix(in srgb, var(--link-color, #007bff) 35%, transparent);
		outline: none;
	}

	.feed-column-header {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 0.25rem 0.5rem;
		align-items: center;
		padding: 0.6rem 0.75rem;
		border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
		/* The column itself is transparent; only its header keeps the section fill. */
		background-color: var(--section-bg-color);
	}

	.feed-column-title {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		min-width: 0;
	}

	.feed-column-title h3 {
		margin: 0;
		font-size: 1rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.feed-column-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 0.15rem;
	}

	.feed-column-actions :slotted(button) {
		padding: 0.2em 0.35em;
	}

	/* One fixed-height line in every column, so all column headers line up.
	   Compact controls placed here size themselves to --feed-column-meta-height. */
	.feed-column-meta {
		--feed-column-meta-height: 1.5rem;
		grid-column: 1 / -1;
		display: flex;
		flex-wrap: nowrap;
		align-items: center;
		gap: 0 0.6rem;
		height: var(--feed-column-meta-height);
		overflow: hidden;
		white-space: nowrap;
		font-size: 0.8rem;
	}

	.feed-column-body {
		flex: 1;
		overflow-y: auto;
		padding: 0.5rem 0.75rem;
	}

	/* Posts draw their own card (PostPreview/PostBoost "filled"); just space them. */
	.feed-column-body :slotted(.feed-column-post) {
		margin-bottom: 0.5rem;
	}

	.feed-column-body :slotted(.feed-column-post:last-of-type) {
		margin-bottom: 0;
	}
</style>
