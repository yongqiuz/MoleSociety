<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ChevronLeft } from 'lucide-vue-next';
import { fetchSocialBootstrap, fetchUser, type SocialPost, type SocialUser } from '../api/socialApi';
import { useAppearance } from '../composables/useAppearance';

const route = useRoute();
const router = useRouter();
const { themeStyles, appearanceSettings } = useAppearance();

const loading = ref(true);
const error = ref('');
const user = ref<SocialUser | null>(null);
const posts = ref<SocialPost[]>([]);

const targetId = computed(() => String(route.params.id || '').trim());

function avatarText(name: string) {
  return name?.slice(0, 1).toUpperCase() || 'U';
}

function formatTimestamp(input: string) {
  if (!input) return '';
  const date = new Date(input);
  if (Number.isNaN(date.getTime())) return input;
  const locale = appearanceSettings.value.language || 'zh-CN';
  const timezone = appearanceSettings.value.timezone || 'UTC';
  try {
    return new Intl.DateTimeFormat(locale, {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
      timeZone: timezone,
    }).format(date);
  } catch {
    return date.toLocaleString(locale);
  }
}

async function loadProfile() {
  const id = targetId.value;
  if (!id) {
    error.value = '用户不存在';
    loading.value = false;
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    const [profile, bootstrap] = await Promise.all([
      fetchUser(id),
      fetchSocialBootstrap(80),
    ]);
    user.value = profile;
    posts.value = bootstrap.feed.filter((post) => post.authorId === id);
  } catch {
    error.value = '加载主页失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}

function goBack() {
  router.back();
}

onMounted(() => {
  void loadProfile();
});

watch(
  () => route.params.id,
  () => {
    void loadProfile();
  },
);
</script>

<template>
  <div class="min-h-screen bg-[var(--app-bg)] text-[color:var(--text-primary)] transition-colors duration-300" :style="themeStyles">
    <div class="mx-auto max-w-4xl px-4 py-6 lg:px-6">
      <button
        @click="goBack"
        class="mb-5 inline-flex items-center gap-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-2 text-sm text-[color:var(--text-secondary)] transition hover:border-emerald-500/40 hover:text-emerald-500"
      >
        <ChevronLeft class="h-4 w-4" />
        返回
      </button>

      <div v-if="loading" class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-6 py-16 text-center text-[color:var(--text-muted)]">
        正在加载用户主页...
      </div>

      <div v-else-if="error" class="rounded-3xl border border-rose-500/20 bg-rose-500/10 px-6 py-12 text-center text-rose-300">
        {{ error }}
      </div>

      <div v-else-if="user" class="space-y-6">
        <section class="overflow-hidden rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)]">
          <div class="h-36 bg-gradient-to-r from-emerald-300/40 via-cyan-300/30 to-blue-300/40">
            <img v-if="user.backgroundUrl" :src="user.backgroundUrl" :alt="`${user.displayName} 背景图`" class="h-full w-full object-cover" />
          </div>
          <div class="px-6 pb-6">
            <div class="-mt-10 flex flex-wrap items-end justify-between gap-4">
              <div class="flex items-end gap-4">
                <div class="flex h-20 w-20 items-center justify-center overflow-hidden rounded-2xl border-4 border-[var(--frame-bg)] bg-gradient-to-br from-lime-200 to-cyan-200 text-2xl font-bold text-slate-900">
                  <img v-if="user.avatarUrl" :src="user.avatarUrl" :alt="user.displayName" class="h-full w-full object-cover" />
                  <template v-else>{{ avatarText(user.displayName) }}</template>
                </div>
                <div>
                  <div class="text-2xl font-semibold text-[color:var(--text-primary)]">{{ user.displayName }}</div>
                  <div class="text-sm text-[color:var(--text-muted)]">{{ user.handle }}@{{ user.instance }}</div>
                </div>
              </div>
            </div>

            <p class="mt-4 whitespace-pre-wrap text-sm leading-7 text-[color:var(--text-secondary)]">{{ user.bio || '这个用户还没有填写简介。' }}</p>

            <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-[color:var(--text-secondary)]">
              <span><strong class="text-[color:var(--text-primary)]">{{ user.followers }}</strong> 关注者</span>
              <span><strong class="text-[color:var(--text-primary)]">{{ user.following }}</strong> 关注中</span>
              <span class="text-[color:var(--text-muted)]">加入于 {{ formatTimestamp(user.createdAt) }}</span>
            </div>
          </div>
        </section>

        <section class="overflow-hidden rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)]">
          <div class="border-b border-[color:var(--border-color)] px-6 py-4 text-base font-semibold text-[color:var(--text-primary)]">最近帖子</div>
          <div v-if="posts.length === 0" class="px-6 py-12 text-center text-sm text-[color:var(--text-muted)]">
            暂无公开帖子
          </div>
          <article v-for="post in posts" :key="post.id" class="border-b border-[color:var(--border-color)] px-6 py-5 last:border-b-0">
            <div class="text-xs text-[color:var(--text-muted)]">{{ formatTimestamp(post.createdAt) }}</div>
            <div class="mt-2 whitespace-pre-wrap text-[15px] leading-7 text-[color:var(--text-secondary)]">{{ post.content }}</div>
            <div v-if="post.tags.length" class="mt-3 flex flex-wrap gap-2">
              <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-emerald-500/10 px-3 py-1 text-xs text-emerald-300">
                #{{ tag }}
              </span>
            </div>
          </article>
        </section>
      </div>
    </div>
  </div>
</template>
