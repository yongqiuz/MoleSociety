import { computed, ref } from 'vue';

export type AuthSession = {
  id: string;
  handle: string;
  displayName: string;
  instance: string;
  bio: string;
  avatarUrl: string;
};

const STORAGE_KEY = 'molesociety-auth-session';

const demoAccounts: AuthSession[] = [
  {
    id: 'user_archive',
    handle: '@archive',
    displayName: 'Whale Archive',
    instance: 'vault.social',
    bio: '为创作者提供永久内容归档与链上身份锚定。',
    avatarUrl: '',
  },
  {
    id: 'user_librarian',
    handle: '@librarian',
    displayName: 'Node Librarian',
    instance: 'readers.polkadot',
    bio: '把书籍确权、媒体存储和去中心化社交连接在一起。',
    avatarUrl: '',
  },
  {
    id: 'user_fedilab',
    handle: '@fedilab',
    displayName: 'Open Federation Lab',
    instance: 'relay.zone',
    bio: '探索 ActivityPub、实时会话和多实例协作。',
    avatarUrl: '',
  },
];

const session = ref<AuthSession | null>(null);
let initialized = false;

function loadSession() {
  if (typeof window === 'undefined' || initialized) return;
  initialized = true;

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    session.value = JSON.parse(raw) as AuthSession;
  } catch {
    session.value = null;
  }
}

function persistSession(nextSession: AuthSession | null) {
  if (typeof window === 'undefined') return;

  if (!nextSession) {
    window.localStorage.removeItem(STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(nextSession));
}

function login(accountId: string, password: string) {
  loadSession();

  const trimmedPassword = password.trim();
  if (!trimmedPassword) {
    throw new Error('请输入密码后再登录');
  }

  const nextSession = demoAccounts.find((account) => account.id === accountId);
  if (!nextSession) {
    throw new Error('未找到对应的登录账号');
  }

  session.value = nextSession;
  persistSession(nextSession);
  return nextSession;
}

function logout() {
  loadSession();
  session.value = null;
  persistSession(null);
}

export function useAuth() {
  loadSession();

  return {
    session,
    demoAccounts,
    isAuthenticated: computed(() => !!session.value),
    login,
    logout,
    loadSession,
  };
}
