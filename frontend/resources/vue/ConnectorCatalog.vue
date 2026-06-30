<template>
	<div class="connector-catalog">
		<p v-if="emptyMessage && !connectors.length && !unregisteredConnectors.length" class="inline-notification note">
			{{ emptyMessage }}
		</p>

		<div v-if="connectors.length">
			<h3 v-if="startedTitle" class="subsection-title">{{ startedTitle }}</h3>
			<Navigation ref="startedNavigation">
				<NavigationGrid />
			</Navigation>
		</div>

		<div v-if="unregisteredConnectors.length" class="unregistered-section">
			<h3 class="subsection-title">{{ unregisteredTitle }}</h3>
			<p class="muted small">{{ unregisteredDescription }}</p>
			<Navigation ref="unregisteredNavigation">
				<NavigationGrid />
			</Navigation>
		</div>
	</div>
</template>

<script setup>
	import { ref, watch, nextTick } from 'vue';
	import { useRouter } from 'vue-router';
	import { connectorDocsUrl, connectorHugeIcon } from '../javascript/util';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';

	const router = useRouter();

	const props = defineProps({
		connectors: {
			type: Array,
			default: () => [],
		},
		unregisteredConnectors: {
			type: Array,
			default: () => [],
		},
		selectionMode: {
			type: String,
			default: 'default',
			validator: (value) => ['default', 'preAdd'].includes(value),
		},
		showOAuthActions: {
			type: Boolean,
			default: false,
		},
		showDocLinks: {
			type: Boolean,
			default: true,
		},
		connectingId: {
			type: String,
			default: null,
		},
		startedTitle: {
			type: String,
			default: '',
		},
		unregisteredTitle: {
			type: String,
			default: 'Unavailable Connectors',
		},
		unregisteredDescription: {
			type: String,
			default: 'These connectors exist but are not currently started.',
		},
		emptyMessage: {
			type: String,
			default: '',
		},
	});

	const emit = defineEmits(['connect-oauth']);

	const startedNavigation = ref(null);
	const unregisteredNavigation = ref(null);

	function connectorTags(connector) {
		const tags = [];
		if (connector.hasOauth) tags.push('OAuth');
		if (connector.usesYamlConfig) tags.push('YAML Config');
		if (connector.supportsSocialAccounts) tags.push('Social Accounts');
		if (connector.supportsChatbot) tags.push('Chat Bot');
		return tags;
	}

	function buildStartedDescription(connector) {
		const parts = [];

		const tags = connectorTags(connector);
		if (tags.length) {
			parts.push(tags.join(' · '));
		}

		if (connector.protocol !== connector.name) {
			parts.push(connector.protocol);
		}

		if (connector.issues?.length) {
			parts.push(connector.issues.join('; '));
		}

		if (props.selectionMode === 'preAdd') {
			parts.push('View details and connect');
		} else if (props.showOAuthActions) {
			if (props.connectingId === connector.name) {
				parts.push('Redirecting to OAuth provider…');
			} else if (connector.hasOauth && !connector.issues?.length) {
				parts.push('Connect with OAuth');
			} else if (connector.hasOauth && connector.issues?.length) {
				parts.push('Resolve issues before connecting');
			} else if (connector.usesYamlConfig) {
				parts.push('Configured via YAML; no OAuth login required');
			} else {
				parts.push('OAuth not supported');
			}
		}

		if (props.showDocLinks && props.selectionMode !== 'preAdd') {
			parts.push('Documentation available');
		}

		return parts.join(' · ') || 'Connector';
	}

	function buildUnregisteredDescription(connector) {
		const parts = [connector.notStartedReason || 'Not started'];
		if (props.selectionMode === 'preAdd') {
			parts.push('View details');
		} else if (props.showDocLinks) {
			parts.push('Documentation available');
		}
		return parts.join(' · ');
	}

	function openDocs(protocol) {
		window.open(connectorDocsUrl(protocol), '_blank', 'noopener,noreferrer');
	}

	function handleStartedClick(connector) {
		if (props.selectionMode === 'preAdd') {
			router.push({
				name: 'socialAccountPreAdd',
				params: { connectorId: connector.name },
			});
			return;
		}

		if (
			props.showOAuthActions &&
			connector.hasOauth &&
			!connector.issues?.length &&
			props.connectingId !== connector.name
		) {
			emit('connect-oauth', connector);
			return;
		}

		if (props.showDocLinks) {
			openDocs(connector.protocol);
		}
	}

	function handleUnregisteredClick(connector) {
		if (props.selectionMode === 'preAdd') {
			router.push({
				name: 'socialAccountPreAdd',
				params: { connectorId: connector.protocol },
				query: { unregistered: '1' },
			});
			return;
		}

		if (props.showDocLinks) {
			openDocs(connector.protocol);
		}
	}

	function updateStartedNavigation() {
		const nav = startedNavigation.value;
		if (!nav) {
			return;
		}

		nav.clearNavigationLinks();

		for (const connector of props.connectors) {
			const displayName = connector.name;
			nav.addCallback(displayName, () => handleStartedClick(connector), {
				icon: connectorHugeIcon(connector.protocol),
				name: `connector-${connector.protocol}-${connector.name}`,
				description: buildStartedDescription(connector),
			});
		}
	}

	function updateUnregisteredNavigation() {
		const nav = unregisteredNavigation.value;
		if (!nav) {
			return;
		}

		nav.clearNavigationLinks();

		for (const connector of props.unregisteredConnectors) {
			nav.addCallback(connector.name, () => handleUnregisteredClick(connector), {
				icon: connectorHugeIcon(connector.protocol),
				name: `unregistered-${connector.protocol}`,
				description: buildUnregisteredDescription(connector),
			});
		}
	}

	async function updateNavigation() {
		await nextTick();
		updateStartedNavigation();
		updateUnregisteredNavigation();
	}

	watch(
		() => [
			props.connectors,
			props.unregisteredConnectors,
			props.selectionMode,
			props.showOAuthActions,
			props.connectingId,
			startedNavigation.value,
			unregisteredNavigation.value,
		],
		() => {
			updateNavigation();
		},
		{ deep: true }
	);
</script>

<style scoped>
	.subsection-title {
		margin: 1.5rem 0 0.5rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.muted {
		opacity: 0.7;
	}

	.small {
		font-size: 0.85em;
	}

	.unregistered-section {
		margin-top: 2rem;
		padding-top: 1rem;
		border-top: 1px solid var(--pico-muted-border-color, rgba(255, 255, 255, 0.1));
	}
</style>
