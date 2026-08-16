<template>
	<Section
		v-if="!loaded"
		title="IAM"
		subtitle="Identity and Access Management — users, groups, account policies, and roles."
		classes="iam"
	>
		<p class="muted">Loading…</p>
	</Section>

	<Section
		v-else-if="canAccessIam"
		title="IAM"
		subtitle="Identity and Access Management — users, groups, account policies, and roles."
		classes="iam"
	>
		<template #toolbar>
			<router-link :to="{ name: 'controlPanel' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Control Panel</span>
			</router-link>
		</template>

		<Navigation ref="localNavigation">
			<NavigationGrid />
		</Navigation>
	</Section>

	<Section v-else title="IAM" subtitle="Identity and Access Management">
		<p class="inline-notification error">You do not have permission to view IAM settings.</p>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { canAccessIamFromStatus } from '../javascript/rbacAccess.js';
	import { setupIamNavigationGrid } from '../javascript/iamNavigation.js';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';

	const iconStrokeWidth = 2.5;
	const loaded = ref(false);
	const systemStatus = ref({});
	const localNavigation = ref(null);

	const canAccessIam = computed(() => canAccessIamFromStatus(systemStatus.value));

	function goToRoute(route) {
		window.router.push(route);
	}

	onMounted(async () => {
		await waitForClient();
		try {
			systemStatus.value = await window.client.getStatus();
		} catch (error) {
			console.error('Error fetching status for IAM:', error);
			loaded.value = true;
			return;
		}

		if (!canAccessIamFromStatus(systemStatus.value)) {
			window.router.push('/');
			return;
		}

		loaded.value = true;
		await nextTick();
		setupIamNavigationGrid(localNavigation, goToRoute, systemStatus.value);
	});
</script>

<style scoped>
	.muted {
		opacity: 0.8;
	}
</style>
