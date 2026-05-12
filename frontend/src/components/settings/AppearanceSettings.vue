<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useAppearance, type ColorScheme, type LanguageCode } from '../../composables/useAppearance';
import { ChevronDown } from 'lucide-vue-next';

const { appearanceSettings, saveAppearanceSettings, setColorSchemePreview, clearColorSchemePreview } = useAppearance();

const appearanceDraft = ref({ ...appearanceSettings.value });
const settingsNotice = ref('');
const languageOpen = ref(false);
const timezoneOpen = ref(false);
const dropdownRootRef = ref<HTMLElement | null>(null);

const timezoneCountries = [
  { value: 'Asia/Shanghai', label: '中国（北京时间） UTC+08:00' },
  { value: 'Asia/Hong_Kong', label: '中国香港 UTC+08:00' },
  { value: 'Asia/Tokyo', label: '日本（东京） UTC+09:00' },
  { value: 'Asia/Seoul', label: '韩国（首尔） UTC+09:00' },
  { value: 'Asia/Singapore', label: '新加坡 UTC+08:00' },
  { value: 'Asia/Bangkok', label: '泰国（曼谷） UTC+07:00' },
  { value: 'Asia/Kolkata', label: '印度（新德里） UTC+05:30' },
  { value: 'Australia/Sydney', label: '澳大利亚（悉尼） UTC+10:00' },
  { value: 'Europe/London', label: '英国（伦敦） UTC+00:00' },
  { value: 'Europe/Paris', label: '法国（巴黎） UTC+01:00' },
  { value: 'Europe/Berlin', label: '德国（柏林） UTC+01:00' },
  { value: 'Europe/Madrid', label: '西班牙（马德里） UTC+01:00' },
  { value: 'Europe/Rome', label: '意大利（罗马） UTC+01:00' },
  { value: 'Europe/Moscow', label: '俄罗斯（莫斯科） UTC+03:00' },
  { value: 'America/New_York', label: '美国（纽约） UTC-05:00' },
  { value: 'America/Chicago', label: '美国（芝加哥） UTC-06:00' },
  { value: 'America/Denver', label: '美国（丹佛） UTC-07:00' },
  { value: 'America/Los_Angeles', label: '美国（洛杉矶） UTC-08:00' },
  { value: 'America/Toronto', label: '加拿大（多伦多） UTC-05:00' },
  { value: 'America/Mexico_City', label: '墨西哥（墨西哥城） UTC-06:00' },
  { value: 'America/Sao_Paulo', label: '巴西（圣保罗） UTC-03:00' },
  { value: 'Africa/Cairo', label: '埃及（开罗） UTC+02:00' },
  { value: 'Africa/Johannesburg', label: '南非（约翰内斯堡） UTC+02:00' },
  { value: 'UTC', label: '协调世界时（UTC） UTC+00:00' },
];

const timezoneOptions = computed(() => {
  const current = appearanceDraft.value.timezone;
  const matched = timezoneCountries.some((item) => item.value === current);
  if (matched) return timezoneCountries;
  if (!current) return timezoneCountries;
  return [
    { value: current, label: `当前时区（${current}）` },
    ...timezoneCountries,
  ];
});

const colorSchemeOptions: { value: ColorScheme; label: string; desc: string }[] = [
  { value: 'auto', label: '自动', desc: '跟随系统外观' },
  { value: 'light', label: '浅色', desc: '高亮背景，适合白天使用' },
  { value: 'dark', label: '深色', desc: '降低亮度，适合夜间使用' },
];

const languageOptions: { value: LanguageCode; label: string }[] = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
];

const selectedLanguageLabel = computed(
  () => languageOptions.find((item) => item.value === appearanceDraft.value.language)?.label || '简体中文',
);
const selectedTimezoneLabel = computed(
  () => timezoneOptions.value.find((item) => item.value === appearanceDraft.value.timezone)?.label || 'UTC',
);

const hasAppearanceChanges = computed(() => JSON.stringify(appearanceDraft.value) !== JSON.stringify(appearanceSettings.value));

function handleSave() {
  saveAppearanceSettings(appearanceDraft.value);
  clearColorSchemePreview();
  settingsNotice.value = '设置已保存';
  setTimeout(() => {
    settingsNotice.value = '';
  }, 3000);
}

watch(
  () => appearanceDraft.value.colorScheme,
  (next) => {
    setColorSchemePreview(next);
  },
  { immediate: true },
);

