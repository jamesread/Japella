<template>
	<Section
		title="User Preferences"
		subtitle="Personal settings for your account"
		classes="user-preferences"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userControlPanel' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to User Control Panel</span>
			</router-link>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading || saving"
				@click="load"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-else-if="loading" class="muted">Loading…</div>

		<FormLayout v-else @submit.prevent="save">
			<FormField
				label="Language"
				for="user-preferences-language"
				description="Choose the language for the Japella interface. Browser default follows your browser settings on the next full page load."
				:disabled="saving"
			>
				<select id="user-preferences-language" v-model="language" :disabled="saving">
					<option value="">Browser default</option>
					<option v-for="code in availableLanguages" :key="code" :value="code">
						{{ languageLabel(code) }}
					</option>
				</select>
			</FormField>

			<FormField
				label="Sidebar"
				fake
				description="Show the navigation sidebar and header menu on wide screens."
				:disabled="saving"
			>
				<RadioGroup
					v-model="sidebarEnabled"
					name="user-preferences-sidebar"
					variant="boolean"
					:options="sidebarOptions"
					:disabled="saving"
					aria-label="Sidebar visibility"
				/>
			</FormField>

			<p v-if="saveMessage" class="inline-notification" :class="saveMessageType">{{ saveMessage }}</p>

			<template #actions>
				<button type="submit" class="good" :disabled="saving || !dirty">
					{{ saving ? 'Saving…' : 'Save preferences' }}
				</button>
			</template>
		</FormLayout>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useI18n } from 'vue-i18n';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, RefreshIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import { waitForClient } from '../javascript/util';
	import { applyUserLanguage, applyUserSidebar } from '../javascript/userPreferences.js';

	const { locale, fallbackLocale } = useI18n();
	const iconStrokeWidth = 2.5;

	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const saveMessage = ref('');
	const saveMessageType = ref('');
	const language = ref('');
	const savedLanguage = ref('');
	const sidebarEnabled = ref(true);
	const savedSidebarEnabled = ref(true);
	const availableLanguages = ref([]);

	const sidebarOptions = [
		{ label: 'Enabled', value: true },
		{ label: 'Disabled', value: false },
	];

	const LANGUAGE_LABELS = {
		en: 'English',
		'de-DE': 'Deutsch',
		'it-IT': 'Italiano',
		'ja-JP': '日本語',
	};

	const dirty = computed(() =>
		language.value !== savedLanguage.value ||
		sidebarEnabled.value !== savedSidebarEnabled.value,
	);

	function languageLabel(code) {
		return LANGUAGE_LABELS[code] || code;
	}

	async function load() {
		loading.value = true;
		errorMessage.value = '';
		saveMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.getUserPreferences({});
			language.value = res.language || '';
			savedLanguage.value = res.language || '';
			sidebarEnabled.value = res.sidebarEnabled !== false;
			savedSidebarEnabled.value = res.sidebarEnabled !== false;
			availableLanguages.value = res.availableLanguages || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load preferences.';
		} finally {
			loading.value = false;
		}
	}

	async function save() {
		if (!dirty.value) {
			return;
		}
		saving.value = true;
		saveMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.saveUserPreferences({
				language: language.value,
				sidebarEnabled: sidebarEnabled.value,
			});
			if (!res.standardResponse?.success) {
				throw new Error(res.standardResponse?.message || 'Failed to save preferences.');
			}
			savedLanguage.value = language.value;
			savedSidebarEnabled.value = sidebarEnabled.value;
			applyUserLanguage(locale, fallbackLocale.value, language.value);
			applyUserSidebar(sidebarEnabled.value);
			saveMessageType.value = 'note';
			saveMessage.value = res.standardResponse.message || 'Preferences saved.';
		} catch (e) {
			console.error(e);
			saveMessageType.value = 'error';
			saveMessage.value = e.message || 'Failed to save preferences.';
		} finally {
			saving.value = false;
		}
	}

	onMounted(() => {
		load();
	});
</script>
