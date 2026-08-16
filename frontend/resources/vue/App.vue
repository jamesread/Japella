<template>
	<Header
		:username="isLoggedIn ? username : ''"
		@toggleSidebar="toggleSidebar"
		@logoClick="goToWelcome"
		@userClick="goToUserControlPanel"
		title="Japella"
		:logoUrl="logoUrl"
		:sidebarEnabled="isLoggedIn && sidebarPreferenceEnabled"
		:themeToggleEnabled="isLoggedIn && themeTogglePreferenceEnabled"
		:breadcrumbs="isLoggedIn"
	>
		<template #toolbar>
			<div v-if="isImpersonating" class="impersonation-banner">
				<Icon icon="mdi:account-switch" width="18" height="18" />
				<span>Impersonating <strong>{{ username }}</strong> (as {{ impersonatorUsername }})</span>
				<button type="button" class="impersonation-exit" @click="stopImpersonation" :disabled="impersonationLoading">
					<Icon icon="mdi:logout" width="16" height="16" />
					Exit
				</button>
			</div>
			<QuickSearch
				v-if="isLoggedIn"
				placeholder="Search pages..."
				:search-fields="['title', 'description', 'category']"
			/>
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

	<NotificationPopupsHost />
</template>

<style scoped>
    footer span {
        display: inline-block;
        margin-right: 10px;
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
        align-self: center;
        margin-right: 0.25rem;
    }

    /* Beat Header header-actions :deep(button) chrome styles */
    .impersonation-exit {
        display: inline-flex;
        align-items: center;
        gap: 0.25rem;
        background: rgba(255,255,255,0.2) !important;
        color: white !important;
        border: 1px solid rgba(255,255,255,0.4) !important;
        border-radius: 3px !important;
        padding: 0.15rem 0.5rem !important;
        cursor: pointer;
        font-size: 0.85em;
        margin-left: 0.5rem;
        height: auto !important;
        align-self: center !important;
    }

    .impersonation-exit:hover {
        background: rgba(255,255,255,0.35) !important;
        color: white !important;
    }
</style>

<script setup>
    import { waitForClient } from '../javascript/util.js'
    import { fetchAppStatus } from '../javascript/status.js'
    import { loadAndApplyUserPreferences, registerSidebarApplier, registerThemeToggleApplier } from '../javascript/userPreferences.js'
    import { ref, onMounted, provide, watch } from 'vue';
    import { useRoute, useRouter } from 'vue-router';
    import { Icon } from '@iconify/vue';
    import { useI18n } from 'vue-i18n';
	import LoginForm from './LoginForm.vue';
	import ErrorDialog from './ErrorDialog.vue';
	import PWAInstallPrompt from './PWAInstallPrompt.vue';
    import Header from 'picocrank/vue/components/Header.vue';
    import Navigation from 'picocrank/vue/components/Navigation.vue';
    import Sidebar from 'picocrank/vue/components/Sidebar.vue';
    import QuickSearch from 'picocrank/vue/components/QuickSearch.vue';
    import NotificationPopupsHost from './NotificationPopupsHost.vue';
	import logoUrl from '../../logo.png';
	import {
		fetchApprovalsCount,
		setupSidebarNavigation,
		showControlPanelInSidebar,
	} from '../javascript/sidebarNavigation.js';

    const { t, locale, fallbackLocale } = useI18n();
    const route = useRoute();
    const router = useRouter();

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
    const sidebarPreferenceEnabled = ref(true);
    const themeTogglePreferenceEnabled = ref(false);
    const errorDialogRef = ref();

	provide('showSectionError', (msg) => {
		errorDialogRef.value?.showError(msg)
	});

	function applySidebarPreference(enabled) {
		sidebarPreferenceEnabled.value = enabled;
		if (!sidebar.value) {
			return;
		}
		if (enabled) {
			sidebar.value.open();
			sidebar.value.stick();
		} else {
			sidebar.value.close();
		}
	}

	function applyThemeTogglePreference(enabled) {
		themeTogglePreferenceEnabled.value = enabled;
	}

	function toggleSidebar() {
		if (sidebar.value) {
			sidebar.value.toggle();
		}
	}

	function goToUserControlPanel() {
		router.push({ name: 'userControlPanel' });
	}

	function goToWelcome() {
		router.push({ name: 'welcome' });
	}

	async function setupNavigation() {
		if (!navigation.value) return;

		const approvalsCount = await fetchApprovalsCount();

		setupSidebarNavigation(navigation.value, {
			approvalsCount,
			showControlPanel: showControlPanelInSidebar({
				rbacIsSuperuser: window.userRbacIsSuperuser,
				rbacPermissions: window.userRbacPermissions,
			}),
		});

		// Apply sidebar visibility preference for logged-in users
		if (isLoggedIn.value && sidebar.value) {
			applySidebarPreference(sidebarPreferenceEnabled.value);
		}
	}

	async function refreshApprovalsNavCount(knownCount) {
		if (!navigation.value || !isLoggedIn.value) return;
		const approvalsCount = knownCount != null ? knownCount : await fetchApprovalsCount();
		navigation.value.addRouterLink('approvals', null, { count: approvalsCount });
	}

	provide('refreshApprovalsNavCount', refreshApprovalsNavCount);

	function checkSecureContext(st) {
		if (st.usesSecureCookies && !window.isSecureContext) {
			statusMessages.value.push({
				type: 'error',
				message: 'Your browser is not running in a secure context, and the server is set to only send secure cookies. You will not be able to stay logged in.',
				url: 'https://jamesread.github.io/Japella/troubleshooting/secure-context-cookies.html'
			});
		}
	}

    function applyStatusToUi(st) {
        statusMessages.value = st.statusMessages || [];

        checkSecureContext(st)

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
            window.isLoggedIn = false;
            window.userRbacPermissions = [];
            window.userRbacIsSuperuser = false;
        }

        currentVersion.value = 'Version: ' + st.version;
    }

    async function loadStatus({ force = false } = {}) {
        try {
            const st = await fetchAppStatus({ force });
            applyStatusToUi(st);
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
        loadAndApplyUserPreferences({ localeRef: locale, fallbackLocale: fallbackLocale.value });
    }

    /**
     * After password login the client only has username until GetStatus returns RBAC.
     */
    async function onLogin() {
        await loadStatus({ force: true });
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
        registerSidebarApplier(applySidebarPreference);
        registerThemeToggleApplier(applyThemeTogglePreference);

        setTimeout(() => {
            loadingWarning.value = 'If you are reading this text after waiting more than a few seconds, something has gone wrong. Please check your browser console for errors.';
        }, 2000);

        await waitForClient();
        clientReady.value = true;
        await loadStatus();
    });

    watch(clientReady, (ready) => {
        if (ready) {
            document.body.setAttribute('loaded-app', 'true');
        } else {
            document.body.removeAttribute('loaded-app');
        }
    });

    watch(isLoggedIn, (loggedIn) => {
        if (loggedIn) {
            document.body.setAttribute('logged-in', 'true');
        } else {
            document.body.removeAttribute('logged-in');
        }
    });

    // Refresh the Approvals badge when leaving the approvals page (e.g. after acting).
    watch(() => route.name, (name, prev) => {
        if (prev === 'approvals' || name === 'approvals') {
            refreshApprovalsNavCount();
        }
    });
</script>
