<template>
	<Header
		:username="isLoggedIn ? username : ''"
		@toggleSidebar="toggleSidebar"
		title="Japella"
		:logoUrl="logoUrl"
		:sidebarEnabled="isLoggedIn"
	>
		<template #toolbar>
			<div v-if="isImpersonating" class="impersonation-banner">
				<Icon icon="mdi:account-switch" width="18" height="18" />
				<span>Impersonating <strong>{{ username }}</strong> (as {{ impersonatorUsername }})</span>
				<button class="impersonation-exit" @click="stopImpersonation" :disabled="impersonationLoading">
					<Icon icon="mdi:logout" width="16" height="16" />
					Exit
				</button>
			</div>
			<QuickSearch v-if="isLoggedIn" />
		</template>
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

    .impersonation-banner {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: #e65100;
        color: white;
        padding: 0.3rem 0.75rem;
        border-radius: 4px;
        font-size: 0.85em;
        white-space: nowrap;
    }

    .impersonation-exit {
        display: inline-flex;
        align-items: center;
        gap: 0.25rem;
        background: rgba(255,255,255,0.2);
        color: white;
        border: 1px solid rgba(255,255,255,0.4);
        border-radius: 3px;
        padding: 0.15rem 0.5rem;
        cursor: pointer;
        font-size: 0.85em;
        margin-left: 0.5rem;
    }

    .impersonation-exit:hover {
        background: rgba(255,255,255,0.35);
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
    import QuickSearch from 'picocrank/vue/components/QuickSearch.vue';
	import logoUrl from '../../logo.png';
	import { canAccessControlPanelFromStatus } from '../javascript/rbacAccess.js';

    const { t } = useI18n();

    const clientReady = ref(false);
    const isLoggedIn = ref(false);
    const currentVersion = ref('');
    const username = ref('');
    const statusMessages = ref([]);
    const isImpersonating = ref(false);
    const impersonatorUsername = ref('');
    const impersonationLoading = ref(false);

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

		if (canAccessControlPanelFromStatus({
			isLoggedIn: isLoggedIn.value,
			rbacIsSuperuser: window.userRbacIsSuperuser,
			rbacPermissions: window.userRbacPermissions,
		})) {
			navigation.value.addRouterLink('controlPanel');
		}

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

            isImpersonating.value = Boolean(st.isImpersonating);
            impersonatorUsername.value = st.impersonatorUsername || '';

            if (st.isLoggedIn) {
                applyAuthFromStatus(st)
            } else {
                isLoggedIn.value = false;
                window.isLoggedIn = false; // Set global auth state for router
                window.userRbacPermissions = [];
                window.userRbacIsSuperuser = false;
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

    function applyAuthFromStatus(st) {
        isLoggedIn.value = true;
        username.value = st.username;
        window.isLoggedIn = true;
        window.userRbacPermissions = st.rbacPermissions || [];
        window.userRbacIsSuperuser = Boolean(st.rbacIsSuperuser);
        setupNavigation();
    }

    /**
     * After password login the client only has username until GetStatus returns RBAC.
     */
    async function onLogin() {
        await getStatus();
    }

    async function stopImpersonation() {
        impersonationLoading.value = true;
        try {
            await window.client.stopImpersonation({});
            window.location.href = '/';
        } catch (e) {
            statusMessages.value.push({ id: Date.now(), type: 'error', message: 'Failed to stop impersonation: ' + e.message });
            impersonationLoading.value = false;
        }
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
