<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ApiError, changePassword } from '../../api/authApi';
import { updateUserProfile } from '../../api/socialApi';
import { useAuth } from '../../composables/useAuth';

type PrivacySettings = {
  profileDiscoverable: boolean;
  searchableByHandle: boolean;
  showOnlineStatus: boolean;
  allowTagging: boolean;
  allowQuoteFrom: 'anyone' | 'followers' | 'none';
  allowDmFrom: 'anyone' | 'followers' | 'none';
};

const STORAGE_KEY = 'mole-privacy-settings';

const defaultSettings: PrivacySettings = {
  profileDiscoverable: true,
  searchableByHandle: true,
  showOnlineStatus: false,
  allowTagging: true,
  allowQuoteFrom: 'followers',
  allowDmFrom: 'followers',
};

const draft = ref<PrivacySettings>({ ...defaultSettings });
const saved = ref<PrivacySettings>({ ...defaultSettings });
const notice = ref('');
const changingPassword = ref(false);
const passwordNotice = ref('');
const passwordError = ref('');
const accountNotice = ref('');
const accountError = ref('');
const savingAccount = ref(false);
const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
});
const { currentUser, updateCurrentUserLocally } = useAuth();
const readonlyUsername = computed(() => String(currentUser.value?.username || currentUser.value?.handle || '').replace(/^@/, ''));
const requireCurrentPassword = computed(() => currentUser.value?.requireCurrentPassword !== false);
const editableUsername = ref('');
watch(
  () => readonlyUsername.value,
  (value) => {
    if (!editableUsername.value) {
      editableUsername.value = value;
    }
  },
  { immediate: true },
);

function loadSettings() {
  if (typeof window === 'undefined') return;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw) as Partial<PrivacySettings>;
    const next = { ...defaultSettings, ...parsed };
    draft.value = next;
    saved.value = next;
  } catch {
    draft.value = { ...defaultSettings };
    saved.value = { ...defaultSettings };
  }
}

function saveSettings() {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(draft.value));
  }
  saved.value = { ...draft.value };
  notice.value = '隐私设置已保存';
  setTimeout(() => {
    notice.value = '';
  }, 3000);
}

const hasChanges = computed(() => JSON.stringify(draft.value) !== JSON.stringify(saved.value));

loadSettings();

async function saveUsername() {
  if (!currentUser.value || savingAccount.value) return;
  accountNotice.value = '';
  accountError.value = '';
  const nextUsername = editableUsername.value.trim();
  if (!nextUsername) {
    accountError.value = '账号不能为空';
    return;
  }
  savingAccount.value = true;
  try {
    await updateUserProfile(currentUser.value.id, { username: nextUsername });
    updateCurrentUserLocally({ username: nextUsername });
    editableUsername.value = nextUsername;
    accountNotice.value = '账号已更新';
  } catch (err) {
    if (err instanceof ApiError && err.code === 'AUTH_USERNAME_TAKEN') {
      accountError.value = '账号已被占用';
    } else {
      accountError.value = '账号更新失败，请稍后重试';
    }
  } finally {
    savingAccount.value = false;
  }
}

async function submitChangePassword() {
  passwordNotice.value = '';
  passwordError.value = '';

  const currentPassword = passwordForm.value.currentPassword.trim();
  const newPassword = passwordForm.value.newPassword.trim();
  const confirmPassword = passwordForm.value.confirmPassword.trim();

  if (!newPassword || !confirmPassword) {
    passwordError.value = '请填写完整密码信息';
    return;
  }
  if (requireCurrentPassword.value && !currentPassword) {
    passwordError.value = '请填写当前密码';
    return;
  }
  if (newPassword.length < 6) {
    passwordError.value = '新密码至少需要 6 位';
    return;
  }
  if (newPassword !== confirmPassword) {
    passwordError.value = '两次输入的新密码不一致';
    return;
  }
  if (currentPassword && currentPassword === newPassword) {
    passwordError.value = '新密码不能与旧密码相同';
    return;
  }

  changingPassword.value = true;
  try {
    await changePassword(currentPassword, newPassword);
    passwordForm.value = {
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
    };
    passwordNotice.value = '密码修改成功';
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.code === 'AUTH_INVALID_PASSWORD') {
        passwordError.value = '当前密码不正确';
      } else if (err.code === 'AUTH_WEAK_PASSWORD') {
        passwordError.value = '新密码至少需要 6 位';
      } else if (err.code === 'AUTH_SESSION_REQUIRED') {
        passwordError.value = '登录状态失效，请重新登录';
      } else {
        passwordError.value = err.message || '修改密码失败，请稍后重试';
      }
    } else {
      passwordError.value = '修改密码失败，请稍后重试';
    }
  } finally {
    changingPassword.value = false;
  }
}
</script>

