<template>
	<Section
		title="Campaign Details"
		subtitle="View and manage a single campaign"
		classes="campaign-details"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refresh" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
			<router-link class="good" :to="{ name: 'postBox', query: { campaignId: campaignId } }">
				<Icon icon="jam:write-f" />
			</router-link>
		</template>

		<div v-if="!clientReady">
			<p>Loading...</p>
		</div>
		<div v-else>
			<h3>{{ campaign?.name || 'Unknown Campaign' }}</h3>
			<p>ID: {{ campaignId }}</p>
			<p>Posts: {{ campaign?.postCount ?? 0 }}</p>
			<p>Last Post: {{ campaign?.lastPostDate || 'Never' }}</p>
		</div>
	</Section>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import Section from 'picocrank/vue/components/Section.vue';

	const route = useRoute();
	const clientReady = ref(false)
	const campaignId = ref(0)
	const campaign = ref(null)

	async function refresh() {
		if (!window.client || !campaignId.value) return
		const res = await window.client.getCampaigns()
		const found = (res.campaigns || []).find(c => c.id === campaignId.value)
		campaign.value = found || null
	}

	onMounted(async () => {
		campaignId.value = parseInt(route.params.id, 10) || 0
		clientReady.value = true
		refresh()
	})
</script>

<style scoped>
</style>
