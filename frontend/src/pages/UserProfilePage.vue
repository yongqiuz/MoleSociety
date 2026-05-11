<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Bookmark, ChevronLeft, Heart, MessageCircle, MoreHorizontal, Repeat } from 'lucide-vue-next';
import {
  fetchSocialBootstrap,
  fetchUser,
  fetchUserFollowers,
  fetchUserFollowing,
  followUser,
  unfollowUser,
  type SocialPost,
  type SocialUser,
} from '../api/socialApi';
import { useAppearance } from '../composables/useAppearance';
import { useAuth } from '../composables/useAuth';

const route = useRoute();
const router = useRouter();
const { themeStyles, appearanceSettings } = useAppearance();
const { currentUser } = useAuth();

const loading = ref(true);
const error = ref('');
const user = ref<SocialUser | null>(null);
const posts = ref<SocialPost[]>([]);
const following = ref(false);
const followLoading = ref(false);
const FOLLOW_STATE_PREFIX = 'mole-follow-state';
const LIKE_STORAGE_PREFIX = 'mole-liked-posts';
const BOOKMARK_STORAGE_PREFIX = 'mole-bookmarked-posts';
const likedPosts = ref<Record<string, boolean>>({});
const bookmarkedPosts = ref<Record<string, boolean>>({});
const followActionLoading = ref<Record<string, boolean>>({});
const relationView = ref<'posts' | 'followers' | 'following'>('posts');
const relationUsers = ref<SocialUser[]>([]);
const relationLoading = ref(false);
const relationError = ref('');
const openPostMenuId = ref('');

const targetKey = computed(() => decodeURIComponent(String(route.params.id || '').trim()));
const isSelfProfile = computed(() => {
  if (!currentUser.value || !user.value) return false;
  if (currentUser.value.id && user.value.id && currentUser.value.id === user.value.id) return true;
  const myHandle = String(currentUser.value.handle || '').replace(/^@/, '').trim().toLowerCase();
  const profileHandle = String(user.value.handle || '').replace(/^@/, '').trim().toLowerCase();
  return Boolean(myHandle && profileHandle && myHandle === profileHandle);
});

function avatarText(name: string) {
  return name?.slice(0, 1).toUpperCase() || 'U';
}

function formatHandleInstance(handle: string, instance: string) {
  const normalizedInstance = String(instance || '').trim();
  if (normalizedInstance === '摩尔1号') {
    return `@${normalizedInstance}`;
  }
  return `${handle}@${normalizedInstance}`;
}

function profileLabel(userInfo: SocialUser | null) {
  if (!userInfo) return '';
  return userInfo.bio || 'MoleSociety member';
}

function relationTitle() {
  if (!user.value) return '';
  return relationView.value === 'followers' ? `${user.value.displayName} 的关注者` : `${user.value.displayName} 的关注中`;
}

