<script setup lang="ts">
import { computed, ref } from 'vue';
import { useAppearance, type ColorScheme } from '../../composables/useAppearance';

const { appearanceSettings, saveAppearanceSettings } = useAppearance();

const appearanceDraft = ref({ ...appearanceSettings.value });
const settingsNotice = ref('');

const timezones = [
  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+08:00)' },
  { value: 'UTC', label: 'UTC (UTC+00:00)' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+09:00)' },
  { value: 'Europe/London', label: 'Europe/London (UTC+00:00)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (UTC-08:00)' },
  { value: 'America/New_York', label: 'America/New_York (UTC-05:00)' },
];

const colorSchemeOptions: { value: ColorScheme; label: string; desc: string }[] = [
  { value: 'auto', label: '自动', desc: '跟随系统外观' },
  { value: 'light', label: '浅色', desc: '高亮背景，适合白天使用' },
  { value: 'dark', label: '深色', desc: '降低亮度，适合夜间使用' },
];

const hasAppearanceChanges = computed(() => JSON.stringify(appearanceDraft.value) !== JSON.stringify(appearanceSettings.value));

function handleSave() {
  saveAppearanceSettings(appearanceDraft.value);
  settingsNotice.value = '设置已保存';
  setTimeout(() => {
    settingsNotice.value = '';
  }, 3000);
}
</script>

<template>
  <div class="px-8 py-8 lg:px-10">
    <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[34px] font-semibold tracking-tight text-[color:var(--text-primary)]">外观</div>
        <div class="mt-2 text-sm text-[color:var(--text-muted)]">调整界面语言、时间显示与整体视觉模式。</div>
      </div>

      <div class="flex items-center gap-3">
        <span v-if="settingsNotice" class="text-sm text-emerald-500">{{ settingsNotice }}</span>
        <button
          :disabled="!hasAppearanceChanges"
          @click="handleSave"
          class="rounded-xl bg-emerald-600 px-6 py-3 text-base font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          保存更改
        </button>
      </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-2">
      <label class="block">
        <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">界面语言</div>
        <select
          v-model="appearanceDraft.language"
          class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
        >
          <option value="zh-CN">简体中文</option>
          <option value="en-US">English</option>
        </select>
      </label>

      <label class="block">
        <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">时区</div>
        <select
          v-model="appearanceDraft.timezone"
          class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
        >
          <option v-for="item in timezones" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>
      </label>
    </div>

    <div class="mt-10 space-y-10">
      <div>
        <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">配色方案</div>
        <div class="grid gap-3 md:grid-cols-3">
          <button
            v-for="option in colorSchemeOptions"
            :key="option.value"
            @click="appearanceDraft.colorScheme = option.value"
            class="rounded-2xl border p-4 text-left transition"
            :class="appearanceDraft.colorScheme === option.value ? 'border-emerald-500/60 bg-emerald-500/10' : 'border-[color:var(--border-color)] hover:border-emerald-500/30 hover:bg-[var(--panel-soft)]'"
          >
            <div class="mb-2 flex items-center justify-between gap-2">
              <span class="text-sm font-semibold text-[color:var(--text-primary)]">{{ option.label }}</span>
              <span
                class="h-4 w-4 rounded-full border-2 transition"
                :class="appearanceDraft.colorScheme === option.value ? 'border-emerald-500 bg-emerald-500' : 'border-[color:var(--text-muted)] bg-transparent'"
              />
            </div>
            <div class="text-xs text-[color:var(--text-muted)]">{{ option.desc }}</div>
          </button>
        </div>
      </div>

      <div class="border-t border-[color:var(--border-color)] pt-8">
        <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">动画与可访问性</div>
        <div class="space-y-5 text-[color:var(--text-primary)]">
          <label class="flex items-start gap-3">
            <input v-model="appearanceDraft.slowMode" type="checkbox" class="mt-1 h-4 w-4 rounded accent-emerald-600" />
            <div>
              <div class="font-medium">慢速模式</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">点击后手动加载动态，减少页面连续刷新。</div>
            </div>
          </label>

          <label class="flex items-start gap-3">
            <input v-model="appearanceDraft.autoplayGif" type="checkbox" class="mt-1 h-4 w-4 rounded accent-emerald-600" />
            <div>
              <div class="font-medium">自动播放 GIF 动画</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">在时间线中直接播放轻量动画内容。</div>
            </div>
          </label>

          <label class="flex items-start gap-3">
            <input v-model="appearanceDraft.reduceMotion" type="checkbox" class="mt-1 h-4 w-4 rounded accent-emerald-600" />
            <div>
              <div class="font-medium">降低动态效果</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">减少转场和悬浮动画带来的视觉刺激。</div>
            </div>
          </label>
        </div>
      </div>
    </div>
  </div>
</template>
