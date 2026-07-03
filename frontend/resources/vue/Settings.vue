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
		<form class="settings-form">
			<template v-for="cvar in category.cvars" :key="cvar.keyName">
				<label :for="cvar.keyName">{{ cvar.title }}:</label>
				<template v-if="cvar.type === 'text' || cvar.type === 'password'">
					<input
						:type="cvar.type"
						:id="cvar.keyName"
						:placeholder="cvar.keyName"
						:value="cvar.valueString"
						@blur="setCvar(cvar)"
					/>
				</template>
				<template v-else-if="cvar.type === 'bool'">
					<input
						type="checkbox"
						:id="cvar.keyName"
						:checked="cvar.valueInt === 1"
						@blur="setCvar(cvar)"
					/>
				</template>
				<span class="fg1"><div v-html="cvar.description"></div></span>
				<span>
					<a v-if="cvar.externalUrl" :href="cvar.externalUrl">Get</a>
				</span>
				<span>
					<a v-if="cvar.docsUrl" :href="cvar.docsUrl">Docs</a>
				</span>
			</template>
		</form>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import InlineNotification from './InlineNotification.vue';

	const route = useRoute();

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

	function setCvar(cvar) {
		const req = {
			keyName: cvar.keyName,
		};

		if (cvar.type === 'text' || cvar.type === 'password') {
			req.valueString = document.getElementById(cvar.keyName).value;
		} else if (cvar.type === 'bool') {
			req.valueInt = document.getElementById(cvar.keyName).checked ? 1 : 0;
		} else if (cvar.type === 'int') {
			req.valueInt = parseFloat(document.getElementById(cvar.keyName).value);
		} else {
			console.warn(`Unsupported cvar type: ${cvar.type}`);
			return;
		}

		window.client.setCvar(req)
			.then(() => {
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
	.settings-form {
		display: grid;
		grid-template-columns: 240px 300px 3fr min-content min-content;
		align-items: center;
	}

	.settings-form label {
		justify-self: end;
	}

	:deep(.settings-field-highlight) {
		outline: 2px solid var(--pico-primary, #1a73e8);
		outline-offset: 2px;
	}
</style>