function relationCount() {
  if (!user.value) return 0;
  return relationView.value === 'followers' ? user.value.followers : user.value.following;
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
  const key = targetKey.value;
  if (!key) {
    error.value = '用户不存在';
    loading.value = false;
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    const bootstrap = await fetchSocialBootstrap(80);
    const normalizedKey = key.replace(/^@/, '');
    const matchedUser = bootstrap.users.find((item) => {
      if (item.id === normalizedKey) return true;
      const handle = String(item.handle || '').replace(/^@/, '');
      return handle === normalizedKey;
    });
    const targetUserId = matchedUser?.id || normalizedKey;
    const profile = await fetchUser(targetUserId);
    user.value = profile;
    posts.value = bootstrap.feed.filter((post) => post.authorId === profile.id);
    loadFollowState();
    const canonicalHandle = String(profile.handle || '').replace(/^@/, '');
    if (canonicalHandle && normalizedKey !== canonicalHandle) {
      void router.replace(`/profile/${encodeURIComponent(canonicalHandle)}`);
    }
  } catch {
    error.value = '加载主页失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}

function goBack() {
  router.back();
}

function goEditProfile() {
  void router.push('/settings/account');
}

function goStartConversation() {
  if (!user.value?.id || isSelfProfile.value) return;
  void router.push({ path: '/app', query: { messageUser: user.value.id } });
}

function goToUserProfile(target: SocialUser) {
  const handle = String(target?.handle || '').replace(/^@/, '').trim();
  const key = handle || target?.id;
  if (!key) return;
  void router.push(`/profile/${encodeURIComponent(key)}`);
}

function followStorageKey() {
  const me = currentUser.value?.id || '';
  const target = user.value?.id || '';
  return me && target ? `${FOLLOW_STATE_PREFIX}:${me}:${target}` : '';
}

function followStorageKeyFor(targetId: string) {
  const me = currentUser.value?.id || '';
  return me && targetId ? `${FOLLOW_STATE_PREFIX}:${me}:${targetId}` : '';
}

function loadFollowState() {
  if (typeof window === 'undefined') return;
  const key = followStorageKey();
  if (!key) return;
  following.value = window.localStorage.getItem(key) === '1';
}

function persistFollowState() {
  if (typeof window === 'undefined') return;
  const key = followStorageKey();
  if (!key) return;
  window.localStorage.setItem(key, following.value ? '1' : '0');
}

function likeStorageKey() {
  const userId = currentUser.value?.id;
  return userId ? `${LIKE_STORAGE_PREFIX}:${userId}` : '';
}

function bookmarkStorageKey() {
  const userId = currentUser.value?.id;
  return userId ? `${BOOKMARK_STORAGE_PREFIX}:${userId}` : '';
}

function loadInteractionState() {
  if (typeof window === 'undefined') return;
  const likeKey = likeStorageKey();
  const bookmarkKey = bookmarkStorageKey();
  likedPosts.value = {};
  bookmarkedPosts.value = {};
  if (likeKey) {
    try {
      likedPosts.value = JSON.parse(window.localStorage.getItem(likeKey) || '{}');
    } catch {
      likedPosts.value = {};
    }
  }
  if (bookmarkKey) {
    try {
      bookmarkedPosts.value = JSON.parse(window.localStorage.getItem(bookmarkKey) || '{}');
    } catch {
      bookmarkedPosts.value = {};
    }
  }
}

function persistInteractionState() {
  if (typeof window === 'undefined') return;
  const likeKey = likeStorageKey();
  const bookmarkKey = bookmarkStorageKey();
  if (likeKey) {
    window.localStorage.setItem(likeKey, JSON.stringify(likedPosts.value));
  }
  if (bookmarkKey) {
    window.localStorage.setItem(bookmarkKey, JSON.stringify(bookmarkedPosts.value));
  }
}

function toggleLike(postId: string) {
  likedPosts.value = { ...likedPosts.value, [postId]: !likedPosts.value[postId] };
  persistInteractionState();
}

function toggleBookmark(postId: string) {
  bookmarkedPosts.value = { ...bookmarkedPosts.value, [postId]: !bookmarkedPosts.value[postId] };
  persistInteractionState();
}

function openPostDetail(postId: string) {
  if (!postId) return;
  void router.push({ path: '/app', query: { post: postId } });
}

function sharePost(postId: string) {
  if (!postId) return;
  void router.push({ path: '/app', query: { sharePost: postId } });
}

function postPublicUrl(postId: string) {
  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  return `${origin}/app?post=${encodeURIComponent(postId)}`;
}

async function copyPostLink(postId: string) {
  if (!postId || typeof window === 'undefined') return;
  const url = postPublicUrl(postId);
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(url);
      return;
    }
  } catch {
    // ignore and fallback
  }
  window.prompt('复制帖子链接', url);
}

async function handlePostMenuAction(action: 'share' | 'copy', postId: string) {
  openPostMenuId.value = '';
  if (!postId) return;
  if (action === 'share') {
    sharePost(postId);
    return;
  }
  await copyPostLink(postId);
}

async function toggleFollow() {
  if (!user.value || isSelfProfile.value || followLoading.value) return;
  followLoading.value = true;
  const next = !following.value;
  try {
    if (next) {
      await followUser(user.value.id);
      following.value = true;
      user.value = { ...user.value, followers: user.value.followers + 1 };
    } else {
      await unfollowUser(user.value.id);
      following.value = false;
      user.value = { ...user.value, followers: Math.max(0, user.value.followers - 1) };
    }
    persistFollowState();
  } finally {
    followLoading.value = false;
  }
}

function isFollowingUser(targetId: string) {
  if (!targetId) return false;
  const key = followStorageKeyFor(targetId);
  if (!key || typeof window === 'undefined') return false;
  return window.localStorage.getItem(key) === '1';
}

async function toggleFollowUser(target: SocialUser) {
  if (!target?.id || followActionLoading.value[target.id]) return;
  followActionLoading.value = { ...followActionLoading.value, [target.id]: true };
  const next = !isFollowingUser(target.id);
  try {
    if (next) {
      await followUser(target.id);
    } else {
      await unfollowUser(target.id);
    }
    const key = followStorageKeyFor(target.id);
    if (key && typeof window !== 'undefined') {
      window.localStorage.setItem(key, next ? '1' : '0');
    }
  } finally {
    followActionLoading.value = { ...followActionLoading.value, [target.id]: false };
  }
}

async function loadRelationUsers(type: 'followers' | 'following') {
  if (!user.value?.id) return;
  relationLoading.value = true;
  relationError.value = '';
  try {
    const list = type === 'followers'
      ? await fetchUserFollowers(user.value.id, 200)
      : await fetchUserFollowing(user.value.id, 200);
    relationUsers.value = list;
  } catch {
    relationError.value = '加载列表失败，请稍后重试';
    relationUsers.value = [];
  } finally {
    relationLoading.value = false;
  }
}

