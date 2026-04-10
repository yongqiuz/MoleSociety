<script setup lang="ts">
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth';
import { registerAccount } from '../api/authApi';
import { Mail, Lock, Eye, EyeOff, Wallet, ArrowRight, Hexagon, Sparkles, User, UserPlus } from 'lucide-vue-next';

const router = useRouter();
const route = useRoute();
const { login, loginWithPassword } = useAuth();

const activeTab = ref<'signin' | 'register' | 'wallet'>('signin');
const loading = ref(false);
const errorMessage = ref('');
const successMessage = ref('');
const showPassword = ref(false);
const showRegPassword = ref(false);

const loginForm = ref({
  identifier: '',
  password: '',
});

const registerForm = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
});

function switchTab(tab: 'signin' | 'register' | 'wallet') {
  activeTab.value = tab;
  errorMessage.value = '';
  successMessage.value = '';
}

function redirectAfterAuth() {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/app';
  router.replace(redirect);
}

async function handleSignIn() {
  if (!loginForm.value.identifier || !loginForm.value.password) {
    errorMessage.value = '请填写用户名和密码';
    return;
  }
  loading.value = true;
  errorMessage.value = '';
  try {
    await loginWithPassword(loginForm.value.identifier, loginForm.value.password);
    redirectAfterAuth();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录失败';
  } finally {
    loading.value = false;
  }
}

async function handleRegister() {
  const { username, email, password, confirmPassword } = registerForm.value;
  if (!username || !password) {
    errorMessage.value = '请填写用户名和密码';
    return;
  }
  if (password.length < 6) {
    errorMessage.value = '密码长度至少 6 位';
    return;
  }
  if (password !== confirmPassword) {
    errorMessage.value = '两次输入的密码不一致';
    return;
  }
  loading.value = true;
  errorMessage.value = '';
  try {
    await registerAccount({ username, email: email || undefined, password });
    // After successful registration, auto-login and redirect
    await loginWithPassword(username, password);
    redirectAfterAuth();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '注册失败';
  } finally {
    loading.value = false;
  }
}

