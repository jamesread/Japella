<template>
	<Section
		title="Conversations"
		subtitle="Message threads across all configured chat bots"
		:icon="MessageMultiple01Icon"
		classes="all-chat-bot-conversations-page"
	>
		<div v-if="!clientReady || loading">
			<p>Loading...</p>
		</div>
		<div v-else-if="error">
			<NotificationBlock type="error" :message="error" />
		</div>
		<ChatBotConversations v-else-if="bots.length > 0" :bots="bots" />
		<NotificationBlock v-else type="note">
			No chat bots are configured. <router-link :to="{ name: 'createChatBot' }">Add a chat bot</router-link> to get started.
		</NotificationBlock>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { MessageMultiple01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
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
