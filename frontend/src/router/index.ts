import { createRouter, createWebHistory } from 'vue-router';
import MainApp from '../pages/MainApp.vue';
import LoginPage from '../pages/LoginPage.vue';
import LogoutPage from '../pages/LogoutPage.vue';
import SettingsPage from '../pages/SettingsPage.vue';
import ProfileEditPage from '../pages/ProfileEditPage.vue';
import NotFoundPage from '../pages/NotFoundPage.vue';
import AppearanceSettings from '../components/settings/AppearanceSettings.vue';
import AccountSettings from '../components/settings/AccountSettings.vue';
import PlaceholderSettings from '../components/settings/PlaceholderSettings.vue';
import { useAuth } from '../composables/useAuth';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/login',
    },
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { guestOnly: true },
    },
    {
      path: '/logout',
      name: 'logout',
      component: LogoutPage,
    },
    {
      path: '/app',
      name: 'app',
      component: MainApp,
      meta: { requiresAuth: true },
    },
    {
      path: '/profile/edit',
      name: 'profile-edit',
      component: ProfileEditPage,
      meta: { requiresAuth: true },
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsPage,
      redirect: '/settings/appearance',
      meta: { requiresAuth: true },
      children: [
        {
          path: 'appearance',
          name: 'settings-appearance',
          component: AppearanceSettings,
        },
        {
          path: 'notifications',
          name: 'settings-notifications',
          component: PlaceholderSettings,
        },
        {
          path: 'privacy',
          name: 'settings-privacy',
          component: PlaceholderSettings,
        },
        {
          path: 'account',
          name: 'settings-account',
          component: AccountSettings,
        },
      ],
    },
    {
      path: '/404',
      name: 'not-found',
      component: NotFoundPage,
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/404',
    },
  ],
});

router.beforeEach((to) => {
  const { isAuthenticated, loadSession } = useAuth();
  return loadSession().then(() => {
    if (to.meta.requiresAuth && !isAuthenticated.value) {
      return {
        path: '/login',
        query: { redirect: to.fullPath },
      };
    }

    if (to.meta.guestOnly && isAuthenticated.value) {
      const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : '/app';
      return redirect;
    }

    return true;
  });
});

export default router;
