<template>
	<Section
		title="Conversations"
		subtitle="Message threads across all configured chat bots"
		classes="all-chat-bot-conversations-page"
	>
		<div v-if="!clientReady || loading">
			<p>Loading...</p>
		</div>
		<div v-else-if="error">
			<p class="inline-notification error">{{ error }}</p>
		</div>
		<ChatBotConversations v-else-if="bots.length > 0" :bots="bots" />
		<p v-else class="inline-notification note">No chat bots are configured. Add a bot under Settings → Chat Bots.</p>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';
	import ChatBotConversations from './ChatBotConversations.vue';

	const clientReady = ref(false);
	const loading = ref(true);
	const error = ref('');
	const bots = ref([]);

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		loading.value = true;
		error.value = '';
		try {
			const res = await window.client.getChatBots({});
			bots.value = res.bots || [];
		} catch (e) {
			error.value = `Failed to load chat bots: ${e.message || e}`;
			bots.value = [];
		} finally {
			loading.value = false;
		}
	});
</script>
