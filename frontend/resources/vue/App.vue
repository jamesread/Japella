<template>
	<header>
		<div class = "logo-and-title fg1">
			<img src = "../../logo.png" class = "logo" />
			<h1>Japella</h1>
		</div>

		<div class = "user-info icon-and-text" v-if = "isLoggedIn">
			<span id = "user-name">{{ username }}</span>
			<Icon icon = "mdi:user" width = "24" height = "24" />
		</div>
	</header>

	<div id = "layout">
		<aside class = "shown stuck" v-show = "isLoggedIn">
			<SectionNavigation @section-change-request = "changeSection"/>
		</aside>

		<div id = "loading" v-if = "!clientReady" class = "icon-and-text" style = "margin: auto; margin-top: 5em;">
			<Icon icon = "eos-icons:loading" width = "48" height = "48" />
			<div>
				Loading...
				<div v-if = "loadingWarning"><br />
					<div class = "error">{{ loadingWarning }}</div>
				</div>
			</div>
		</div>
		<div id = "content" v-else>
			<main v-if = "!isLoggedIn">
				<LoginForm @login-success = "onLogin" />
			</main>
			<main v-else>
				<KeepAlive>
					<component :is = "currentSection" />
				</KeepAlive>
			</main>
			<footer>
				<span><a href = "https://github.com/jamesread/Japella">Japella on GitHub</a></span>
				<span><a href = "https://jamesread.github.io/Japella/">Documentation</a></span>
				<span id = "currentVersion" v-if = "isLoggedIn">{{ currentVersion }}</span>
			</footer>
		</div>
	</div>
</template>

<style scoped>
	footer span {
		display: inline-block;
		margin-right: 10px;
	}
</style>

<script setup>
	import { waitForClient } from '../javascript/util.js'
	import { ref, computed, defineAsyncComponent, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';

	const clientReady = ref(false);
	const isLoggedIn = ref(false);
	const currentVersion = ref('');
	const username = ref('');

	import Welcome from './Welcome.vue';
	import Timeline from './Timeline.vue';
	import AppStatus from './AppStatus.vue';
	import PostBox from './PostBox.vue';
	import Calendar from './Calendar.vue';
	import Settings from './Settings.vue';
	import CannedPosts from './CannedPosts.vue';
	import UserList from './UserList.vue';
	import SocialAccounts from './SocialAccounts.vue';
	import OAuthServices from './OAuthServices.vue';
	import ApiKeys from './ApiKeys.vue';

	const sections = {
		welcome: Welcome,
		timeline: Timeline,
		appStatus: AppStatus,
		postBox: PostBox,
		calendar: Calendar,
		settings: Settings,
		cannedPosts: CannedPosts,
		settingsUsers: UserList,
		settingsApiKeys: ApiKeys,
		socialAccounts: SocialAccounts,
		oauthServices: OAuthServices
	}

	const currentSectionName = ref('welcome');
	const currentSection = computed(() => sections[currentSectionName.value]);
	const loadingWarning = ref('');

	function changeSection(sectionName) {
		if (sections[sectionName]) {
			currentSectionName.value = sectionName;
		} else {
			console.warn(`Section "${sectionName}" does not exist.`);
		}
	}

	async function getStatus() {
		const st = await window.client.getStatus();

		if (st.isLoggedIn) {
			onLogin(st)
		} else {
			isLoggedIn.value = false;
		}

		currentVersion.value = 'Version: ' + st.version;
	}

	/**
	 * ret could be from getStatus, or from loginWith... - bot hhave the ".username" property.
	 */
	function onLogin(ret) {
		isLoggedIn.value = true;
		username.value = ret.username
	}

	onMounted(async () => {
		setTimeout(() => {
		    loadingWarning.value = 'If you are reading this text after waiting more than a few seconds, something has gone wrong. Please check your browser console for errors.';
		}, 2000);

		await waitForClient();
		clientReady.value = true;
		getStatus()
	});
</script>
