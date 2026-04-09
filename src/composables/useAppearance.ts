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

export function useAppearance() {
  ensureInitialized();

  return {
    appearanceSettings: settings,
    defaultAppearanceSettings,
    resolvedTheme,
    saveAppearanceSettings,
  };
}
