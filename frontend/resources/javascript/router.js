import { createRouter, createWebHistory } from 'vue-router'
import { useI18n } from 'vue-i18n'

// Import all the components
import Welcome from '../vue/Welcome.vue'
import Timeline from '../vue/Timeline.vue'
import Campaigns from '../vue/Campaigns.vue'
import CampaignDetails from '../vue/CampaignDetails.vue'
import AppStatus from '../vue/AppStatus.vue'
import PostBox from '../vue/PostBox.vue'
import Calendar from '../vue/Calendar.vue'
import Settings from '../vue/Settings.vue'
import CannedPosts from '../vue/CannedPosts.vue'
import UserList from '../vue/UserList.vue'
import SocialAccounts from '../vue/SocialAccounts.vue'
import SocialAccountDetails from '../vue/SocialAccountDetails.vue'
import OAuthServices from '../vue/OAuthServices.vue'
import ApiKeys from '../vue/ApiKeys.vue'
import Media from '../vue/Media.vue'
import ControlPanel from '../vue/ControlPanel.vue'

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
    path: '/social-accounts/:id',
    name: 'socialAccountDetails',
    component: SocialAccountDetails,
    meta: {
      title: 'Social Account',
      requiresAuth: true
    }
  },
  {
    path: '/oauth-services',
    name: 'oauthServices',
    component: OAuthServices,
    meta: {
      title: 'OAuth Services',
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
    path: '/status',
    name: 'appStatus',
    component: AppStatus,
    meta: {
      icon: ActivityIcon,
      title: 'Status',
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

  next()
})

export default router
