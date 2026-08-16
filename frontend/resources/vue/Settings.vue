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
			<p class="inline-notification error">You do not have permission to view system settings.</p>
		</div>
		<template v-else>
			<p>This page allows you to configure all of the cvars available in the app.</p>
			<InlineNotification
				message="Settings are saved immediately when you change them on this page."
				type="warning"
			/>
		</template>
	</Section>

	<Section
		v-for="category in categoryList"
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
				:fake="cvar.type === 'bool'"
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
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
	import { waitForClient, decodeHtmlEntities } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import RadioGroup from 'picocrank/vue/components/RadioGroup.vue';
	import InlineNotification from './InlineNotification.vue';

	const route = useRoute();

	const boolOptions = [
		{ label: 'On', value: true },
		{ label: 'Off', value: false },
	];

	const categories = ref({});
	const loaded = ref(false);
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
			await refreshCvars();
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

	:deep(.settings-field-highlight) {
		outline: 2px solid var(--pico-primary, #1a73e8);
		outline-offset: 2px;
	}
</style>
