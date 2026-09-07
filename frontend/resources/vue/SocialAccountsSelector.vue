<template>
	<div class="social-accounts-selector">
		<div v-if="loading || internalLoading" class="muted">Loading social accounts…</div>
		<NotificationBlock
			v-else-if="loadError"
			type="error"
			:message="loadError"
		/>
		<NotificationBlock
			v-else-if="!displayAccounts.length"
			type="note"
			:message="emptyMessage"
		/>
		<div
			v-else
			class="account-list"
			role="group"
			:aria-label="ariaLabel"
		>
			<label
				v-for="acc in displayAccounts"
				:key="acc.id"
				class="check-list"
			>
				<input
					type="checkbox"
					:value="acc.id"
					:checked="selectedSet.has(acc.id)"
					:disabled="disabled"
					@change="onToggle(acc.id, $event.target.checked)"
				/>
				<span>
					<Icon :icon="acc.icon || 'mdi:account'" />
					{{ accountLabel(acc) }}
				</span>
			</label>

			<label v-if="showCannedPostOption" class="check-list">
				<input
					type="checkbox"
					:checked="cannedPost"
					:disabled="disabled"
					@change="onCannedPostChange($event.target.checked)"
				/>
				<span>
					<Icon icon="jam:box" />
					Canned post
				</span>
			</label>
		</div>
	</div>
</template>

<script setup>
	import { computed, ref, watch } from 'vue';
	import { Icon } from '@iconify/vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import { waitForClient } from '../javascript/util';

	const props = defineProps({
		/** Selected social account IDs. */
		modelValue: { type: Array, default: () => [] },
		/** Optional preloaded accounts; when omitted the selector loads them. */
		accounts: { type: Array, default: null },
		onlyActive: { type: Boolean, default: false },
		/** When true, only accounts the current user can post to. */
		onlyCanPost: { type: Boolean, default: false },
		/** External loading flag (e.g. parent still fetching membership). */
		loading: { type: Boolean, default: false },
		disabled: { type: Boolean, default: false },
		emptyMessage: {
			type: String,
			default: 'No social accounts available.',
		},
		ariaLabel: { type: String, default: 'Select social accounts' },
		showConnector: { type: Boolean, default: false },
		/** PostBox: include a “Canned post” checkbox after the account list. */
		showCannedPostOption: { type: Boolean, default: false },
		cannedPost: { type: Boolean, default: false },
		/** When true, fetch accounts on mount / when filter props change. */
		autoLoad: { type: Boolean, default: true },
	});

	const emit = defineEmits({
		'update:modelValue': (ids) => Array.isArray(ids),
		'update:cannedPost': (v) => typeof v === 'boolean',
		loaded: (accounts) => Array.isArray(accounts),
	});

	const internalAccounts = ref([]);
	const internalLoading = ref(false);
	const loadError = ref('');

	const displayAccounts = computed(() => {
		if (Array.isArray(props.accounts)) {
			return props.accounts;
		}
		return internalAccounts.value;
	});

	const selectedSet = computed(() => new Set(
		(props.modelValue || []).map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0),
	));

	function accountLabel(acc) {
		if (props.showConnector && acc.connector) {
			return `${acc.identity} (${acc.connector})`;
		}
		return acc.identity || `Account #${acc.id}`;
	}

	function onToggle(id, checked) {
		const numId = Number(id);
		const next = new Set(selectedSet.value);
		if (checked) {
			next.add(numId);
		} else {
			next.delete(numId);
		}
		emit('update:modelValue', [...next]);
	}

	function onCannedPostChange(checked) {
		emit('update:cannedPost', checked);
	}

	async function loadAccounts() {
		if (Array.isArray(props.accounts)) {
			return;
		}
		internalLoading.value = true;
		loadError.value = '';
		try {
			await waitForClient();
			const res = await window.client.getSocialAccounts({ onlyActive: props.onlyActive });
			let list = res.accounts || [];
			if (props.onlyCanPost) {
				list = list.filter((a) => a.canPost);
			}
			internalAccounts.value = list.map((a) => ({
				id: a.id,
				identity: a.identity,
				icon: a.icon,
				connector: a.connector,
				canPost: a.canPost,
				active: a.active,
			}));
			emit('loaded', internalAccounts.value);
		} catch (e) {
			loadError.value = e.message || 'Failed to load social accounts';
			internalAccounts.value = [];
			emit('loaded', []);
		} finally {
			internalLoading.value = false;
		}
	}

	watch(
		() => [props.onlyActive, props.onlyCanPost, props.accounts, props.autoLoad],
		() => {
			if (!props.autoLoad) return;
			if (Array.isArray(props.accounts)) return;
			loadAccounts();
		},
		{ immediate: true },
	);

	defineExpose({
		loadAccounts,
		reload: loadAccounts,
	});
</script>

<style scoped>
	.account-list {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.check-list {
		display: inline-block;
		margin: 0;
	}

	.check-list label,
	.check-list {
		cursor: pointer;
	}

	.check-list > span {
		display: inline-flex;
		align-items: center;
		gap: 0.4em;
		padding: 0.4em 0.6em;
		border-radius: 0.5em;
		background-color: #f5f5f5;
		color: #333;
		border: 1px solid #e0e0e0;
		font-weight: 500;
		user-select: none;
		white-space: nowrap;
		transition: all 0.2s ease;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.check-list input[type="checkbox"] {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}

	.check-list input[type="checkbox"]:checked + span {
		background-color: #e3f2fd;
		border-color: #2196f3;
		color: #1565c0;
		box-shadow: 0 2px 4px rgba(33, 150, 243, 0.2);
	}

	.check-list:hover > span {
		background-color: #e8e8e8;
		border-color: #d0d0d0;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		transform: translateY(-1px);
	}

	.check-list input[type="checkbox"]:checked + span:hover {
		background-color: #bbdefb;
		border-color: #1976d2;
		box-shadow: 0 2px 4px rgba(33, 150, 243, 0.3);
	}

	.check-list input[type="checkbox"]:disabled + span {
		opacity: 0.65;
		cursor: not-allowed;
		transform: none;
	}

	html[data-theme="dark"] {
		.check-list > span {
			background-color: #2a2a2a;
			color: #e0e0e0;
			border-color: #404040;
			box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
		}

		.check-list input[type="checkbox"]:checked + span {
			background-color: #1e3a5f;
			border-color: #2196f3;
			color: #90caf9;
			box-shadow: 0 2px 4px rgba(33, 150, 243, 0.3);
		}

		.check-list:hover > span {
			background-color: #353535;
			border-color: #505050;
			box-shadow: 0 2px 4px rgba(0, 0, 0, 0.4);
		}

		.check-list input[type="checkbox"]:checked + span:hover {
			background-color: #2a4a6f;
			border-color: #42a5f5;
			box-shadow: 0 2px 4px rgba(33, 150, 243, 0.4);
		}
	}

	.muted {
		opacity: 0.75;
	}
</style>