async function signInWithWallet() {
  loading.value = true;
  errorMessage.value = '';
  try {
    await login();
    redirectAfterAuth();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '钱包登录失败';
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="relative min-h-screen bg-slate-950 text-slate-100 overflow-hidden font-sans">
    <!-- Ambient Background -->
    <div class="absolute inset-0 pointer-events-none">
      <div class="absolute -top-[30%] -left-[10%] w-[70vw] h-[70vw] rounded-full bg-emerald-900/20 blur-[120px] mix-blend-screen opacity-50 animate-pulse-slow"></div>
      <div class="absolute top-[40%] -right-[20%] w-[60vw] h-[60vw] rounded-full bg-teal-900/20 blur-[120px] mix-blend-screen opacity-50"></div>
    </div>

    <!-- Header -->
    <header class="absolute top-0 left-0 right-0 p-8 flex justify-between items-center z-10">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-400 to-teal-600 text-white shadow-lg shadow-emerald-500/20">
          <Hexagon class="w-6 h-6 fill-emerald-100 text-emerald-100" />
        </div>
        <span class="text-xl font-bold tracking-tight text-white drop-shadow-sm">MoleSociety</span>
      </div>
    </header>

    <div class="relative z-10 mx-auto flex min-h-screen max-w-6xl items-center px-4 py-20 lg:px-8">
      <div class="grid w-full gap-16 lg:grid-cols-2 items-center">
        
        <!-- Left: Value Proposition -->
        <section class="hidden lg:block lg:max-w-lg">
          <div class="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-4 py-1.5 text-sm font-semibold text-emerald-300 tracking-wide mb-8">
            <Sparkles class="w-4 h-4" /> Next-Gen Social
          </div>
          <h1 class="text-5xl font-bold leading-[1.1] tracking-tight text-white m-0">
            Own your identity. <br>
            <span class="text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-teal-400">Secure your network.</span>
          </h1>
          <p class="mt-6 text-lg leading-relaxed text-slate-300">
            MoleSociety 是去中心化社交的下一步。绑定钱包验证身份，使用账号密码快捷登录日常操作。高敏感操作依然由钱包签名保护。
          </p>
          
          <div class="mt-12 space-y-6">
            <div class="flex gap-4">
              <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white/5 border border-white/10 text-emerald-400">
                <Wallet class="w-6 h-6" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-slate-100">钱包即根身份</h3>
                <p class="mt-1 text-slate-400">链上钱包不可撤销地证明你是谁，日常操作无需反复弹窗。</p>
              </div>
            </div>
            <div class="flex gap-4">
              <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white/5 border border-white/10 text-emerald-400">
                <Lock class="w-6 h-6" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-slate-100">账号密码快捷登录</h3>
                <p class="mt-1 text-slate-400">注册后使用密码即可浏览、发帖、聊天，无缝的 Web2 体验。</p>
              </div>
            </div>
          </div>
        </section>

        <!-- Right: Auth Card -->
        <section class="mx-auto w-full max-w-md rounded-[2.5rem] border border-white/10 bg-slate-900/60 p-8 shadow-[0_30px_90px_rgba(4,47,46,0.35)] backdrop-blur-2xl">
          
          <!-- 3-Tab Navigation -->
          <div class="mb-8 flex rounded-[1.25rem] border border-white/5 bg-slate-950/50 p-1.5">
            <button 
              class="flex-1 rounded-2xl py-2.5 text-sm font-semibold transition-all duration-300 flex items-center justify-center gap-1.5"
              :class="activeTab === 'signin' ? 'bg-slate-800 text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-white/5'"
              @click="switchTab('signin')"
            >
              <User class="w-4 h-4" /> 登录
            </button>
            <button 
              class="flex-1 rounded-2xl py-2.5 text-sm font-semibold transition-all duration-300 flex items-center justify-center gap-1.5"
              :class="activeTab === 'register' ? 'bg-slate-800 text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-white/5'"
              @click="switchTab('register')"
            >
              <UserPlus class="w-4 h-4" /> 注册
            </button>
            <button 
              class="flex-1 rounded-2xl py-2.5 text-sm font-semibold transition-all duration-300 flex items-center justify-center gap-1.5"
              :class="activeTab === 'wallet' ? 'bg-slate-800 text-white shadow' : 'text-slate-400 hover:text-slate-200 hover:bg-white/5'"
              @click="switchTab('wallet')"
            >
              <Wallet class="w-4 h-4" /> 钱包
            </button>
          </div>

          <!-- Error / Success Messages -->
          <div v-if="errorMessage" class="mb-6 rounded-2xl border border-rose-400/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200 flex items-center gap-2">
            <div class="w-1.5 h-1.5 rounded-full bg-rose-400 shrink-0"></div>
            {{ errorMessage }}
          </div>
          <div v-if="successMessage" class="mb-6 rounded-2xl border border-emerald-400/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200 flex items-center gap-2">
            <div class="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0"></div>
            {{ successMessage }}
          </div>

          <!-- Tab: Sign In -->
          <div v-show="activeTab === 'signin'">
            <h2 class="text-2xl font-semibold mb-6">欢迎回来</h2>
            
            <form @submit.prevent="handleSignIn" class="space-y-5">
              <div class="space-y-1.5">
                <label class="text-sm font-medium text-slate-300 ml-1">用户名 / 邮箱</label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Mail class="h-5 w-5 text-slate-500" />
                  </div>
                  <input 
                    v-model="loginForm.identifier"
                    type="text" 
                    class="w-full rounded-2xl border border-white/10 bg-slate-950/60 py-3.5 pl-12 pr-4 text-slate-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:bg-slate-900/80 focus:outline-none focus:ring-1 focus:ring-emerald-500/50 transition-all font-medium"
                    placeholder="输入用户名或邮箱"
                  >
                </div>
              </div>

              <div class="space-y-1.5">
                <div class="flex items-center justify-between ml-1">
                  <label class="text-sm font-medium text-slate-300">密码</label>
                </div>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Lock class="h-5 w-5 text-slate-500" />
                  </div>
                  <input 
                    v-model="loginForm.password"
                    :type="showPassword ? 'text' : 'password'" 
                    class="w-full rounded-2xl border border-white/10 bg-slate-950/60 py-3.5 pl-12 pr-12 text-slate-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:bg-slate-900/80 focus:outline-none focus:ring-1 focus:ring-emerald-500/50 transition-all font-medium"
                    placeholder="请输入密码"
                  >
                  <button type="button" @click="showPassword = !showPassword" class="absolute inset-y-0 right-0 pr-4 flex items-center text-slate-500 hover:text-slate-300 transition-colors">
                    <EyeOff v-if="!showPassword" class="h-5 w-5" />
                    <Eye v-else class="h-5 w-5" />
                  </button>
                </div>
              </div>

              <button 
                type="submit"
                :disabled="loading"
                class="mt-2 group relative w-full flex justify-center items-center gap-2 rounded-2xl bg-emerald-600 px-6 py-4 text-base font-semibold text-white transition-all hover:bg-emerald-500 hover:shadow-lg hover:shadow-emerald-500/25 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-70 disabled:transform-none"
              >
                {{ loading ? '登录中...' : '登录' }}
                <ArrowRight v-if="!loading" class="w-5 h-5 transition-transform group-hover:translate-x-1" />
              </button>
            </form>

            <div class="mt-8 text-center text-sm text-slate-400">
              还没有账号？ 
              <button @click="switchTab('register')" class="font-semibold text-white hover:text-emerald-400 transition-colors">立即注册</button>
            </div>
          </div>

          <!-- Tab: Register -->
          <div v-show="activeTab === 'register'">
            <h2 class="text-2xl font-semibold mb-6">创建账号</h2>
            
            <form @submit.prevent="handleRegister" class="space-y-5">
              <div class="space-y-1.5">
                <label class="text-sm font-medium text-slate-300 ml-1">用户名</label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <User class="h-5 w-5 text-slate-500" />
                  </div>
                  <input 
                    v-model="registerForm.username"
                    type="text" 
                    class="w-full rounded-2xl border border-white/10 bg-slate-950/60 py-3.5 pl-12 pr-4 text-slate-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:bg-slate-900/80 focus:outline-none focus:ring-1 focus:ring-emerald-500/50 transition-all font-medium"
                    placeholder="选择一个用户名"
                  >
                </div>
              </div>

              <div class="space-y-1.5">
                <label class="text-sm font-medium text-slate-300 ml-1">邮箱 <span class="text-slate-500">(可选)</span></label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Mail class="h-5 w-5 text-slate-500" />
                  </div>
                  <input 
                    v-model="registerForm.email"
                    type="email" 
                    class="w-full rounded-2xl border border-white/10 bg-slate-950/60 py-3.5 pl-12 pr-4 text-slate-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:bg-slate-900/80 focus:outline-none focus:ring-1 focus:ring-emerald-500/50 transition-all font-medium"
                    placeholder="可选，用于找回密码"
                  >
                </div>
              </div>

              <div class="space-y-1.5">
                <label class="text-sm font-medium text-slate-300 ml-1">密码</label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Lock class="h-5 w-5 text-slate-500" />
                  </div>
                  <input 
                    v-model="registerForm.password"
                    :type="showRegPassword ? 'text' : 'password'" 
                    class="w-full rounded-2xl border border-white/10 bg-slate-950/60 py-3.5 pl-12 pr-12 text-slate-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:bg-slate-900/80 focus:outline-none focus:ring-1 focus:ring-emerald-500/50 transition-all font-medium"
                    placeholder="至少 6 位"
                  >
                  <button type="button" @click="showRegPassword = !showRegPassword" class="absolute inset-y-0 right-0 pr-4 flex items-center text-slate-500 hover:text-slate-300 transition-colors">
                    <EyeOff v-if="!showRegPassword" class="h-5 w-5" />
                    <Eye v-else class="h-5 w-5" />
                  </button>
                </div>
              </div>

              <div class="space-y-1.5">
                <label class="text-sm font-medium text-slate-300 ml-1">确认密码</label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Lock class="h-5 w-5 text-slate-500" />
                  </div>
                  <input 
                    v-model="registerForm.confirmPassword"
                    type="password" 
                    class="w-full rounded-2xl border border-white/10 bg-slate-950/60 py-3.5 pl-12 pr-4 text-slate-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:bg-slate-900/80 focus:outline-none focus:ring-1 focus:ring-emerald-500/50 transition-all font-medium"
                    placeholder="再次输入密码"
                  >
                </div>
              </div>

              <button 
                type="submit"
                :disabled="loading"
                class="mt-2 group relative w-full flex justify-center items-center gap-2 rounded-2xl bg-emerald-600 px-6 py-4 text-base font-semibold text-white transition-all hover:bg-emerald-500 hover:shadow-lg hover:shadow-emerald-500/25 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-70 disabled:transform-none"
              >
                {{ loading ? '注册中...' : '注册并登录' }}
                <ArrowRight v-if="!loading" class="w-5 h-5 transition-transform group-hover:translate-x-1" />
              </button>
            </form>

            <div class="mt-8 text-center text-sm text-slate-400">
              已有账号？ 
              <button @click="switchTab('signin')" class="font-semibold text-white hover:text-emerald-400 transition-colors">返回登录</button>
            </div>
          </div>

          <!-- Tab: Wallet -->
          <div v-show="activeTab === 'wallet'">
            <h2 class="text-2xl font-semibold mb-2">Web3 身份</h2>
            <p class="text-sm text-slate-400 leading-relaxed mb-8">
              使用 EVM 钱包签名挑战来证明你的身份。此过程不会产生链上交易。
            </p>

            <div class="space-y-4">
              <div class="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 p-4 flex gap-3">
                <div class="mt-0.5">
                  <div class="w-5 h-5 rounded-full bg-emerald-500/20 flex items-center justify-center">
                    <div class="w-2 h-2 rounded-full bg-emerald-400"></div>
                  </div>
                </div>
                <div class="text-sm font-medium text-emerald-200">
                  请确保 MetaMask 或其他 EVM 钱包已安装并解锁。
                </div>
              </div>
            </div>

            <button
              :disabled="loading"
              type="button"
              class="mt-10 group relative w-full flex justify-center items-center gap-3 rounded-2xl bg-slate-100 px-6 py-4 text-base font-semibold text-slate-900 transition-all hover:bg-white hover:shadow-lg hover:shadow-white/10 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-70 disabled:transform-none"
              @click="signInWithWallet"
            >
              <Wallet v-if="!loading" class="w-5 h-5 text-slate-700" />
              {{ loading ? '等待钱包签名...' : '连接钱包登录' }}
            </button>

            <div class="mt-8 text-center text-sm text-slate-400">
              更便捷的方式？ 
              <button @click="switchTab('register')" class="font-semibold text-white hover:text-emerald-400 transition-colors">注册账号密码</button>
            </div>
          </div>

        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes pulse-slow {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.6; }
}
.animate-pulse-slow {
  animation: pulse-slow 8s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
</style>
