<script setup lang="ts">
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth';

const router = useRouter();
const { logout } = useAuth();

onMounted(async () => {
  await logout();
  await router.replace('/login');
});
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-slate-950 px-6 text-slate-100">
    <div class="rounded-[28px] border border-white/10 bg-white/5 px-8 py-10 text-center shadow-[0_30px_90px_rgba(15,23,42,0.35)]">
      <div class="text-2xl font-semibold">Signing you out</div>
      <div class="mt-3 text-sm text-slate-400">Your secure session is being cleared. You will be redirected to login in a moment.</div>
    </div>
  </div>
</template>
