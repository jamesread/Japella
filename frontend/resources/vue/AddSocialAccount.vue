<template>
	<Section
		title="Add Social Account"
		subtitle="Connect a new social account by choosing a connector below."
		classes="add-social-account"
	>
		<template #toolbar>
			<router-link :to="{ name: 'socialAccounts' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to Accounts
			</router-link>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="fetchConnectors">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-if="loading && !connectors.length" class="muted">Loading connectors…</div>

		<template v-else>
			<p v-if="!connectors.length && !unregisteredConnectors.length" class="inline-notification note">
				No connectors are available. Ask an administrator to configure connectors in the system settings.
			</p>

			<div v-if="connectors.length" class="connector-grid">
				<div
					v-for="c in connectors"
					:key="c.name"
					class="connector-card"
					:class="{ 'has-oauth': c.hasOauth, 'no-oauth': !c.hasOauth }"
				>
					<div class="connector-header">
						<Icon :icon="c.icon || 'mdi:help-circle-outline'" width="28" height="28" />
						<div>
							<strong class="connector-name">{{ c.name }}</strong>
							<span v-if="c.protocol !== c.name" class="connector-protocol">{{ c.protocol }}</span>
						</div>
					</div>

					<div class="connector-tags">
						<span v-if="c.hasOauth" class="tag oauth-tag">OAuth</span>
						<span v-if="c.usesYamlConfig" class="tag yaml-tag">YAML Config</span>
						<span v-if="c.supportsSocialAccounts" class="tag social-tag">Social Accounts</span>
						<span v-if="c.supportsChatbot" class="tag chatbot-tag">Chat Bot</span>
					</div>

					<div v-if="c.issues && c.issues.length" class="connector-issues">
						<p v-for="issue in c.issues" :key="issue" class="inline-notification error small">{{ issue }}</p>
					</div>

					<div class="connector-actions">
						<button
							v-if="c.hasOauth && !c.issues.length"
							type="button"
							class="good connect-btn"
							:disabled="connectingId === c.name"
							@click="startOAuth(c)"
						>
							<Icon v-if="connectingId === c.name" icon="eos-icons:loading" width="16" height="16" />
							<Icon v-else icon="material-symbols:login" width="16" height="16" />
							{{ connectingId === c.name ? 'Redirecting…' : 'Connect with ' + c.name }}
						</button>
						<p v-else-if="c.hasOauth && c.issues.length" class="muted small">
							Resolve issues above before connecting.
						</p>
						<p v-else-if="c.usesYamlConfig" class="muted small">
							This connector is configured via YAML and does not require OAuth login.
						</p>
						<p v-else class="muted small">
							This connector does not support OAuth.
						</p>
					</div>
				</div>
			</div>

			<div v-if="unregisteredConnectors.length" class="unregistered-section">
				<h3 class="subsection-title">Unavailable Connectors</h3>
				<p class="muted small">These connectors exist but are not currently started.</p>
				<div class="connector-grid">
					<div
						v-for="c in unregisteredConnectors"
						:key="c.protocol"
						class="connector-card unregistered"
					>
						<div class="connector-header">
							<Icon :icon="c.icon || 'mdi:help-circle-outline'" width="28" height="28" />
							<strong class="connector-name">{{ c.name }}</strong>
						</div>
						<p class="muted small">{{ c.notStartedReason || 'Not started' }}</p>
					</div>
				</div>
			</div>
		</template>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';

	const connectors = ref([]);
	const unregisteredConnectors = ref([]);
	const loading = ref(true);
	const errorMessage = ref('');
	const connectingId = ref(null);

	async function fetchConnectors() {
		loading.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.getConnectors({ onlyWantOauth: false });

			const list = (res.connectors || []).map((c) => ({
				name: c.name,
				protocol: c.protocol || c.name,
				icon: c.icon,
				hasOauth: !!c.hasOauth,
				usesYamlConfig: !!c.usesYamlConfig,
				supportsSocialAccounts: !!c.supportsSocialAccounts,
				supportsChatbot: !!c.supportsChatbot,
				issues: c.issues || [],
			}));
			list.sort((a, b) => {
				if (a.hasOauth !== b.hasOauth) return a.hasOauth ? -1 : 1;
				return a.name.localeCompare(b.name);
			});
			connectors.value = list;

			const unregisteredList = (res.unregisteredConnectors || []).map((c) => ({
				protocol: c.protocol,
				name: c.name || c.protocol,
				icon: c.icon,
				notStartedReason: c.notStartedReason || '',
			}));
			unregisteredList.sort((a, b) => a.name.localeCompare(b.name));
			unregisteredConnectors.value = unregisteredList;
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load connectors.';
		} finally {
			loading.value = false;
		}
	}

	async function startOAuth(connector) {
		connectingId.value = connector.name;
		errorMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.startOAuth({ connectorId: connector.name });
			if (res.url) {
				window.location.href = res.url;
			} else {
				errorMessage.value = 'No OAuth URL returned. Check connector configuration.';
				connectingId.value = null;
			}
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to start OAuth flow.';
			connectingId.value = null;
		}
	}

	onMounted(() => {
		fetchConnectors();
	});
</script>

<style scoped>
	.subsection-title {
		margin: 1.5rem 0 0.5rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.connector-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 1rem;
		margin-top: 1rem;
	}

	.connector-card {
		border: 1px solid var(--pico-muted-border-color, rgba(255, 255, 255, 0.15));
		border-radius: 0.5rem;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.connector-card.has-oauth {
		border-color: var(--pico-primary, #1a73e8);
	}

	.connector-card.unregistered {
		opacity: 0.6;
	}

	.connector-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.connector-name {
		display: block;
		font-size: 1.05em;
		text-transform: capitalize;
	}

	.connector-protocol {
		display: block;
		font-size: 0.8em;
		opacity: 0.7;
	}

	.connector-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
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

	.connector-issues {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.connector-actions {
		margin-top: auto;
	}

	.connect-btn {
		width: 100%;
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
