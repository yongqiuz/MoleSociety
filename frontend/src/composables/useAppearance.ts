import { computed, ref, watch } from 'vue';

export type ColorScheme = 'auto' | 'light' | 'dark';
export type ContrastMode = 'auto' | 'high';

export type AppearanceSettings = {
  language: string;
  timezone: string;
  colorScheme: ColorScheme;
  contrast: ContrastMode;
  emojiStyle: string;
  slowMode: boolean;
  autoplayGif: boolean;
  reduceMotion: boolean;
};

const STORAGE_KEY = 'whale-vault-appearance-settings';

const defaultAppearanceSettings: AppearanceSettings = {
  language: '简体中文',
  timezone: '(GMT+08:00) Asia/Shanghai',
  colorScheme: 'light',
  contrast: 'auto',
  emojiStyle: '自动',
  slowMode: false,
  autoplayGif: true,
  reduceMotion: false,
};

const settings = ref<AppearanceSettings>({ ...defaultAppearanceSettings });
const systemPrefersDark = ref(false);

let initialized = false;
let mediaQueryList: MediaQueryList | null = null;

function loadSettings() {
  if (typeof window === 'undefined') return;

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw) as Partial<AppearanceSettings>;
    settings.value = {
      ...defaultAppearanceSettings,
      ...parsed,
    };
  } catch {
    settings.value = { ...defaultAppearanceSettings };
  }
}

function persistSettings() {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings.value));
}

function applyResolvedTheme(theme: 'light' | 'dark') {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.theme = theme;
}

function handleSystemThemeChange(event: MediaQueryListEvent | MediaQueryList) {
  systemPrefersDark.value = event.matches;
}

function ensureInitialized() {
  if (initialized || typeof window === 'undefined') return;

  initialized = true;
  loadSettings();

  mediaQueryList = window.matchMedia('(prefers-color-scheme: dark)');
  handleSystemThemeChange(mediaQueryList);

  if (typeof mediaQueryList.addEventListener === 'function') {
    mediaQueryList.addEventListener('change', handleSystemThemeChange);
  } else if (typeof mediaQueryList.addListener === 'function') {
    mediaQueryList.addListener(handleSystemThemeChange);
  }
}

const resolvedTheme = computed<'light' | 'dark'>(() => {
  if (settings.value.colorScheme === 'auto') {
    return systemPrefersDark.value ? 'dark' : 'light';
  }
  return settings.value.colorScheme;
});

watch(
  settings,
  () => {
    if (!initialized) return;
    persistSettings();
  },
  { deep: true },
);

watch(
  resolvedTheme,
  (theme) => {
    applyResolvedTheme(theme);
  },
  { immediate: true },
);

function saveAppearanceSettings(next: AppearanceSettings) {
  settings.value = { ...next };
}

const themeStyles = computed<Record<string, string>>(() => {
  if (resolvedTheme.value === 'dark') {
    return {
      '--app-bg': '#0f1117',
      '--frame-bg': '#171a22',
      '--panel-bg': '#131720',
      '--panel-soft': '#1b2130',
      '--panel-muted': '#202738',
      '--panel-contrast': '#0f1320',
      '--border-color': 'rgba(148, 163, 184, 0.18)',
      '--text-primary': '#f8fafc',
      '--text-secondary': '#cbd5e1',
      '--text-muted': '#94a3b8',
      '--text-soft': '#e2e8f0',
      '--chip-bg': 'rgba(255,255,255,0.06)',
      '--chip-hover': 'rgba(255,255,255,0.1)',
    };
  }

  return {
    '--app-bg': '#f4f7fb',
    '--frame-bg': '#ffffff',
    '--panel-bg': '#f8fafc',
    '--panel-soft': '#ffffff',
    '--panel-muted': '#eef2f7',
    '--panel-contrast': '#edf2f9',
    '--border-color': 'rgba(15, 23, 42, 0.08)',
    '--text-primary': '#0f172a',
    '--text-secondary': '#334155',
    '--text-muted': '#64748b',
    '--text-soft': '#1e293b',
    '--chip-bg': 'rgba(148, 163, 184, 0.12)',
    '--chip-hover': 'rgba(148, 163, 184, 0.2)',
  };
});

export function useAppearance() {
  ensureInitialized();

  return {
    appearanceSettings: settings,
    defaultAppearanceSettings,
    resolvedTheme,
    saveAppearanceSettings,
    themeStyles,
  };
}
