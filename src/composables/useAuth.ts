import { computed, ref } from 'vue';
import { ApiError, connectWalletAndLogin, fetchCurrentSession, logoutSession, type AuthSession } from '../api/authApi';

const session = ref<AuthSession | null>(null);
const loading = ref(false);
const ready = ref(false);
let pendingLoad: Promise<AuthSession | null> | null = null;

async function loadSession(force = false) {
  if (pendingLoad && !force) {
    return pendingLoad;
  }

  pendingLoad = (async () => {
    loading.value = true;
    try {
      const nextSession = await fetchCurrentSession();
      session.value = nextSession;
      ready.value = true;
      return nextSession;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        session.value = null;
        ready.value = true;
        return null;
      }
      throw error;
    } finally {
      loading.value = false;
      pendingLoad = null;
    }
  })();

  return pendingLoad;
}

async function login() {
  const nextSession = await connectWalletAndLogin();
  session.value = nextSession;
  ready.value = true;
  return nextSession;
}

async function logout() {
  try {
    await logoutSession();
  } finally {
    session.value = null;
    ready.value = true;
  }
}

export function useAuth() {
  return {
    session,
    isAuthenticated: computed(() => !!session.value),
    isLoading: computed(() => loading.value),
    isReady: computed(() => ready.value),
    login,
    logout,
    loadSession,
  };
}