onUnmounted(() => {
  clearColorSchemePreview();
});

function toggleLanguageOpen() {
  languageOpen.value = !languageOpen.value;
  if (languageOpen.value) timezoneOpen.value = false;
}

function toggleTimezoneOpen() {
  timezoneOpen.value = !timezoneOpen.value;
  if (timezoneOpen.value) languageOpen.value = false;
}

function pickLanguage(value: LanguageCode) {
  appearanceDraft.value.language = value;
  languageOpen.value = false;
}

function pickTimezone(value: string) {
  appearanceDraft.value.timezone = value;
  timezoneOpen.value = false;
}

function handleOutsideClick(event: MouseEvent) {
  const target = event.target as Node | null;
  if (!dropdownRootRef.value || !target) return;
  if (!dropdownRootRef.value.contains(target)) {
    languageOpen.value = false;
    timezoneOpen.value = false;
  }
}

onMounted(() => {
  document.addEventListener('click', handleOutsideClick);
});

onUnmounted(() => {
  document.removeEventListener('click', handleOutsideClick);
});
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

    <div ref="dropdownRootRef" class="grid gap-6 lg:grid-cols-2">
      <label class="block">
        <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">界面语言</div>
        <div class="relative">
          <button
            type="button"
            @click.stop="toggleLanguageOpen"
            class="flex w-full items-center justify-between rounded-2xl border border-emerald-500/20 bg-[var(--panel-soft)] px-4 py-3.5 text-left text-base text-[color:var(--text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.06)] transition hover:border-emerald-500/45"
          >
            <span>{{ selectedLanguageLabel }}</span>
            <ChevronDown class="h-4 w-4 text-[color:var(--text-muted)]" :class="languageOpen ? 'rotate-180' : ''" />
          </button>
          <div
            v-if="languageOpen"
            class="absolute left-0 right-0 z-30 mt-2 overflow-hidden rounded-2xl border border-emerald-500/25 bg-[var(--frame-bg)] shadow-[0_16px_42px_rgba(0,0,0,0.32)]"
          >
            <button
              v-for="item in languageOptions"
              :key="item.value"
              type="button"
              @click="pickLanguage(item.value)"
              class="flex w-full items-center justify-between px-4 py-3 text-left text-sm transition hover:bg-emerald-500/10"
              :class="appearanceDraft.language === item.value ? 'text-emerald-400' : 'text-[color:var(--text-primary)]'"
            >
              <span>{{ item.label }}</span>
              <span v-if="appearanceDraft.language === item.value" class="text-xs font-semibold">已选</span>
            </button>
          </div>
        </div>
      </label>

      <label class="block">
        <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">时区</div>
        <div class="relative">
          <button
            type="button"
            @click.stop="toggleTimezoneOpen"
            class="flex w-full items-center justify-between rounded-2xl border border-emerald-500/20 bg-[var(--panel-soft)] px-4 py-3.5 text-left text-base text-[color:var(--text-primary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.06)] transition hover:border-emerald-500/45"
          >
            <span class="truncate pr-3">{{ selectedTimezoneLabel }}</span>
            <ChevronDown class="h-4 w-4 shrink-0 text-[color:var(--text-muted)]" :class="timezoneOpen ? 'rotate-180' : ''" />
          </button>
          <div
            v-if="timezoneOpen"
            class="absolute left-0 right-0 z-30 mt-2 max-h-72 overflow-y-auto rounded-2xl border border-emerald-500/25 bg-[var(--frame-bg)] shadow-[0_16px_42px_rgba(0,0,0,0.32)]"
          >
            <button
              v-for="item in timezoneOptions"
              :key="item.value"
              type="button"
              @click="pickTimezone(item.value)"
              class="flex w-full items-center justify-between px-4 py-3 text-left text-sm transition hover:bg-emerald-500/10"
              :class="appearanceDraft.timezone === item.value ? 'text-emerald-400' : 'text-[color:var(--text-primary)]'"
            >
              <span class="pr-3">{{ item.label }}</span>
              <span v-if="appearanceDraft.timezone === item.value" class="text-xs font-semibold">已选</span>
            </button>
          </div>
        </div>
        <div class="mt-2 text-xs text-[color:var(--text-muted)]">帖子时间与发布时间将按所选国家时区显示。</div>
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

    </div>
  </div>
</template>
