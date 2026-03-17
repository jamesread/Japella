<template>
	<Header
		:username="isLoggedIn ? username : ''"
		@toggleSidebar="toggleSidebar"
		title="Japella"
		:logoUrl="logoUrl"
	>
		<template #user-info>
			<div class="user-info icon-and-text logo-with-title" v-if="isLoggedIn">
				<router-link to="/user-control-panel" class="username-link">
					<span id="user-name">{{ username }}</span>
				</router-link>
				<Icon icon="mdi:user" width="24" height="24" />
			</div>
		</template>
	</Header>

	<Navigation ref="navigation">
		<div id="layout">
			<Sidebar ref="sidebar" />

			<div id="loading" v-if="!clientReady" class="icon-and-text" style="margin: auto; margin-top: 5em;">
				<Icon icon="eos-icons:loading" width="48" height="48" />
				<div>
					Loading...
					<div v-if="loadingWarning"><br />
						<div class="error">{{ loadingWarning }}</div>
					</div>
				</div>
			</div>
			<div id="content" v-else>
				<main>
					<div v-if="statusMessages.length > 0" class="messages">
						<div v-for="message in statusMessages" :key="message.id" :class="message.type + ' notification'">
							<strong>Server Message: </strong> {{ message.message }}
							<a v-if="message.url" :href="message.url" target="_blank">More info</a>
						</div>
					</div>

					<div v-if="!isLoggedIn">
						<LoginForm @login-success="onLogin" />
					</div>
					<div v-else>
						<router-view />
					</div>
					<ErrorDialog ref="errorDialogRef" />
					<PWAInstallPrompt />
				</main>
				<footer>
					<small>
						<span><a href="https://github.com/jamesread/Japella">Japella on GitHub</a></span>
						<span><a href="https://jamesread.github.io/Japella/">Documentation</a></span>
						<span id="currentVersion" v-if="isLoggedIn">{{ currentVersion }}</span>
					</small>
				</footer>
			</div>
		</div>
	</Navigation>
</template>

<style scoped>
    footer span {
        display: inline-block;
        margin-right: 10px;
    }

    .username-link {
        color: white;
        text-decoration: none;
        cursor: pointer;
        transition: color 0.2s ease;
    }

    .username-link:hover {
        color: #4CAF50;
        text-decoration: underline;
    }

    .username-link:visited {
        color: white;
    }
</style>

<script setup>
    import { waitForClient } from '../javascript/util.js'
    import { ref, computed, onMounted, provide } from 'vue';
    import { Icon } from '@iconify/vue';
    import { useI18n } from 'vue-i18n';
	import LoginForm from './LoginForm.vue';
	import ErrorDialog from './ErrorDialog.vue';
	import PWAInstallPrompt from './PWAInstallPrompt.vue';
    import Header from 'picocrank/vue/components/Header.vue';
    import Navigation from 'picocrank/vue/components/Navigation.vue';
    import Sidebar from 'picocrank/vue/components/Sidebar.vue';
	import logoUrl from '../../logo.png';

    const { t } = useI18n();

    const clientReady = ref(false);
    const isLoggedIn = ref(false);
    const currentVersion = ref('');
    const username = ref('');
    const statusMessages = ref([]);

    // Router will handle component loading
    const loadingWarning = ref('');
    const navigation = ref(null);
    const sidebar = ref(null);
    const errorDialogRef = ref();

	provide('showSectionError', (msg) => {
		errorDialogRef.value?.showError(msg)
	});

	function toggleSidebar() {
		if (sidebar.value) {
			sidebar.value.toggle();
		}
	}

	function setupNavigation() {
		if (!navigation.value) return;

		// Clear existing navigation
		navigation.value.clearNavigationLinks();

		navigation.value.addHtml('<h2 style = "padding-left: .5em; margin-top: 1em; margin-bottom: .5em;">Write</h2>', {
			name: 'title-post',
		});

		// Add router links
		navigation.value.addRouterLink('postBox');

		navigation.value.addRouterLink('media');

		navigation.value.addRouterLink('campaigns');

		navigation.value.addRouterLink('cannedPosts');

		navigation.value.addRouterLink('calendar');

		navigation.value.addRouterLink('timeline');

		navigation.value.addHtml('<h2 style = "padding-left: .5em; margin-top: 1em; margin-bottom: .5em;">Read</h2>', {
			name: 'title-read',
		})

		navigation.value.addRouterLink('feed');

		navigation.value.addSeparator('connections-separator');

		navigation.value.addHtml('<h2 style = "padding-left: .5em; margin-top: 1em; margin-bottom: .5em;">Settings</h2>', {
			name: 'title-shared',
		});

		navigation.value.addRouterLink('socialAccounts');

		navigation.value.addRouterLink('chatBots');

		navigation.value.addRouterLink('controlPanel');

		navigation.value.addRouterLink('appStatus');

		// Open and stick the sidebar for logged-in users
		if (isLoggedIn.value && sidebar.value) {
			sidebar.value.open();
			sidebar.value.stick();
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

            // Check if database schema is dirty and show error
            if (st.databaseSchemaDirty) {
                statusMessages.value.push({
                    id: Date.now() + '_dirty_db',
                    type: 'error',
                    message: 'Database schema is in a dirty state. Please run database migrations to fix this issue.',
                    url: 'https://jamesread.github.io/Japella/troubleshooting/database-migrations.html'
                });
            }

            if (st.isLoggedIn) {
                onLogin(st)
            } else {
                isLoggedIn.value = false;
                window.isLoggedIn = false; // Set global auth state for router
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
        window.isLoggedIn = true; // Set global auth state for router

        // Setup navigation for logged-in user
        setupNavigation();
    }

    onMounted(async () => {
        setTimeout(() => {
            loadingWarning.value = 'If you are reading this text after waiting more than a few seconds, something has gone wrong. Please check your browser console for errors.';
        }, 2000);

        await waitForClient();
        clientReady.value = true;
        getStatus();
    });
</script>
