<template>
	<router-link
		:to="{ name: 'socialAccountDetails', params: { id: socialAccountId } }"
		class="social-account-chip"
		:class="{ 'social-account-chip-transparent': transparent }"
		:title="chipTitle"
	>
		<Icon :icon="icon || 'mdi:account'" />
		<span>{{ identity }}</span>
	</router-link>
</template>

<script setup>
	import { computed } from 'vue';
	import { Icon } from '@iconify/vue';

	const props = defineProps({
		socialAccountId: {
			type: [Number, String],
			required: true,
		},
		identity: {
			type: String,
			default: '',
		},
		icon: {
			type: String,
			default: 'mdi:account',
		},
		transparent: {
			type: Boolean,
			default: false,
		},
	});

	const chipTitle = computed(() => {
		if (props.identity) {
			return `View social account ${props.identity}`;
		}
		return 'View social account';
	});
</script>

<style scoped>
	.social-account-chip {
		white-space: nowrap;
		border-radius: 0.5em;
		background-color: #f5f5f5;
		color: #333;
		border: 1px solid #e0e0e0;
		font-weight: 500;
		padding: 0.4em 0.6em;
		display: inline-flex;
		align-items: center;
		gap: 0.4em;
		user-select: none;
		text-decoration: none;
		transition: all 0.2s ease;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.social-account-chip:hover {
		background-color: #e8e8e8;
		border-color: #d0d0d0;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		transform: translateY(-1px);
		text-decoration: none;
	}

	.social-account-chip-transparent {
		background-color: transparent;
		border-color: transparent;
		box-shadow: none;
	}

	.social-account-chip-transparent:hover {
		background-color: rgba(0, 0, 0, 0.05);
		border-color: transparent;
		box-shadow: none;
		transform: none;
	}

	html[data-theme="dark"] {
		.social-account-chip {
			background-color: #2a2a2a;
			color: #e0e0e0;
			border-color: #404040;
			box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
		}

		.social-account-chip:hover {
			background-color: #353535;
			border-color: #505050;
			box-shadow: 0 2px 4px rgba(0, 0, 0, 0.4);
		}

		.social-account-chip-transparent {
			background-color: transparent;
			border-color: transparent;
			box-shadow: none;
		}

		.social-account-chip-transparent:hover {
			background-color: rgba(255, 255, 255, 0.08);
			border-color: transparent;
			box-shadow: none;
		}
	}
</style>
