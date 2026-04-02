import { createRouter, createWebHistory } from 'vue-router'
import { useI18n } from 'vue-i18n'

// Import all the components
import Welcome from '../vue/Welcome.vue'
import Timeline from '../vue/Timeline.vue'
import Campaigns from '../vue/Campaigns.vue'
import CampaignDetails from '../vue/CampaignDetails.vue'
import PostBox from '../vue/PostBox.vue'
import Calendar from '../vue/Calendar.vue'
import Settings from '../vue/Settings.vue'
import CannedPosts from '../vue/CannedPosts.vue'
import UserList from '../vue/UserList.vue'
import UserDetails from '../vue/UserDetails.vue'
import UserDetailsRoles from '../vue/UserDetailsRoles.vue'
import UserDetailsPassword from '../vue/UserDetailsPassword.vue'
import RbacSettings from '../vue/RbacSettings.vue'
import SocialAccounts from '../vue/SocialAccounts.vue'
import SocialAccountDetails from '../vue/SocialAccountDetails.vue'
import ApiKeys from '../vue/ApiKeys.vue'
import Media from '../vue/Media.vue'
import ControlPanel from '../vue/ControlPanel.vue'
import UserControlPanel from '../vue/UserControlPanel.vue'
import ChangePassword from '../vue/ChangePassword.vue'
import PostDetails from '../vue/PostDetails.vue'
import Feed from '../vue/Feed.vue'
import Logs from '../vue/Logs.vue'
import BrowserDiagnostics from '../vue/BrowserDiagnostics.vue'
import ChatBots from '../vue/ChatBots.vue'
import ChatBotDetails from '../vue/ChatBotDetails.vue'
import ChatBotConversationsPage from '../vue/ChatBotConversationsPage.vue'
import AllChatBotConversationsPage from '../vue/AllChatBotConversationsPage.vue'
import ChatBotHooksPage from '../vue/ChatBotHooksPage.vue'
import CreateUser from '../vue/CreateUser.vue'
import Connectors from '../vue/Connectors.vue'
import SystemArchitecture from '../vue/SystemArchitecture.vue'
import UserGroups from '../vue/UserGroups.vue'
import AddSocialAccount from '../vue/AddSocialAccount.vue'
import { canAccessControlPanelFromStatus } from './rbacAccess.js'

import {
  SettingsIcon,
  EditIcon,
  ImageIcon,
  VolumeUpIcon,
  PackageIcon,
  ClockIcon,
  CalendarIcon,
  UserMultiple03Icon,
  KeyIcon,
  UserGroupIcon,
  ActivityIcon,
  HomeIcon,
  Robot01Icon,
  WebSecurityIcon,
  SecurityValidationIcon,
  MessageMultiple01Icon,
} from '@hugeicons/core-free-icons'

