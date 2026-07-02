<template>
	<Section
		:title="connector?.name || 'Connector'"
		:subtitle="connector ? connector.protocol : 'Loading connector…'"
		classes="social-account-pre-add"
	>
		<template #toolbar>
			<router-link :to="{ name: 'addSocialAccount' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Choose Connector
			</router-link>
			<a
				v-if="connector"
				:href="connectorDocsUrl(connector.protocol)"
				target="_blank"
				rel="noopener noreferrer"
				class="button neutral"
				title="Open documentation"
			>
				<Icon icon="material-symbols:menu-book" />
				Documentation
			</a>
		</template>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-if="loading" class="muted">Loading connector…</div>

		<div v-else-if="!connector">
			<p class="inline-notification note">Connector not found.</p>
		</div>

		<template v-else>
			<div class="connector-hero">
				<HugeiconsIcon :icon="connectorHugeIcon(connector.protocol)" width="48" height="48" />
				<p class="connector-intro">{{ connectorIntroText(connector.protocol) }}</p>
			</div>

			<dl class="connector-details">
				<dt>Protocol</dt>
				<dd>{{ connector.protocol }}</dd>

				<dt>Display name</dt>
				<dd>{{ connector.name }}</dd>

				<dt>Configuration</dt>
				<dd>
					<span v-if="connector.hasOauth" class="tag oauth-tag">OAuth</span>
					<span v-if="connector.usesYamlConfig" class="tag yaml-tag">YAML Config</span>
					<span v-if="!connector.hasOauth && !connector.usesYamlConfig" class="muted">—</span>
				</dd>

				<dt>Capabilities</dt>
				<dd>
					<span v-if="connector.supportsSocialAccounts" class="tag">Social Accounts</span>
					<span v-if="connector.supportsChatbot" class="tag">Chat Bot</span>
					<span v-if="!connector.supportsSocialAccounts && !connector.supportsChatbot" class="muted">—</span>
				</dd>

				<dt>Status</dt>
				<dd>
					<span v-if="isUnregistered" class="inline-notification warning small">
						{{ connector.notStartedReason || 'Not started' }}
					</span>
					<span v-else class="tag good-tag">Available</span>
				</dd>
			</dl>

			<ConnectorConfigurationIssues
				:issues="connector.issues"
				:registering="registeringClient"
				@register-client="registerOAuthClient"
			/>
		</template>
	</Section>

	<Section
		v-if="connector && !loading"
		title="Connect Account"
		subtitle="Sign in to link this connector to Japella."
	>
		<div v-if="isUnregistered" class="inline-notification note">
			This connector is not running. Ask an administrator to configure and start it before you can connect an account.
		</div>

		<template v-else-if="connector.hasOauth">
			<p v-if="connectorHasBlockingIssues(connector.issues)" class="muted">
				Resolve the configuration issues above before connecting.
			</p>
			<button
				v-else
				type="button"
				class="good connect-btn"
				:disabled="connecting"
				@click="startOAuth"
			>
				<Icon v-if="connecting" icon="eos-icons:loading" width="16" height="16" />
				<Icon v-else icon="material-symbols:login" width="16" height="16" />
				{{ connecting ? 'Redirecting…' : 'Connect with ' + connector.name }}
			</button>
		</template>

		<p v-else-if="connector.usesYamlConfig" class="inline-notification note">
			This connector is configured via YAML and does not require OAuth login. Contact an administrator if you need access.
		</p>

		<p v-else class="inline-notification note">
			This connector does not support OAuth account linking.
		</p>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import ConnectorConfigurationIssues from './ConnectorConfigurationIssues.vue';
	import {
		waitForClient,
		connectorDocsUrl,
		connectorHugeIcon,
		connectorIntroText,
		connectorUsesOauth,
		connectorUsesYamlConfig,
		normalizeConnectorIssues,
		connectorHasBlockingIssues,
	} from '../javascript/util';

	const route = useRoute();

	const loading = ref(true);
	const errorMessage = ref('');
	const connecting = ref(false);
	const registeringClient = ref(false);
	const connectors = ref([]);
	const unregisteredConnectors = ref([]);

	const connectorId = computed(() => String(route.params.connectorId || ''));
	const isUnregistered = computed(() => route.query.unregistered === '1');

	const connector = computed(() => {
		if (isUnregistered.value) {
			return unregisteredConnectors.value.find(
				(c) => c.protocol === connectorId.value || c.name === connectorId.value
			);
		}

		return connectors.value.find(
			(c) => c.name === connectorId.value || c.protocol === connectorId.value
		);
	});

	function normalizeStarted(c) {
		return {
			name: c.name,
			protocol: c.protocol || c.name,
			icon: c.icon,
			hasOauth: !!c.hasOauth,
			usesYamlConfig: !!c.usesYamlConfig,
			supportsSocialAccounts: !!c.supportsSocialAccounts,
			supportsChatbot: !!c.supportsChatbot,
			supportsClientRegistration: !!c.supportsClientRegistration,
			isRegistered: !!c.isRegistered,
			issues: normalizeConnectorIssues(c.issues),
		};
	}

	function normalizeUnregistered(c) {
		const protocol = c.protocol;
		return {
			protocol,
			name: c.name || protocol,
			icon: c.icon,
			notStartedReason: c.notStartedReason || '',
			hasOauth: connectorUsesOauth(protocol),
			usesYamlConfig: connectorUsesYamlConfig(protocol),
			supportsSocialAccounts: connectorUsesOauth(protocol),
			supportsChatbot: connectorUsesYamlConfig(protocol),
			issues: [],
		};
	}

	async function fetchConnectors() {
		loading.value = true;
		errorMessage.value = '';

		try {
			await waitForClient();
			const res = await window.client.getConnectors({ onlyWantOauth: false });

			connectors.value = (res.connectors || []).map(normalizeStarted);
			unregisteredConnectors.value = (res.unregisteredConnectors || []).map(normalizeUnregistered);
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load connector.';
		} finally {
			loading.value = false;
		}
	}

	async function registerOAuthClient() {
		if (!connector.value?.supportsClientRegistration || registeringClient.value) {
			return;
		}

		registeringClient.value = true;
		errorMessage.value = '';

		try {
			await waitForClient();
			const res = await window.client.registerConnector({ name: connector.value.name });
			if (!res.standardResponse?.success) {
				errorMessage.value = res.standardResponse?.message || 'Failed to register OAuth application.';
				return;
			}
			await fetchConnectors();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to register OAuth application.';
		} finally {
			registeringClient.value = false;
		}
	}

	async function startOAuth() {
		if (!connector.value?.hasOauth || connectorHasBlockingIssues(connector.value.issues)) {
			return;
		}

		connecting.value = true;
		errorMessage.value = '';

		try {
			await waitForClient();
			const res = await window.client.startOAuth({ connectorId: connector.value.name });
			if (res.url) {
				window.location.href = res.url;
			} else {
				errorMessage.value = 'No OAuth URL returned. Check connector configuration.';
				connecting.value = false;
			}
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to start OAuth flow.';
			connecting.value = false;
		}
	}

	onMounted(() => {
		fetchConnectors();
	});
</script>

<style scoped>
	.connector-hero {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.connector-intro {
		margin: 0;
		line-height: 1.5;
		max-width: 42rem;
	}

	.connector-details {
		display: grid;
		grid-template-columns: 10rem 1fr;
		gap: 0.75rem 1rem;
		margin: 0;
	}

	.connector-details dt {
		font-weight: 600;
		opacity: 0.85;
	}

	.connector-details dd {
		margin: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		align-items: center;
	}

	.tag {
		display: inline-block;
		padding: 0.15rem 0.5rem;
		font-size: 0.75rem;
		font-weight: 600;
		border-radius: 3px;
		background: var(--pico-muted-border-color, rgba(255, 255, 255, 0.1));
	}

	.oauth-tag {
		background: rgba(26, 115, 232, 0.2);
		color: var(--pico-primary, #1a73e8);
	}

	.good-tag {
		background: rgba(46, 125, 50, 0.2);
		color: #2e7d32;
	}

	.connect-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
	}

	.muted {
		opacity: 0.7;
	}

	.small {
		font-size: 0.85em;
	}
</style>