<template>
  <div class="px-8 py-8 lg:px-10">
    <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[34px] font-semibold tracking-tight text-[color:var(--text-primary)]">隐私与安全</div>
        <div class="mt-2 text-sm text-[color:var(--text-muted)]">控制个人信息可见范围与互动权限。</div>
      </div>
      <div class="flex items-center gap-3">
        <span v-if="notice" class="text-sm text-emerald-500">{{ notice }}</span>
        <button
          :disabled="!hasChanges"
          @click="saveSettings"
          class="rounded-xl bg-emerald-600 px-6 py-3 text-base font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          保存更改
        </button>
      </div>
    </div>

    <div class="space-y-4">
      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">公开展示个人资料</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">允许未登录用户访问你的公开主页。</div>
        </div>
        <input v-model="draft.profileDiscoverable" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">可被用户名搜索</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">允许其他人通过 handle 快速找到你。</div>
        </div>
        <input v-model="draft.searchableByHandle" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">显示在线状态</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">向已关注你的用户展示活跃状态。</div>
        </div>
        <input v-model="draft.showOnlineStatus" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">允许被他人标签提及</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">他人可以在内容中通过 # 相关标签联动你的帖子。</div>
        </div>
        <input v-model="draft.allowTagging" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <div class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div class="text-sm font-semibold text-[color:var(--text-primary)]">谁可以引用你的帖子</div>
        <div class="mt-3 grid gap-2 sm:grid-cols-3">
          <button
            @click="draft.allowQuoteFrom = 'anyone'"
            class="rounded-xl border px-3 py-2 text-sm transition"
            :class="draft.allowQuoteFrom === 'anyone' ? 'border-cyan-500/60 bg-cyan-500/10 text-cyan-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)]'"
          >
            任何人
          </button>
          <button
            @click="draft.allowQuoteFrom = 'followers'"
            class="rounded-xl border px-3 py-2 text-sm transition"
            :class="draft.allowQuoteFrom === 'followers' ? 'border-cyan-500/60 bg-cyan-500/10 text-cyan-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)]'"
          >
            仅关注者
          </button>
          <button
            @click="draft.allowQuoteFrom = 'none'"
            class="rounded-xl border px-3 py-2 text-sm transition"
            :class="draft.allowQuoteFrom === 'none' ? 'border-cyan-500/60 bg-cyan-500/10 text-cyan-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)]'"
          >
            禁止
          </button>
        </div>
      </div>

      <div class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div class="text-sm font-semibold text-[color:var(--text-primary)]">谁可以向你发私信</div>
        <div class="mt-3 grid gap-2 sm:grid-cols-3">
          <button
            @click="draft.allowDmFrom = 'anyone'"
            class="rounded-xl border px-3 py-2 text-sm transition"
            :class="draft.allowDmFrom === 'anyone' ? 'border-emerald-500/60 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)]'"
          >
            任何人
          </button>
          <button
            @click="draft.allowDmFrom = 'followers'"
            class="rounded-xl border px-3 py-2 text-sm transition"
            :class="draft.allowDmFrom === 'followers' ? 'border-emerald-500/60 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)]'"
          >
            仅关注者
          </button>
          <button
            @click="draft.allowDmFrom = 'none'"
            class="rounded-xl border px-3 py-2 text-sm transition"
            :class="draft.allowDmFrom === 'none' ? 'border-emerald-500/60 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)]'"
          >
            禁止
          </button>
        </div>
      </div>

      <div class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-5">
        <div class="text-base font-semibold text-[color:var(--text-primary)]">修改密码</div>
        <div class="mt-1 text-sm text-[color:var(--text-muted)]">
          {{ requireCurrentPassword ? '输入当前密码并设置新密码，至少 6 位。' : '首次设置密码，无需填写当前密码。' }}
        </div>
        <div class="mt-4">
          <div class="mb-2 text-xs font-semibold uppercase tracking-wider text-[color:var(--text-muted)]">账号</div>
          <div class="flex items-center gap-3">
            <input
              v-model="editableUsername"
              type="text"
              class="flex-1 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-3 text-sm text-[color:var(--text-primary)] outline-none focus:border-emerald-500/60"
            />
            <button
              type="button"
              :disabled="savingAccount || editableUsername.trim() === readonlyUsername"
              @click="saveUsername"
              class="rounded-xl border border-emerald-500/40 bg-emerald-500/10 px-4 py-3 text-sm font-semibold text-emerald-300 transition hover:bg-emerald-500/20 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {{ savingAccount ? '保存中...' : '保存账号' }}
            </button>
          </div>
          <div v-if="accountNotice" class="mt-2 text-xs text-emerald-500">{{ accountNotice }}</div>
          <div v-if="accountError" class="mt-2 text-xs text-rose-400">{{ accountError }}</div>
        </div>
        <form class="mt-4 grid gap-3 sm:grid-cols-2" @submit.prevent="submitChangePassword">
          <input
            v-if="requireCurrentPassword"
            v-model="passwordForm.currentPassword"
            type="password"
            autocomplete="current-password"
            placeholder="当前密码"
            class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-3 text-sm text-[color:var(--text-primary)] outline-none focus:border-emerald-500/60 sm:col-span-2"
          />
          <input
            v-model="passwordForm.newPassword"
            type="password"
            autocomplete="new-password"
            placeholder="新密码"
            class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-3 text-sm text-[color:var(--text-primary)] outline-none focus:border-emerald-500/60"
          />
          <input
            v-model="passwordForm.confirmPassword"
            type="password"
            autocomplete="new-password"
            placeholder="确认新密码"
            class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-3 text-sm text-[color:var(--text-primary)] outline-none focus:border-emerald-500/60"
          />
          <div class="sm:col-span-2 flex items-center justify-between gap-4">
            <div>
              <div v-if="passwordNotice" class="text-sm text-emerald-500">{{ passwordNotice }}</div>
              <div v-if="passwordError" class="text-sm text-rose-400">{{ passwordError }}</div>
            </div>
            <button
              type="submit"
              :disabled="changingPassword"
              class="rounded-xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {{ changingPassword ? '提交中...' : '更新密码' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
