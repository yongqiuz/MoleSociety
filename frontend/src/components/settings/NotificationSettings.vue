<script setup lang="ts">
import { computed, ref } from 'vue';

type NotificationSettings = {
  mentions: boolean;
  replies: boolean;
  follows: boolean;
  directMessages: boolean;
  systemUpdates: boolean;
  emailDigest: boolean;
  quietHours: boolean;
};

const STORAGE_KEY = 'mole-notification-settings';

const defaultSettings: NotificationSettings = {
  mentions: true,
  replies: true,
  follows: true,
  directMessages: true,
  systemUpdates: true,
  emailDigest: false,
  quietHours: false,
};

const draft = ref<NotificationSettings>({ ...defaultSettings });
const saved = ref<NotificationSettings>({ ...defaultSettings });
const notice = ref('');

function loadSettings() {
  if (typeof window === 'undefined') return;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw) as Partial<NotificationSettings>;
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
  notice.value = '通知设置已保存';
  setTimeout(() => {
    notice.value = '';
  }, 3000);
}

const hasChanges = computed(() => JSON.stringify(draft.value) !== JSON.stringify(saved.value));

loadSettings();
</script>

<template>
  <div class="px-8 py-8 lg:px-10">
    <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[34px] font-semibold tracking-tight text-[color:var(--text-primary)]">通知</div>
        <div class="mt-2 text-sm text-[color:var(--text-muted)]">管理哪些行为会触发提醒，以及通知到达方式。</div>
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
          <div class="font-medium text-[color:var(--text-primary)]">@提及通知</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">当其他用户在帖子或回复中提到你时通知。</div>
        </div>
        <input v-model="draft.mentions" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">回复通知</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">当有人回复你的帖子时通知。</div>
        </div>
        <input v-model="draft.replies" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">关注通知</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">当新用户关注你时通知。</div>
        </div>
        <input v-model="draft.follows" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">私信通知</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">收到新私信时通知。</div>
        </div>
        <input v-model="draft.directMessages" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">系统更新通知</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">版本更新、维护和安全公告。</div>
        </div>
        <input v-model="draft.systemUpdates" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">邮件摘要</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">每天发送一次活动摘要到绑定邮箱。</div>
        </div>
        <input v-model="draft.emailDigest" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>

      <label class="flex items-center justify-between rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4">
        <div>
          <div class="font-medium text-[color:var(--text-primary)]">免打扰模式</div>
          <div class="mt-1 text-sm text-[color:var(--text-muted)]">减少非关键通知打断，仅保留高优先级提醒。</div>
        </div>
        <input v-model="draft.quietHours" type="checkbox" class="h-4 w-4 rounded accent-emerald-600" />
      </label>
    </div>
  </div>
</template>
