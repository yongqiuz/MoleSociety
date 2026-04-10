import { createRouter, createWebHistory } from 'vue-router';
import App from '../App.vue';
import LoginPage from '../pages/LoginPage.vue';
import LogoutPage from '../pages/LogoutPage.vue';
import { useAuth } from '../composables/useAuth';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/app',
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
      component: App,
      meta: { requiresAuth: true },
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
