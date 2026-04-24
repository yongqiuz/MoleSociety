<script setup lang="ts">
import { Home, LogIn } from 'lucide-vue-next';
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-[var(--app-bg,#f8fafc)] px-6 text-[color:var(--text-primary,#0f172a)]">
    <section class="w-full max-w-md text-center">
      <div class="text-sm font-semibold uppercase tracking-[0.18em] text-emerald-500">404</div>
      <h1 class="mt-4 text-3xl font-bold">页面不可用</h1>
      <p class="mt-3 text-base leading-7 text-[color:var(--text-secondary,#475569)]">
        后端数据请求没有成功，当前页面不展示本地写死数据。
      </p>
      <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-center">
        <RouterLink
          to="/app"
          class="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-5 py-3 text-sm font-semibold text-white transition hover:bg-emerald-500"
        >
          <Home class="h-4 w-4" />
          返回应用
        </RouterLink>
        <RouterLink
          to="/login"
          class="inline-flex items-center justify-center gap-2 rounded-xl border border-[color:var(--border-color,#cbd5e1)] px-5 py-3 text-sm font-semibold text-[color:var(--text-primary,#0f172a)] transition hover:bg-[var(--panel-soft,#f1f5f9)]"
        >
          <LogIn class="h-4 w-4" />
          重新登录
        </RouterLink>
      </div>
    </section>
  </main>
</template>
