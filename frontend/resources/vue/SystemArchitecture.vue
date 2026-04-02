<template>
	<Section
		title="System architecture"
		subtitle="High-level view of Japella components, persistence, and asynchronous messaging"
		classes="system-architecture"
	>
		<template #toolbar>
			<router-link :to="{ name: 'controlPanel' }" class="button neutral">
				<Icon icon="mdi:arrow-left" />
				Control Panel
			</router-link>
		</template>

		<p class="intro">
			The browser loads the Vue admin UI from the same HTTP server as the Control API. That process
			runs in-process workers and platform connectors. Health tinting for the database and message
			broker reflects live status from the server when this page loads.
		</p>

		<p v-if="statusError" class="inline-notification error">{{ statusError }}</p>
		<p v-else-if="diagramLoading" class="muted">Loading diagram…</p>
		<div
			v-show="!diagramLoading && !statusError"
			ref="diagramRoot"
			class="mermaid-wrap"
			aria-label="System architecture diagram"
		/>
		<div v-if="diagramError" class="inline-notification error">{{ diagramError }}</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted, nextTick } from 'vue';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';

	const diagramRoot = ref(null);
	const diagramError = ref('');
	const diagramLoading = ref(true);
	const statusError = ref('');

	function buildDiagramSource(status) {
		const dbBad = !status.databaseConnected || status.databaseSchemaDirty;
		const dbClass = dbBad ? 'healthBad' : 'healthOk';

		let dbLabel = 'Database<br/>MySQL / MariaDB';
		if (!status.databaseConnected) {
			dbLabel += '<br/>(not connected)';
		} else if (status.databaseSchemaDirty) {
			dbLabel += '<br/>(schema migration dirty)';
		}

		let mqLabel = 'Message broker<br/>AMQP / RabbitMQ';
		let mqClass = 'healthNeutral';
		if (status.amqpEnabled) {
			mqClass = status.amqpConnected ? 'healthOk' : 'healthBad';
			if (!status.amqpConnected) {
				mqLabel += '<br/>(disconnected)';
			}
		} else {
			mqLabel = 'Message broker<br/>disabled in configuration';
		}

		return `flowchart TB
  classDef healthOk fill:#14532d,stroke:#22c55e,color:#dcfce7
  classDef healthBad fill:#7f1d1d,stroke:#f87171,color:#fee2e2
  classDef healthNeutral fill:#374151,stroke:#9ca3af,color:#e5e7eb

  subgraph clients["Clients"]
    Browser["Web browser"]
  end

  subgraph japella["Japella Container"]
    API["HTTP server<br/>Vue SPA, Control API, static assets"]
    subgraph runtime["In-process runtime"]
      Workers["Nanoservices & Jobs"]
      Conn["Connectors<br/>chat and social platforms"]
    end
  end

  DB[("${dbLabel}")]
  MQ[["${mqLabel}"]]
  Ext["Third-party APIs<br/>Telegram, Discord, Bluesky, …"]

  Browser --> API
  API --> DB
  API --> MQ
  MQ <--> Workers
  Workers <--> Conn
  Conn <--> Ext
  API -.-> Conn

  class DB ${dbClass}
  class MQ ${mqClass}
`;
	}

	onMounted(async () => {
		await waitForClient();
		let status;
		try {
			status = await window.client.getStatus({});
		} catch (e) {
			statusError.value = e?.message || String(e);
			diagramLoading.value = false;
			return;
		}

		diagramLoading.value = false;
		await nextTick();

		const el = diagramRoot.value;
		if (!el) {
			return;
		}

		const diagramSource = buildDiagramSource(status);

		try {
			const { default: mermaid } = await import('mermaid');
			mermaid.initialize({
				startOnLoad: false,
				theme: 'dark',
				securityLevel: 'strict',
				fontFamily: 'ui-sans-serif, system-ui, sans-serif',
				flowchart: { htmlLabels: true, curve: 'basis' },
			});
			const id = `mermaid-arch-${Math.random().toString(36).slice(2, 10)}`;
			const { svg } = await mermaid.render(id, diagramSource);
			el.innerHTML = svg;
		} catch (e) {
			diagramError.value = e?.message || String(e);
			console.error('Mermaid render failed:', e);
		}
	});
</script>

<style scoped>
	.intro {
		max-width: 44rem;
		margin: 0 0 1rem;
		opacity: 0.9;
		line-height: 1.5;
	}

	.muted {
		opacity: 0.75;
		margin: 0.5rem 0;
	}

	.mermaid-wrap {
		overflow-x: auto;
		margin-top: 0.5rem;
		padding: 1rem 1.25rem;
		border-radius: 0.35rem;
		border: 1px solid rgba(255, 255, 255, 0.12);
		background: rgba(0, 0, 0, 0.28);
	}

	.mermaid-wrap :deep(svg) {
		display: block;
		max-width: 100%;
		height: auto;
		margin: 0 auto;
	}
</style>
