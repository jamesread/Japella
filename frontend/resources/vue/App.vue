<template>
    <header>
		<div @click = "clickSidebarToggle" id = "sidebar-button" ref = "toggleButton">
			<div class = "logo-and-title">
				<img src = "../../logo.png" class = "logo" />
				<h1>Japella</h1>
			</div>
			<span class = "menu-icon" ref = "menuIcon" hidden>
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
        <aside class = "" v-show = "isLoggedIn" ref = "sectionNavigation">
			<div ref = "stickIcon" class = "stick-icon icon-and-text" @click = "clickSidebarStick">
				<Icon :icon = "pinIcon" width = "24" height = "24" />
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

						<a v-if = "message.url" :href = "message.url" target = "_blank">More info</a>
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
				<ErrorDialog ref = "errorDialogRef" />
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


	#sidebar-button {
		border-right: 0;
	}

	#sidebar-button:hover {
		background-color: #1d345c;
	}

	.stick-icon {
		padding: 0.1em;
		padding-top: 0.5em;
		display: flex;
		justify-content: end;
	}
</style>

<script setup>
    import { waitForClient } from '../javascript/util.js'
    import { ref, computed, onMounted, provide } from 'vue';
    import { Icon } from '@iconify/vue';

    const clientReady = ref(false);
    const isLoggedIn = ref(false);
    const currentVersion = ref('');
    const username = ref('');
    const statusMessages = ref([]);
	const menuIcon = ref(null);

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
	const stickIcon = ref(null);
	const sectionNavigation = ref(null);
	const toggleButton = ref(null);
	const pinIcon = ref('mdi:pin-outline');
	const errorDialogRef = ref();

	provide('showSectionError', (msg) => {
		errorDialogRef.value?.showError(msg)
	});

    function changeSection(sectionName) {
        if (sections[sectionName]) {
            currentSectionName.value = sectionName;

			if (!sectionNavigation.value.classList.contains('stuck')) {
				hideSidebar();
			}
        } else {
            console.warn(`Section "${sectionName}" does not exist.`);
        }
    }

	function checkSecureContext(st) {
		if (st.usesSecureCookies && !window.isSecureContext) {
			statusMessages.value.push({
				type: 'error',
				message: 'Your browser is not running in a secure context, and the server is set to only send secure cookies. You will not be able to stay logged in.',
				url: 'https://jamesread.github.io/Japella/troubleshooting/secure-context-cookies.html'
			});
		}
	}

    async function getStatus() {
        try {
            const st = await window.client.getStatus();

            statusMessages.value = st.statusMessages || [];

            checkSecureContext(st)

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

		menuIcon.value.hidden = false;
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
	  if (sectionNavigation.value.classList.contains('shown')) {
	    hideSidebar();
	  } else {
		sectionNavigation.value.classList.add('shown');
		stickIcon.value.classList.remove('vh')
	  }
	}

	function hideSidebar() {
	  sectionNavigation.value.classList.remove('shown');
	  stickIcon.value.classList.add('vh');
	}

	function clickSidebarStick() {
	  if (sectionNavigation.value.classList.contains('stuck')) {
	    pinIcon.value = 'mdi:pin-outline';
		sectionNavigation.value.classList.remove('stuck');
	  } else {
	    pinIcon.value = 'mdi:pin';
		sectionNavigation.value.classList.add('stuck');
	  }

	  console.log('Sidebar stick toggled');
	}
</script>
