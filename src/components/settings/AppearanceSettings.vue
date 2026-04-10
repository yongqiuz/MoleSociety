<script setup lang="ts">
import { ref, computed } from 'vue';
import { useAppearance } from '../../composables/useAppearance';

const {
  appearanceSettings,
  saveAppearanceSettings,
} = useAppearance();

const appearanceDraft = ref({ ...appearanceSettings.value });
const settingsNotice = ref('');

const hasAppearanceChanges = computed(() => {
  return JSON.stringify(appearanceDraft.value) !== JSON.stringify(appearanceSettings.value);
});

function handleSave() {
  saveAppearanceSettings(appearanceDraft.value);
  settingsNotice.value = '设置已保存';
  setTimeout(() => (settingsNotice.value = ''), 3000);
}

function toneClass(value: string) {
  return appearanceDraft.value.colorScheme === value ? 'text-emerald-500 font-semibold' : 'text-[color:var(--text-secondary)]';
}
</script>

<template>
  <div class="px-8 py-8 lg:px-10">
    <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[34px] font-semibold tracking-tight text-[color:var(--text-primary)]">外观</div>
        <div class="mt-2 text-sm text-[color:var(--text-muted)]">调整界面语言、配色方案和可访问性偏好。</div>
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
          <option>简体中文</option>
          <option>English</option>
        </select>
      </label>

      <label class="block">
        <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">时区</div>
        <select
          v-model="appearanceDraft.timezone"
          class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
        >
          <option>(GMT+08:00) Asia/Shanghai</option>
          <option>(GMT+00:00) UTC</option>
          <option>(GMT-07:00) America/Los_Angeles</option>
        </select>
      </label>
    </div>

    <div class="mt-10 space-y-10">
      <div>
        <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">配色方案</div>
        <div class="flex flex-wrap gap-8 text-base">
          <label class="flex items-center gap-2" :class="toneClass('auto')">
            <input v-model="appearanceDraft.colorScheme" type="radio" value="auto" class="accent-emerald-600" />
            <span>自动</span>
          </label>
          <label class="flex items-center gap-2" :class="toneClass('light')">
            <input v-model="appearanceDraft.colorScheme" type="radio" value="light" class="accent-emerald-600" />
            <span>浅色</span>
          </label>
          <label class="flex items-center gap-2" :class="toneClass('dark')">
            <input v-model="appearanceDraft.colorScheme" type="radio" value="dark" class="accent-emerald-600" />
            <span>深色</span>
          </label>
        </div>
      </div>

      <div>
        <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">对比度</div>
        <div class="flex flex-wrap gap-8 text-base text-[color:var(--text-primary)]">
          <label class="flex items-center gap-2">
            <input v-model="appearanceDraft.contrast" type="radio" value="auto" class="accent-emerald-600" />
            <span>自动</span>
          </label>
          <label class="flex items-center gap-2">
            <input v-model="appearanceDraft.contrast" type="radio" value="high" class="accent-emerald-600" />
            <span>高</span>
          </label>
        </div>
      </div>

      <label class="block">
        <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">表情符号样式</div>
        <select
          v-model="appearanceDraft.emojiStyle"
          class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
        >
          <option>自动</option>
          <option>原生 Emoji</option>
          <option>Twemoji</option>
        </select>
        <div class="mt-3 rounded-2xl border border-emerald-500/30 bg-emerald-500/5 px-4 py-4 text-sm leading-7 text-[color:var(--text-secondary)]">
          你可以在这里选择社区默认外观。配色方案默认显示为浅色，保存后会在整个前端界面生效。
        </div>
      </label>

      <div class="border-t border-[color:var(--border-color)] pt-8">
        <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">动画与可访问性</div>
        <div class="space-y-5 text-[color:var(--text-primary)]">
          <label class="flex items-start gap-3">
            <input v-model="appearanceDraft.slowMode" type="checkbox" class="mt-1 h-4 w-4 rounded accent-emerald-600" />
            <div>
              <div class="font-medium">慢速模式</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">点击查看时间线更新，而非自动滚动更新动态。</div>
            </div>
          </label>

          <label class="flex items-start gap-3">
            <input v-model="appearanceDraft.autoplayGif" type="checkbox" class="mt-1 h-4 w-4 rounded accent-emerald-600" />
            <div>
              <div class="font-medium">自动播放 GIF 动画</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">推荐开启，在动态流中直接预览轻量动画内容。</div>
            </div>
          </label>

          <label class="flex items-start gap-3">
            <input v-model="appearanceDraft.reduceMotion" type="checkbox" class="mt-1 h-4 w-4 rounded accent-emerald-600" />
            <div>
              <div class="font-medium">降低动态效果</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">减少转场、悬浮和自动播放带来的视觉刺激。</div>
            </div>
          </label>
        </div>
      </div>
    </div>
  </div>
</template>
