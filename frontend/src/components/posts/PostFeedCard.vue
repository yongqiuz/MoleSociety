<script setup lang="ts">
import { Bookmark, Heart, MessageCircle, MoreHorizontal, Repeat } from 'lucide-vue-next';
import MentionText from '../common/MentionText.vue';

type PollOption = {
  label: string;
  votes: number;
};

type Poll = {
  options: PollOption[];
  expiresAt: string;
  multiple: boolean;
  voters: string[];
};

type FeedCard = {
  id: string;
  authorId: string;
  author: string;
  handle: string;
  instance: string;
  kind: string;
  chainProof: string;
  type: string;
  interaction: string;
  time: string;
  content: string;
  bio?: string;
  tags: string[];
  media?: {
    name: string;
    preview: string;
    type: string;
    sizeLabel: string;
  };
  stats: {
    replies: number;
    boosts: number;
    likes: number;
    bookmarks: number;
  };
  poll?: Poll;
};

const props = defineProps<{
  post: FeedCard;
  avatarUrl?: string;
  liked?: boolean;
  bookmarked?: boolean;
  currentUserId?: string;
  showMoreMenu?: boolean;
  moreMenuOpen?: boolean;
  mentionUsers?: Array<{
    id: string;
    handle: string;
    displayName: string;
    instance?: string;
  }>;
}>();

const emit = defineEmits<{
  (e: 'open-profile', userId: string): void;
  (e: 'open-detail', postId: string): void;
  (e: 'forward', post: any): void;
  (e: 'toggle-like', postId: string): void;
  (e: 'toggle-bookmark', postId: string): void;
  (e: 'toggle-more', postId: string): void;
  (e: 'menu-action', action: string, post: any): void;
  (e: 'vote', post: any, optionIndices: number[]): void;
}>();

function avatarText(name: string) {
  return name?.slice(0, 1).toUpperCase() || 'U';
}
</script>

<template>
  <article class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
    <div class="flex gap-3">
      <button
        @click="emit('open-profile', post.authorId)"
        class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
        title="查看用户主页"
      >
        <img v-if="avatarUrl" :src="avatarUrl" class="h-full w-full object-cover" />
        <template v-else>{{ avatarText(post.author) }}</template>
      </button>
      <div class="min-w-0 flex-1">
        <div class="cursor-pointer" @click="emit('open-detail', post.id)">
          <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
            <button
              @click.stop="emit('open-profile', post.authorId)"
              class="text-lg font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
            >
              {{ post.author }}
            </button>
            <span class="text-sm text-[color:var(--text-secondary)]">@{{ post.instance }}</span>
            <span class="text-xs text-[color:var(--text-muted)]">{{ post.time }}</span>
          </div>
          <div v-if="post.bio" class="mt-0.5 text-xs text-[color:var(--text-muted)]">{{ post.bio }}</div>
          <div class="mt-3 text-[15px] leading-7 text-[color:var(--text-soft)]">
            <MentionText :text="post.content" :users="mentionUsers" @open-profile="(id) => emit('open-profile', id)" />
          </div>

          <div v-if="post.poll" class="mt-3 space-y-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-3">
            <div v-for="(opt, idx) in post.poll.options" :key="idx" class="relative">
              <div v-if="post.poll.voters.includes(currentUserId || '') || new Date(post.poll.expiresAt) < new Date()" class="group overflow-hidden rounded-lg bg-[var(--frame-bg)]">
                <div
                  class="absolute inset-y-0 left-0 bg-emerald-500/20 transition-all duration-1000"
                  :style="{ width: `${(opt.votes / Math.max(1, post.poll.options.reduce((a, b) => a + b.votes, 0))) * 100}%` }"
                />
                <div class="relative flex items-center justify-between px-4 py-2 text-[13px]">
                  <span class="font-medium text-[color:var(--text-primary)]">{{ opt.label }}</span>
                  <span class="font-bold text-emerald-400">
                    {{ Math.round((opt.votes / Math.max(1, post.poll.options.reduce((a, b) => a + b.votes, 0))) * 100) }}%
                  </span>
                </div>
              </div>
              <button
                v-else
                @click.stop="emit('vote', post, [idx])"
                class="w-full rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-left text-[13px] font-medium text-emerald-400 transition-all hover:bg-emerald-500/10 hover:border-emerald-500/50"
              >
                {{ opt.label }}
              </button>
            </div>
            <div class="mt-2 flex items-center justify-between text-[10px] font-bold uppercase tracking-wider text-[color:var(--text-muted)]">
              <span>{{ post.poll.options.reduce((a, b) => a + b.votes, 0) }} 票</span>
              <span>{{ new Date(post.poll.expiresAt) < new Date() ? '已结束' : '进行中' }}</span>
            </div>
          </div>

          <div v-if="post.media" class="mt-4 overflow-hidden rounded-2xl">
            <img :src="post.media.preview" :alt="post.media.name" class="max-h-[60vh] w-full h-auto object-contain" />
          </div>

          <div v-if="post.tags.length" class="mt-4 flex flex-wrap gap-2">
            <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-emerald-600 px-3 py-1 text-sm font-semibold text-white shadow-sm">
              #{{ tag }}
            </span>
          </div>
        </div>

        <div class="mt-5 flex flex-wrap items-center gap-3 text-sm" @click.stop>
          <button
            @click="emit('open-detail', post.id)"
            class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
          >
            <MessageCircle class="mr-1.5 h-[18px] w-[18px]" /> {{ post.stats.replies || '' }}
          </button>
          <button
            @click="emit('forward', post)"
            class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200"
          >
            <Repeat class="mr-1.5 h-[18px] w-[18px]" /> 转发
          </button>
          <button
            @click="emit('toggle-like', post.id)"
            class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
            :class="liked ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
          >
            <Heart :class="{ 'fill-current': liked }" class="mr-1.5 h-[18px] w-[18px]" /> {{ post.stats.likes + (liked ? 1 : 0) || '' }}
          </button>
          <button
            @click="emit('toggle-bookmark', post.id)"
            class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
            :class="bookmarked ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
          >
            <Bookmark :class="{ 'fill-current': bookmarked }" class="mr-1.5 h-[18px] w-[18px]" />
            {{ post.stats.bookmarks || '' }}
          </button>

          <div v-if="showMoreMenu" class="relative ml-auto">
            <button
              @click="emit('toggle-more', post.id)"
              class="inline-flex items-center rounded-lg px-2 py-1.5 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
            >
              <MoreHorizontal class="h-5 w-5" />
            </button>

            <div
              v-if="moreMenuOpen"
              class="absolute right-0 top-full z-50 mt-2 w-56 overflow-hidden rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] text-sm shadow-[0_10px_40px_rgba(0,0,0,0.5)]"
            >
              <div class="py-1">
                <button @click="emit('menu-action', 'share', post)" class="w-full px-4 py-2.5 text-left hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">分享</button>
                <button @click="emit('menu-action', 'mention', post)" class="w-full px-4 py-2.5 text-left hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)] font-medium">提及 {{ post.handle }}</button>
                <button
                  v-if="currentUserId && post.authorId === currentUserId"
                  @click="emit('menu-action', 'delete', post)"
                  class="w-full px-4 py-2.5 text-left text-rose-500 hover:bg-rose-500/10"
                >
                  删除帖子
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </article>
</template>
