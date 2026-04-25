<script setup lang="ts">
import { computed, ref } from 'vue';

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
    </div>
  </div>
</template>
