<script setup lang="ts">
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth';

const router = useRouter();
const route = useRoute();
const { login } = useAuth();

const loading = ref(false);
const errorMessage = ref('');

async function signInWithWallet() {
  loading.value = true;
  errorMessage.value = '';

  try {
    await login();
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/app';
    await router.replace(redirect);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Wallet sign-in failed.';
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
              <div class="mt-1 text-sm text-slate-400">Wallet signature authentication</div>
            </div>
          </div>

          <div class="mt-10 max-w-xl space-y-6">
            <div>
              <div class="text-5xl font-semibold leading-tight">
                Sign in with your
                <span class="text-violet-400">wallet identity</span>
              </div>
              <div class="mt-4 text-lg leading-8 text-slate-300">
                MoleSociety now uses wallet signatures for authentication. Your wallet proves identity, and the backend
                issues a secure session cookie for posting, messaging, and media upload.
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3">
              <article class="rounded-3xl border border-white/10 bg-slate-900/60 p-5">
                <div class="text-sm uppercase tracking-[0.2em] text-violet-300">1</div>
                <div class="mt-3 text-lg font-semibold">Connect</div>
                <div class="mt-2 text-sm leading-7 text-slate-400">
                  Use MetaMask or any injected EVM wallet to expose your address.
                </div>
              </article>

              <article class="rounded-3xl border border-white/10 bg-slate-900/60 p-5">
                <div class="text-sm uppercase tracking-[0.2em] text-violet-300">2</div>
                <div class="mt-3 text-lg font-semibold">Sign</div>
                <div class="mt-2 text-sm leading-7 text-slate-400">
                  The app asks your wallet to sign a short-lived login challenge.
                </div>
              </article>

              <article class="rounded-3xl border border-white/10 bg-slate-900/60 p-5">
                <div class="text-sm uppercase tracking-[0.2em] text-violet-300">3</div>
                <div class="mt-3 text-lg font-semibold">Enter</div>
                <div class="mt-2 text-sm leading-7 text-slate-400">
                  The backend verifies the signature and creates your MoleSociety session.
                </div>
              </article>
            </div>
          </div>
        </section>

        <section class="rounded-[32px] border border-white/10 bg-slate-900/90 p-8 shadow-[0_30px_90px_rgba(15,23,42,0.45)]">
          <div class="text-sm font-semibold uppercase tracking-[0.24em] text-violet-400">Wallet Login</div>
          <div class="mt-4 text-3xl font-semibold">Authenticate with an EVM wallet</div>
          <div class="mt-2 text-base leading-7 text-slate-400">
            The signed challenge never creates an on-chain transaction. It only proves wallet ownership and opens a
            server session for this app.
          </div>

          <div class="mt-8 rounded-3xl border border-white/10 bg-white/5 p-5">
            <div class="text-sm font-medium text-slate-200">Before you sign in</div>
            <ul class="mt-3 space-y-3 text-sm leading-7 text-slate-300">
              <li>Install MetaMask or another injected EVM wallet.</li>
              <li>Unlock the wallet in this browser.</li>
              <li>Approve the signature request when the wallet prompt appears.</li>
            </ul>
          </div>

          <div v-if="errorMessage" class="mt-6 rounded-2xl border border-rose-400/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
            {{ errorMessage }}
          </div>

          <button
            :disabled="loading"
            type="button"
            class="mt-8 w-full rounded-2xl bg-violet-600 px-6 py-4 text-base font-semibold text-white transition hover:bg-violet-500 disabled:cursor-not-allowed disabled:opacity-60"
            @click="signInWithWallet"
          >
            {{ loading ? 'Waiting for wallet signature...' : 'Connect wallet and sign in' }}
          </button>
        </section>
      </div>
    </div>
  </div>
</template>
