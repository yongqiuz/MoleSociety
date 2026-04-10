<script setup lang="ts">
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { 
  Palette, 
  Bell, 
  Shield, 
  User,
  ChevronLeft
} from 'lucide-vue-next';
import { useAppearance } from '../composables/useAppearance';

const router = useRouter();
const route = useRoute();
const { themeStyles } = useAppearance();

const settingsMenu = [
  { id: 'appearance', label: '外观', icon: Palette, path: '/settings/appearance' },
  { id: 'notifications', label: '通知', icon: Bell, path: '/settings/notifications' },
  { id: 'privacy', label: '隐私与安全', icon: Shield, path: '/settings/privacy' },
  { id: 'account', label: '个人资料', icon: User, path: '/settings/account' },
];

function goBack() {
  router.push('/app');
}
</script>

<template>
  <div class="min-h-screen bg-[var(--app-bg)] text-[color:var(--text-primary)] transition-colors duration-300" :style="themeStyles">
    <div class="mx-auto max-w-[1540px] px-4 py-8 lg:px-6">
      <div class="overflow-hidden rounded-[28px] border border-[color:var(--border-color)] bg-[var(--frame-bg)] shadow-[0_20px_60px_rgba(15,23,42,0.08)]">
        <div class="grid min-h-[calc(100vh-140px)] lg:grid-cols-[280px_1fr]">
          <!-- Sidebar -->
          <aside class="border-r border-[color:var(--border-color)] bg-[var(--panel-bg)] px-6 py-8">
            <button
              @click="goBack"
              class="mb-8 flex items-center gap-3 text-base font-medium text-[color:var(--text-secondary)] transition hover:text-emerald-500"
            >
              <ChevronLeft class="w-5 h-5" />
              <span>返回社区</span>
            </button>

            <div class="space-y-1">
              <router-link
                v-for="item in settingsMenu"
                :key="item.id"
                :to="item.path"
                class="flex w-full items-center gap-3 rounded-[1.2rem] px-4 py-3.5 text-left text-base font-medium transition-all hover:translate-x-1"
                :class="route.path === item.path ? 'bg-emerald-600/15 text-emerald-600 shadow-sm' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-hover)]'"
              >
                <component :is="item.icon" class="w-5 h-5 stroke-[1.5]" />
                <span>{{ item.label }}</span>
              </router-link>
            </div>
          </aside>

          <!-- Content -->
          <main class="bg-[var(--frame-bg)] overflow-y-auto no-scrollbar">
            <router-view />
          </main>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.no-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.no-scrollbar::-webkit-scrollbar {
  width: 0;
  height: 0;
}
</style>
