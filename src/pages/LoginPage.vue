<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth';

const router = useRouter();
const route = useRoute();
const { demoAccounts, login } = useAuth();

const selectedAccountId = ref(demoAccounts[0]?.id ?? '');
const password = ref('');
const loading = ref(false);
const errorMessage = ref('');

const selectedAccount = computed(
  () => demoAccounts.find((account) => account.id === selectedAccountId.value) ?? demoAccounts[0] ?? null,
);

async function submitLogin() {
  if (!selectedAccountId.value) {
    errorMessage.value = '请先选择一个演示账号';
    return;
  }

  loading.value = true;
  errorMessage.value = '';

  try {
    login(selectedAccountId.value, password.value);
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/app';
    await router.replace(redirect);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="min-h-screen bg-slate-950 text-slate-100">
    <div class="mx-auto flex min-h-screen max-w-7xl items-center px-6 py-12">
      <div class="grid w-full gap-8 lg:grid-cols-[1.05fr_0.95fr]">
        <section class="rounded-[32px] border border-white/10 bg-white/5 p-8 shadow-[0_30px_90px_rgba(15,23,42,0.35)] backdrop-blur">
          <div class="flex items-center gap-4">
            <div class="flex h-16 w-16 items-center justify-center rounded-3xl bg-violet-600 text-3xl font-bold text-white">
              m
            </div>
            <div>
              <div class="text-3xl font-semibold tracking-tight">MoleSociety</div>
              <div class="mt-1 text-sm text-slate-400">去中心化社交平台登录入口</div>
            </div>
          </div>

          <div class="mt-10 max-w-xl space-y-6">
            <div>
              <div class="text-5xl font-semibold leading-tight">
                登录后进入你的
                <span class="text-violet-400">社交主场</span>
              </div>
              <div class="mt-4 text-lg leading-8 text-slate-300">
                当前先提供前端原型级登录流程，使用系统内置演示账号进入社区。后续可以平滑接到真实后端认证、钱包签名登录或 SSO。
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3">
              <button
                v-for="account in demoAccounts"
                :key="account.id"
                @click="selectedAccountId = account.id"
                class="rounded-3xl border p-4 text-left transition"
                :class="
                  selectedAccountId === account.id
                    ? 'border-violet-400/60 bg-violet-500/15 shadow-[0_0_0_1px_rgba(167,139,250,0.25)]'
                    : 'border-white/10 bg-slate-900/60 hover:border-white/20 hover:bg-slate-900'
                "
              >
                <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                  {{ account.displayName.slice(0, 1) }}
                </div>
                <div class="mt-4 text-lg font-semibold">{{ account.displayName }}</div>
                <div class="mt-1 text-sm text-slate-400">{{ account.handle }}@{{ account.instance }}</div>
              </button>
            </div>
          </div>
        </section>

        <section class="rounded-[32px] border border-white/10 bg-slate-900/90 p-8 shadow-[0_30px_90px_rgba(15,23,42,0.45)]">
          <div class="text-sm font-semibold uppercase tracking-[0.24em] text-violet-400">Sign In</div>
          <div class="mt-4 text-3xl font-semibold">登录 MoleSociety</div>
          <div class="mt-2 text-base leading-7 text-slate-400">
            选择一个账号并输入密码即可进入。当前密码仅作为前端原型校验，后续可以替换成真实登录 API。
          </div>

          <form class="mt-8 space-y-6" @submit.prevent="submitLogin">
            <label class="block">
              <div class="mb-3 text-sm font-medium text-slate-200">账号身份</div>
              <select
                v-model="selectedAccountId"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-4 text-base text-slate-100 outline-none transition focus:border-violet-400/60"
              >
                <option v-for="account in demoAccounts" :key="account.id" :value="account.id">
                  {{ account.displayName }} · {{ account.handle }}@{{ account.instance }}
                </option>
              </select>
            </label>

            <label class="block">
              <div class="mb-3 text-sm font-medium text-slate-200">密码</div>
              <input
                v-model="password"
                type="password"
                placeholder="输入任意非空密码进入原型"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-4 text-base text-slate-100 outline-none placeholder:text-slate-500 transition focus:border-violet-400/60"
              />
            </label>

            <div v-if="selectedAccount" class="rounded-3xl border border-white/10 bg-white/5 p-5">
              <div class="text-sm font-medium text-slate-200">将以该身份进入社区</div>
              <div class="mt-3 text-xl font-semibold">{{ selectedAccount.displayName }}</div>
              <div class="mt-1 text-sm text-slate-400">{{ selectedAccount.handle }}@{{ selectedAccount.instance }}</div>
              <div class="mt-3 text-sm leading-7 text-slate-300">{{ selectedAccount.bio }}</div>
            </div>

            <div v-if="errorMessage" class="rounded-2xl border border-rose-400/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
              {{ errorMessage }}
            </div>

            <button
              :disabled="loading"
              type="submit"
              class="w-full rounded-2xl bg-violet-600 px-6 py-4 text-base font-semibold text-white transition hover:bg-violet-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {{ loading ? '登录中...' : '进入社区' }}
            </button>
          </form>
        </section>
      </div>
    </div>
  </div>
</template>
