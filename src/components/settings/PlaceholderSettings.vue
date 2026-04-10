<template>
  <div class="flex h-full flex-col items-center justify-center px-10 text-center">
    <div class="mb-6 rounded-full bg-emerald-600/10 p-6">
      <component :is="icon" class="h-12 w-12 text-emerald-600" />
    </div>
    <h2 class="mb-2 text-2xl font-bold text-[color:var(--text-primary)]">{{ title }} 正在开发中</h2>
    <p class="max-w-md text-[color:var(--text-muted)]">我们正在努力完善这个功能模块。请稍后再来查看最新的更新。</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { Bell, Shield, User } from 'lucide-vue-next';

const route = useRoute();

const title = computed(() => {
  switch (route.name) {
    case 'settings-notifications': return '通知设置';
    case 'settings-privacy': return '隐私与安全';
    case 'settings-account': return '个人账户';
    default: return '设置项';
  }
});

const icon = computed(() => {
  switch (route.name) {
    case 'settings-notifications': return Bell;
    case 'settings-privacy': return Shield;
    case 'settings-account': return User;
    default: return User;
  }
});
</script>