function openRelationView(type: 'followers' | 'following') {
  relationView.value = type;
  void loadRelationUsers(type);
}

function closeRelationView() {
  relationView.value = 'posts';
}

onMounted(() => {
  loadInteractionState();
  void loadProfile();
});

watch(
  () => route.params.id,
  () => {
    openPostMenuId.value = '';
    void loadProfile();
  },
);

watch(
  () => currentUser.value?.id,
  () => {
    loadInteractionState();
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
                  <div class="text-sm text-[color:var(--text-muted)]">{{ formatHandleInstance(user.handle, user.instance) }}</div>
                </div>
              </div>
              <button
                v-if="isSelfProfile"
                @click="goEditProfile"
                class="rounded-xl border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold text-emerald-400 transition hover:bg-emerald-500/20"
              >
                编辑资料
              </button>
              <div v-else class="flex items-center gap-2">
                <button
                  @click="toggleFollow"
                  :disabled="followLoading"
                  class="rounded-xl border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold text-emerald-400 transition hover:bg-emerald-500/20 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {{ followLoading ? '处理中...' : following ? '取消关注' : '关注' }}
                </button>
                <button
                  @click="goStartConversation"
                  class="rounded-xl border border-cyan-500/40 bg-cyan-500/10 px-4 py-2 text-sm font-semibold text-cyan-300 transition hover:bg-cyan-500/20"
                >
                  发消息
                </button>
              </div>
            </div>

            <p class="mt-4 whitespace-pre-wrap text-sm leading-7 text-[color:var(--text-secondary)]">{{ user.bio || '这个用户还没有填写简介。' }}</p>

            <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-[color:var(--text-secondary)]">
              <button
                @click="openRelationView('followers')"
                class="rounded-lg px-1 py-0.5 transition hover:bg-[var(--panel-soft)]"
              >
                <strong class="text-[color:var(--text-primary)]">{{ user.followers }}</strong> 关注者
              </button>
              <button
                @click="openRelationView('following')"
                class="rounded-lg px-1 py-0.5 transition hover:bg-[var(--panel-soft)]"
              >
                <strong class="text-[color:var(--text-primary)]">{{ user.following }}</strong> 关注中
              </button>
              <span class="text-[color:var(--text-muted)]">加入于 {{ formatTimestamp(user.createdAt) }}</span>
            </div>
          </div>
        </section>

        <section v-if="relationView === 'posts'" class="overflow-visible rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)]">
          <div class="border-b border-[color:var(--border-color)] px-6 py-4 text-base font-semibold text-[color:var(--text-primary)]">最近帖子</div>
          <div v-if="posts.length === 0" class="px-6 py-12 text-center text-sm text-[color:var(--text-muted)]">
            暂无公开帖子
          </div>
          <article v-for="post in posts" :key="post.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
            <div class="flex gap-3">
              <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                <img v-if="user.avatarUrl" :src="user.avatarUrl" :alt="user.displayName" class="h-full w-full object-cover" />
                <template v-else>{{ avatarText(user.displayName) }}</template>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                  <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ user.displayName }}</div>
                  <div class="text-sm text-[color:var(--text-secondary)]">{{ formatHandleInstance(user.handle, user.instance) }}</div>
                  <div class="text-xs text-[color:var(--text-muted)]">{{ formatTimestamp(post.createdAt) }}</div>
                </div>
                <div class="mt-0.5 text-xs text-[color:var(--text-muted)]">{{ profileLabel(user) }}</div>
                <div class="mt-2 whitespace-pre-wrap text-[15px] leading-7 text-[color:var(--text-secondary)]">{{ post.content }}</div>
                <div
                  v-if="Array.isArray(post.media) && post.media.length > 0"
                  class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]"
                >
                  <img
                    :src="post.media[0].url"
                    :alt="post.media[0].name || '帖子图片'"
                    class="max-h-[70vh] w-full object-contain bg-[var(--panel-contrast)]"
                  />
                </div>
                <div v-if="post.tags.length" class="mt-3 flex flex-wrap gap-2">
                  <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-emerald-500/10 px-3 py-1 text-xs text-emerald-300">
                    #{{ tag }}
                  </span>
                </div>
                <div class="mt-5 flex flex-wrap items-center gap-3 text-sm">
                  <button
                    @click="openPostDetail(post.id)"
                    class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                  >
                    <MessageCircle class="mr-1.5 h-[18px] w-[18px]" /> {{ post.replies || '' }}
                  </button>
                  <button
                    @click="sharePost(post.id)"
                    class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:border-emerald-300/30 hover:text-emerald-200"
                  >
                    <Repeat class="mr-1.5 h-[18px] w-[18px]" /> 转发
                  </button>
                  <button
                    @click="toggleLike(post.id)"
                    class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                    :class="likedPosts[post.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                  >
                    <Heart :class="{ 'fill-current': likedPosts[post.id] }" class="mr-1.5 h-[18px] w-[18px]" /> {{ post.likes + (likedPosts[post.id] ? 1 : 0) || '' }}
                  </button>
                  <button
                    @click="toggleBookmark(post.id)"
                    class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                    :class="bookmarkedPosts[post.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                  >
                    <Bookmark :class="{ 'fill-current': bookmarkedPosts[post.id] }" class="mr-1.5 h-[18px] w-[18px]" />
                  </button>
                  <div class="relative ml-auto">
                    <button
                      @click="openPostMenuId = openPostMenuId === post.id ? '' : post.id"
                      class="inline-flex items-center rounded-lg px-2 py-1.5 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                    >
                      <MoreHorizontal class="h-5 w-5" />
                    </button>
                    <div
                      v-if="openPostMenuId === post.id"
                      class="absolute right-0 top-full z-30 mt-2 w-44 overflow-hidden rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] text-sm shadow-[0_10px_40px_rgba(0,0,0,0.5)]"
                    >
                      <button
                        @click="handlePostMenuAction('share', post.id)"
                        class="w-full px-4 py-2.5 text-left text-[color:var(--text-primary)] hover:bg-[var(--panel-soft)]"
                      >
                        分享
                      </button>
                      <button
                        @click="handlePostMenuAction('copy', post.id)"
                        class="w-full px-4 py-2.5 text-left text-[color:var(--text-primary)] hover:bg-[var(--panel-soft)]"
                      >
                        复制链接
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </article>
        </section>

        <section v-else class="overflow-hidden rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)]">
          <div class="border-b border-[color:var(--border-color)] px-5 py-4">
            <button
              @click="closeRelationView"
              class="mb-3 inline-flex items-center gap-2 text-sm font-medium text-emerald-500 transition hover:text-emerald-400"
            >
              <ChevronLeft class="h-4 w-4" />
              返回
            </button>
            <div class="text-[26px] font-semibold text-[color:var(--text-primary)]">{{ relationTitle() }}</div>
            <div class="mt-1 text-sm text-[color:var(--text-muted)]">{{ relationCount() }} 个账号</div>
          </div>

          <div v-if="relationLoading" class="px-6 py-12 text-center text-sm text-[color:var(--text-muted)]">
            正在加载列表...
          </div>

          <div v-else-if="relationError" class="px-6 py-12 text-center text-sm text-rose-300">
            {{ relationError }}
          </div>

          <div v-else-if="relationUsers.length === 0" class="px-6 py-12 text-center text-sm text-[color:var(--text-muted)]">
            暂无可展示账号
          </div>

          <article v-for="item in relationUsers" :key="item.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex items-start gap-3">
                <button
                  @click="goToUserProfile(item)"
                  class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                >
                  <img v-if="item.avatarUrl" :src="item.avatarUrl" :alt="item.displayName" class="h-full w-full object-cover" />
                  <template v-else>{{ avatarText(item.displayName) }}</template>
                </button>
                <div class="min-w-0">
                  <button
                    @click="goToUserProfile(item)"
                    class="text-left text-lg font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                  >
                    {{ item.displayName }}
                  </button>
                  <button
                    @click="goToUserProfile(item)"
                    class="block text-left text-sm text-[color:var(--text-muted)] transition hover:text-emerald-400"
                  >
                    {{ formatHandleInstance(item.handle, item.instance) }}
                  </button>
                  <div class="mt-3 grid grid-cols-3 gap-5 text-sm">
                    <div>
                      <div class="text-[color:var(--text-muted)]">关注者</div>
                      <div class="font-semibold text-[color:var(--text-primary)]">{{ item.followers }}</div>
                    </div>
                    <div>
                      <div class="text-[color:var(--text-muted)]">嘟文</div>
                      <div class="font-semibold text-[color:var(--text-primary)]">0</div>
                    </div>
                    <div>
                      <div class="text-[color:var(--text-muted)]">上次活跃</div>
                      <div class="font-semibold text-[color:var(--text-primary)]">{{ formatTimestamp(item.createdAt) || '-' }}</div>
                    </div>
                  </div>
                </div>
              </div>

              <button
                @click="toggleFollowUser(item)"
                :disabled="followActionLoading[item.id]"
                class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {{
                  followActionLoading[item.id]
                    ? '处理中...'
                    : isFollowingUser(item.id)
                      ? '取消关注'
                      : '关注'
                }}
              </button>
            </div>
          </article>
        </section>
      </div>
    </div>
  </div>
</template>
