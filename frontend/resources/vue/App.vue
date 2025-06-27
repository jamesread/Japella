<template>
    <header>
		<div @click = "clickSidebarToggle" id = "sidebar-button">
			<div class = "logo-and-title">
				<img src = "../../logo.png" class = "logo" />
				<h1>Japella</h1>
			</div>
			<span class = "menu-icon">
				<Icon icon = "material-symbols:menu-rounded" width = "24" height = "24" />
			</span>
        </div>

		<div class = "fg1"></div>

        <div class = "user-info icon-and-text logo-with-title" v-if = "isLoggedIn">
            <span id = "user-name">{{ username }}</span>
            <Icon icon = "mdi:user" width = "24" height = "24" />
        </div>
    </header>

    <div id = "layout">
        <aside class = "shown stuck" v-show = "isLoggedIn" id = "section-navigation">
			<div id = "stick-icon" class = "icon-and-text vh" @click = "clickSidebarStick">
				<Icon icon = "mdi:pin" width = "24" height = "24" />
			</div>
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
            <main>
                <div v-if = "statusMessages.length > 0" class = "messages">
                    <div v-for = "message in statusMessages" :key = "message.id" :class = "message.type + ' notification'">
                        <strong>Server Message: </strong> {{ message.message }}
                    </div>
                </div>

                <div v-if = "!isLoggedIn">
                    <LoginForm @login-success = "onLogin" />
                </div>
                <div v-else>
                    <KeepAlive>
                        <component :is = "currentSection" />
                    </KeepAlive>
                </div>
            </main>
            <footer>
				<small>
                <span><a href = "https://github.com/jamesread/Japella">Japella on GitHub</a></span>
                <span><a href = "https://jamesread.github.io/Japella/">Documentation</a></span>
                <span id = "currentVersion" v-if = "isLoggedIn">{{ currentVersion }}</span>
				</small>
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
    import { ref, computed, onMounted } from 'vue';
    import { Icon } from '@iconify/vue';

    const clientReady = ref(false);
    const isLoggedIn = ref(false);
    const currentVersion = ref('');
    const username = ref('');
    const statusMessages = ref([]);

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
    import Media from './Media.vue';

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
        oauthServices: OAuthServices,
        media: Media
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
        try {
            const st = await window.client.getStatus();

            statusMessages.value = st.statusMessages || [];

            if (st.isLoggedIn) {
                onLogin(st)
            } else {
                isLoggedIn.value = false;
            }

            currentVersion.value = 'Version: ' + st.version;
        } catch (error) {
            statusMessages.value.push({
                id: Date.now(),
                type: 'error',
                message: 'Failed to fetch status from the server: ' + error.message
            });
        }
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

	function clickSidebarToggle() {
	  const sidebar = document.getElementById('section-navigation');
	  const stickIcon = document.getElementById('stick-icon');
	  const toggleButton = document.getElementById('sidebar-button');

	  if (sidebar.classList.contains('shown')) {
		sidebar.classList.remove('shown');
		stickIcon.classList.add('vh')
		toggleButton.style.position = '';
	  } else {
		sidebar.classList.add('shown');
		stickIcon.classList.remove('vh')
		toggleButton.style.position = 'fixed';
	  }
	}

	function clickSidebarStick() {
	  const sidebar = document.getElementById('sidebar');
	  const toggleButton = document.getElementById('sidebar-button');

	  if (sidebar.classList.contains('stuck')) {
		sidebar.classList.remove('stuck');
		toggleButton.style.position = 'fixed';
	  } else {
		sidebar.classList.add('stuck');
		toggleButton.style.position = '';
	  }

	  console.log('Sidebar stick toggled');
	}
</script>
