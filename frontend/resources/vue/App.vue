<template>
	<Navigation ref="navigation">
		<AppShell>
			<template #header>
				<Header
					:username="isLoggedIn ? username : ''"
					@toggleSidebar="toggleSidebar"
					@logoClick="goToWelcome"
					@userClick="goToUserControlPanel"
					title="Japella"
					:logoUrl="logoUrl"
					:sidebarEnabled="isLoggedIn && sidebarPreferenceEnabled"
					:themeToggleEnabled="isLoggedIn && themeColorSchemeSwitcherEnabled"
					:breadcrumbs="isLoggedIn"
				>
					<template #toolbar>
						<button
							v-if="isLoggedIn && route.name !== 'postBox'"
							type="button"
							class="header-post-button"
							title="Write a new post"
							@click="showPostDialog = true"
						>
							<Icon icon="mdi:pencil-plus-outline" width="18" height="18" aria-hidden="true" />
							<span>Post</span>
						</button>
						<QuickSearch
							v-if="isLoggedIn"
							placeholder="Search pages..."
							:search-fields="['title', 'description', 'category']"
						/>
					</template>
				</Header>
			</template>

			<template #sidebar>
				<Sidebar v-if="isLoggedIn && sidebarPreferenceEnabled" ref="sidebar" />
			</template>

			<template #banners>
				<NotificationBlock
					v-if="isImpersonating"
					type="warning"
					role="status"
				>
					<span class="impersonation-banner">
						<span>Impersonating <strong>{{ username }}</strong> (as {{ impersonatorUsername }})</span>
						<button type="button" class="impersonation-exit inline-icon" @click.stop="stopImpersonation" :disabled="impersonationLoading">
							<Icon icon="mdi:logout" width="16" height="16" aria-hidden="true" />
							<span>Exit</span>
						</button>
					</span>
				</NotificationBlock>
				<NotificationBlock
					v-for="(message, index) in statusMessages"
					:key="message.id ?? index"
					:type="message.type || 'info'"
					:role="message.type === 'error' ? 'alert' : 'status'"
					:message="message.message"
					:message-prefix="message.url ? message.message + ' ' : ''"
					:link-label="message.url ? 'More info' : ''"
					:link-href="message.url || ''"
				/>
			</template>

			<div id="loading" v-if="!clientReady" class="icon-and-text" style="margin: auto; margin-top: 5em;">
				<Icon icon="eos-icons:loading" width="48" height="48" />
				<div>
					Loading...
					<div v-if="loadingWarning"><br />
						<div class="error">{{ loadingWarning }}</div>
					</div>
				</div>
			</div>
			<template v-else>
				<div v-if="!isLoggedIn">
					<LoginForm @login-success="onLogin" />
				</div>
				<div v-else>
					<router-view />
				</div>
				<ErrorDialog ref="errorDialogRef" />
				<PWAInstallPrompt />
			</template>

			<template v-if="clientReady && !route.meta.hideFooter" #footer>
				<small>
					<span><a href="https://github.com/jamesread/Japella">Japella on GitHub</a></span>
					<span><a href="https://jamesread.github.io/Japella/">Documentation</a></span>
					<span id="currentVersion" v-if="isLoggedIn">{{ currentVersion }}</span>
				</small>
			</template>
		</AppShell>
	</Navigation>

	<PostBoxDialog v-if="showPostDialog" @close="showPostDialog = false" />

	<NotificationPopupsHost />
</template>

<style scoped>
    footer span {
        display: inline-block;
        margin-right: 10px;
    }

    .impersonation-banner {
        display: inline-flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.5rem 1rem;
    }

    .header-post-button {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        align-self: center;
        white-space: nowrap;
    }

    .impersonation-exit {
        padding: 0.15rem 0.6rem;
        font-size: 0.9em;
    }
</style>

<script setup>
    import { waitForClient } from '../javascript/util.js'
    import { fetchAppStatus } from '../javascript/status.js'
    import { loadAndApplyUserPreferences, registerSidebarApplier } from '../javascript/userPreferences.js'
    import '../javascript/theme.js';
    import {
        applyAppTheming,
        themeColorSchemeSwitcherEnabledFromFeatures,
        themeFeaturesFromStatus,
    } from '../javascript/applyAppTheming.js'
    import { useCustomTheme } from '../javascript/theme.js'
    import { ref, onMounted, onUnmounted, provide, watch, nextTick } from 'vue';
    import { useRoute, useRouter } from 'vue-router';
    import { Icon } from '@iconify/vue';
    import { useI18n } from 'vue-i18n';
	import LoginForm from './LoginForm.vue';
	import ErrorDialog from './ErrorDialog.vue';
	import PWAInstallPrompt from './PWAInstallPrompt.vue';
    import AppShell from 'picocrank/vue/components/AppShell.vue';
    import Header from 'picocrank/vue/components/Header.vue';
    import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
    import PostBoxDialog from './PostBoxDialog.vue';
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
	import { HEALTH_CHANGED_EVENT, fetchHealthSummaryCount } from '../javascript/health.js';

    const { t, locale, fallbackLocale } = useI18n();
    const route = useRoute();
    const router = useRouter();

    // The compose page, opened over whatever page is showing.
    const showPostDialog = ref(false);

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
    const themeColorSchemeSwitcherEnabled = ref(false);
    const appThemeFeatures = ref({});
    const errorDialogRef = ref();

	provide('showSectionError', (msg) => {
		errorDialogRef.value?.showError(msg)
	});
	provide('appThemeFeatures', appThemeFeatures);

	async function applySidebarPreference(enabled) {
		sidebarPreferenceEnabled.value = enabled;
		if (!enabled) {
			return;
		}
		await nextTick();
		if (!sidebar.value) {
			return;
		}
		sidebar.value.open();
		sidebar.value.stick();
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

		const [approvalsCount, healthCount] = await Promise.all([
			fetchApprovalsCount(),
			fetchHealthSummaryCount(),
		]);

		setupSidebarNavigation(navigation.value, {
			approvalsCount,
			healthCount,
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

	async function refreshHealthNavCount() {
		if (!navigation.value || !isLoggedIn.value) return;
		const showControlPanel = showControlPanelInSidebar({
			rbacIsSuperuser: window.userRbacIsSuperuser,
			rbacPermissions: window.userRbacPermissions,
		});
		if (!showControlPanel) return;
		const healthCount = await fetchHealthSummaryCount();
		navigation.value.addRouterLink('controlPanel', null, healthCount > 0 ? { count: healthCount } : {});
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

        appThemeFeatures.value = themeFeaturesFromStatus(st);
        themeColorSchemeSwitcherEnabled.value = themeColorSchemeSwitcherEnabledFromFeatures(appThemeFeatures.value);
        applyAppTheming(appThemeFeatures.value);

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

    const { discoverThemes } = useCustomTheme();

    onMounted(async () => {
        window.addEventListener(HEALTH_CHANGED_EVENT, refreshHealthNavCount);
        registerSidebarApplier(applySidebarPreference);

        setTimeout(() => {
            loadingWarning.value = 'If you are reading this text after waiting more than a few seconds, something has gone wrong. Please check your browser console for errors.';
        }, 2000);

        await waitForClient();
        await discoverThemes();
        clientReady.value = true;
        await loadStatus();
    });

    onUnmounted(() => {
        window.removeEventListener(HEALTH_CHANGED_EVENT, refreshHealthNavCount);
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
