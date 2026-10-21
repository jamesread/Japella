<template>
	<Section
		title="User Preferences"
		subtitle="Personal settings for your account"
		:icon="Settings01Icon"
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

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />
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
				component-has-label
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

			<FormField
				label="Color scheme preference"
				component-has-label
				description="Choose light, dark, or match your operating system. Saved in this browser."
			>
				<RadioGroup
					v-model="theme"
					name="user-preferences-color-scheme"
					variant="list"
					:options="colorSchemeOptions"
					aria-label="Color scheme preference"
				/>
			</FormField>

			<FormField
				v-if="userThemeControl"
				label="Theme name"
				for="user-preferences-theme-name"
				description="Override the administrator default theme. Saved in this browser."
			>
				<select
					id="user-preferences-theme-name"
					:value="themePreference"
					@change="onThemeNameChange"
				>
					<option value="">Default (Femtocrank only)</option>
					<option v-for="name in themeNameOptions" :key="name" :value="name">
						{{ themeLabels[name] || name }}
					</option>
				</select>
			</FormField>

			<p v-else-if="adminThemeName" class="muted">
				Theme name is set by the administrator ({{ adminThemeName }}).
			</p>

			<NotificationBlock v-if="saveError" type="error" :message="saveError" />

			<template #actions>
				<button type="submit" class="good" :disabled="saving || !dirty">
					{{ saving ? 'Saving…' : 'Save preferences' }}
				</button>
			</template>
		</FormLayout>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, inject } from 'vue';
	import { useI18n } from 'vue-i18n';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, RefreshIcon, Settings01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import { waitForClient } from '../javascript/util';
	import { showNotification } from '../javascript/notifications.js';
	import { applyUserLanguage, applyUserSidebar } from '../javascript/userPreferences.js';
	import { themeControlFromFeatures, ThemeControlUser } from '../javascript/applyAppTheming.js';
	import { useTheme, useCustomTheme } from '../javascript/theme.js';

	const { locale, fallbackLocale } = useI18n();
	const { theme } = useTheme();
	const { availableThemes, themeLabels, themePreference, setTheme } = useCustomTheme();
	const appThemeFeatures = inject('appThemeFeatures', ref({}));
	const iconStrokeWidth = 2.5;

	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const saveError = ref('');
	const language = ref('');
	const savedLanguage = ref('');
	const sidebarEnabled = ref(true);
	const savedSidebarEnabled = ref(true);
	const availableLanguages = ref([]);

	const sidebarOptions = [
		{ label: 'Enabled', value: true },
		{ label: 'Disabled', value: false },
	];

	const themeOptions = [
		{ label: 'Auto (system)', value: 'auto' },
		{ label: 'Light', value: 'light' },
		{ label: 'Dark', value: 'dark' },
	];
	const colorSchemeOptions = themeOptions;

	const themeControl = computed(() => themeControlFromFeatures(appThemeFeatures.value));
	const userThemeControl = computed(() => themeControl.value === ThemeControlUser);
	const adminThemeName = computed(() => (appThemeFeatures.value.themeName || '').trim());
	const themeNameOptions = computed(() => {
		const fromClient = availableThemes.value || [];
		if (fromClient.length > 0) {
			return fromClient;
		}
		return appThemeFeatures.value.availableThemes || [];
	});

	function onThemeNameChange(event) {
		setTheme(event.target.value);
	}

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
		saveError.value = '';
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
		saveError.value = '';
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
			showNotification(
				'success',
				'Preferences saved',
				res.standardResponse.message || 'Preferences saved successfully.',
			);
		} catch (e) {
			console.error(e);
			saveError.value = e.message || 'Failed to save preferences.';
		} finally {
			saving.value = false;
		}
	}

	onMounted(() => {
		load();
	});
</script>
