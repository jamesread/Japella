<template>
	<Section
		title="Settings"
		subtitle="System configuration variables (cvars)."
		classes="settings"
	>
		<div v-if="!loaded">
			<p>Loading...</p>
		</div>
		<div v-else-if="!canAccess">
			<NotificationBlock type="error" message="You do not have permission to view system settings." />
		</div>
		<template v-else>
			<p>This page allows you to configure all of the cvars available in the app.</p>
			<NotificationBlock
				type="warning"
				message="Settings are saved immediately when you change them on this page."
			/>
		</template>
	</Section>

	<Section
		v-for="category in nonThemeCategoryList"
		:key="category.name"
		:title="category.name"
		classes="settings-category"
	>
		<FormLayout @submit.prevent>
			<FormField
				v-for="cvar in category.cvars"
				:key="cvar.keyName"
				:label="cvar.title"
				:for="cvar.type === 'bool' ? '' : cvar.keyName"
				:component-has-label="cvar.type === 'bool'"
				:description="decodeHtmlEntities(cvar.description)"
				:docs-url="cvar.docsUrl || ''"
				docs-url-title="Docs"
			>
				<div
					class="settings-field"
					:id="cvar.type === 'bool' ? cvar.keyName : undefined"
					:tabindex="cvar.type === 'bool' ? -1 : undefined"
				>
					<input
						v-if="cvar.type === 'text' || cvar.type === 'password'"
						:type="cvar.type"
						:id="cvar.keyName"
						:placeholder="cvar.keyName"
						:value="cvar.valueString"
						@blur="setCvar(cvar)"
					/>
					<RadioGroup
						v-else-if="cvar.type === 'bool'"
						:name="cvar.keyName"
						variant="boolean"
						:aria-label="cvar.title"
						:model-value="cvar.valueInt === 1"
						:options="boolOptions"
						@update:model-value="setBoolCvar(cvar, $event)"
					/>
					<input
						v-else-if="cvar.type === 'int'"
						type="number"
						:id="cvar.keyName"
						:placeholder="cvar.keyName"
						:value="cvar.valueInt"
						@blur="setCvar(cvar)"
					/>
					<p v-if="cvar.externalUrl" class="settings-links">
						<a :href="cvar.externalUrl">Get</a>
					</p>
				</div>
			</FormField>
		</FormLayout>
	</Section>

	<Section
		v-if="canAccess && themeSectionReady"
		title="Theme"
		subtitle="Appearance and theme policy"
		classes="settings-category settings-theme"
	>
		<FormLayout @submit.prevent="saveThemeSection">
			<FormField label="Color scheme switcher" component-has-label>
				<RadioGroup
					v-model="themeEdits.colorSchemeSwitcher"
					name="settings-theme-color-scheme-switcher"
					variant="boolean"
					:options="boolOptions"
					aria-label="Color scheme switcher"
					@update:model-value="themeSectionDirty = true"
				/>
				<p class="subtle">Show the auto/light/dark control in the header.</p>
			</FormField>

			<FormField label="Theme name" for="settings-theme-name">
				<select
					id="settings-theme-name"
					v-model="themeEdits.themeName"
					@change="themeSectionDirty = true"
				>
					<option value="">Default (Femtocrank only)</option>
					<option v-for="name in themeNameOptions" :key="name" :value="name">
						{{ themeLabels[name] || name }}
					</option>
				</select>
				<p class="subtle">Drop-in CSS theme for the app.</p>
			</FormField>

			<FormField label="Theme control" component-has-label>
				<RadioGroup
					v-model="themeEdits.themeControl"
					name="settings-theme-control"
					variant="list"
					:options="themeControlOptions"
					aria-label="Theme control"
					@update:model-value="themeSectionDirty = true"
				/>
				<p class="subtle">
					System preference forces the theme name for all users.
					User preference uses this theme as the default; users may override on User Preferences.
				</p>
			</FormField>

			<template #actions>
				<button type="submit" class="good" :disabled="!themeSectionDirty || themeSaving">
					{{ themeSaving ? 'Saving…' : 'Save theme settings' }}
				</button>
			</template>
		</FormLayout>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
	import { waitForClient, decodeHtmlEntities } from '../javascript/util';
	import { fetchAppStatus, invalidateAppStatus } from '../javascript/status.js';
	import {
		applyAppTheming,
		themeFeaturesFromStatus,
		ThemeControlSystem,
		ThemeControlUser,
	} from '../javascript/applyAppTheming.js';
	import { useCustomTheme } from '../javascript/theme.js';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';

	const route = useRoute();
	const { themeLabels, availableThemes, discoverThemes } = useCustomTheme();

	const boolOptions = [
		{ label: 'On', value: true },
		{ label: 'Off', value: false },
	];

	const themeControlOptions = [
		{ label: 'System preference', value: ThemeControlSystem },
		{ label: 'User preference', value: ThemeControlUser },
	];

	const categories = ref({});
	const loaded = ref(false);
	const themeSectionReady = ref(false);
	const themeSectionDirty = ref(false);
	const themeSaving = ref(false);
	const themeEdits = ref({
		colorSchemeSwitcher: false,
		themeName: '',
		themeControl: ThemeControlUser,
	});
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const canAccess = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.settings'))
	);

	const categoryList = computed(() =>
		Object.values(categories.value).sort((a, b) =>
			String(a.name || '').localeCompare(String(b.name || ''))
		)
	);

	const nonThemeCategoryList = computed(() =>
		categoryList.value.filter((category) => category.name !== 'Theme')
	);

	const statusThemeNames = ref([]);

	const themeNameOptions = computed(() => {
		const fromClient = availableThemes.value || [];
		if (fromClient.length > 0) {
			return fromClient;
		}
		return statusThemeNames.value;
	});

	function findThemeCvar(keyName) {
		const themeCategory = categories.value.Theme;
		return themeCategory?.cvars?.find((c) => c.keyName === keyName) || null;
	}

	function syncThemeEditsFromCvars() {
		const switcher = findThemeCvar('theme_color_scheme_switcher_enabled');
		const themeName = findThemeCvar('theme_name');
		const themeControl = findThemeCvar('theme_control');
		themeEdits.value = {
			colorSchemeSwitcher: switcher?.valueInt === 1,
			themeName: themeName?.valueString || '',
			themeControl: themeControl?.valueString === ThemeControlSystem ? ThemeControlSystem : ThemeControlUser,
		};
		themeSectionDirty.value = false;
		themeSectionReady.value = Boolean(switcher && themeName && themeControl);
	}

	async function reloadAppTheming() {
		invalidateAppStatus();
		const st = await fetchAppStatus({ force: true });
		statusThemeNames.value = st.availableThemes || [];
		applyAppTheming(themeFeaturesFromStatus(st));
	}

	async function saveThemeSection() {
		if (!themeSectionDirty.value || themeSaving.value) {
			return;
		}
		themeSaving.value = true;
		try {
			await window.client.setCvar({
				keyName: 'theme_color_scheme_switcher_enabled',
				valueInt: themeEdits.value.colorSchemeSwitcher ? 1 : 0,
			});
			await window.client.setCvar({
				keyName: 'theme_name',
				valueString: themeEdits.value.themeName || '',
			});
			await window.client.setCvar({
				keyName: 'theme_control',
				valueString: themeEdits.value.themeControl,
			});
			await refreshCvars();
			syncThemeEditsFromCvars();
			await reloadAppTheming();
		} catch (error) {
			console.error('Error saving theme settings:', error);
		} finally {
			themeSaving.value = false;
		}
	}

	function refreshCvars() {
		categories.value = {};
		return window.client.getCvars().then((ret) => {
			const filtered = {};
			for (const [name, category] of Object.entries(ret.cvarCategories || {})) {
				if (name === 'Chat Bots') {
					continue;
				}
				const cvars = (category.cvars || []).filter((c) => !String(c.keyName || '').startsWith('bot.'));
				if (cvars.length > 0) {
					filtered[name] = { ...category, cvars };
				}
			}
			categories.value = filtered;
			syncThemeEditsFromCvars();
		}).catch((error) => {
			console.error('Error fetching cvars:', error);
		});
	}

	async function focusCvarFromHash() {
		const hash = route.hash?.replace(/^#/, '');
		if (!hash) {
			return;
		}

		await nextTick();
		const field = document.getElementById(hash);
		if (!field) {
			return;
		}

		field.scrollIntoView({ behavior: 'smooth', block: 'center' });
		field.focus({ preventScroll: true });
		field.classList.add('settings-field-highlight');
		setTimeout(() => field.classList.remove('settings-field-highlight'), 2500);
	}

	function setBoolCvar(cvar, enabled) {
		const req = {
			keyName: cvar.keyName,
			valueInt: enabled ? 1 : 0,
		};

		window.client.setCvar(req)
			.then(() => {
				cvar.valueInt = req.valueInt;
				console.log(`Cvar ${cvar.keyName} set.`);
			})
			.catch((error) => {
				console.error(`Error setting cvar ${cvar.keyName}:`, error);
			});
	}

	function setCvar(cvar) {
		const req = {
			keyName: cvar.keyName,
		};

		if (cvar.type === 'text' || cvar.type === 'password') {
			req.valueString = document.getElementById(cvar.keyName).value;
		} else if (cvar.type === 'int') {
			req.valueInt = parseFloat(document.getElementById(cvar.keyName).value);
		} else {
			console.warn(`Unsupported cvar type: ${cvar.type}`);
			return;
		}

		window.client.setCvar(req)
			.then(() => {
				if (cvar.type === 'int') {
					cvar.valueInt = req.valueInt;
				} else if (cvar.type === 'text' || cvar.type === 'password') {
					cvar.valueString = req.valueString;
				}
				console.log(`Cvar ${cvar.keyName} set.`);
			})
			.catch((error) => {
				console.error(`Error setting cvar ${cvar.keyName}:`, error);
			});
	}

	onMounted(async () => {
		await waitForClient();

		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
		loaded.value = true;

		if (canAccess.value) {
			await discoverThemes();
			await refreshCvars();
			const st = await window.client.getStatus({});
			statusThemeNames.value = st.availableThemes || [];
			await focusCvarFromHash();
		}
	});
</script>

<style scoped>
	.settings-field {
		display: flex;
		flex-direction: column;
		gap: 0.4em;
		min-width: 0;
	}

	.settings-links {
		margin: 0;
		display: flex;
		gap: 0.75em;
	}

	.subtle {
		margin: 0.35em 0 0;
		font-size: 0.9em;
		color: var(--muted-text-color, #666);
	}

	:deep(.settings-field-highlight) {
		outline: 2px solid var(--pico-primary, #1a73e8);
		outline-offset: 2px;
	}
</style>