const routes = [
  {
    path: '/',
    name: 'welcome',
    component: Welcome,
    meta: {
      title: 'Welcome',
      requiresAuth: false
    }
  },
  {
    path: '/connectors',
    name: 'connectors',
    component: Connectors,
    meta: {
      title: 'Connectors',
      requiresAuth: true
    }
  },
  {
    path: '/chat-bots',
    name: 'chatBots',
    component: ChatBots,
    meta: {
      icon: Robot01Icon,
      title: 'Chat Bots',
      requiresAuth: true
    }
  },
  {
    path: '/chat-bots/:connector/:identity/conversations',
    name: 'chatBotConversations',
    component: ChatBotConversationsPage,
    meta: {
      title: 'Bot Conversations',
      requiresAuth: true
    }
  },
  {
    path: '/chat-bots/:connector/:identity/hooks',
    name: 'chatBotHooks',
    component: ChatBotHooksPage,
    meta: {
      title: 'Bot Message Hooks',
      requiresAuth: true
    }
  },
  {
    path: '/chat-bots/:connector/:identity?',
    name: 'chatBotDetails',
    component: ChatBotDetails,
    meta: {
      title: 'Chat Bot Details',
      requiresAuth: true
    }
  },
  {
    path: '/post',
    name: 'postBox',
    component: PostBox,
    meta: {
      icon: EditIcon,
      title: 'Post',
      requiresAuth: true
    }
  },
  {
    path: '/media',
    name: 'media',
    component: Media,
    meta: {
      icon: ImageIcon,
      title: 'Media',
      requiresAuth: true
    }
  },
  {
    path: '/campaigns',
    name: 'campaigns',
    component: Campaigns,
    meta: {
      icon: VolumeUpIcon,
      title: 'Campaigns',
      requiresAuth: true
    }
  },
  {
    path: '/campaigns/:id',
    name: 'campaignDetails',
    component: CampaignDetails,
    meta: {
      title: 'Campaign Details',
      requiresAuth: true
    }
  },
  {
    path: '/canned-posts',
    name: 'cannedPosts',
    component: CannedPosts,
    meta: {
      icon: PackageIcon,
      title: 'Canned Posts',
      requiresAuth: true
    }
  },
  {
    path: '/timeline',
    name: 'timeline',
    component: Timeline,
    meta: {
      icon: ClockIcon,
      title: 'Timeline',
      requiresAuth: true
    }
  },
  {
    path: '/timeline/:id',
    name: 'postDetails',
    component: PostDetails,
    meta: {
      title: 'Post Details',
      requiresAuth: true
    }
  },
  {
    path: '/feed',
    name: 'feed',
    component: Feed,
    meta: {
      icon: ActivityIcon,
      title: 'Feed',
      requiresAuth: true
    }
  },
  {
    path: '/conversations',
    name: 'chatBotConversationsAll',
    component: AllChatBotConversationsPage,
    meta: {
      icon: MessageMultiple01Icon,
      title: 'Conversations',
      requiresAuth: true
    }
  },
  {
    path: '/logs',
    name: 'logs',
    component: Logs,
    meta: {
      icon: ActivityIcon,
      title: 'Logs',
      requiresAuth: true
    }
  },
  {
    path: '/calendar',
    name: 'calendar',
    component: Calendar,
    meta: {
      icon: CalendarIcon,
      title: 'Calendar',
      requiresAuth: true
    }
  },
  {
    path: '/social-accounts',
    name: 'socialAccounts',
    component: SocialAccounts,
    meta: {
      icon: UserMultiple03Icon,
      title: 'Social Accounts',
      requiresAuth: true
    }
  },
  {
    path: '/social-accounts/add',
    name: 'addSocialAccount',
    component: AddSocialAccount,
    meta: {
      title: 'Add Social Account',
      requiresAuth: true
    }
  },
  {
    path: '/social-accounts/:id',
    name: 'socialAccountDetails',
    component: SocialAccountDetails,
    meta: {
      title: 'Social Account',
      requiresAuth: true
    }
  },
  {
    path: '/settings',
    name: 'settings',
    component: Settings,
    meta: {
      icon: SettingsIcon,
      title: 'Settings',
      requiresAuth: true
    }
  },
  {
    path: '/settings/rbac',
    name: 'rbacSettings',
    component: RbacSettings,
    meta: {
      title: 'Roles & permissions',
      requiresAuth: true
    }
  },
  {
    path: '/api-keys',
    name: 'settingsApiKeys',
    component: ApiKeys,
    meta: {
      icon: KeyIcon,
      title: 'API Keys',
      requiresAuth: true
    }
  },
  {
    path: '/users',
    name: 'settingsUsers',
    component: UserList,
    meta: {
      icon: UserGroupIcon,
      title: 'Users',
      requiresAuth: true
    }
  },
  {
    path: '/create-user',
    name: 'createUser',
    component: CreateUser,
    meta: {
      title: 'Create User',
      requiresAuth: true
    }
  },
  {
    path: '/users/:id',
    name: 'userDetails',
    component: UserDetails,
    meta: {
      title: 'User',
      requiresAuth: true
    }
  },
  {
    path: '/users/:id/roles',
    name: 'userDetailsRoles',
    component: UserDetailsRoles,
    meta: {
      icon: WebSecurityIcon,
      title: 'Roles & permissions',
      description: 'Assign RBAC roles and review effective permissions',
      requiresAuth: true
    }
  },
  {
    path: '/users/:id/password',
    name: 'userDetailsPassword',
    component: UserDetailsPassword,
    meta: {
      icon: SecurityValidationIcon,
      title: 'Reset password',
      description: 'Set a new password for this user',
      requiresAuth: true
    }
  },
  {
    path: '/users/:id/api-keys',
    name: 'userDetailsApiKeys',
    component: ApiKeys,
    meta: {
      icon: KeyIcon,
      title: 'API Keys',
      description: 'View and manage API keys for this user',
      requiresAuth: true
    }
  },
  {
    path: '/user-groups',
    name: 'userGroups',
    component: UserGroups,
    meta: {
      title: 'User Groups',
      requiresAuth: true
    }
  },
  {
    path: '/change-password',
    name: 'changePassword',
    component: ChangePassword,
    meta: {
      title: 'Change Password',
      requiresAuth: true
    }
  },
  {
    path: '/control-panel',
    name: 'controlPanel',
    component: ControlPanel,
    meta: {
      icon: HomeIcon,
      title: 'Control Panel',
      requiresAuth: true,
      requiresControlPanel: true
    }
  },
  {
    path: '/control-panel/system-architecture',
    name: 'systemArchitecture',
    component: SystemArchitecture,
    meta: {
      title: 'System Architecture',
      requiresAuth: true,
      requiresControlPanel: true
    }
  },
  {
    path: '/user-control-panel',
    name: 'userControlPanel',
    component: UserControlPanel,
    meta: {
      title: 'User Control Panel',
      requiresAuth: true
    }
  },
  {
    path: '/browser-diagnostics',
    name: 'browserDiagnostics',
    component: BrowserDiagnostics,
    meta: {
      title: 'Browser Diagnostics',
      requiresAuth: true
    }
  },
  // Catch all route - redirect to welcome
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Helper function to wait for client to be ready
function waitForClient() {
  return new Promise((resolve) => {
    const check = () => {
      if (window.client) {
        resolve(true);
      } else {
        setTimeout(check, 100);
      }
    };
    check();
  });
}

// Navigation guard to handle authentication
router.beforeEach(async (to, from, next) => {
  // Check if route requires authentication
  if (to.meta.requiresAuth) {
    // If we already know the auth state, use it
    if (window.isLoggedIn !== undefined) {
      if (!window.isLoggedIn) {
        next('/')
        return
      }
    } else {
      // If auth state is unknown, wait for client and check with server
      try {
        await waitForClient()
        const status = await window.client.getStatus()
        window.isLoggedIn = status.isLoggedIn

        if (!status.isLoggedIn) {
          next('/')
          return
        }
      } catch (error) {
        console.error('Error checking authentication status:', error)
        // If we can't check auth status, redirect to login
        next('/')
        return
      }
    }
  }

  if (to.meta.requiresControlPanel) {
    try {
      await waitForClient()
      const status = await window.client.getStatus()
      if (!canAccessControlPanelFromStatus(status)) {
        next('/')
        return
      }
    } catch (error) {
      console.error('Error checking control panel access:', error)
      next('/')
      return
    }
  }

  next()
})

export default router
