<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import 'emoji-picker-element';
import {
  bookmarkPost,
  createConversation,
  createConversationMessage,
  createMediaAsset,
  deletePost as deletePostApi,
  createPost,
  fetchPostReplies,
  fetchPostThread,
  fetchSocialBootstrap,
  fetchSocialUsers,
  fetchSocialHot,
  fetchSocialLatest,
  fetchSocialNews,
  fetchSocialInstances,
  fetchSocialBootstrapMine,
  fetchUserFollowers,
  fetchUserFollowing,
  followUser,
  getConversation,
  listConversations,
  unfollowUser,
  unbookmarkPost,
  updateUserProfile,
  voteOnPoll,
  type BootstrapPayload,
  type FederationInstance,
  type MediaAsset,
  type SocialConversation,
  type SocialPost,
  type SocialUser,
  type Poll,
} from '../api/socialApi';
import { ApiError } from '../api/apiError';
import { useAuth } from '../composables/useAuth';
import type { Component } from 'vue';
import {
  Home, Compass, Bell, Hash, Star, Bookmark, AtSign, Settings,
  MoreHorizontal, Shield, PenTool, Mail, AlignJustify, Users,
  Filter, Trash2, Image as ImageIcon, CheckSquare, Smile, Search,
  ArrowLeft, ChevronLeft, LogOut, MessageCircle, Repeat, Heart, Pencil, TrendingUp, Newspaper,
  Globe, Moon, Lock, ChevronDown, ChevronUp, X, BarChart3, RefreshCw
} from 'lucide-vue-next';
import { useAppearance } from '../composables/useAppearance';
import PostFeedCard from '../components/posts/PostFeedCard.vue';

type Section =
  | 'home'
  | 'search'
  | 'postDetail'
  | 'explore'
  | 'messages'
  | 'notifications'
  | 'lists'
  | 'topics'
  | 'likes'
  | 'bookmarks'
  | 'mentions'
  | 'followers'
  | 'following'
  | 'preferences';

type ExploreTab = 'posts' | 'latest' | 'topics' | 'users' | 'news';

type SettingsTab =
  | 'profile'
  | 'privacy'
  | 'preferences'
  | 'appearance'
  | 'posting'
  | 'notifications'
  | 'other'
  | 'follows'
  | 'filters'
  | 'cleanup';

type FeedCard = {
  id: string;
  authorId: string;
  author: string;
  handle: string;
  instance: string;
  kind: string;
  parentPostId?: string;
  rootPostId?: string;
  replyDepth?: number;
  time: string;
  content: string;
  bio?: string;
  isBot?: boolean;
  tags: string[];
  chainProof: string;
  chainId?: string;
  txHash?: string;
  contractAddress?: string;
  explorerUrl?: string;
  media?: {
    name: string;
    preview: string;
    type: string;
    sizeLabel: string;
  };
  type: string;
  stats: {
    replies: number;
    boosts: number;
    likes: number;
    bookmarks: number;
  };
  interaction: string;
  poll?: Poll;
  createdAt?: string;
};

type NotificationItem = {
  id: string;
  kind: 'mention' | 'reply' | 'follow' | 'directMessage' | 'system' | 'digest';
  title: string;
  body: string;
  time: string;
  sortAt: number;
};

type AssetCard = {
  id: string;
  title: string;
  network: string;
  cid: string;
  size: string;
  retention: string;
  url: string;
};

type MessageCard = {
  id: string;
  from: 'me' | 'peer';
  text: string;
  imageEmoji?: {
    name: string;
    preview: string;
    type: string;
    sizeLabel: string;
  } | null;
  time: string;
  createdAt?: string;
  forwardedPost?: {
    id: string;
    author: string;
    handle: string;
    instance: string;
    content: string;
    createdAt?: string;
    media?: {
      name: string;
      preview: string;
      type: string;
      sizeLabel: string;
    };
  } | null;
  assetUri?: string;
  chainId?: string;
  txHash?: string;
  contractAddress?: string;
  explorerUrl?: string;
};

type ConversationCard = {
  id: string;
  name: string;
  handle: string;
  avatarUrl?: string;
  backgroundUrl?: string;
  status: string;
  crossInstance: boolean;
  federationRoute: string;
  assetUri?: string;
  chainId?: string;
  txHash?: string;
  contractAddress?: string;
  explorerUrl?: string;
  avatarLabel: string;
  participantId?: string;
  messages: MessageCard[];
};

const primaryNavItems: { label: string; key: Section; icon: Component }[] = [
  { label: '主页', key: 'home', icon: Home },
  { label: '当前热门', key: 'explore', icon: TrendingUp },
  { label: '消息', key: 'messages', icon: Mail },
  { label: '通知', key: 'notifications', icon: Bell },
];

const secondaryNavItems: { label: string; key: Section; icon: Component }[] = [
  { label: '联邦切换', key: 'lists', icon: Globe },
  { label: '话题', key: 'topics', icon: Hash },
  { label: '喜欢', key: 'likes', icon: Star },
  { label: '书签', key: 'bookmarks', icon: Bookmark },
  { label: '提及', key: 'mentions', icon: AtSign },
];

const utilityNavItems: { label: string; key: Section; icon: Component }[] = [
  { label: '偏好设置', key: 'preferences', icon: Settings },
];

// settingsMenu removed - moved to SettingsPage.vue

const currentSection = ref<Section>('home');
const currentUser = ref<SocialUser | null>(null);
const people = ref<SocialUser[]>([]);
const posts = ref<FeedCard[]>([]);
const myPosts = ref<FeedCard[]>([]);
const assets = ref<AssetCard[]>([]);
const conversations = ref<ConversationCard[]>([]);
const instances = ref<FederationInstance[]>([]);
const postDraft = ref('');
const showPollEditor = ref(false);
const showTagPicker = ref(false);
const showEmojiPicker = ref(false);
const selectedPostTags = ref<string[]>([]);
const customTagInput = ref('');
const defaultTagOptions = ['二次元', '游戏', '影视'];
const recentPostTags = ref<string[]>([]);
const pollOptions = ref(['', '']);
const pollExpiresIn = ref(1440); // 1 day
const pollMultiple = ref(false);
const messageDraft = ref('');
const messageMediaPreview = ref<string | null>(null);
const messageMediaMeta = ref<{ name: string; sizeLabel: string; type: string; sizeBytes: number } | null>(null);
const showMessageEmojiPicker = ref(false);
const showMessageStickerPanel = ref(false);
const messageEmojiPanelRef = ref<HTMLElement | null>(null);
const messageEmojiTriggerRef = ref<HTMLElement | null>(null);
const messageStickerPanelRef = ref<HTMLElement | null>(null);
const messageStickerTriggerRef = ref<HTMLElement | null>(null);
const messageImageInputRef = ref<HTMLInputElement | null>(null);
const messageInputRef = ref<HTMLTextAreaElement | null>(null);
const recentMessageStickers = ref<Array<{ name: string; preview: string; type: string; sizeLabel: string }>>([]);
const searchQuery = ref('');
const selectedConversationId = ref('');
const selectedTopicTag = ref('');
const mediaPreview = ref<string | null>(null);
const mediaMeta = ref<{ name: string; sizeLabel: string; type: string; sizeBytes: number } | null>(null);
const replyDraft = ref('');
const replyMediaPreview = ref<string | null>(null);
const replyMediaMeta = ref<{ name: string; sizeLabel: string; type: string; sizeBytes: number } | null>(null);
const messageListRef = ref<HTMLElement | null>(null);
const loading = ref(true);
const saving = ref(false);
const apiOnline = ref(false);
const errorMessage = ref('');
const followedUsers = ref<Record<string, boolean>>({});
const likedPosts = ref<Record<string, boolean>>({});
const bookmarkedPosts = ref<Record<string, boolean>>({});
const postEngagement = ref<Record<string, { likes: number; bookmarks: number }>>({});
const showForwardDialog = ref(false);
const forwardingPost = ref<FeedCard | null>(null);
const forwardingConversationId = ref('');
const forwarding = ref(false);
const showDeletePostDialog = ref(false);
const deletingPost = ref<FeedCard | null>(null);
const deletingPostLoading = ref(false);
const selectedInstanceName = ref('all');
const selectedPostId = ref('');
const threadLoading = ref(false);
const threadError = ref('');
const mainContentRef = ref<HTMLElement | null>(null);
const isPullingHome = ref(false);
const isRefreshingHome = ref(false);
const pullDistance = ref(0);
const pullStartY = ref(0);
const pullEligible = ref(false);
const threadFocusPost = ref<FeedCard | null>(null);
const threadAncestors = ref<FeedCard[]>([]);
const threadReplies = ref<FeedCard[]>([]);
const activeReplyTarget = ref<FeedCard | null>(null);
const replyTextareaRef = ref<HTMLTextAreaElement | null>(null);
const replyFileInputRef = ref<HTMLInputElement | null>(null);
const postComposerRef = ref<HTMLTextAreaElement | null>(null);
const emojiPickerPanelRef = ref<HTMLElement | null>(null);
const emojiTriggerRef = ref<HTMLElement | null>(null);
const emojiPickerFloatingStyle = ref<Record<string, string>>({});
const postSelectionStart = ref(0);
const postSelectionEnd = ref(0);
const replySelectionStart = ref(0);
const replySelectionEnd = ref(0);
const showReplyEmojiPicker = ref(false);
const replyEmojiPickerPanelRef = ref<HTMLElement | null>(null);
const replyEmojiTriggerRef = ref<HTMLElement | null>(null);
const processingMessageIntent = ref(false);
const notificationFeed = ref<NotificationItem[]>([]);
const relationUsers = ref<SocialUser[]>([]);
const relationLoading = ref(false);
const relationError = ref('');
const followActionLoading = ref<Record<string, boolean>>({});
const bookmarkActionLoading = ref<Record<string, boolean>>({});

const threadedReplies = computed(() => {
  const root = threadFocusPost.value;
  if (!root) {
    return threadReplies.value.map((post) => ({ post, depth: Math.max(2, (post.replyDepth ?? 1) + 1) }));
  }

  const items = [...threadReplies.value];
  const byId = new Map(items.map((post) => [post.id, post]));
  const children = new Map<string, FeedCard[]>();
  const rootKey = root.id;

  const pushChild = (parentId: string, post: FeedCard) => {
    const bucket = children.get(parentId);
    if (bucket) bucket.push(post);
    else children.set(parentId, [post]);
  };

  for (const post of items) {
    const parentId = String(post.parentPostId || '').trim();
    if (!parentId || parentId === rootKey || !byId.has(parentId)) {
      pushChild(rootKey, post);
      continue;
    }
    pushChild(parentId, post);
  }

  const toSortTime = (post: FeedCard) => {
    const raw = post.createdAt || '';
    const ts = raw ? Date.parse(raw) : NaN;
    return Number.isNaN(ts) ? 0 : ts;
  };

  for (const [, arr] of children.entries()) {
    arr.sort((a, b) => toSortTime(a) - toSortTime(b) || a.id.localeCompare(b.id));
  }

  const flattened: Array<{ post: FeedCard; depth: number }> = [];
  const walk = (parentId: string, depth: number) => {
    const bucket = children.get(parentId) || [];
    for (const post of bucket) {
      flattened.push({ post, depth });
      walk(post.id, depth + 1);
    }
  };

  walk(rootKey, 2);
  return flattened;
});
const twoLevelReplies = computed(() => {
  const flattened = threadedReplies.value;
  const byId = new Map(flattened.map((item) => [item.post.id, item.post]));
  const groups: Array<{ parent: FeedCard; children: Array<{ post: FeedCard; replyTo?: FeedCard | null }> }> = [];
  const groupByParentId = new Map<string, { parent: FeedCard; children: Array<{ post: FeedCard; replyTo?: FeedCard | null }> }>();

  for (const item of flattened) {
    if (item.depth <= 2) {
      const group = { parent: item.post, children: [] as Array<{ post: FeedCard; replyTo?: FeedCard | null }> };
      groups.push(group);
      groupByParentId.set(item.post.id, group);
    }
  }

  if (groups.length === 0) {
    for (const item of flattened) {
      const group = { parent: item.post, children: [] as Array<{ post: FeedCard; replyTo?: FeedCard | null }> };
      groups.push(group);
      groupByParentId.set(item.post.id, group);
    }
  }

  for (const item of flattened) {
    if (item.depth <= 2) continue;
    const directParentId = String(item.post.parentPostId || '').trim();
    const directParent = byId.get(directParentId) || null;
    let anchorParentId = directParentId;
    while (anchorParentId) {
      const anchor = byId.get(anchorParentId);
      if (!anchor) break;
      const anchorDepth = flattened.find((x) => x.post.id === anchor.id)?.depth ?? 3;
      if (anchorDepth <= 2) break;
      anchorParentId = String(anchor.parentPostId || '').trim();
    }
    const group = groupByParentId.get(anchorParentId) || groups[0];
    if (!group) continue;
    group.children.push({ post: item.post, replyTo: directParent });
  }

  return groups;
});
const route = useRoute();
const router = useRouter();
const { session: authSession } = useAuth();

const MAX_POST_LENGTH = 500;
const MAX_POST_TAGS = 5;
const MAX_TAG_LENGTH = 24;
const RECENT_TAGS_STORAGE_KEY_PREFIX = 'mole-compose-recent-tags';
const POSTING_PRIVACY_STORAGE_KEY = 'mole-posting-privacy-settings';
const PRIVACY_SETTINGS_STORAGE_KEY = 'mole-privacy-settings';
const PULL_MAX_DISTANCE = 120;
const PULL_REFRESH_THRESHOLD = 72;
const LIKE_STORAGE_PREFIX = 'mole-liked-posts';
const BOOKMARK_STORAGE_PREFIX = 'mole-bookmarked-posts';
const POST_ENGAGEMENT_STORAGE_PREFIX = 'mole-post-engagement';
const FORWARDED_POST_PREFIX = '[FORWARDED_POST]';
const MESSAGE_IMAGE_EMOJI_PREFIX = '[IMAGE_EMOJI]';
const NOTIFICATION_READ_STORAGE_PREFIX = 'mole-notification-read';
const MENTION_READ_STORAGE_PREFIX = 'mole-mention-read';
const FOLLOWER_SEEN_IDS_STORAGE_PREFIX = 'mole-follower-seen-ids';
const MESSAGE_READ_AT_STORAGE_PREFIX = 'mole-message-read-at';
const MESSAGE_STICKER_STORAGE_PREFIX = 'mole-message-stickers';
const ENCRYPTED_MESSAGE_CACHE_STORAGE_PREFIX = 'mole-encrypted-message-cache';
const MESSAGE_DEVICE_ID_STORAGE_PREFIX = 'mole-message-device-id';
const MESSAGE_ENCRYPTION_KEY_STORAGE_PREFIX = 'mole-message-encryption-key';
const INSTANCE_POLL_INTERVAL_MS = 2000;
const DEFAULT_FEED_LIMIT = 12;
const NOTIFICATION_SETTINGS_STORAGE_KEY = 'mole-notification-settings';

type NotificationSettings = {
  mentions: boolean;
  replies: boolean;
  follows: boolean;
  directMessages: boolean;
  systemUpdates: boolean;
  emailDigest: boolean;
  quietHours: boolean;
};

const defaultNotificationSettings: NotificationSettings = {
  mentions: true,
  replies: true,
  follows: true,
  directMessages: true,
  systemUpdates: true,
  emailDigest: false,
  quietHours: false,
};

type EncryptedMessageCacheEntry = {
  version: 1;
  ciphertext: string;
  iv: string;
  tag: string;
  algorithm: 'AES-256-GCM';
  createdAt: number;
  senderDeviceId: string;
};

const isLoggedIn = computed(() => !!authSession.value);
const isReplyingRoot = computed(() => !!threadFocusPost.value && activeReplyTarget.value?.id === threadFocusPost.value.id);

const { themeStyles, appearanceSettings } = useAppearance();

const activeExploreTab = ref<ExploreTab>('posts');
const newsTimeline = ref<FeedCard[]>([]);
const newsLoading = ref(false);
const explorePostsTimeline = ref<FeedCard[]>([]);
const explorePostsLoading = ref(false);
const latestPostsLoading = ref(false);
const exploreUsersLoading = ref(false);
const exploreUsers = ref<SocialUser[]>([]);
const searchWarmupLoading = ref(false);
const instancePollingTimer = ref<number | null>(null);
const instancePollingInFlight = ref(false);

function parseNewsContent(content: string) {
  const text = String(content || '').trim();
  const lines = text.split('\n').map((line) => line.trim()).filter(Boolean);
  const first = lines[0] || '';
  const link = lines.find((line) => /^https?:\/\//i.test(line)) || '';
  const sourceMatch = first.match(/^【([^】]+)】(.*)$/);
  if (!sourceMatch) {
    return { source: '新闻', title: first || text, link };
  }
  return {
    source: sourceMatch[1] || '新闻',
    title: (sourceMatch[2] || '').trim() || first,
    link,
  };
}

async function loadNewsTimeline(force = false) {
  if (newsLoading.value) return;
  if (!force && newsTimeline.value.length > 0) return;
  newsLoading.value = true;
  try {
    if (!apiOnline.value) return;
    const payload = await fetchSocialNews(DEFAULT_FEED_LIMIT, 0);
    newsTimeline.value = hydrateFeedCardList((payload.items || []).map(toFeedCard));
  } catch {
    // keep current timeline, avoid interrupting main flow
  } finally {
    newsLoading.value = false;
  }
}

function stopInstancePolling() {
  if (instancePollingTimer.value !== null && typeof window !== 'undefined') {
    window.clearTimeout(instancePollingTimer.value);
  }
  instancePollingTimer.value = null;
}

function scheduleNextInstancePoll() {
  if (typeof window === 'undefined') return;
  stopInstancePolling();
  instancePollingTimer.value = window.setTimeout(() => {
    void pollInstancesSafely();
  }, INSTANCE_POLL_INTERVAL_MS);
}

async function pollInstancesSafely() {
  if (instancePollingInFlight.value) {
    scheduleNextInstancePoll();
    return;
  }
  if (!apiOnline.value) {
    scheduleNextInstancePoll();
    return;
  }
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') {
    scheduleNextInstancePoll();
    return;
  }
  instancePollingInFlight.value = true;
  try {
    const latest = await fetchSocialInstances();
    if (Array.isArray(latest)) {
      instances.value = latest;
    }
  } catch {
    // swallow polling errors; next round will retry
  } finally {
    instancePollingInFlight.value = false;
    scheduleNextInstancePoll();
  }
}

function startInstancePolling() {
  if (typeof window === 'undefined') return;
  if (instancePollingTimer.value !== null) return;
  scheduleNextInstancePoll();
}

function handleVisibilityChange() {
  if (typeof document === 'undefined') return;
  if (document.visibilityState === 'visible') {
    if (!instancePollingInFlight.value) {
      void pollInstancesSafely();
    }
    return;
  }
  stopInstancePolling();
}
const activeMoreMenuId = ref<string | null>(null);

function toggleMoreMenu(postId: string) {
  activeMoreMenuId.value = activeMoreMenuId.value === postId ? null : postId;
}

function toPostPublicUrl(post: FeedCard) {
  if (typeof window === 'undefined') return '';
  const origin = window.location.origin || '';
  return `${origin}/app?post=${encodeURIComponent(post.id)}`;
}

function toMentionToken(post: FeedCard) {
  const handle = String(post.handle || '').replace(/^@/, '').trim();
  const instance = String(post.instance || '').trim();
  if (!handle && !instance) return '';
  if (!handle) return `@${instance}`;
  if (!instance) return `@${handle}`;
  return `@${handle}@${instance}`;
}

async function injectMentionIntoComposer(post: FeedCard) {
  const mentionToken = toMentionToken(post);
  if (!mentionToken) return;

  const hasMention = postDraft.value.includes(mentionToken);
  if (!hasMention) {
    postDraft.value = postDraft.value.trim()
      ? `${postDraft.value.trim()} ${mentionToken} `
      : `${mentionToken} `;
  }

  currentSection.value = 'home';
  showTagPicker.value = false;
  showEmojiPicker.value = false;
  syncPostCursor();
  await nextTick();
  if (postComposerRef.value) {
    postComposerRef.value.focus();
    const pos = postDraft.value.length;
    postComposerRef.value.setSelectionRange(pos, pos);
  }
}

async function handleMenuAction(action: string, post: FeedCard) {
  activeMoreMenuId.value = null;
  if (action === 'share') {
    await openForwardDialog(post);
    return;
  }
  if (action === 'mention') {
    await injectMentionIntoComposer(post);
    return;
  }
  if (action === 'copyLink') {
    const postUrl = toPostPublicUrl(post);
    if (!postUrl || typeof navigator === 'undefined' || !navigator.clipboard) return;
    try {
      await navigator.clipboard.writeText(postUrl);
    } catch {
      errorMessage.value = '复制链接失败，请稍后重试。';
    }
    return;
  }
  if (action === 'delete') {
    requestDeletePost(post);
    return;
  }
}

function requestDeletePost(post: FeedCard) {
  if (!currentUser.value || post.authorId !== currentUser.value.id) {
    errorMessage.value = '只能删除自己发布的帖子。';
    return;
  }
  deletingPost.value = post;
  showDeletePostDialog.value = true;
}

function cancelDeletePost() {
  if (deletingPostLoading.value) return;
  showDeletePostDialog.value = false;
  deletingPost.value = null;
}

async function confirmDeletePost() {
  if (!deletingPost.value || deletingPostLoading.value) return;
  deletingPostLoading.value = true;
  const target = deletingPost.value;
  try {
    await deletePostApi(target.id);
    const drop = (list: FeedCard[]) => list.filter((item) => item.id !== target.id);
    posts.value = drop(posts.value);
    myPosts.value = drop(myPosts.value);
    threadAncestors.value = drop(threadAncestors.value);
    threadReplies.value = drop(threadReplies.value);
    if (threadFocusPost.value?.id === target.id) {
      threadFocusPost.value = null;
      currentSection.value = 'home';
    }
    const nextLiked = { ...likedPosts.value };
    delete nextLiked[target.id];
    likedPosts.value = nextLiked;
    const nextBookmarked = { ...bookmarkedPosts.value };
    delete nextBookmarked[target.id];
    bookmarkedPosts.value = nextBookmarked;
    showDeletePostDialog.value = false;
    deletingPost.value = null;
  } catch {
    errorMessage.value = '删除失败，请稍后重试。';
  } finally {
    deletingPostLoading.value = false;
  }
}

// Visibility and Interaction State
const visibility = ref('public');
const interaction = ref('anyone');
const showVisibilityModal = ref(false);
const tempVisibility = ref('public');
const tempInteraction = ref('anyone');

const visibilityOptions = [
  { id: 'public', label: '公开', description: '所有人可见', icon: Globe },
  { id: 'unlisted', label: '悄悄公开', description: '不出现在搜索或公共时间线', icon: Moon },
  { id: 'private', label: '关注者', description: '仅限你的关注者', icon: Lock },
  { id: 'direct', label: '私下提及', description: '仅提到的用户可见', icon: AtSign },
];

const interactionOptions = [
  { id: 'anyone', label: '任何人' },
  { id: 'followers', label: '仅关注者' },
  { id: 'me', label: '仅限自己' },
];

const selectedVisibilityItem = computed(() => 
  visibilityOptions.find(opt => opt.id === visibility.value) || visibilityOptions[0]
);

const selectedInteractionItem = computed(() => 
  interactionOptions.find(opt => opt.id === interaction.value) || interactionOptions[0]
);

const interactionSummary = computed(() => {
  if (interaction.value === 'anyone') return '允许引用';
  if (interaction.value === 'followers') return '关注者可引用';
  return '禁止引用';
});

const remainingPostChars = computed(() => MAX_POST_LENGTH - postDraft.value.length);
const isPostOverLimit = computed(() => remainingPostChars.value < 0);
const pullRefreshHint = computed(() => {
  if (isRefreshingHome.value) return '刷新中...';
  return pullDistance.value >= PULL_REFRESH_THRESHOLD ? '松开刷新' : '下拉刷新';
});

const activeConversation = computed(() =>
  conversations.value.find((conversation) => conversation.id === selectedConversationId.value),
);

const activeConversationPeer = computed(() => {
  const conversation = activeConversation.value;
  if (!conversation) return null;
  return people.value.find((person) => person.id === conversation.participantId) ?? null;
});

function isCrossInstanceUser(user: SocialUser) {
  return Boolean(currentUser.value && currentUser.value.instance !== user.instance);
}

const mediaCount = computed(() => posts.value.filter((post) => post.media).length + assets.value.length);

const myTimeline = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return myPosts.value;
  return myPosts.value.filter((post) =>
    [post.author, post.handle, post.content, ...post.tags].join(' ').toLowerCase().includes(query),
  );
});

const timeline = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return posts.value;
  return posts.value.filter((post) =>
    [post.author, post.handle, post.content, ...post.tags].join(' ').toLowerCase().includes(query),
  );
});

function postCreatedAtTs(post: FeedCard) {
  if (!post.createdAt) return 0;
  const ts = Date.parse(post.createdAt);
  return Number.isNaN(ts) ? 0 : ts;
}

const exploreTimeline = computed(() =>
  [...explorePostsTimeline.value]
    .sort((a, b) => {
    const likeDelta = (b.stats.likes || 0) - (a.stats.likes || 0);
    if (likeDelta !== 0) return likeDelta;
    const bookmarkDelta = (b.stats.bookmarks || 0) - (a.stats.bookmarks || 0);
    if (bookmarkDelta !== 0) return bookmarkDelta;
    return postCreatedAtTs(b) - postCreatedAtTs(a);
  }),
);

async function loadExplorePostsTimeline(force = false) {
  if (explorePostsLoading.value) return;
  if (!force && explorePostsTimeline.value.length > 0) return;
  explorePostsLoading.value = true;
  try {
    if (!apiOnline.value) return;
    // "当前热门" should come from hot ranking endpoint.
    const payload = await fetchSocialHot(DEFAULT_FEED_LIMIT, 0);
    explorePostsTimeline.value = hydrateFeedCardList((payload.items || []).map(toFeedCard));
  } catch {
    // keep current timeline, avoid interrupting main flow
  } finally {
    explorePostsLoading.value = false;
  }
}

async function loadLatestPostsTimeline(force = false) {
  if (latestPostsLoading.value) return;
  if (!force && posts.value.length > 0) return;
  latestPostsLoading.value = true;
  try {
    if (!apiOnline.value) return;
    const payload = await fetchSocialLatest(DEFAULT_FEED_LIMIT, 0);
    posts.value = hydrateFeedCardList((payload.items || []).map(toFeedCard));
  } finally {
    latestPostsLoading.value = false;
  }
}

async function warmupSearchPostPools() {
  if (searchWarmupLoading.value) return;
  if (!apiOnline.value) return;
  searchWarmupLoading.value = true;
  try {
    await Promise.allSettled([
      loadLatestPostsTimeline(true),
      loadExplorePostsTimeline(true),
      loadNewsTimeline(true),
    ]);
  } finally {
    searchWarmupLoading.value = false;
  }
}

const selectedInstance = computed(() =>
  instances.value.find((instance) => instance.name === selectedInstanceName.value) ?? null,
);

const activeProfileInstanceName = computed(() =>
  selectedInstanceName.value === 'all'
    ? currentUser.value?.instance || '摩尔1号'
    : selectedInstanceName.value,
);

const homeTimeline = computed(() => {
  if (selectedInstanceName.value === 'all') return myTimeline.value;
  return myTimeline.value.filter((post) => post.instance === selectedInstanceName.value);
});

async function selectInstance(name: string) {
  const previousInstance = selectedInstanceName.value;
  selectedInstanceName.value = name;
  if (name === 'all') return;
  if (!currentUser.value) return;
  if (!apiOnline.value) {
    goToNotFound();
    return;
  }

  try {
    // Optimistic update: immediately reflect instance switch in local UI.
    currentUser.value = { ...currentUser.value, instance: name };
    people.value = people.value.map((person) =>
      person.id === currentUser.value?.id ? { ...person, instance: name } : person,
    );
    applyInstanceMemberSwitch(previousInstance, name);
    refreshConversationFederationRoutes();

    const updatedUser = await updateUserProfile(currentUser.value.id, { instance: name });
    currentUser.value = updatedUser;
    people.value = people.value.map((person) => (person.id === updatedUser.id ? updatedUser : person));
    refreshConversationFederationRoutes();
    apiOnline.value = true;
    // Silent refresh in background, do not block the interaction.
    void fetchSocialBootstrap()
      .then(async (payload) => {
        applyBootstrap(payload);
        await loadLatestPostsTimeline(true);
      })
      .catch(() => {});
  } catch (error) {
    // rollback optimistic selection
    selectedInstanceName.value = previousInstance;
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    goToNotFound();
  }
}

const recommendedPeople = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  const source = exploreUsers.value.length > 0 ? exploreUsers.value : people.value;
  return source
    .filter((person) => person.id !== currentUser.value?.id)
    .filter((person) =>
      !query ||
      [person.displayName, person.handle, person.instance, person.bio]
        .join(' ')
        .toLowerCase()
        .includes(query),
    );
});

async function loadExploreUsers(force = false) {
  if (exploreUsersLoading.value) return;
  if (!force && exploreUsers.value.length > 0) return;
  if (!apiOnline.value) return;
  exploreUsersLoading.value = true;
  try {
    const users = await fetchSocialUsers();
    exploreUsers.value = users;
    people.value = users;
  } catch {
    // keep existing list on failure
  } finally {
    exploreUsersLoading.value = false;
  }
}

const trendingTags = computed(() => {
  const bucket = new Map<string, number>();
  allKnownPosts.value.forEach((post) => {
    const tags = [
      ...(Array.isArray(post.tags) ? post.tags : []),
      ...extractContentTopicTags(post.content),
    ]
      .map(normalizeTopicTag)
      .filter(Boolean);
    tags.forEach((tag) => {
      bucket.set(tag, (bucket.get(tag) ?? 0) + 1);
    });
  });
  return [...bucket.entries()]
    .sort((left, right) => right[1] - left[1])
    .slice(0, 6)
    .map(([tag, count]) => ({ tag, count }));
});

const availablePostTags = computed(() => {
  const pool = [...defaultTagOptions, ...recentPostTags.value];
  return [...new Set(pool.map((tag) => tag.trim()).filter(Boolean))];
});

const serviceNotice = computed(() =>
  errorMessage.value || '跨实例动态正在持续刷新。',
);

function allowNotificationKind(kind: NotificationItem['kind']) {
  if (notificationSettings.value.quietHours) {
    return kind === 'mention' || kind === 'directMessage' || kind === 'system';
  }
  if (kind === 'mention') return notificationSettings.value.mentions;
  if (kind === 'reply') return notificationSettings.value.replies;
  if (kind === 'follow') return notificationSettings.value.follows;
  if (kind === 'directMessage') return notificationSettings.value.directMessages;
  if (kind === 'system') return notificationSettings.value.systemUpdates;
  if (kind === 'digest') return notificationSettings.value.emailDigest;
  return true;
}

const currentSectionInfo = computed(() => {
  const allNavItems = [...primaryNavItems, ...secondaryNavItems, ...utilityNavItems];
  const navItem = allNavItems.find(item => item.key === currentSection.value);
  
  if (navItem) return navItem;
  
  if (currentSection.value === 'postDetail') {
    return { label: '摩文详情', icon: MessageCircle };
  }
  if (currentSection.value === 'search') {
    return { label: '搜索', icon: Search };
  }
  
  return { label: '主页', icon: Home };
});

const allKnownPosts = computed(() => {
  const pool: FeedCard[] = [
    ...posts.value,
    ...myPosts.value,
    ...explorePostsTimeline.value,
    ...newsTimeline.value,
    ...threadAncestors.value,
    ...threadReplies.value,
    ...(threadFocusPost.value ? [threadFocusPost.value] : []),
  ];
  const uniq = new Map<string, FeedCard>();
  pool.forEach((post) => {
    if (!post?.id || uniq.has(post.id)) return;
    uniq.set(post.id, post);
  });
  return [...uniq.values()];
});

const likedTimeline = computed(() => allKnownPosts.value.filter((post) => likedPosts.value[post.id]));

const bookmarkedTimeline = computed(() => allKnownPosts.value.filter((post) => bookmarkedPosts.value[post.id]));

function normalizeSearchText(raw: string) {
  return String(raw || '').trim().toLowerCase();
}

const searchKeyword = computed(() => normalizeSearchText(searchQuery.value));

const searchedUsers = computed(() => {
  const q = searchKeyword.value;
  if (!q) return [];
  const userPool = [
    ...people.value,
    ...(currentUser.value ? [currentUser.value] : []),
  ];
  const uniq = new Map<string, SocialUser>();
  userPool.forEach((user) => {
    if (!user?.id || uniq.has(user.id)) return;
    uniq.set(user.id, user);
  });
  return [...uniq.values()]
    .filter((user) => [
      user.displayName,
      user.handle,
      user.bio,
      user.instance,
      user.wallet,
    ].join(' ').toLowerCase().includes(q))
    .sort((a, b) => {
      const ah = String(a.handle || '').replace(/^@/, '').toLowerCase();
      const bh = String(b.handle || '').replace(/^@/, '').toLowerCase();
      const an = String(a.displayName || '').toLowerCase();
      const bn = String(b.displayName || '').toLowerCase();
      const aScore = Number(ah === q) * 4 + Number(an === q) * 3 + Number(ah.startsWith(q)) * 2 + Number(an.startsWith(q));
      const bScore = Number(bh === q) * 4 + Number(bn === q) * 3 + Number(bh.startsWith(q)) * 2 + Number(bn.startsWith(q));
      return bScore - aScore;
    })
    .slice(0, 20);
});

const searchedPosts = computed(() => {
  const q = searchKeyword.value;
  if (!q) return [];
  const postPool = [...posts.value, ...myPosts.value, ...newsTimeline.value];
  const uniq = new Map<string, FeedCard>();
  postPool.forEach((post) => {
    if (!post?.id || uniq.has(post.id)) return;
    uniq.set(post.id, post);
  });
  return [...uniq.values()]
    .filter((post) => [
      post.author,
      post.handle,
      post.instance,
      post.bio,
      post.content,
      ...(post.tags || []),
    ].join(' ').toLowerCase().includes(q))
    .sort((a, b) => postCreatedAtTs(b) - postCreatedAtTs(a))
    .slice(0, 30);
});

const notificationItems = computed(() => {
  const feedById = new Map<string, FeedCard>();
  [...posts.value, ...myPosts.value].forEach((post) => {
    if (!feedById.has(post.id)) feedById.set(post.id, post);
  });
  const myPostIds = new Set(myPosts.value.map((post) => post.id));

  const mentionEvents = currentUser.value
    ? posts.value
        .filter((post) => post.authorId !== currentUser.value?.id)
        .filter(isPostMentioningCurrentUser)
        .slice(0, 5)
        .map((post) => ({
          id: `mention-${post.id}`,
          kind: 'mention' as const,
          title: `${post.author} 提及了你`,
          body: post.content,
          time: post.time || '刚刚',
          sortAt: post.createdAt ? new Date(post.createdAt).getTime() : 0,
        }))
    : [];

  const replyEvents = currentUser.value
    ? [...feedById.values()]
      .filter((post) => post.authorId !== currentUser.value?.id)
      .filter((post) => post.kind === 'reply' && Boolean(post.parentPostId) && myPostIds.has(String(post.parentPostId)))
      .slice(0, 5)
      .map((post) => {
        const parent = post.parentPostId ? feedById.get(post.parentPostId) : null;
        return {
          id: `reply-${post.id}`,
          kind: 'reply' as const,
          title: `${post.author} 回复了你的帖子`,
          body: parent ? `回复「${parent.content.slice(0, 32)}」：${post.content}` : post.content,
          time: post.time || '刚刚',
          sortAt: post.createdAt ? new Date(post.createdAt).getTime() : 0,
        };
      })
    : [];

  const postEvents = timeline.value
    .filter((post) => post.authorId !== currentUser.value?.id)
    .filter((post) => Boolean(followedUsers.value[post.authorId]))
    .slice(0, 2)
    .map((post) => ({
    id: `post-${post.id}`,
    kind: 'system' as const,
    title: `${post.author} 发布了新内容`,
    body: post.content,
    time: post.time,
    sortAt: post.createdAt ? new Date(post.createdAt).getTime() : 0,
  }));

  const messageEvents = conversations.value
    .flatMap((conversation) => {
      const latestPeerMessage = [...conversation.messages]
        .filter((message) => message.from === 'peer')
        .sort((a, b) => {
          const left = a.createdAt ? Date.parse(a.createdAt) : 0;
          const right = b.createdAt ? Date.parse(b.createdAt) : 0;
          return right - left;
        })[0];
      if (!latestPeerMessage) return [];
      return [{
        id: `message-${conversation.id}-${latestPeerMessage.id}`,
        kind: 'directMessage' as const,
        title: `${conversation.name} 给你发了消息`,
        body: latestPeerMessage.forwardedPost
          ? forwardedSummaryText(latestPeerMessage.forwardedPost, 'peer')
          : latestPeerMessage.text,
        time: latestPeerMessage.time || '刚刚',
        sortAt: latestPeerMessage.createdAt ? Date.parse(latestPeerMessage.createdAt) : 0,
      }];
    })
    .slice(0, 5);

  const welcomeEvent = currentUser.value
    ? [{
        id: `welcome-${currentUser.value.id}`,
        kind: 'system' as const,
        title: '欢迎来到鼹鼠社区',
        body: '这里是一个去中心化社区，你可以在这里畅所欲言',
        time: '刚刚',
        sortAt: 0,
      }]
    : [];

  const digestEvent = notificationSettings.value.emailDigest
    ? [{
        id: 'digest-enabled',
        kind: 'digest' as const,
        title: '邮件摘要已开启',
        body: '系统会每天向你的绑定邮箱发送活动摘要。',
        time: '刚刚',
        sortAt: Date.now(),
      }]
    : [];

  const merged = [
    ...welcomeEvent,
    ...followerNotifications.value,
    ...messageEvents,
    ...mentionEvents,
    ...replyEvents,
    ...postEvents,
    ...digestEvent,
  ];

  return merged.filter((item) => allowNotificationKind(item.kind));
});

const orderedNotificationItems = computed(() =>
  [...notificationFeed.value]
    .filter((item) => allowNotificationKind(item.kind))
    .sort((a, b) => b.sortAt - a.sortAt),
);

function normalizeTopicTag(raw: string) {
  return String(raw || '').replace(/#/g, '').trim();
}

function extractContentTopicTags(content: string) {
  const text = String(content || '');
  const tags: string[] = [];
  const wrapped = text.match(/#([^#\s]{1,24})#/g) || [];
  wrapped.forEach((item) => {
    const cleaned = normalizeTopicTag(item);
    if (cleaned) tags.push(cleaned);
  });
  const loose = text.match(/(^|\s)#([^\s#]{1,24})/g) || [];
  loose.forEach((item) => {
    const cleaned = normalizeTopicTag(item);
    if (cleaned) tags.push(cleaned);
  });
  return tags;
}

const followedTopicCards = computed(() => {
  const now = Date.now();
  const bucket = new Map<string, { count: number; latestAt: number; mineCount: number; mineLatestAt: number }>();
  for (const post of posts.value) {
    const tags = [
      ...(Array.isArray(post.tags) ? post.tags : []),
      ...extractContentTopicTags(post.content),
    ].map(normalizeTopicTag).filter(Boolean);
    if (!tags.length) continue;
    const createdAt = post.createdAt ? Date.parse(post.createdAt) : 0;
    const isMine = Boolean(currentUser.value?.id && post.authorId === currentUser.value.id);
    for (const tag of tags) {
      const prev = bucket.get(tag) || { count: 0, latestAt: 0, mineCount: 0, mineLatestAt: 0 };
      bucket.set(tag, {
        count: prev.count + 1,
        latestAt: Math.max(prev.latestAt, Number.isNaN(createdAt) ? 0 : createdAt),
        mineCount: prev.mineCount + (isMine ? 1 : 0),
        mineLatestAt: isMine ? Math.max(prev.mineLatestAt, Number.isNaN(createdAt) ? 0 : createdAt) : prev.mineLatestAt,
      });
    }
  }

  const ranked = [...bucket.entries()]
    .map(([tag, stat]) => {
      const hours = stat.latestAt > 0 ? Math.max(1, (now - stat.latestAt) / 3600000) : 9999;
      const freshness = 24 / Math.min(24, hours);
      const mineHours = stat.mineLatestAt > 0 ? Math.max(1, (now - stat.mineLatestAt) / 3600000) : 9999;
      const mineFreshness = stat.mineLatestAt > 0 ? 12 / Math.min(24, mineHours) : 0;
      const mineBoost = stat.mineCount > 0 ? 25 + stat.mineCount * 8 + mineFreshness : 0;
      const score = stat.count * 10 + freshness + mineBoost;
      const cleanTag = normalizeTopicTag(tag);
      return { tag: cleanTag, label: cleanTag || '未命名话题', score, mineCount: stat.mineCount };
    })
    .filter((item) => item.tag.length > 0)
    .sort((a, b) => {
      if (a.mineCount > 0 && b.mineCount === 0) return -1;
      if (a.mineCount === 0 && b.mineCount > 0) return 1;
      return b.score - a.score;
    })
    .slice(0, 24);

  return ranked;
});

const visibleInstances = computed(() => {
  return (Array.isArray(instances.value) ? instances.value : []).map((item) => ({
    ...item,
    members: String(item.members || '0 人在线'),
    latency: String(item.latency || '未探测'),
  }));
});

function applyInstanceMemberSwitch(fromName: string, toName: string) {
  // no-op: instance members should come from backend snapshots only
  void fromName;
  void toName;
}

const topicTimeline = computed(() => {
  const tag = normalizeTopicTag(selectedTopicTag.value);
  if (!tag) return [];
  return allKnownPosts.value
    .filter((post) => {
    const tags = [
      ...(Array.isArray(post.tags) ? post.tags : []),
      ...extractContentTopicTags(post.content),
    ].map(normalizeTopicTag);
    return tags.includes(tag);
    })
    .sort((a, b) => postCreatedAtTs(b) - postCreatedAtTs(a));
});

function openTopicTagFeed(tag: string) {
  selectedTopicTag.value = String(tag || '').trim();
}

function clearTopicTagFeed() {
  selectedTopicTag.value = '';
}

function backToExploreTop() {
  selectedTopicTag.value = '';
  activeExploreTab.value = 'posts';
  setSection('explore');
}

function escapeRegex(input: string) {
  return input.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function isPostMentioningCurrentUser(post: FeedCard) {
  if (!currentUser.value) return false;
  const handle = String(currentUser.value.handle || '').replace(/^@/, '').trim();
  const instance = String(currentUser.value.instance || '').trim();
  if (!handle) return false;

  const escapedHandle = escapeRegex(handle);
  const escapedInstance = escapeRegex(instance);
  const fullPattern = new RegExp(`(^|\\s)@${escapedHandle}@${escapedInstance}(?=\\s|$)`, 'i');
  const shortPattern = new RegExp(`(^|\\s)@${escapedHandle}(?=\\s|$)`, 'i');
  const content = String(post.content || '');
  return fullPattern.test(content) || shortPattern.test(content);
}

const mentionItems = computed(() => {
  if (!currentUser.value) return [];
  return allKnownPosts.value
    .filter((post) => post.authorId !== currentUser.value?.id)
    .filter(isPostMentioningCurrentUser)
    .map((post) => ({
      id: post.id,
      title: `${post.author} 提及了你`,
      body: post.content,
      time: post.time || '刚刚',
    }));
});

const outgoingMentionItems = computed(() => {
  if (!currentUser.value) return [];
  return allKnownPosts.value
    .filter((post) => post.authorId === currentUser.value?.id)
    .filter((post) => {
      const content = String(post.content || '');
      return /(^|\s)@[^\s@]+/.test(content);
    })
    .sort((a, b) => postCreatedAtTs(b) - postCreatedAtTs(a))
    .map((post) => ({
      id: `outgoing-${post.id}`,
      postId: post.id,
      title: '你提及了其他用户',
      body: post.content,
      time: post.time || '刚刚',
    }));
});

const unreadNotificationCount = computed(() =>
  orderedNotificationItems.value.filter((item) => !readNotificationIds.value[item.id]).length,
);

const unreadMentionCount = computed(() =>
  mentionItems.value.filter((item) => !readMentionIds.value[item.id]).length,
);

function latestPeerMessageTimestamp(conversation: ConversationCard) {
  const latest = [...conversation.messages]
    .filter((message) => message.from === 'peer')
    .sort((a, b) => {
      const left = a.createdAt ? Date.parse(a.createdAt) : 0;
      const right = b.createdAt ? Date.parse(b.createdAt) : 0;
      return right - left;
    })[0];
  if (!latest) return 0;
  if (latest.createdAt) {
    const ts = Date.parse(latest.createdAt);
    if (!Number.isNaN(ts)) return ts;
  }
  return 0;
}

const unreadConversationCount = computed(() =>
  conversations.value.filter((conversation) => {
    const latestPeerTs = latestPeerMessageTimestamp(conversation);
    if (!latestPeerTs) return false;
    const readAtTs = readConversationAt.value[conversation.id] || 0;
    return latestPeerTs > readAtTs;
  }).length,
);

function markNotificationsAsRead() {
  if (orderedNotificationItems.value.length === 0) return;
  const next = { ...readNotificationIds.value };
  orderedNotificationItems.value.forEach((item) => {
    next[item.id] = true;
  });
  readNotificationIds.value = next;
  persistReadState();
}

function markMentionsAsRead() {
  if (mentionItems.value.length === 0) return;
  const next = { ...readMentionIds.value };
  mentionItems.value.forEach((item) => {
    next[item.id] = true;
  });
  readMentionIds.value = next;
  persistReadState();
}

function markConversationAsRead(conversationId: string) {
  const conversation = conversations.value.find((item) => item.id === conversationId);
  if (!conversation) return;
  const latestPeerTs = latestPeerMessageTimestamp(conversation);
  const seenAt = latestPeerTs || Date.now();
  if ((readConversationAt.value[conversationId] || 0) >= seenAt) return;
  readConversationAt.value = {
    ...readConversationAt.value,
    [conversationId]: seenAt,
  };
  persistReadState();
}

// themeStyles moved to composable

// appearance computed removed

// appearance watches removed

function setSection(section: Section) {
  if (section === 'notifications' || section === 'mentions') {
    loadNotificationSettings();
  }
  currentSection.value = section;
  // settings transition removed
  if (section !== 'postDetail') {
    threadError.value = '';
    threadLoading.value = false;
    replyDraft.value = '';
    activeReplyTarget.value = null;
  }
}

function updatePostInList(list: FeedCard[], postId: string, updater: (post: FeedCard) => FeedCard) {
  return list.map((item) => (item.id === postId ? updater(item) : item));
}

function upsertPostEngagement(postId: string, likes: number, bookmarks: number) {
  if (!postId) return;
  postEngagement.value = {
    ...postEngagement.value,
    [postId]: {
      likes: Math.max(0, Number(likes || 0)),
      bookmarks: Math.max(0, Number(bookmarks || 0)),
    },
  };
}

function bumpPostStatEverywhere(postId: string, field: 'likes' | 'bookmarks', delta: number) {
  const current = postEngagement.value[postId] || { likes: 0, bookmarks: 0 };
  const nextLikes = field === 'likes' ? Math.max(0, current.likes + delta) : current.likes;
  const nextBookmarks = field === 'bookmarks' ? Math.max(0, current.bookmarks + delta) : current.bookmarks;
  upsertPostEngagement(postId, nextLikes, nextBookmarks);
  const apply = (post: FeedCard): FeedCard => ({
    ...post,
    stats: {
      ...post.stats,
      likes: field === 'likes' ? nextLikes : Math.max(0, Number(post.stats.likes || 0)),
      bookmarks: field === 'bookmarks' ? nextBookmarks : Math.max(0, Number(post.stats.bookmarks || 0)),
    },
  });
  posts.value = updatePostInList(posts.value, postId, apply);
  myPosts.value = updatePostInList(myPosts.value, postId, apply);
  explorePostsTimeline.value = updatePostInList(explorePostsTimeline.value, postId, apply);
  newsTimeline.value = updatePostInList(newsTimeline.value, postId, apply);
  threadAncestors.value = updatePostInList(threadAncestors.value, postId, apply);
  threadReplies.value = updatePostInList(threadReplies.value, postId, apply);
  if (threadFocusPost.value?.id === postId) {
    threadFocusPost.value = apply(threadFocusPost.value);
  }
}

function toggleLike(postId: string) {
  const next = !likedPosts.value[postId];
  likedPosts.value = { ...likedPosts.value, [postId]: next };
  bumpPostStatEverywhere(postId, 'likes', next ? 1 : -1);
}

function applyUpdatedPostEverywhere(updatedPost: SocialPost) {
  const card = toFeedCard(updatedPost);
  upsertPostEngagement(card.id, card.stats.likes, card.stats.bookmarks);
  posts.value = posts.value.map((item) => (item.id === card.id ? card : item));
  myPosts.value = myPosts.value.map((item) => (item.id === card.id ? card : item));
  explorePostsTimeline.value = explorePostsTimeline.value.map((item) => (item.id === card.id ? card : item));
  newsTimeline.value = newsTimeline.value.map((item) => (item.id === card.id ? card : item));
  threadAncestors.value = threadAncestors.value.map((item) => (item.id === card.id ? card : item));
  threadReplies.value = threadReplies.value.map((item) => (item.id === card.id ? card : item));
  if (threadFocusPost.value?.id === card.id) {
    threadFocusPost.value = card;
  }
}

async function toggleBookmark(postId: string) {
  if (!postId || bookmarkActionLoading.value[postId]) return;
  const next = !bookmarkedPosts.value[postId];
  bookmarkActionLoading.value = { ...bookmarkActionLoading.value, [postId]: true };
  try {
    const updated = next ? await bookmarkPost(postId) : await unbookmarkPost(postId);
    bookmarkedPosts.value = { ...bookmarkedPosts.value, [postId]: next };
    applyUpdatedPostEverywhere(updated);
    if (!updated) {
      bumpPostStatEverywhere(postId, 'bookmarks', next ? 1 : -1);
    }
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    errorMessage.value = '收藏操作失败，请稍后重试。';
  } finally {
    bookmarkActionLoading.value = { ...bookmarkActionLoading.value, [postId]: false };
  }
}

function likeStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${LIKE_STORAGE_PREFIX}:${userId}` : '';
}

function bookmarkStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${BOOKMARK_STORAGE_PREFIX}:${userId}` : '';
}

function postEngagementStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${POST_ENGAGEMENT_STORAGE_PREFIX}:${userId}` : '';
}

function loadInteractionState() {
  if (typeof window === 'undefined') return;
  const likeKey = likeStorageKey();
  const bookmarkKey = bookmarkStorageKey();
  const engagementKey = postEngagementStorageKey();
  likedPosts.value = {};
  bookmarkedPosts.value = {};
  postEngagement.value = {};
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
  if (engagementKey) {
    try {
      postEngagement.value = JSON.parse(window.localStorage.getItem(engagementKey) || '{}');
    } catch {
      postEngagement.value = {};
    }
  }
}

function persistInteractionState() {
  if (typeof window === 'undefined') return;
  const likeKey = likeStorageKey();
  const bookmarkKey = bookmarkStorageKey();
  const engagementKey = postEngagementStorageKey();
  if (likeKey) {
    window.localStorage.setItem(likeKey, JSON.stringify(likedPosts.value));
  }
  if (bookmarkKey) {
    window.localStorage.setItem(bookmarkKey, JSON.stringify(bookmarkedPosts.value));
  }
  if (engagementKey) {
    window.localStorage.setItem(engagementKey, JSON.stringify(postEngagement.value));
  }
}

function notificationReadStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${NOTIFICATION_READ_STORAGE_PREFIX}:${userId}` : '';
}

function mentionReadStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${MENTION_READ_STORAGE_PREFIX}:${userId}` : '';
}

function messageReadAtStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${MESSAGE_READ_AT_STORAGE_PREFIX}:${userId}` : '';
}

function encryptedMessageCacheStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${ENCRYPTED_MESSAGE_CACHE_STORAGE_PREFIX}:${userId}` : '';
}

function messageDeviceIdStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${MESSAGE_DEVICE_ID_STORAGE_PREFIX}:${userId}` : '';
}

function messageEncryptionKeyStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${MESSAGE_ENCRYPTION_KEY_STORAGE_PREFIX}:${userId}` : '';
}

function bytesToBase64(bytes: Uint8Array) {
  let raw = '';
  bytes.forEach((v) => {
    raw += String.fromCharCode(v);
  });
  return btoa(raw);
}

function base64ToBytes(base64: string) {
  const raw = atob(base64);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i += 1) out[i] = raw.charCodeAt(i);
  return out;
}

function randomBytes(length: number) {
  if (typeof window === 'undefined') return new Uint8Array(length);
  const bytes = new Uint8Array(length);
  window.crypto.getRandomValues(bytes);
  return bytes;
}

function getOrCreateMessageDeviceId() {
  if (typeof window === 'undefined') return 'dev_unknown';
  const key = messageDeviceIdStorageKey();
  if (!key) return 'dev_unknown';
  const existing = window.localStorage.getItem(key);
  if (existing) return existing;
  const id = `dev_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
  window.localStorage.setItem(key, id);
  return id;
}

async function getOrCreateMessageCryptoKey() {
  if (typeof window === 'undefined') return null;
  const keyStorageKey = messageEncryptionKeyStorageKey();
  if (!keyStorageKey) return null;
  const rawExisting = window.localStorage.getItem(keyStorageKey);
  if (rawExisting) {
    const imported = await window.crypto.subtle.importKey(
      'raw',
      base64ToBytes(rawExisting),
      'AES-GCM',
      false,
      ['encrypt', 'decrypt'],
    );
    return imported;
  }
  const generated = randomBytes(32);
  window.localStorage.setItem(keyStorageKey, bytesToBase64(generated));
  return window.crypto.subtle.importKey('raw', generated, 'AES-GCM', false, ['encrypt', 'decrypt']);
}

function loadEncryptedMessageCache() {
  if (typeof window === 'undefined') return {} as Record<string, EncryptedMessageCacheEntry>;
  const key = encryptedMessageCacheStorageKey();
  if (!key) return {} as Record<string, EncryptedMessageCacheEntry>;
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return {} as Record<string, EncryptedMessageCacheEntry>;
    const parsed = JSON.parse(raw) as Record<string, EncryptedMessageCacheEntry>;
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch {
    return {} as Record<string, EncryptedMessageCacheEntry>;
  }
}

function persistEncryptedMessageCache(cache: Record<string, EncryptedMessageCacheEntry>) {
  if (typeof window === 'undefined') return;
  const key = encryptedMessageCacheStorageKey();
  if (!key) return;
  window.localStorage.setItem(key, JSON.stringify(cache));
}

async function cacheEncryptedConversationPayload(conversationId: string, messageBody: string) {
  if (!conversationId) return;
  if (typeof window === 'undefined') return;
  const cryptoKey = await getOrCreateMessageCryptoKey();
  if (!cryptoKey) return;
  const iv = randomBytes(12);
  const encoded = new TextEncoder().encode(messageBody);
  const encrypted = await window.crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    cryptoKey,
    encoded,
  );
  const encryptedBytes = new Uint8Array(encrypted);
  const tagLength = 16;
  const tag = encryptedBytes.slice(encryptedBytes.length - tagLength);
  const ciphertext = encryptedBytes.slice(0, encryptedBytes.length - tagLength);
  const cache = loadEncryptedMessageCache();
  cache[conversationId] = {
    version: 1,
    ciphertext: bytesToBase64(ciphertext),
    iv: bytesToBase64(iv),
    tag: bytesToBase64(tag),
    algorithm: 'AES-256-GCM',
    createdAt: Date.now(),
    senderDeviceId: getOrCreateMessageDeviceId(),
  };
  persistEncryptedMessageCache(cache);
}

function followerSeenIdsStorageKey() {
  const userId = authSession.value?.id;
  return userId ? `${FOLLOWER_SEEN_IDS_STORAGE_PREFIX}:${userId}` : '';
}

const readNotificationIds = ref<Record<string, boolean>>({});
const readMentionIds = ref<Record<string, boolean>>({});
const readConversationAt = ref<Record<string, number>>({});
const seenFollowerIds = ref<string[]>([]);
const followerNotifications = ref<NotificationItem[]>([]);
const notificationSettings = ref<NotificationSettings>({ ...defaultNotificationSettings });

function loadNotificationSettings() {
  if (typeof window === 'undefined') return;
  try {
    const raw = window.localStorage.getItem(NOTIFICATION_SETTINGS_STORAGE_KEY);
    if (!raw) {
      notificationSettings.value = { ...defaultNotificationSettings };
      return;
    }
    const parsed = JSON.parse(raw) as Partial<NotificationSettings>;
    notificationSettings.value = { ...defaultNotificationSettings, ...parsed };
  } catch {
    notificationSettings.value = { ...defaultNotificationSettings };
  }
}

function handleStorageChange(event: StorageEvent) {
  if (event.key === NOTIFICATION_SETTINGS_STORAGE_KEY) {
    loadNotificationSettings();
  }
}

function loadReadState() {
  if (typeof window === 'undefined') return;
  readNotificationIds.value = {};
  readMentionIds.value = {};
  readConversationAt.value = {};
  seenFollowerIds.value = [];
  const notificationKey = notificationReadStorageKey();
  const mentionKey = mentionReadStorageKey();
  const messageReadAtKey = messageReadAtStorageKey();
  if (notificationKey) {
    try {
      readNotificationIds.value = JSON.parse(window.localStorage.getItem(notificationKey) || '{}');
    } catch {
      readNotificationIds.value = {};
    }
  }
  if (mentionKey) {
    try {
      readMentionIds.value = JSON.parse(window.localStorage.getItem(mentionKey) || '{}');
    } catch {
      readMentionIds.value = {};
    }
  }
  if (messageReadAtKey) {
    try {
      readConversationAt.value = JSON.parse(window.localStorage.getItem(messageReadAtKey) || '{}');
    } catch {
      readConversationAt.value = {};
    }
  }
  const followerSeenKey = followerSeenIdsStorageKey();
  if (followerSeenKey) {
    try {
      const raw = window.localStorage.getItem(followerSeenKey);
      seenFollowerIds.value = raw ? JSON.parse(raw) : [];
    } catch {
      seenFollowerIds.value = [];
    }
  }
}

function persistReadState() {
  if (typeof window === 'undefined') return;
  const notificationKey = notificationReadStorageKey();
  const mentionKey = mentionReadStorageKey();
  const messageReadAtKey = messageReadAtStorageKey();
  if (notificationKey) {
    window.localStorage.setItem(notificationKey, JSON.stringify(readNotificationIds.value));
  }
  if (mentionKey) {
    window.localStorage.setItem(mentionKey, JSON.stringify(readMentionIds.value));
  }
  if (messageReadAtKey) {
    window.localStorage.setItem(messageReadAtKey, JSON.stringify(readConversationAt.value));
  }
  const followerSeenKey = followerSeenIdsStorageKey();
  if (followerSeenKey) {
    window.localStorage.setItem(followerSeenKey, JSON.stringify(seenFollowerIds.value));
  }
}

function toForwardedPostBody(post: FeedCard) {
  const payload = {
    id: post.id,
    author: post.author,
    handle: post.handle,
    instance: post.instance,
    content: post.content,
    createdAt: post.createdAt || post.time,
    media: post.media
      ? {
          name: post.media.name,
          preview: post.media.preview,
          type: post.media.type,
          sizeLabel: post.media.sizeLabel,
        }
      : null,
  };
  return `${FORWARDED_POST_PREFIX}${JSON.stringify(payload)}`;
}

function toMessageImageBody(text: string, image: { name: string; preview: string; type: string; sizeLabel: string }) {
  const payload = {
    text,
    media: image,
  };
  return `${MESSAGE_IMAGE_EMOJI_PREFIX}${JSON.stringify(payload)}`;
}

function parseMessageImageBody(text: string) {
  if (!text.startsWith(MESSAGE_IMAGE_EMOJI_PREFIX)) return null;
  const raw = text.slice(MESSAGE_IMAGE_EMOJI_PREFIX.length);
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return null;
    const media = parsed.media && typeof parsed.media === 'object'
      ? {
          name: String(parsed.media.name || ''),
          preview: String(parsed.media.preview || ''),
          type: String(parsed.media.type || 'image'),
          sizeLabel: String(parsed.media.sizeLabel || ''),
        }
      : null;
    if (!media?.preview) return null;
    return {
      text: String(parsed.text || ''),
      media,
    };
  } catch {
    return null;
  }
}

function parseForwardedPostBody(text: string) {
  if (!text.startsWith(FORWARDED_POST_PREFIX)) return null;
  const raw = text.slice(FORWARDED_POST_PREFIX.length);
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return null;
    return {
      id: String(parsed.id || ''),
      author: String(parsed.author || ''),
      handle: String(parsed.handle || ''),
      instance: String(parsed.instance || ''),
      content: String(parsed.content || ''),
      createdAt: typeof parsed.createdAt === 'string' ? parsed.createdAt : '',
      media: parsed.media && typeof parsed.media === 'object'
        ? {
            name: String(parsed.media.name || ''),
            preview: String(parsed.media.preview || ''),
            type: String(parsed.media.type || 'image'),
            sizeLabel: String(parsed.media.sizeLabel || ''),
          }
        : undefined,
    };
  } catch {
    return null;
  }
}

function forwardedSummaryText(forwardedPost: MessageCard['forwardedPost'], from: 'me' | 'peer') {
  if (!forwardedPost) return '';
  const actor = from === 'me' ? '你' : '对方';
  const sourcePost = posts.value.find((item) => item.id === forwardedPost.id)
    || myPosts.value.find((item) => item.id === forwardedPost.id)
    || threadAncestors.value.find((item) => item.id === forwardedPost.id)
    || threadReplies.value.find((item) => item.id === forwardedPost.id)
    || (threadFocusPost.value?.id === forwardedPost.id ? threadFocusPost.value : null);
  const byHandle = people.value.find((person) => person.handle === forwardedPost.handle)
    || (currentUser.value?.handle === forwardedPost.handle ? currentUser.value : null);
  const owner = sourcePost?.author || byHandle?.displayName || forwardedPost.author || '某用户';
  const raw = String(forwardedPost.content || '').trim();
  const brief = raw.length > 36 ? `${raw.slice(0, 36)}...` : raw || '帖子';
  return `${actor}转发了${owner}的《${brief}》`;
}

function forwardedPostAuthorName(forwardedPost: MessageCard['forwardedPost']) {
  if (!forwardedPost) return '某用户';
  const sourcePost = posts.value.find((item) => item.id === forwardedPost.id)
    || myPosts.value.find((item) => item.id === forwardedPost.id)
    || threadAncestors.value.find((item) => item.id === forwardedPost.id)
    || threadReplies.value.find((item) => item.id === forwardedPost.id)
    || (threadFocusPost.value?.id === forwardedPost.id ? threadFocusPost.value : null);
  const byHandle = people.value.find((person) => person.handle === forwardedPost.handle)
    || (currentUser.value?.handle === forwardedPost.handle ? currentUser.value : null);
  return sourcePost?.author || byHandle?.displayName || forwardedPost.author || '某用户';
}

function openForwardedPostDetail(message: MessageCard) {
  if (!message.forwardedPost?.id) return;
  void openPostDetail(message.forwardedPost.id, false);
}

// toneClass removed - moved to AppearanceSettings.vue

// settings functions removed - moved to SettingsPage.vue / AppearanceSettings.vue

function formatTimestamp(input: string) {
  if (!input) return '刚刚';
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
    return date.toLocaleString(locale, {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  }
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 MB';
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

function avatarText(name: string) {
  return name.slice(0, 1).toUpperCase();
}

function findPersonById(userId: string) {
  if (!userId) return null;
  if (currentUser.value?.id === userId) return currentUser.value;
  return people.value.find((item) => item.id === userId) ?? null;
}

function hydrateFeedCardAuthor(post: FeedCard): FeedCard {
  const person = findPersonById(post.authorId);
  if (!person) return post;
  return {
    ...post,
    author: person.displayName || post.author,
    handle: person.handle || post.handle,
    bio: person.bio || post.bio,
    instance: person.instance || post.instance,
  };
}

function hydrateFeedCardList(items: FeedCard[]) {
  return items.map(hydrateFeedCardAuthor);
}

function userAvatarUrl(userId: string) {
  return findPersonById(userId)?.avatarUrl || '';
}

function userBackgroundUrl(userId: string) {
  return findPersonById(userId)?.backgroundUrl || '';
}

function formatHandleInstance(handle: string, instance: string) {
  const normalizedInstance = String(instance || '').trim();
  if (normalizedInstance === '摩尔1号') {
    return `@${normalizedInstance}`;
  }
  return `${handle}@${normalizedInstance}`;
}

function profileLabel(user: SocialUser | null) {
  if (!user) return '';
  return `@${activeProfileInstanceName.value}`;
}

function resolveAuthenticatedUser(users: SocialUser[]) {
  const sessionUser = authSession.value;
  if (!sessionUser) return null;

  const matchedUser = users.find((user) => user.id === sessionUser.id);
  if (matchedUser) return matchedUser;

  return {
    id: sessionUser.id,
    handle: sessionUser.handle,
    displayName: sessionUser.displayName,
    bio: sessionUser.bio,
    instance: sessionUser.instance,
    wallet: '0xauth',
    avatarUrl: sessionUser.avatarUrl,
    fields: sessionUser.fields || [],
    featuredTags: sessionUser.featuredTags || [],
    isBot: sessionUser.isBot || false,
    followers: 0,
    following: 0,
    createdAt: new Date().toISOString(),
  } satisfies SocialUser;
}

function goToLogout() {
  void router.push('/logout');
}

function toFeedCard(post: SocialPost): FeedCard {
  const firstMedia = Array.isArray(post.media) ? post.media[0] : undefined;
  const person = findPersonById(post.authorId);
  const savedEngagement = postEngagement.value[post.id];
  const likes = savedEngagement ? Math.max(savedEngagement.likes, Number(post.likes || 0)) : Number(post.likes || 0);
  const bookmarks = savedEngagement ? Math.max(savedEngagement.bookmarks, Number(post.bookmarks || 0)) : Number(post.bookmarks || 0);
  return {
    id: post.id,
    authorId: post.authorId,
    author: person?.displayName || post.authorName,
    handle: person?.handle || post.authorHandle,
    instance: person?.instance || post.instance,
    kind: post.kind || (post.parentPostId ? 'reply' : 'post'),
    parentPostId: post.parentPostId,
    rootPostId: post.rootPostId,
    replyDepth: post.replyDepth ?? 0,
    time: formatTimestamp(post.createdAt),
    content: post.content,
    type: post.type || 'post',
    interaction: post.interaction || 'anyone',
    createdAt: post.createdAt,
    bio: person?.bio,
    isBot: person?.isBot || false,
    tags: post.tags,
    chainProof: post.txHash || post.attestationUri || post.storageUri || 'unverified://pending',
    chainId: post.chainId,
    txHash: post.txHash,
    contractAddress: post.contractAddress,
    explorerUrl: post.explorerUrl,
    media: firstMedia
      ? {
          name: firstMedia.name,
          preview: firstMedia.url,
          type: firstMedia.kind,
          sizeLabel: '已同步',
        }
      : undefined,
    stats: {
      replies: post.replies,
      boosts: post.boosts,
      likes,
      bookmarks,
    },
    poll: post.poll,
  };
}

function toAssetCard(asset: MediaAsset): AssetCard {
  return {
    id: asset.id,
    title: asset.name,
    network: asset.storageUri ? `Indexed · ${asset.storageUri}` : 'Uploaded',
    cid: asset.cid || asset.storageUri || 'pending',
    size: formatBytes(asset.sizeBytes),
    retention: asset.status,
    url: asset.url,
  };
}

function toConversationCard(conversation: SocialConversation, userId: string | null): ConversationCard {
  const participantIds = [...new Set(conversation.participantIds.filter(Boolean))];
  const otherParticipantIds = participantIds.filter((id) => id !== userId);
  const participantUsers = participantIds
    .map((participantId) => people.value.find((person) => person.id === participantId))
    .filter((person): person is SocialUser => Boolean(person));
  const otherParticipants = otherParticipantIds
    .map((participantId) => people.value.find((person) => person.id === participantId))
    .filter((person): person is SocialUser => Boolean(person));

  const fallbackPeer = participantUsers.find((person) => person.id !== userId) ?? otherParticipants[0] ?? null;
  const displayParticipants = otherParticipants.length ? otherParticipants : fallbackPeer ? [fallbackPeer] : [];

  const normalizedTitle = conversation.title
    .trim()
    .replace(/^跨联邦(?:会话)?\s*[-:：]?\s*/u, '');
  const resolvedTitle =
    normalizedTitle ||
    displayParticipants.map((person) => person.displayName).join('、') ||
    otherParticipantIds.join(', ') ||
    participantIds.join(', ') ||
    '新会话';

  const resolvedHandle =
    displayParticipants.map((person) => formatHandleInstance(person.handle, person.instance)).join(', ') ||
    (fallbackPeer ? formatHandleInstance(fallbackPeer.handle, fallbackPeer.instance) : '') ||
    otherParticipantIds.join(', ') ||
    participantIds.join(', ');

  const selfUser = userId ? (people.value.find((person) => person.id === userId) || currentUser.value) : currentUser.value;
  const selfInstance = String(selfUser?.instance || '').trim();
  const peerInstance = String(fallbackPeer?.instance || '').trim();
  const resolvedRoute = conversation.crossInstance
    ? (selfInstance && peerInstance ? `${selfInstance} → ${peerInstance}` : (conversation.federationRoute || ''))
    : '';

  return {
    id: conversation.id,
    name: resolvedTitle,
    handle: resolvedHandle,
    status: conversation.crossInstance ? '跨联邦会话' : conversation.encrypted ? '端到端加密会话' : '同实例会话',
    crossInstance: Boolean(conversation.crossInstance),
    federationRoute: resolvedRoute,
    assetUri: conversation.assetUri,
    chainId: conversation.chainId,
    txHash: conversation.txHash,
    contractAddress: conversation.contractAddress,
    explorerUrl: conversation.explorerUrl,
    avatarUrl: fallbackPeer?.avatarUrl,
    backgroundUrl: fallbackPeer?.backgroundUrl,
    avatarLabel: avatarText(resolvedTitle),
    participantId: fallbackPeer?.id,
    messages: conversation.messages.map((message) => {
      const from: 'me' | 'peer' = message.senderId === userId ? 'me' : 'peer';
      const forwardedPost = parseForwardedPostBody(message.body);
      const imageEmoji = forwardedPost ? null : parseMessageImageBody(message.body);
      return {
      id: message.id,
      from,
      text: forwardedPost
        ? forwardedSummaryText(forwardedPost, from)
        : (imageEmoji?.text || (imageEmoji ? '[图片表情]' : message.body)),
      imageEmoji: imageEmoji?.media || null,
      forwardedPost,
      time: formatTimestamp(message.createdAt),
      createdAt: message.createdAt,
      assetUri: message.assetUri,
      chainId: message.chainId,
      txHash: message.txHash,
      contractAddress: message.contractAddress,
      explorerUrl: message.explorerUrl,
      };
    }),
  };
}

function syncNotificationFeed() {
  const existing = new Map(notificationFeed.value.map((item) => [item.id, item]));
  for (const item of notificationItems.value) {
    if (!existing.has(item.id)) {
      existing.set(item.id, item);
    }
  }
  notificationFeed.value = [...existing.values()]
    .sort((a, b) => b.sortAt - a.sortAt)
    .slice(0, 200);
}

function conversationLatestAt(conversation: ConversationCard) {
  const latest = conversation.messages[conversation.messages.length - 1];
  if (!latest?.createdAt) return 0;
  const ts = Date.parse(latest.createdAt);
  return Number.isNaN(ts) ? 0 : ts;
}

function sortConversationsByLatestMessage(items: ConversationCard[]) {
  return [...items].sort((a, b) => {
    const delta = conversationLatestAt(b) - conversationLatestAt(a);
    if (delta !== 0) return delta;
    return b.id.localeCompare(a.id);
  });
}

function upsertConversation(conversation: ConversationCard) {
  if (!conversation.participantId || conversation.participantId === currentUser.value?.id) return;
  const remaining = conversations.value.filter((item) => item.id !== conversation.id);
  conversations.value = sortConversationsByLatestMessage([conversation, ...remaining]);
}

async function scrollMessagesToBottom() {
  await nextTick();
  if (!messageListRef.value) return;
  messageListRef.value.scrollTop = messageListRef.value.scrollHeight;
}

async function loadConversationMessages(conversationId: string) {
  if (!currentUser.value) return;
  if (!apiOnline.value) return;
  try {
    const detail = await getConversation(conversationId);
    const mapped = toConversationCard(detail, currentUser.value.id);
    upsertConversation(mapped);
    selectedConversationId.value = mapped.id;
    await scrollMessagesToBottom();
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    errorMessage.value = '会话加载失败，请稍后重试。';
  }
}

async function refreshConversations(keepSelection = true) {
  if (!currentUser.value) return;
  if (!apiOnline.value) return;
  try {
    const items = await listConversations(50);
    const mapped = items
      .map((conversation) => toConversationCard(conversation, currentUser.value?.id ?? null))
      .filter((conversation) => conversation.participantId && conversation.participantId !== currentUser.value?.id);
    conversations.value = sortConversationsByLatestMessage(mapped);
    if (!keepSelection || !selectedConversationId.value) {
      selectedConversationId.value = conversations.value[0]?.id ?? '';
    } else if (!conversations.value.find((item) => item.id === selectedConversationId.value)) {
      selectedConversationId.value = conversations.value[0]?.id ?? '';
    }
  } catch (error) {
    errorMessage.value = '会话列表刷新失败，请稍后再试。';
  }
}

async function openConversation(conversationId: string) {
  selectedConversationId.value = conversationId;
  markConversationAsRead(conversationId);
  await loadConversationMessages(conversationId);
  markConversationAsRead(conversationId);
}

function findDirectConversationWith(userId: string) {
  return conversations.value.find((conversation) => conversation.participantId === userId);
}

function goToNotFound() {
  void router.replace('/404');
}

async function startConversation(targetUser: SocialUser) {
  if (!currentUser.value || saving.value) return;

  const existingConversation = findDirectConversationWith(targetUser.id);
  if (existingConversation) {
    selectedConversationId.value = existingConversation.id;
    currentSection.value = 'messages';
    errorMessage.value = '';
    return;
  }

  saving.value = true;
  try {
    if (!apiOnline.value) {
      goToNotFound();
      return;
    }

    const createdConversation = await createConversation({
      title: targetUser.displayName,
      participantIds: [targetUser.id],
      encrypted: false,
    });

    const mappedConversation = toConversationCard(createdConversation, currentUser.value.id);
    upsertConversation(mappedConversation);
    selectedConversationId.value = mappedConversation.id;

    currentSection.value = 'messages';
    messageDraft.value = '';
    errorMessage.value = '';
    await scrollMessagesToBottom();
  } catch (error) {
    if (error instanceof ApiError && error.code === 'AUTH_SESSION_REQUIRED') {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    goToNotFound();
  } finally {
    saving.value = false;
  }
}

function applyBootstrap(payload: BootstrapPayload) {
  currentUser.value = resolveAuthenticatedUser(payload.users) ?? payload.currentUser ?? payload.users[0] ?? null;
  loadRecentMessageStickers();
  people.value = payload.users;
  assets.value = payload.media.map(toAssetCard);
  const mappedConversations = payload.conversations
    .map((conversation) => toConversationCard(conversation, currentUser.value?.id ?? null))
    .filter((conversation) => conversation.participantId && conversation.participantId !== currentUser.value?.id);
  conversations.value = sortConversationsByLatestMessage(mappedConversations);
  instances.value = payload.instances;
  if (instances.value.length > 0) {
    const currentInstance = String(currentUser.value?.instance || '').trim();
    const hasCurrent = instances.value.some((item) => item.name === currentInstance);
    const hasSelected = instances.value.some((item) => item.name === selectedInstanceName.value);
    if (selectedInstanceName.value === 'all') {
      selectedInstanceName.value = hasCurrent ? currentInstance : instances.value[0].name;
    } else if (!hasSelected) {
      selectedInstanceName.value = hasCurrent ? currentInstance : instances.value[0].name;
    }
  }
  selectedConversationId.value = conversations.value[0]?.id ?? '';
  void syncFollowerNotifications();
}

watch([people, currentUser], () => {
  posts.value = hydrateFeedCardList(posts.value);
  myPosts.value = hydrateFeedCardList(myPosts.value);
  threadAncestors.value = hydrateFeedCardList(threadAncestors.value);
  threadReplies.value = hydrateFeedCardList(threadReplies.value);
  if (threadFocusPost.value) {
    threadFocusPost.value = hydrateFeedCardAuthor(threadFocusPost.value);
  }
  if (activeReplyTarget.value) {
    activeReplyTarget.value = hydrateFeedCardAuthor(activeReplyTarget.value);
  }
});

async function loadMyFeed() {
  if (!authSession.value) {
    myPosts.value = [];
    return;
  }

  const payload = await fetchSocialBootstrapMine(DEFAULT_FEED_LIMIT);
  myPosts.value = payload.feed.map(toFeedCard);
}

function bumpReplyCount(postId: string) {
  posts.value = posts.value.map((post) =>
    post.id === postId
      ? {
          ...post,
          stats: {
            ...post.stats,
            replies: post.stats.replies + 1,
          },
        }
      : post,
  );

  threadAncestors.value = threadAncestors.value.map((post) =>
    post.id === postId
      ? {
          ...post,
          stats: {
            ...post.stats,
            replies: post.stats.replies + 1,
          },
        }
      : post,
  );

  threadReplies.value = threadReplies.value.map((post) =>
    post.id === postId
      ? {
          ...post,
          stats: {
            ...post.stats,
            replies: post.stats.replies + 1,
          },
        }
      : post,
  );

  if (threadFocusPost.value?.id === postId) {
    threadFocusPost.value = {
      ...threadFocusPost.value,
      stats: {
        ...threadFocusPost.value.stats,
        replies: threadFocusPost.value.stats.replies + 1,
      },
    };
  }
}

function setReplyTarget(target: FeedCard) {
  if (activeReplyTarget.value?.id !== target.id) {
    replyDraft.value = '';
    clearReplyMedia();
    showReplyEmojiPicker.value = false;
  }
  activeReplyTarget.value = target;
  void focusReplyComposer();
}

async function focusReplyComposer() {
  await nextTick();
  const textarea = document.querySelector('textarea[placeholder^="回复"]');
  if (textarea instanceof HTMLTextAreaElement) {
    textarea.focus();
  }
}

function openVisibilityModal() {
  tempVisibility.value = visibility.value;
  tempInteraction.value = interaction.value;
  showVisibilityModal.value = true;
}

function closeVisibilityModal() {
  showVisibilityModal.value = false;
}

function saveVisibilitySettings() {
  visibility.value = tempVisibility.value;
  interaction.value = tempInteraction.value;
  persistPostingPrivacySettings();
  showVisibilityModal.value = false;
}

function normalizeInteractionFromPrivacy(value: string) {
  if (value === 'none') return 'me';
  if (value === 'followers') return 'followers';
  return 'anyone';
}

function normalizeVisibility(value: string) {
  if (['public', 'unlisted', 'private', 'direct'].includes(value)) return value;
  return 'public';
}

function normalizeInteraction(value: string) {
  if (['anyone', 'followers', 'me'].includes(value)) return value;
  return 'anyone';
}

function loadPostingPrivacySettings() {
  if (typeof window === 'undefined') return;
  try {
    const rawPosting = window.localStorage.getItem(POSTING_PRIVACY_STORAGE_KEY);
    if (rawPosting) {
      const parsed = JSON.parse(rawPosting) as Partial<{ visibility: string; interaction: string }>;
      visibility.value = normalizeVisibility(String(parsed.visibility || 'public'));
      interaction.value = normalizeInteraction(String(parsed.interaction || 'anyone'));
      return;
    }
  } catch {
    // ignore invalid local storage data
  }

  try {
    const rawPrivacy = window.localStorage.getItem(PRIVACY_SETTINGS_STORAGE_KEY);
    if (!rawPrivacy) return;
    const parsed = JSON.parse(rawPrivacy) as Partial<{ allowQuoteFrom: string }>;
    interaction.value = normalizeInteractionFromPrivacy(String(parsed.allowQuoteFrom || 'anyone'));
  } catch {
    // ignore invalid local storage data
  }
}

function persistPostingPrivacySettings() {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(POSTING_PRIVACY_STORAGE_KEY, JSON.stringify({
    visibility: visibility.value,
    interaction: interaction.value,
  }));
}

async function loadBootstrap() {
  if (!authSession.value) {
    await router.replace({ path: '/login', query: { redirect: '/app' } });
    return;
  }

  loading.value = true;
  try {
    const payload = await fetchSocialBootstrap(DEFAULT_FEED_LIMIT);
    applyBootstrap(payload);
    apiOnline.value = true;
    errorMessage.value = '';
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      await router.replace({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    goToNotFound();
  } finally {
    loading.value = false;
  }
}

async function warmupHomeDataAfterFirstPaint() {
  await nextTick();
  await Promise.allSettled([
    loadMyFeed(),
    loadLatestPostsTimeline(true),
  ]);
}

async function handleRouteMessageIntent() {
  if (processingMessageIntent.value) return;
  const targetUserId = typeof route.query.messageUser === 'string' ? route.query.messageUser.trim() : '';
  const targetPostId = typeof route.query.post === 'string' ? route.query.post.trim() : '';
  const sharePostId = typeof route.query.sharePost === 'string' ? route.query.sharePost.trim() : '';
  if (!targetUserId && !targetPostId && !sharePostId) return;
  if (!currentUser.value) return;

  processingMessageIntent.value = true;
  try {
    if (targetUserId) {
      const targetUser = people.value.find((person) => person.id === targetUserId);
      if (targetUser && targetUser.id !== currentUser.value.id) {
        await startConversation(targetUser);
      }
    }

    if (targetPostId) {
      await openPostDetail(targetPostId, false);
    }

    if (sharePostId) {
      const targetPost = posts.value.find((post) => post.id === sharePostId)
        || myPosts.value.find((post) => post.id === sharePostId)
        || threadReplies.value.find((post) => post.id === sharePostId)
        || threadAncestors.value.find((post) => post.id === sharePostId)
        || (threadFocusPost.value?.id === sharePostId ? threadFocusPost.value : null);
      if (targetPost) {
        await openForwardDialog(targetPost);
      }
    }
  } finally {
    await router.replace('/app');
    processingMessageIntent.value = false;
  }
}

function isHomeTopReached() {
  if (currentSection.value !== 'home') return false;
  const el = mainContentRef.value;
  if (el && el.scrollHeight > el.clientHeight) {
    return el.scrollTop <= 0;
  }
  if (typeof window !== 'undefined') {
    return window.scrollY <= 0;
  }
  return true;
}

function resetPullRefreshState() {
  isPullingHome.value = false;
  pullEligible.value = false;
  pullDistance.value = 0;
}

function onHomeTouchStart(event: TouchEvent) {
  if (currentSection.value !== 'home' || isRefreshingHome.value) return;
  if (!isHomeTopReached()) return;
  const touch = event.touches[0];
  if (!touch) return;
  pullStartY.value = touch.clientY;
  pullDistance.value = 0;
  pullEligible.value = true;
  isPullingHome.value = true;
}

function onHomeTouchMove(event: TouchEvent) {
  if (!isPullingHome.value || !pullEligible.value || isRefreshingHome.value) return;
  const touch = event.touches[0];
  if (!touch) return;
  const delta = touch.clientY - pullStartY.value;
  if (delta <= 0) {
    pullDistance.value = 0;
    return;
  }
  pullDistance.value = Math.min(PULL_MAX_DISTANCE, delta * 0.45);
}

function onHomeTouchEnd() {
  if (!isPullingHome.value) return;
  const shouldRefresh = pullDistance.value >= PULL_REFRESH_THRESHOLD && pullEligible.value;
  resetPullRefreshState();
  if (shouldRefresh) {
    void refreshHomeTimeline();
  }
}

async function refreshHomeTimeline() {
  if (isRefreshingHome.value || saving.value) return;

  isRefreshingHome.value = true;
  try {
    if (!apiOnline.value) {
      goToNotFound();
      return;
    }
    const [payload, minePayload] = await Promise.all([
      fetchSocialBootstrap(DEFAULT_FEED_LIMIT),
      authSession.value ? fetchSocialBootstrapMine(DEFAULT_FEED_LIMIT) : Promise.resolve(null),
    ]);
    applyBootstrap(payload);
    myPosts.value = minePayload ? minePayload.feed.map(toFeedCard) : [];
    await loadLatestPostsTimeline(true);
    errorMessage.value = '';
    apiOnline.value = true;
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    errorMessage.value = '刷新失败，请稍后再试。';
  } finally {
    isRefreshingHome.value = false;
    pullDistance.value = 0;
  }
}

async function openPostDetail(postId: string, focusComposer = true) {
  selectedPostId.value = postId;
  currentSection.value = 'postDetail';
  threadLoading.value = true;
  threadError.value = '';
  replyDraft.value = '';
  clearReplyMedia();
  showReplyEmojiPicker.value = false;
  activeReplyTarget.value = null;
  threadFocusPost.value = null;
  threadAncestors.value = [];
  threadReplies.value = [];

  if (!apiOnline.value) {
    goToNotFound();
    return;
  }

  try {
    const [thread, replies] = await Promise.all([fetchPostThread(postId), fetchPostReplies(postId)]);
    threadFocusPost.value = toFeedCard(thread.post);
    threadAncestors.value = thread.ancestors.map(toFeedCard);
    threadReplies.value = (replies.length ? replies : thread.replies).map(toFeedCard);
    activeReplyTarget.value = toFeedCard(thread.post);
    errorMessage.value = '';
    if (focusComposer) {
      await focusReplyComposer();
    }
  } catch (error) {
    goToNotFound();
  } finally {
    threadLoading.value = false;
  }
}

async function submitReply() {
  if ((!replyDraft.value.trim() && !replyMediaPreview.value) || !currentUser.value || !threadFocusPost.value || !activeReplyTarget.value || saving.value) return;

  const targetPost = activeReplyTarget.value;
  const rootPostId = threadFocusPost.value.rootPostId || threadFocusPost.value.id;
  const pendingTargetId = targetPost.id;
  saving.value = true;
  try {
    if (!apiOnline.value) {
      goToNotFound();
      return;
    }

    let createdAsset: AssetCard | null = null;
    if (replyMediaPreview.value && replyMediaMeta.value) {
      createdAsset = await createMediaAsset({
        ownerId: currentUser.value.id,
        name: replyMediaMeta.value.name,
        kind: replyMediaMeta.value.type.startsWith('video') ? 'video' : 'image',
        url: replyMediaPreview.value,
        storageUri: `media://reply/${Date.now()}`,
        cid: `cid_${Date.now()}`,
        sizeBytes: replyMediaMeta.value.sizeBytes,
        status: 'ready',
      }).then(toAssetCard);
    }

    await createPost({
      authorId: currentUser.value.id,
      instance: targetPost.instance || activeProfileInstanceName.value,
      kind: 'reply',
      content: replyDraft.value.trim(),
      visibility: 'public',
      interaction: 'anyone',
      storageUri: `draft://reply/${Date.now()}`,
      attestationUri: `attestation://reply/${Date.now()}`,
      tags: targetPost.tags.slice(0, 3),
      mediaIds: createdAsset ? [createdAsset.id] : [],
      parentPostId: targetPost.id,
      rootPostId,
      type: 'post',
    });
    bumpReplyCount(targetPost.id);
    await openPostDetail(selectedPostId.value || rootPostId);

    replyDraft.value = '';
    clearReplyMedia();
    if (threadFocusPost.value?.id === pendingTargetId) {
      activeReplyTarget.value = threadFocusPost.value;
    } else {
      const nextTarget = threadReplies.value.find((post) => post.id === pendingTargetId);
      activeReplyTarget.value = nextTarget || threadFocusPost.value;
    }
    errorMessage.value = '';
    threadError.value = '';
    await focusReplyComposer();
  } catch (error) {
    goToNotFound();
  } finally {
    saving.value = false;
  }
}

async function handleVote(post: FeedCard, optionIndices: number[]) {
  if (!currentUser.value) return;
  if (!apiOnline.value) {
    goToNotFound();
    return;
  }
  try {
    const updatedPost = await voteOnPoll(post.id, optionIndices);
    // Update local state
    const postIdx = posts.value.findIndex(p => p.id === post.id);
    if (postIdx !== -1) {
      posts.value[postIdx] = toFeedCard(updatedPost);
    }
    if (threadFocusPost.value?.id === post.id) {
      threadFocusPost.value = toFeedCard(updatedPost);
    }
  } catch {
    goToNotFound();
  }
}

async function refreshPost(postId: string) {
  if (!apiOnline.value) {
    goToNotFound();
    return;
  }
  try {
    // We can use getPost API if we had one exported, otherwise reuse thread or feed.
    // Assuming getPost is available via search_web investigation or standard patterns.
    // Actually socialApi.ts has fetchPostReplies and fetchPostThread.
    // Let's assume we use fetchPostThread to get the latest post state.
    const thread = await fetchPostThread(postId, 0);
    const postIdx = posts.value.findIndex(p => p.id === postId);
    if (postIdx !== -1) {
      posts.value[postIdx] = toFeedCard(thread.post);
    }
    if (threadFocusPost.value?.id === postId) {
      threadFocusPost.value = toFeedCard(thread.post);
    }
  } catch {
    goToNotFound();
  }
}

function handleMediaChange(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = () => {
    const result = typeof reader.result === 'string' ? reader.result : null;
    mediaPreview.value = result;
    mediaMeta.value = {
      name: file.name,
      sizeLabel: formatBytes(file.size),
      type: file.type || 'image',
      sizeBytes: file.size,
    };
  };
  reader.readAsDataURL(file);
}

function clearMedia() {
  mediaPreview.value = null;
  mediaMeta.value = null;
}

function handleReplyMediaChange(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => {
    const result = typeof reader.result === 'string' ? reader.result : null;
    replyMediaPreview.value = result;
    replyMediaMeta.value = {
      name: file.name,
      sizeLabel: formatBytes(file.size),
      type: file.type || 'image',
      sizeBytes: file.size,
    };
  };
  reader.readAsDataURL(file);
  target.value = '';
}

function clearReplyMedia() {
  replyMediaPreview.value = null;
  replyMediaMeta.value = null;
}

function applyFollowStatsDelta(userId: string, followed: boolean) {
  const delta = followed ? 1 : -1;
  if (currentUser.value) {
    currentUser.value = {
      ...currentUser.value,
      following: Math.max(0, (currentUser.value.following || 0) + delta),
    };
  }
  people.value = people.value.map((person) => {
    if (person.id !== userId) return person;
    return {
      ...person,
      followers: Math.max(0, (person.followers || 0) + delta),
    };
  });
  relationUsers.value = relationUsers.value.map((person) => {
    if (person.id !== userId) return person;
    return {
      ...person,
      followers: Math.max(0, (person.followers || 0) + delta),
    };
  });
}

async function toggleFollow(userId: string) {
  await toggleFollowRelationUser(userId);
}

async function loadRelationUsers(type: 'followers' | 'following') {
  if (!currentUser.value?.id) return;
  relationLoading.value = true;
  relationError.value = '';
  try {
    relationUsers.value = type === 'followers'
      ? await fetchUserFollowers(currentUser.value.id, 200)
      : await fetchUserFollowing(currentUser.value.id, 200);
    if (type === 'following') {
      const next = { ...followedUsers.value };
      relationUsers.value.forEach((item) => {
        next[item.id] = true;
      });
      followedUsers.value = next;
    }
  } catch {
    relationUsers.value = [];
    relationError.value = '加载列表失败，请稍后重试';
  } finally {
    relationLoading.value = false;
  }
}

async function openRelationSection(type: 'followers' | 'following') {
  currentSection.value = type;
  await loadRelationUsers(type);
}

async function syncFollowerNotifications() {
  if (!currentUser.value?.id || !apiOnline.value) return;
  if (!notificationSettings.value.follows || notificationSettings.value.quietHours) return;
  try {
    const followers = await fetchUserFollowers(currentUser.value.id, 200);
    const currentIds = new Set(followers.map((item) => item.id));
    const seenIds = new Set(seenFollowerIds.value);
    const newFollowers = followers.filter((item) => !seenIds.has(item.id));

    if (newFollowers.length > 0) {
      const now = Date.now();
      const newItems = newFollowers.map((item, idx) => ({
        id: `follow-${item.id}-${now}-${idx}`,
        kind: 'follow' as const,
        title: `${item.displayName} 关注了你`,
        body: `${formatHandleInstance(item.handle, item.instance)} 刚刚关注了你`,
        time: '刚刚',
        sortAt: now - idx,
      }));
      followerNotifications.value = [...newItems, ...followerNotifications.value].slice(0, 20);
    }

    seenFollowerIds.value = [...currentIds];
    persistReadState();
  } catch {
    // ignore follower notification sync errors
  }
}

async function toggleFollowRelationUser(userId: string) {
  if (!userId || followActionLoading.value[userId]) return;
  followActionLoading.value = { ...followActionLoading.value, [userId]: true };
  const next = !followedUsers.value[userId];
  try {
    if (next) {
      await followUser(userId);
    } else {
      await unfollowUser(userId);
    }
    followedUsers.value = { ...followedUsers.value, [userId]: next };
    applyFollowStatsDelta(userId, next);
  } finally {
    followActionLoading.value = { ...followActionLoading.value, [userId]: false };
  }
}

function goToUserProfile(userId: string) {
  if (!userId) return;
  const matched = (currentUser.value?.id === userId ? currentUser.value : null)
    || people.value.find((person) => person.id === userId);
  const handle = String(matched?.handle || '').replace(/^@/, '').trim();
  void router.push(`/profile/${encodeURIComponent(handle || userId)}`);
}

function togglePollEditor() {
  showPollEditor.value = !showPollEditor.value;
  if (showPollEditor.value) {
    pollMultiple.value = false;
  }
}

function normalizeTag(raw: string) {
  return raw.replace(/^#/, '').trim().replace(/\s+/g, '').slice(0, MAX_TAG_LENGTH);
}

function toggleTagPicker() {
  showTagPicker.value = !showTagPicker.value;
}

function togglePostTag(tag: string) {
  const normalized = normalizeTag(tag);
  if (!normalized) return;

  if (selectedPostTags.value.includes(normalized)) {
    selectedPostTags.value = selectedPostTags.value.filter((item) => item !== normalized);
    return;
  }

  if (selectedPostTags.value.length >= MAX_POST_TAGS) return;
  selectedPostTags.value = [...selectedPostTags.value, normalized];
}

function addCustomTag() {
  const normalized = normalizeTag(customTagInput.value);
  if (!normalized) return;
  if (selectedPostTags.value.includes(normalized)) {
    customTagInput.value = '';
    return;
  }
  if (selectedPostTags.value.length >= MAX_POST_TAGS) return;
  selectedPostTags.value = [...selectedPostTags.value, normalized];
  customTagInput.value = '';
}

function removePostTag(tag: string) {
  selectedPostTags.value = selectedPostTags.value.filter((item) => item !== tag);
}

function recentTagsStorageKey() {
  const userId = authSession.value?.id || currentUser.value?.id || '';
  return userId ? `${RECENT_TAGS_STORAGE_KEY_PREFIX}:${userId}` : '';
}

function messageStickerStorageKey() {
  const userId = authSession.value?.id || currentUser.value?.id || '';
  return userId ? `${MESSAGE_STICKER_STORAGE_PREFIX}:${userId}` : '';
}

function loadRecentMessageStickers() {
  const key = messageStickerStorageKey();
  if (!key || typeof window === 'undefined') {
    recentMessageStickers.value = [];
    return;
  }
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) {
      recentMessageStickers.value = [];
      return;
    }
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      recentMessageStickers.value = [];
      return;
    }
    recentMessageStickers.value = parsed
      .filter((item) => item && typeof item === 'object' && String(item.preview || '').trim())
      .map((item) => ({
        name: String(item.name || '图片表情'),
        preview: String(item.preview || ''),
        type: String(item.type || 'image'),
        sizeLabel: String(item.sizeLabel || ''),
      }))
      .slice(0, 30);
  } catch {
    recentMessageStickers.value = [];
  }
}

function persistRecentMessageStickers() {
  const key = messageStickerStorageKey();
  if (!key || typeof window === 'undefined') return;
  window.localStorage.setItem(key, JSON.stringify(recentMessageStickers.value.slice(0, 30)));
}

function addRecentMessageSticker(image: { name: string; preview: string; type: string; sizeLabel: string }) {
  if (!image.preview) return;
  const deduped = recentMessageStickers.value.filter((item) => item.preview !== image.preview);
  recentMessageStickers.value = [image, ...deduped].slice(0, 30);
  persistRecentMessageStickers();
}

function loadRecentPostTags() {
  if (typeof window === 'undefined') return;
  const key = recentTagsStorageKey();
  if (!key) {
    recentPostTags.value = [];
    return;
  }
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return;
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return;
    recentPostTags.value = parsed
      .map((tag) => normalizeTag(String(tag ?? '')))
      .filter(Boolean)
      .slice(0, 10);
  } catch {
    recentPostTags.value = [];
  }
}

function persistRecentPostTags() {
  if (typeof window === 'undefined') return;
  const key = recentTagsStorageKey();
  if (!key) return;
  window.localStorage.setItem(key, JSON.stringify(recentPostTags.value));
}

function updateRecentTags(tags: string[]) {
  const normalized = tags.map((tag) => normalizeTag(tag)).filter(Boolean);
  const merged = [...normalized, ...recentPostTags.value];
  recentPostTags.value = [...new Set(merged)].slice(0, 10);
  persistRecentPostTags();
}

function syncPostCursor(event?: Event) {
  const target = (event?.target as HTMLTextAreaElement | undefined) ?? postComposerRef.value;
  if (!target) return;
  postSelectionStart.value = target.selectionStart ?? postDraft.value.length;
  postSelectionEnd.value = target.selectionEnd ?? postDraft.value.length;
}

function syncReplyCursor(event?: Event) {
  const target = (event?.target as HTMLTextAreaElement | undefined) ?? replyTextareaRef.value;
  if (!target) return;
  replySelectionStart.value = target.selectionStart ?? replyDraft.value.length;
  replySelectionEnd.value = target.selectionEnd ?? replyDraft.value.length;
}

function toggleEmojiPicker() {
  showEmojiPicker.value = !showEmojiPicker.value;
  if (showEmojiPicker.value) {
    syncPostCursor();
    void nextTick(updateEmojiPickerFloatingPosition);
  }
}

function updateEmojiPickerFloatingPosition() {
  if (!showEmojiPicker.value || !emojiTriggerRef.value || typeof window === 'undefined') return;
  const rect = emojiTriggerRef.value.getBoundingClientRect();
  const panelWidth = Math.min(352, window.innerWidth - 16);
  const desiredLeft = rect.right - panelWidth;
  const left = Math.max(8, Math.min(desiredLeft, window.innerWidth - panelWidth - 8));
  const maxHeight = Math.min(420, Math.floor(window.innerHeight * 0.55));
  let top = rect.top - maxHeight - 10;
  if (top < 8) {
    top = Math.min(window.innerHeight - maxHeight - 8, rect.bottom + 10);
  }
  emojiPickerFloatingStyle.value = {
    position: 'fixed',
    left: `${left}px`,
    top: `${Math.max(8, top)}px`,
    width: `${panelWidth}px`,
    maxHeight: `${maxHeight}px`,
  };
}

function refreshConversationFederationRoutes() {
  const selfInstance = String(currentUser.value?.instance || '').trim();
  if (!selfInstance) return;
  conversations.value = conversations.value.map((conversation) => {
    if (!conversation.crossInstance) return conversation;
    const peer = conversation.participantId
      ? people.value.find((person) => person.id === conversation.participantId)
      : null;
    const peerInstance = String(peer?.instance || '').trim();
    if (!peerInstance) return conversation;
    return {
      ...conversation,
      federationRoute: `${selfInstance} → ${peerInstance}`,
    };
  });
}

async function insertEmojiAtCursor(emoji: string) {
  if (!emoji) return;

  const start = postSelectionStart.value;
  const end = postSelectionEnd.value;
  postDraft.value = `${postDraft.value.slice(0, start)}${emoji}${postDraft.value.slice(end)}`;

  const nextPos = start + emoji.length;
  postSelectionStart.value = nextPos;
  postSelectionEnd.value = nextPos;

  await nextTick();
  if (postComposerRef.value) {
    postComposerRef.value.focus();
    postComposerRef.value.setSelectionRange(nextPos, nextPos);
  }
}

async function handleEmojiPick(event: Event) {
  const detail = (event as Event & { detail?: { unicode?: string; emoji?: { unicode?: string } | string } }).detail;
  const unicode = detail?.unicode || (typeof detail?.emoji === 'string' ? detail.emoji : detail?.emoji?.unicode) || '';
  if (!unicode) return;
  await insertEmojiAtCursor(unicode);
  showEmojiPicker.value = false;
}

function toggleReplyEmojiPicker() {
  showReplyEmojiPicker.value = !showReplyEmojiPicker.value;
  if (showReplyEmojiPicker.value) {
    syncReplyCursor();
  }
}

async function insertReplyEmojiAtCursor(emoji: string) {
  if (!emoji) return;
  const start = replySelectionStart.value;
  const end = replySelectionEnd.value;
  replyDraft.value = `${replyDraft.value.slice(0, start)}${emoji}${replyDraft.value.slice(end)}`;
  const nextPos = start + emoji.length;
  replySelectionStart.value = nextPos;
  replySelectionEnd.value = nextPos;
  await nextTick();
  if (replyTextareaRef.value) {
    replyTextareaRef.value.focus();
    replyTextareaRef.value.setSelectionRange(nextPos, nextPos);
  }
}

async function handleReplyEmojiPick(event: Event) {
  const detail = (event as Event & { detail?: { unicode?: string; emoji?: { unicode?: string } | string } }).detail;
  const unicode = detail?.unicode || (typeof detail?.emoji === 'string' ? detail.emoji : detail?.emoji?.unicode) || '';
  if (!unicode) return;
  await insertReplyEmojiAtCursor(unicode);
  showReplyEmojiPicker.value = false;
}

function handleDocumentClick(event: MouseEvent) {
  const target = event.target as Node | null;
  if (!target) return;
  if (showEmojiPicker.value) {
    if (!emojiPickerPanelRef.value?.contains(target) && !emojiTriggerRef.value?.contains(target)) {
      showEmojiPicker.value = false;
    }
  }
  if (showReplyEmojiPicker.value) {
    if (!replyEmojiPickerPanelRef.value?.contains(target) && !replyEmojiTriggerRef.value?.contains(target)) {
      showReplyEmojiPicker.value = false;
    }
  }
  if (showMessageEmojiPicker.value) {
    if (!messageEmojiPanelRef.value?.contains(target) && !messageEmojiTriggerRef.value?.contains(target)) {
      showMessageEmojiPicker.value = false;
    }
  }
  if (showMessageStickerPanel.value) {
    if (!messageStickerPanelRef.value?.contains(target) && !messageStickerTriggerRef.value?.contains(target)) {
      showMessageStickerPanel.value = false;
    }
  }
}

function addPollOption() {
  if (pollOptions.value.length < 4) {
    pollOptions.value.push('');
  }
}

function removePollOption(index: number) {
  if (pollOptions.value.length > 2) {
    pollOptions.value.splice(index, 1);
  }
}

async function publishPost() {
  if ((!postDraft.value.trim() && !mediaPreview.value) || !currentUser.value || saving.value || isPostOverLimit.value) return;

  saving.value = true;
  try {
    if (!apiOnline.value) {
      goToNotFound();
      return;
    }

    let createdAsset: MediaAsset | null = null;

    if (mediaPreview.value && mediaMeta.value) {
      createdAsset = await createMediaAsset({
        ownerId: currentUser.value.id,
        name: mediaMeta.value.name,
        kind: mediaMeta.value.type.startsWith('video') ? 'video' : 'image',
        url: mediaPreview.value,
        storageUri: `preview://${Date.now()}`,
        cid: `draft-${Date.now().toString(36)}`,
        sizeBytes: mediaMeta.value.sizeBytes,
        status: 'uploaded',
      });
      assets.value = [toAssetCard(createdAsset), ...assets.value];
    }

    const createdPost = await createPost({
      authorId: currentUser.value.id,
      instance: activeProfileInstanceName.value,
      content: postDraft.value.trim() || '分享了一条新的媒体动态。',
      visibility: visibility.value,
      interaction: interaction.value,
      storageUri: createdAsset?.storageUri || `draft://post/${Date.now()}`,
      attestationUri: `attestation://frontend/${Date.now()}`,
      tags: selectedPostTags.value,
      mediaIds: createdAsset ? [createdAsset.id] : [],
      type: 'post',
      pollOptions: showPollEditor.value ? pollOptions.value.filter(o => o.trim()) : [],
      pollExpiresIn: showPollEditor.value ? pollExpiresIn.value : 0,
      pollMultiple: false,
    });
    const nextCard = toFeedCard(createdPost);
    posts.value = [nextCard, ...posts.value];
    myPosts.value = [nextCard, ...myPosts.value];
    errorMessage.value = '';

    postDraft.value = '';
    mediaPreview.value = null;
    mediaMeta.value = null;
    updateRecentTags(selectedPostTags.value);
    selectedPostTags.value = [];
    customTagInput.value = '';
    showTagPicker.value = false;
    showEmojiPicker.value = false;
    showPollEditor.value = false;
    pollOptions.value = ['', ''];
    pollExpiresIn.value = 1440;
    pollMultiple.value = false;
  } catch (error) {
    if (error instanceof ApiError && error.code === 'AUTH_SESSION_REQUIRED') {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    goToNotFound();
  } finally {
    saving.value = false;
  }
}

async function sendMessage() {
  if ((!messageDraft.value.trim() && !messageMediaPreview.value) || !currentUser.value || !activeConversation.value || saving.value) return;

  saving.value = true;
  try {
    const targetConversation = activeConversation.value;
    if (!targetConversation) return;

    if (!apiOnline.value) {
      goToNotFound();
      return;
    }

    const body = messageMediaPreview.value && messageMediaMeta.value
      ? toMessageImageBody(messageDraft.value.trim(), {
          name: messageMediaMeta.value.name,
          preview: messageMediaPreview.value,
          type: messageMediaMeta.value.type || 'image',
          sizeLabel: messageMediaMeta.value.sizeLabel || '',
        })
      : messageDraft.value.trim();
    await cacheEncryptedConversationPayload(targetConversation.id, body);

    const updatedConversation = await createConversationMessage(targetConversation.id, {
      senderId: currentUser.value.id,
      body,
    });

    const mapped = toConversationCard(updatedConversation, currentUser.value?.id ?? null);
    upsertConversation(mapped);
    selectedConversationId.value = mapped.id;

    if (messageMediaPreview.value && messageMediaMeta.value) {
      addRecentMessageSticker({
        name: messageMediaMeta.value.name,
        preview: messageMediaPreview.value,
        type: messageMediaMeta.value.type || 'image',
        sizeLabel: messageMediaMeta.value.sizeLabel || '',
      });
    }
    messageDraft.value = '';
    messageMediaPreview.value = null;
    messageMediaMeta.value = null;
    showMessageEmojiPicker.value = false;
    showMessageStickerPanel.value = false;
    errorMessage.value = '';
    await scrollMessagesToBottom();
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === 'AUTH_SESSION_REQUIRED') {
        void router.push({ path: '/login', query: { redirect: '/app' } });
        return;
      }

      if (typeof error.message === 'string') {
        if (error.message.includes('the other user has not followed you back yet')) {
          errorMessage.value = '对方还没有关注你，暂时不能回复。';
          return;
        }
        if (error.message.includes('awaiting follow-back: only one message is allowed')) {
          errorMessage.value = '在对方关注你之前，只能先发送一条消息。';
          return;
        }
      }
    }

    goToNotFound();
  } finally {
    saving.value = false;
  }
}

function triggerMessageImagePicker() {
  messageImageInputRef.value?.click();
}

function clearMessageMedia() {
  messageMediaPreview.value = null;
  messageMediaMeta.value = null;
  if (messageImageInputRef.value) messageImageInputRef.value.value = '';
}

function handleMessageMediaChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  if (!file.type.startsWith('image/')) {
    errorMessage.value = '消息仅支持发送图片表情。';
    input.value = '';
    return;
  }
  const reader = new FileReader();
  reader.onload = () => {
    const result = typeof reader.result === 'string' ? reader.result : null;
    if (!result) return;
    messageMediaPreview.value = result;
    messageMediaMeta.value = {
      name: file.name,
      sizeLabel: formatBytes(file.size),
      type: file.type || 'image',
      sizeBytes: file.size,
    };
  };
  reader.readAsDataURL(file);
  input.value = '';
}

function toggleMessageEmojiPicker() {
  showMessageEmojiPicker.value = !showMessageEmojiPicker.value;
  if (showMessageEmojiPicker.value) showMessageStickerPanel.value = false;
}

function toggleMessageStickerPanel() {
  showMessageStickerPanel.value = !showMessageStickerPanel.value;
  if (showMessageStickerPanel.value) showMessageEmojiPicker.value = false;
}

function useStickerAsMessage(sticker: { name: string; preview: string; type: string; sizeLabel: string }) {
  messageMediaPreview.value = sticker.preview;
  messageMediaMeta.value = {
    name: sticker.name || '图片表情',
    sizeLabel: sticker.sizeLabel || '',
    type: sticker.type || 'image',
    sizeBytes: 0,
  };
  addRecentMessageSticker(sticker);
  showMessageStickerPanel.value = false;
}

async function insertMessageEmojiAtCursor(emoji: string) {
  if (!emoji) return;
  const textarea = messageInputRef.value;
  const start = textarea?.selectionStart ?? messageDraft.value.length;
  const end = textarea?.selectionEnd ?? start;
  messageDraft.value = `${messageDraft.value.slice(0, start)}${emoji}${messageDraft.value.slice(end)}`;
  await nextTick();
  const nextPos = start + emoji.length;
  if (textarea) {
    textarea.focus();
    textarea.setSelectionRange(nextPos, nextPos);
  }
}

function handleMessageEmojiPick(event: Event) {
  const detail = (event as Event & { detail?: { unicode?: string; emoji?: { unicode?: string } | string } }).detail;
  const unicode = detail?.unicode || (typeof detail?.emoji === 'string' ? detail.emoji : detail?.emoji?.unicode) || '';
  if (!unicode) return;
  void insertMessageEmojiAtCursor(unicode);
}

async function openForwardDialog(post: FeedCard) {
  forwardingPost.value = post;
  showForwardDialog.value = true;
  errorMessage.value = '';
  if (!conversations.value.length) {
    await refreshConversations(true);
  }
  forwardingConversationId.value = selectedConversationId.value || conversations.value[0]?.id || '';
}

function closeForwardDialog() {
  showForwardDialog.value = false;
  forwardingPost.value = null;
  forwardingConversationId.value = '';
}

async function forwardPostToConversation() {
  if (!forwardingPost.value || !forwardingConversationId.value || !currentUser.value || forwarding.value) return;
  forwarding.value = true;
  try {
    const body = toForwardedPostBody(forwardingPost.value);
    await cacheEncryptedConversationPayload(forwardingConversationId.value, body);
    const updatedConversation = await createConversationMessage(forwardingConversationId.value, {
      senderId: currentUser.value.id,
      body,
    });
    const mapped = toConversationCard(updatedConversation, currentUser.value?.id ?? null);
    upsertConversation(mapped);
    selectedConversationId.value = mapped.id;
    currentSection.value = 'messages';
    closeForwardDialog();
    await scrollMessagesToBottom();
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    errorMessage.value = '转发失败，请稍后重试。';
  } finally {
    forwarding.value = false;
  }
}

onMounted(() => {
  loadNotificationSettings();
  void (async () => {
    await loadBootstrap();
    void warmupHomeDataAfterFirstPaint();
    await handleRouteMessageIntent();
  })();
  loadPostingPrivacySettings();
  loadRecentPostTags();
  loadInteractionState();
  loadReadState();
  document.addEventListener('click', handleDocumentClick);
  document.addEventListener('visibilitychange', handleVisibilityChange);
  window.addEventListener('storage', handleStorageChange);
  window.addEventListener('resize', updateEmojiPickerFloatingPosition);
  window.addEventListener('scroll', updateEmojiPickerFloatingPosition, true);
  startInstancePolling();
});

watch(
  () => currentUser.value?.id || '',
  () => {
    loadRecentMessageStickers();
  },
);

watch(
  [() => currentSection.value, () => activeExploreTab.value],
  ([section, tab]) => {
    if (section !== 'explore') return;
    if (tab === 'posts') {
      void loadExplorePostsTimeline(true);
      return;
    }
    if (tab === 'latest') {
      void loadLatestPostsTimeline(true);
      return;
    }
    if (tab === 'topics') {
      void warmupSearchPostPools();
      return;
    }
    if (tab === 'users') {
      void loadExploreUsers(true);
      return;
    }
    if (tab !== 'news') return;
    void loadNewsTimeline(true);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick);
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  window.removeEventListener('storage', handleStorageChange);
  window.removeEventListener('resize', updateEmojiPickerFloatingPosition);
  window.removeEventListener('scroll', updateEmojiPickerFloatingPosition, true);
  stopInstancePolling();
});

watch(
  () => currentSection.value,
  (section) => {
    if (section !== 'messages') return;
    void refreshConversations(true);
    if (selectedConversationId.value) {
      markConversationAsRead(selectedConversationId.value);
      void loadConversationMessages(selectedConversationId.value);
    }
  },
);

watch(
  () => selectedConversationId.value,
  (conversationId) => {
    if (!conversationId) return;
    if (currentSection.value !== 'messages') return;
    markConversationAsRead(conversationId);
  },
);

watch(
  () => authSession.value?.id,
  () => {
    conversations.value = [];
    selectedConversationId.value = '';
    messageDraft.value = '';
    loadInteractionState();
    loadReadState();
    loadRecentPostTags();
  },
);

watch([likedPosts, bookmarkedPosts, postEngagement], () => {
  persistInteractionState();
}, { deep: true });

watch(
  () => currentSection.value,
  (section) => {
    if (section === 'notifications') {
      markNotificationsAsRead();
    }
    if (section === 'mentions') {
      markMentionsAsRead();
    }
  },
);

watch(notificationItems, () => {
  syncNotificationFeed();
  if (currentSection.value === 'notifications') {
    markNotificationsAsRead();
  }
});

watch(mentionItems, () => {
  if (currentSection.value === 'mentions') {
    markMentionsAsRead();
  }
});

watch(
  () => authSession.value?.id,
  () => {
    notificationFeed.value = [];
    syncNotificationFeed();
  },
);

watch(
  () => searchQuery.value,
  (value) => {
    const hasQuery = Boolean(value.trim());
    if (hasQuery) {
      currentSection.value = 'search';
      void warmupSearchPostPools();
      return;
    }
    if (currentSection.value === 'search') {
      currentSection.value = 'home';
    }
  },
);

function triggerSearchFromInput() {
  const keyword = searchQuery.value.trim();
  if (!keyword) {
    if (currentSection.value === 'search') currentSection.value = 'home';
    return;
  }
  currentSection.value = 'search';
  void warmupSearchPostPools();
}
</script>

<template>
  <div class="min-h-screen bg-[var(--app-bg)] text-[color:var(--text-primary)] transition-colors duration-300 lg:h-screen lg:overflow-hidden" :style="themeStyles">
    <div class="mx-auto max-w-[1440px] px-0 lg:h-screen lg:px-4 lg:overflow-hidden">
      <div v-if="errorMessage" class="mb-4 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
        {{ serviceNotice }}
      </div>

      <div v-if="loading" class="rounded-[24px] border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-6 py-12 text-center text-[color:var(--text-secondary)]">
        正在载入社区内容...
      </div>

      <div v-if="!loading" class="grid gap-0 overflow-visible lg:h-[calc(100vh-24px)] lg:grid-cols-[260px_minmax(0,1fr)_240px]">
        <aside class="relative z-[80] min-h-0 max-h-[calc(100vh-24px)] overflow-visible border-b border-[color:var(--border-color)] bg-[var(--panel-bg)] lg:h-[calc(100vh-32px)] lg:max-h-none lg:border-b-0 lg:border-r">
          <div class="max-h-[calc(100vh-24px)] min-h-0 space-y-3 overflow-y-auto overscroll-contain p-4 no-scrollbar lg:h-full lg:max-h-none">
            <div class="rounded-2xl border border-emerald-500/25 bg-[var(--panel-soft)] px-3 py-3 shadow-[0_8px_24px_rgba(0,0,0,0.12)]">
              <div class="flex items-center gap-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-3 py-2.5 focus-within:border-emerald-500/60 focus-within:shadow-[0_0_0_2px_rgba(16,185,129,0.15)]">
                <Search class="h-4 w-4 shrink-0 text-emerald-400" />
                <input
                  v-model="searchQuery"
                  @focus="currentSection = 'search'"
                  @keydown.enter.prevent="triggerSearchFromInput"
                  placeholder="搜索用户或帖子"
                  class="w-full bg-transparent text-sm text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
                />
                <button
                  v-if="searchQuery.trim().length > 0"
                  @click="searchQuery = ''"
                  class="inline-flex h-7 w-7 items-center justify-center rounded-md text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                  title="清空"
                >
                  <X class="h-4 w-4" />
                </button>
                <button
                  @click="triggerSearchFromInput"
                  class="inline-flex h-8 shrink-0 items-center justify-center rounded-lg bg-emerald-600 px-3 text-xs font-semibold text-white transition hover:bg-emerald-500"
                  title="搜索"
                >
                  搜索
                </button>
              </div>
            </div>

            <div
              class="relative flex cursor-pointer items-center gap-3 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-gradient-to-r from-emerald-300/40 via-cyan-300/30 to-blue-300/40 px-3 py-3"
              :style="currentUser?.backgroundUrl ? { backgroundImage: `linear-gradient(rgba(0,0,0,0.30), rgba(0,0,0,0.30)), url(${currentUser.backgroundUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : {}"
              @click="currentUser?.id && goToUserProfile(currentUser.id)"
            >
              <button
                type="button"
                class="absolute inset-0 flex items-center justify-center bg-black/20 opacity-0 transition hover:opacity-100"
                title="查看我的主页"
                @click.stop="currentUser?.id && goToUserProfile(currentUser.id)"
              />
              <button
                @click="currentUser?.id && goToUserProfile(currentUser.id)"
                class="relative z-10 flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                title="查看我的主页"
              >
                <img v-if="currentUser?.avatarUrl" :src="currentUser.avatarUrl" class="h-full w-full object-cover" />
                <template v-else>{{ avatarText(currentUser?.displayName || 'W') }}</template>
              </button>
              <div class="relative z-10 min-w-0">
                <button
                  @click="currentUser?.id && goToUserProfile(currentUser.id)"
                  class="truncate text-[17px] font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                >
                  {{ currentUser?.displayName }}
                </button>
                <div class="truncate text-sm text-[color:var(--text-secondary)]">{{ profileLabel(currentUser) }}</div>
              </div>
            </div>

            <div class="flex items-center justify-between rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-3 py-2 text-[11px]">
              <div class="flex items-center gap-3 text-[color:var(--text-secondary)]">
                <button
                  @click="openRelationSection('followers')"
                  class="rounded-md px-1.5 py-0.5 transition hover:bg-[var(--chip-hover)]"
                >
                  <strong class="text-[color:var(--text-primary)]">{{ currentUser?.followers ?? 0 }}</strong> 关注者
                </button>
                <button
                  @click="openRelationSection('following')"
                  class="rounded-md px-1.5 py-0.5 transition hover:bg-[var(--chip-hover)]"
                >
                  <strong class="text-[color:var(--text-primary)]">{{ currentUser?.following ?? 0 }}</strong> 关注中
                </button>
              </div>
              <div class="flex items-center gap-2">
                <button
                  @click="router.push('/settings/account')"
                  class="text-[color:var(--text-secondary)] hover:text-emerald-500 transition-colors"
                >
                  修改
                </button>
              </div>
            </div>

            <div class="rounded-[22px] border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-4">
              <!-- Visibility Selection Button -->
              <div class="mb-4 flex flex-wrap gap-2">
                <button
                  @click="openVisibilityModal"
                  class="group flex items-center gap-2 rounded-xl border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-sm font-medium text-emerald-300 transition-all hover:bg-emerald-500/10 hover:border-emerald-500/50"
                  title="控制可见性和互动权限"
                >
                  <component :is="selectedVisibilityItem.icon" class="w-4 h-4 text-emerald-400 group-hover:scale-110 transition-transform" />
                  <span>{{ selectedVisibilityItem.label }}，{{ interactionSummary }}</span>
                  <ChevronDown class="w-4 h-4 opacity-50 ml-1" />
                </button>
              </div>

              <textarea
                ref="postComposerRef"
                v-model="postDraft"
                @click="syncPostCursor"
                @keyup="syncPostCursor"
                @select="syncPostCursor"
                placeholder="想写什么？"
                class="min-h-[100px] w-full resize-none bg-transparent text-base leading-relaxed text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
              />

              <!-- Media Preview (above poll) -->
              <div v-if="mediaPreview && mediaMeta" class="relative mt-3 overflow-hidden rounded-2xl border border-[color:var(--border-color)] group">
                <img :src="mediaPreview" :alt="mediaMeta.name" class="max-h-48 w-full object-contain bg-[var(--panel-contrast)]" />
                <!-- Cancel Button -->
                <button
                  @click="clearMedia"
                  class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-black/60 text-white opacity-0 backdrop-blur-sm transition-opacity hover:bg-black/80 group-hover:opacity-100"
                  title="移除图片"
                >
                  <X class="w-4 h-4" />
                </button>
                <div class="absolute bottom-0 left-0 right-0 flex items-center justify-between bg-black/40 px-3 py-1.5 text-xs text-white/80 backdrop-blur-sm">
                  <span class="truncate">{{ mediaMeta.name }}</span>
                  <span class="ml-2 shrink-0">{{ mediaMeta.sizeLabel }}</span>
                </div>
              </div>

              <!-- Poll Editor -->
              <Transition name="expand">
                <div v-if="showPollEditor" class="mt-4 space-y-4 rounded-2xl border border-emerald-500/20 bg-emerald-500/5 p-4">
                  <div class="space-y-3">
                    <div v-for="(opt, index) in pollOptions" :key="index" class="flex items-center gap-3">
                      <div class="h-6 w-6 flex-none rounded-full border-2 border-[color:var(--border-color)] bg-transparent"></div>
                      <div class="relative flex-1">
                        <input
                          v-model="pollOptions[index]"
                          :placeholder="`选项 ${index + 1}`"
                          class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-2 text-sm text-[color:var(--text-primary)] outline-none focus:border-emerald-500"
                        />
                        <button v-if="pollOptions.length > 2" @click="removePollOption(index)" class="absolute right-3 top-1/2 -translate-y-1/2 text-[color:var(--text-muted)] hover:text-rose-500">
                          <X class="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                    <button v-if="pollOptions.length < 4" @click="addPollOption" class="ml-9 text-xs font-bold text-emerald-400 hover:text-emerald-300">
                      + 添加选项
                    </button>
                  </div>

                  <div class="flex gap-4 border-t border-emerald-500/10 pt-4">
                    <div class="flex-1 space-y-1">
                      <label class="text-[10px] font-bold uppercase tracking-wider text-[color:var(--text-muted)]">投票期限</label>
                      <select v-model="pollExpiresIn" class="w-full bg-transparent text-sm font-bold text-emerald-400 outline-none">
                        <option :value="60" class="bg-[var(--panel-bg)]">1 小时</option>
                        <option :value="1440" class="bg-[var(--panel-bg)]">1 天</option>
                        <option :value="4320" class="bg-[var(--panel-bg)]">3 天</option>
                        <option :value="10080" class="bg-[var(--panel-bg)]">7 天</option>
                      </select>
                    </div>
                    <div class="flex-1 space-y-1 border-l border-emerald-500/10 pl-4">
                      <label class="text-[10px] font-bold uppercase tracking-wider text-[color:var(--text-muted)]">类型</label>
                      <div class="block w-full text-left text-sm font-bold text-emerald-400">
                        单选
                      </div>
                    </div>
                  </div>
                </div>
              </Transition>

              <div class="relative mt-4 flex flex-col gap-3">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2 text-lg text-[color:var(--text-secondary)]">
                    <label class="cursor-pointer transition hover:text-emerald-300 rounded-lg p-1.5 hover:bg-emerald-500/10" title="上传图片或视频">
                      <ImageIcon class="w-5 h-5 stroke-[1.5] transition-transform hover:scale-110" />
                      <input type="file" accept="image/*,video/*" class="hidden" @change="handleMediaChange" />
                    </label>
                    <button @click="togglePollEditor" class="rounded-lg p-1.5 transition-colors" :class="showPollEditor ? 'text-emerald-400 bg-emerald-500/10' : 'hover:bg-[var(--chip-bg)]'">
                      <BarChart3 class="w-5 h-5 hover:text-emerald-400 cursor-pointer transition-transform hover:scale-110" />
                    </button>
                    <button
                      @click="toggleTagPicker"
                      class="rounded-lg p-1.5 transition-colors"
                      :class="showTagPicker ? 'text-emerald-400 bg-emerald-500/10' : 'hover:bg-emerald-500/10'"
                      title="选择标签"
                    >
                      <Hash class="w-5 h-5 cursor-pointer transition-transform hover:scale-110" :class="showTagPicker ? 'text-emerald-400' : 'hover:text-emerald-400'" />
                    </button>
                    <button ref="emojiTriggerRef" @click.stop="toggleEmojiPicker" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" :class="showEmojiPicker ? 'text-emerald-400 bg-emerald-500/10' : ''" title="表情">
                      <Smile class="w-5 h-5 hover:text-emerald-400 cursor-pointer transition-transform hover:scale-110" />
                    </button>
                  </div>
                  <span
                    class="text-sm font-medium pr-1 transition-colors"
                    :class="remainingPostChars <= 0 ? 'text-rose-400 font-bold' : remainingPostChars <= 50 ? 'text-amber-400' : 'text-[color:var(--text-muted)]'"
                  >{{ remainingPostChars }}</span>
                </div>

                <div
                  v-if="showEmojiPicker"
                  ref="emojiPickerPanelRef"
                  @click.stop
                  :style="emojiPickerFloatingStyle"
                  class="fixed z-[99999] overflow-hidden rounded-2xl border border-yellow-400/25 bg-[var(--panel-bg)] p-2 shadow-[0_20px_50px_rgba(0,0,0,0.38)]"
                >
                  <emoji-picker @emoji-click="handleEmojiPick" locale="zh-Hans" preview-position="none" skin-tone-emoji="👍"></emoji-picker>
                  <div class="mt-2 px-2 text-[11px] text-[color:var(--text-muted)]">点击表情即可插入</div>
                </div>

                <Transition name="expand">
                  <div v-if="showTagPicker" class="rounded-2xl border border-emerald-500/25 bg-emerald-500/5 p-4">
                    <div class="mb-3 text-xs font-semibold uppercase tracking-wider text-[color:var(--text-muted)]">选择标签（点击 #XXX，最多 5 个）</div>
                    <div class="max-h-44 overflow-y-auto pr-1 no-scrollbar">
                      <div class="flex flex-wrap gap-2">
                      <button
                        v-for="tag in availablePostTags"
                        :key="tag"
                        @click="togglePostTag(tag)"
                        class="rounded-full border px-3 py-1 text-xs font-semibold transition"
                        :class="selectedPostTags.includes(tag) ? 'border-emerald-400/60 bg-emerald-500/20 text-emerald-300' : 'border-[color:var(--border-color)] bg-[var(--panel-bg)] text-[color:var(--text-secondary)] hover:border-emerald-400/50 hover:text-emerald-300'"
                      >
                        #{{ tag }}
                      </button>
                      </div>
                    </div>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <input
                        v-model="customTagInput"
                        @keydown.enter.prevent="addCustomTag"
                        placeholder="输入自定义标签，例如 #开发日志"
                        class="min-w-0 flex-1 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-3 py-2 text-sm text-[color:var(--text-primary)] outline-none focus:border-emerald-400"
                      />
                      <button
                        @click="addCustomTag"
                        class="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-500"
                      >
                        添加
                      </button>
                    </div>
                  </div>
                </Transition>

                <div v-if="selectedPostTags.length > 0" class="flex flex-wrap gap-2">
                  <button
                    v-for="tag in selectedPostTags"
                    :key="tag"
                    @click="removePostTag(tag)"
                    class="inline-flex items-center gap-1 rounded-full border border-emerald-500/40 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300 transition hover:bg-emerald-500/20"
                    title="点击移除标签"
                  >
                    <span>#{{ tag }}</span>
                    <X class="h-3.5 w-3.5" />
                  </button>
                </div>

                <button
                  :disabled="saving || isPostOverLimit"
                  @click="publishPost"
                  class="w-full rounded-xl bg-emerald-600 py-2.5 text-[15px] font-bold tracking-wider text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-emerald-500 hover:shadow-emerald-500/25 disabled:opacity-50 disabled:hover:translate-y-0"
                >
                  {{ saving ? '发布中...' : '发 布' }}
                </button>
              </div>
            </div>

          </div>
        </aside>

        <main
          ref="mainContentRef"
          class="relative z-0 bg-[var(--frame-bg)] lg:h-[calc(100vh-32px)] lg:overflow-y-auto no-scrollbar"
          @touchstart="onHomeTouchStart"
          @touchmove="onHomeTouchMove"
          @touchend="onHomeTouchEnd"
          @touchcancel="onHomeTouchEnd"
        >
          <div
            v-if="currentSection === 'home'"
            class="overflow-hidden transition-[height] duration-200"
            :style="{ height: `${isRefreshingHome ? 56 : pullDistance}px` }"
          >
            <div class="flex h-14 items-center justify-center text-sm text-[color:var(--text-muted)]">
              {{ pullRefreshHint }}
            </div>
          </div>
          <div
            v-if="currentSection !== 'followers' && currentSection !== 'following'"
            class="border-b border-[color:var(--border-color)] px-6 py-6 transition-all duration-300"
          >
            <div class="flex items-center justify-between gap-4">
              <div class="flex items-center gap-4 text-2xl font-bold text-[color:var(--text-primary)]">
                <component :is="currentSectionInfo.icon" class="w-7 h-7 text-emerald-500" />
                <span>{{ currentSectionInfo.label }}</span>
              </div>
              <button
                v-if="currentSection === 'home'"
                :disabled="isRefreshingHome"
                @click="refreshHomeTimeline"
                class="inline-flex items-center gap-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-3 py-2 text-sm font-medium text-[color:var(--text-secondary)] transition hover:border-emerald-500/40 hover:text-emerald-500 disabled:opacity-50"
              >
                <RefreshCw class="h-4 w-4" :class="isRefreshingHome ? 'animate-spin' : ''" />
                <span>{{ isRefreshingHome ? '刷新中' : '刷新' }}</span>
              </button>
            </div>
          </div>

          <section v-if="currentSection === 'home'" class="divide-y divide-[color:var(--border-color)]">
            <div v-if="homeTimeline.length === 0" class="px-6 py-16 text-center text-sm text-[color:var(--text-muted)]">
              你还没有发布过摩文哦，快来发布第一条吧
            </div>
            <article v-for="post in homeTimeline" :key="post.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="flex gap-3">
                <button
                  @click.stop="goToUserProfile(post.authorId)"
                  class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                  title="查看用户主页"
                >
                  <img v-if="userAvatarUrl(post.authorId)" :src="userAvatarUrl(post.authorId)" class="h-full w-full object-cover" />
                  <template v-else>{{ avatarText(post.author) }}</template>
                </button>
                <div class="min-w-0 flex-1">
                  <div class="cursor-pointer" @click="openPostDetail(post.id, false)">
                    <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                    <button
                      @click.stop="goToUserProfile(post.authorId)"
                      class="text-lg font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                    >
                      {{ post.author }}
                    </button>
                    <span class="text-sm text-[color:var(--text-secondary)]">@{{ post.instance }}</span>
                    <span class="text-xs text-[color:var(--text-muted)]">{{ post.time }}</span>
                    </div>
                    <div v-if="post.bio" class="mt-0.5 text-xs text-[color:var(--text-muted)]">{{ post.bio }}</div>
                    <div class="mt-3 whitespace-pre-wrap text-[15px] leading-7 text-[color:var(--text-soft)]">{{ post.content }}</div>

                    <!-- Poll Display -->
                    <div v-if="post.poll" class="mt-3 space-y-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-3">
                      <div v-for="(opt, idx) in post.poll.options" :key="idx" class="relative">
                        <!-- Voted or Expired: Show results -->
                        <div v-if="post.poll.voters.includes(currentUser?.id || '') || new Date(post.poll.expiresAt) < new Date()" class="group overflow-hidden rounded-lg bg-[var(--frame-bg)]">
                          <div 
                            class="absolute inset-y-0 left-0 bg-emerald-500/20 transition-all duration-1000"
                            :style="{ width: `${(opt.votes / Math.max(1, post.poll.options.reduce((a, b) => a + b.votes, 0))) * 100}%` }"
                          ></div>
                          <div class="relative flex items-center justify-between px-4 py-2 text-[13px]">
                            <span class="font-medium text-[color:var(--text-primary)]">{{ opt.label }}</span>
                            <span class="font-bold text-emerald-400">
                              {{ Math.round((opt.votes / Math.max(1, post.poll.options.reduce((a, b) => a + b.votes, 0))) * 100) }}%
                            </span>
                          </div>
                        </div>
                        <!-- Not voted and Active: Show voting buttons -->
                        <button 
                          v-else 
                          @click.stop="handleVote(post, [idx])"
                          class="w-full rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-4 py-2 text-left text-[13px] font-medium text-emerald-400 transition-all hover:bg-emerald-500/10 hover:border-emerald-500/50"
                        >
                          {{ opt.label }}
                        </button>
                      </div>
                      
                      <div class="mt-2 flex items-center justify-between text-[10px] font-bold uppercase tracking-wider text-[color:var(--text-muted)]">
                        <div class="flex items-center gap-2">
                          <span>{{ post.poll.options.reduce((a, b) => a + b.votes, 0) }} 票</span>
                          <span class="opacity-30">·</span>
                          <span>{{ new Date(post.poll.expiresAt) < new Date() ? '已结束' : '进行中' }}</span>
                        </div>
                        <button @click.stop="refreshPost(post.id)" class="text-emerald-500/70 hover:text-emerald-400 transition-colors">刷新</button>
                      </div>
                    </div>

                    <div v-if="post.media" class="mt-4 overflow-hidden rounded-2xl">
                      <img :src="post.media.preview" :alt="post.media.name" class="max-h-[60vh] w-full h-auto object-contain" />
                    </div>

                    <div v-if="post.tags.length" class="mt-4 flex flex-wrap gap-2">
                      <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-emerald-600 px-3 py-1 text-sm font-bold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-emerald-500 hover:shadow-emerald-500/25">
                        #{{ tag }}
                      </span>
                    </div>
                  </div>

                  <div class="mt-5 flex flex-wrap items-center gap-3 text-sm" @click.stop>
                    <button
                      @click="openPostDetail(post.id)"
                      class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                    >
                      <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.replies ?? 0 }}
                    </button>
                    <button
                      @click="openForwardDialog(post)"
                      class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200"
                    >
                      <Repeat class="w-[18px] h-[18px] mr-1.5" /> 转发
                    </button>
                    <button
                      @click="toggleLike(post.id)"
                      class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                      :class="likedPosts[post.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                    >
                      <Heart :class="{'fill-current': likedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.likes ?? 0 }}
                    </button>
                    <button
                      @click="toggleBookmark(post.id)"
                      class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                      :class="bookmarkedPosts[post.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                    >
                      <Bookmark :class="{'fill-current': bookmarkedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.bookmarks ?? 0 }}
                    </button>
                    
                    <!-- More Menu Wrapper -->
                    <div class="relative ml-auto">
                      <button 
                        @click="toggleMoreMenu(post.id)"
                        class="inline-flex items-center rounded-lg px-2 py-1.5 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        <MoreHorizontal class="w-5 h-5" />
                      </button>
                      
                      <!-- Dropdown Menu -->
                      <div 
                        v-if="activeMoreMenuId === post.id" 
                        class="absolute right-0 top-full mt-2 w-56 rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] shadow-[0_10px_40px_rgba(0,0,0,0.5)] z-50 text-sm overflow-hidden"
                      >
                        <div class="py-1">
                          <button @click="handleMenuAction('share', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">分享</button>
                          <button @click="handleMenuAction('mention', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)] font-medium">提及 {{ post.handle }}</button>
                          <button
                            v-if="currentUser?.id && post.authorId === currentUser.id"
                            @click="handleMenuAction('delete', post)"
                            class="w-full text-left px-4 py-2.5 text-rose-500 hover:bg-rose-500/10"
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
          </section>

          <section v-else-if="currentSection === 'search'" class="divide-y divide-[color:var(--border-color)]">
            <div class="px-6 py-4">
              <div class="text-xs uppercase tracking-wider text-[color:var(--text-muted)]">关键词</div>
              <div class="mt-1 text-lg font-semibold text-[color:var(--text-primary)]">{{ searchQuery }}</div>
              <div class="mt-1 text-xs text-[color:var(--text-muted)]">
                找到 {{ searchedUsers.length }} 位用户，{{ searchedPosts.length }} 条帖子
              </div>
            </div>

            <div v-if="searchWarmupLoading" class="px-6 py-4">
              <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">搜索中...</div>
              <div class="space-y-3">
                <div v-for="idx in 4" :key="`search-skeleton-${idx}`" class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3">
                  <div class="h-3.5 w-28 animate-pulse rounded bg-[var(--chip-bg)]"></div>
                  <div class="mt-2 h-3 w-full animate-pulse rounded bg-[var(--chip-bg)]"></div>
                  <div class="mt-1.5 h-3 w-4/5 animate-pulse rounded bg-[var(--chip-bg)]"></div>
                </div>
              </div>
            </div>

            <div class="px-6 py-4">
              <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">用户</div>
              <div v-if="searchedUsers.length === 0" class="rounded-xl border border-dashed border-[color:var(--border-color)] px-4 py-6 text-sm text-[color:var(--text-muted)]">
                没有匹配到相关用户
              </div>
              <div v-else class="space-y-3">
                <button
                  v-for="user in searchedUsers"
                  :key="user.id"
                  @click="goToUserProfile(user.id)"
                  class="flex w-full items-center gap-3 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-3 py-3 text-left transition hover:border-emerald-500/35 hover:bg-emerald-500/5"
                >
                  <div class="flex h-10 w-10 items-center justify-center overflow-hidden rounded-lg bg-gradient-to-br from-lime-200 to-cyan-200 font-bold text-slate-900">
                    <img v-if="user.avatarUrl" :src="user.avatarUrl" class="h-full w-full object-cover" />
                    <template v-else>{{ avatarText(user.displayName) }}</template>
                  </div>
                  <div class="min-w-0">
                    <div class="truncate font-semibold text-[color:var(--text-primary)]">{{ user.displayName }}</div>
                    <div class="truncate text-xs text-[color:var(--text-muted)]">{{ formatHandleInstance(user.handle, user.instance) }}</div>
                  </div>
                </button>
              </div>
            </div>

            <div class="px-6 py-4">
              <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">帖子</div>
              <div v-if="searchedPosts.length === 0" class="rounded-xl border border-dashed border-[color:var(--border-color)] px-4 py-6 text-sm text-[color:var(--text-muted)]">
                没有匹配到相关帖子
              </div>
              <div v-else class="space-y-3">
                <button
                  v-for="post in searchedPosts"
                  :key="post.id"
                  @click="openPostDetail(post.id, false)"
                  class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-left transition hover:border-emerald-500/35 hover:bg-emerald-500/5"
                >
                  <div class="flex items-center gap-2 text-xs text-[color:var(--text-muted)]">
                    <span class="font-semibold text-[color:var(--text-primary)]">{{ post.author }}</span>
                    <span>{{ formatHandleInstance(post.handle, post.instance) }}</span>
                    <span>{{ post.time }}</span>
                  </div>
                  <div class="mt-2 max-h-[5.5rem] overflow-hidden whitespace-pre-wrap text-sm leading-6 text-[color:var(--text-soft)]">{{ post.content }}</div>
                  <div v-if="post.tags.length" class="mt-2 flex flex-wrap gap-1.5">
                    <span
                      v-for="tag in post.tags.slice(0, 4)"
                      :key="`${post.id}-${tag}`"
                      class="rounded-full bg-emerald-500/12 px-2 py-0.5 text-[11px] text-emerald-300"
                    >
                      #{{ tag }}
                    </span>
                  </div>
                </button>
              </div>
            </div>
          </section>

          <section v-else-if="currentSection === 'postDetail'" class="min-h-[calc(100vh-140px)]">
            <div class="border-b border-[color:var(--border-color)] px-6 py-4">
              <button
                @click="backToExploreTop"
                class="inline-flex items-center gap-2 rounded-full border border-[color:var(--border-color)] px-4 py-2 text-sm text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
              >
                <span>←</span>
                <span>返回当前热门</span>
              </button>
            </div>

            <div v-if="threadLoading" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              正在载入讨论串...
            </div>

            <div v-else-if="threadFocusPost" class="divide-y divide-[color:var(--border-color)]">
              <div v-if="threadError" class="mx-6 mt-6 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-300">
                {{ threadError }}
              </div>

              <div v-if="threadAncestors.length" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
                <div class="mb-3 text-[10px] font-bold uppercase tracking-[0.2em] text-[color:var(--text-muted)]">
                  上下文
                </div>
                <div class="space-y-3">
                  <article
                    v-for="ancestor in threadAncestors"
                    :key="ancestor.id"
                    class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3"
                  >
                    <div class="flex items-center gap-2 text-sm">
                      <button
                        @click="goToUserProfile(ancestor.authorId)"
                        class="font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                      >
                        {{ ancestor.author }}
                      </button>
                      <span class="text-[color:var(--text-secondary)]">@{{ ancestor.instance }}</span>
                      <span class="text-xs text-[color:var(--text-muted)]">{{ ancestor.time }}</span>
                    </div>
                    <div class="mt-2 whitespace-pre-wrap text-sm leading-6 text-[color:var(--text-secondary)]">
                      {{ ancestor.content }}
                    </div>
                  </article>
                </div>
              </div>

              <article class="px-5 py-6 transition hover:bg-[var(--panel-soft)]">
                <div class="flex gap-4">
                  <button
                    @click="goToUserProfile(threadFocusPost.authorId)"
                    class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                    title="查看用户主页"
                  >
                    <img v-if="userAvatarUrl(threadFocusPost.authorId)" :src="userAvatarUrl(threadFocusPost.authorId)" class="h-full w-full object-cover" />
                    <template v-else>{{ avatarText(threadFocusPost.author) }}</template>
                  </button>
                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                      <button
                        @click="goToUserProfile(threadFocusPost.authorId)"
                        class="text-[20px] font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                      >
                        {{ threadFocusPost.author }}
                      </button>
                      <span class="text-base text-[color:var(--text-secondary)]">@{{ threadFocusPost.instance }}</span>
                      <span class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.1em] text-emerald-500">
                        {{ threadFocusPost.kind === 'reply' ? '回复' : '帖子' }}
                      </span>
                      <span class="text-xs text-[color:var(--text-muted)]">{{ threadFocusPost.time }}</span>
                    </div>
                    <div v-if="threadFocusPost.bio" class="mt-0.5 text-xs text-[color:var(--text-muted)]">{{ threadFocusPost.bio }}</div>
                    <div class="mt-4 whitespace-pre-wrap text-base leading-7 text-[color:var(--text-soft)]">{{ threadFocusPost.content }}</div>

                    <!-- Detail Poll Display -->
                    <div v-if="threadFocusPost.poll" class="mt-6 space-y-4 rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-6">
                      <div v-for="(opt, idx) in threadFocusPost.poll.options" :key="idx" class="relative">
                        <div v-if="threadFocusPost.poll.voters.includes(currentUser?.id || '') || new Date(threadFocusPost.poll.expiresAt) < new Date()" class="group overflow-hidden rounded-xl bg-[var(--frame-bg)]">
                          <div 
                            class="absolute inset-y-0 left-0 bg-emerald-500/20 transition-all duration-1000"
                            :style="{ width: `${(opt.votes / Math.max(1, threadFocusPost.poll.options.reduce((a, b) => a + b.votes, 0))) * 100}%` }"
                          ></div>
                          <div class="relative flex items-center justify-between px-5 py-4 text-base">
                            <span class="font-medium text-[color:var(--text-primary)]">{{ opt.label }}</span>
                            <span class="font-bold text-emerald-400">
                              {{ Math.round((opt.votes / Math.max(1, threadFocusPost.poll.options.reduce((a, b) => a + b.votes, 0))) * 100) }}%
                            </span>
                          </div>
                        </div>
                        <button 
                          v-else 
                          @click="handleVote(threadFocusPost, [idx])"
                          class="w-full rounded-xl border border-emerald-500/30 bg-emerald-500/5 px-5 py-4 text-left text-base font-medium text-emerald-400 transition-all hover:bg-emerald-500/10 hover:border-emerald-500/50"
                        >
                          {{ opt.label }}
                        </button>
                      </div>
                      
                      <div class="mt-4 flex items-center justify-between text-xs font-bold uppercase tracking-wider text-[color:var(--text-muted)]">
                        <div class="flex items-center gap-3">
                          <span>{{ threadFocusPost.poll.options.reduce((a, b) => a + b.votes, 0) }} 票</span>
                          <span class="opacity-30">·</span>
                          <span>{{ new Date(threadFocusPost.poll.expiresAt) < new Date() ? '已结束' : '进行中' }}</span>
                          <span class="opacity-30">·</span>
                          <span v-if="new Date(threadFocusPost.poll.expiresAt) > new Date()">剩余时间: {{ formatTimestamp(threadFocusPost.poll.expiresAt) }}</span>
                        </div>
                      </div>
                    </div>

                    <div
                      v-if="threadFocusPost.media"
                      class="mt-4 overflow-hidden rounded-2xl"
                    >
                      <img :src="threadFocusPost.media.preview" :alt="threadFocusPost.media.name" class="max-h-[70vh] w-full h-auto object-contain" />
                    </div>

                    <div v-if="threadFocusPost.tags.length" class="mt-4 flex flex-wrap gap-2">
                      <span
                        v-for="tag in threadFocusPost.tags"
                        :key="tag"
                        class="rounded-full bg-emerald-600 px-3 py-1 text-sm font-bold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-emerald-500 hover:shadow-emerald-500/25"
                      >
                        #{{ tag }}
                      </span>
                    </div>

                    <div class="mt-5 flex flex-wrap items-center gap-3 text-sm">
                      <button
                        class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> {{ threadFocusPost.stats.replies ?? 0 }}
                      </button>
                      <button
                        @click="openForwardDialog(threadFocusPost)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200"
                      >
                        <Repeat class="w-[18px] h-[18px] mr-1.5" /> 转发
                      </button>
                      <button
                        @click="toggleLike(threadFocusPost.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="likedPosts[threadFocusPost.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                      >
                        <Heart :class="{'fill-current': likedPosts[threadFocusPost.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ threadFocusPost.stats.likes ?? 0 }}
                      </button>
                      <button
                        @click="toggleBookmark(threadFocusPost.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="bookmarkedPosts[threadFocusPost.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                      >
                        <Bookmark :class="{'fill-current': bookmarkedPosts[threadFocusPost.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ threadFocusPost.stats.bookmarks ?? 0 }}
                      </button>
                    </div>
                  </div>
                </div>
              </article>

              <div v-if="isReplyingRoot" ref="replyComposerRef" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
                <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-5">
                  <div class="flex items-start gap-4">
                    <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                      <img v-if="currentUser?.avatarUrl" :src="currentUser.avatarUrl" class="h-full w-full object-cover" />
                      <template v-else>{{ avatarText(currentUser?.displayName || 'U') }}</template>
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">
                        回复给 {{ activeReplyTarget?.author || threadFocusPost.author }}
                      </div>
                      <div
                        v-if="activeReplyTarget"
                        class="mb-4 rounded-2xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-4 py-3"
                      >
                        <div class="flex flex-wrap items-center justify-between gap-3">
                          <div class="min-w-0">
                            <div class="text-sm font-medium text-[color:var(--text-primary)]">
                              {{ activeReplyTarget.id === threadFocusPost.id ? '正在回复主帖' : `正在回复 ${activeReplyTarget.author}` }}
                            </div>
                            <div class="mt-1 line-clamp-2 text-sm text-[color:var(--text-muted)]">
                              {{ activeReplyTarget.content }}
                            </div>
                          </div>
                          <button
                            v-if="activeReplyTarget.id !== threadFocusPost.id"
                            @click="setReplyTarget(threadFocusPost)"
                            class="rounded-full border border-[color:var(--border-color)] px-3 py-2 text-xs text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                          >
                            改为回复主帖
                          </button>
                        </div>
                      </div>
                      <textarea
                        ref="replyTextareaRef"
                        v-model="replyDraft"
                        @click="syncReplyCursor"
                        @keyup="syncReplyCursor"
                        @select="syncReplyCursor"
                        rows="4"
                        maxlength="500"
                        placeholder="写下你的看法，让讨论继续发生"
                        class="w-full resize-none rounded-2xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-4 py-4 text-base leading-7 text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
                      />
                      <div v-if="replyMediaPreview && replyMediaMeta" class="relative mt-3 overflow-hidden rounded-2xl border border-[color:var(--border-color)] group">
                        <img :src="replyMediaPreview" :alt="replyMediaMeta.name" class="max-h-48 w-full object-contain bg-[var(--panel-contrast)]" />
                        <button
                          type="button"
                          @click="clearReplyMedia"
                          class="absolute right-2 top-2 rounded-full bg-black/60 p-1 text-white opacity-0 transition-opacity group-hover:opacity-100"
                          title="移除图片"
                        >
                          <X class="w-4 h-4" />
                        </button>
                      </div>
                      <div class="mt-4 flex items-center justify-between gap-4">
                        <div class="flex items-center gap-2">
                          <label class="cursor-pointer rounded-lg p-1.5 transition-colors hover:bg-emerald-400/10" title="添加图片">
                            <ImageIcon class="w-4 h-4 text-emerald-300" />
                            <input ref="replyFileInputRef" type="file" accept="image/*" class="hidden" @change="handleReplyMediaChange" />
                          </label>
                          <button ref="replyEmojiTriggerRef" @click.stop="toggleReplyEmojiPicker" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" :class="showReplyEmojiPicker ? 'text-emerald-400 bg-emerald-500/10' : ''" title="表情">
                            <Smile class="w-4 h-4" />
                          </button>
                        </div>
                        <div class="text-sm text-[color:var(--text-muted)]">
                          {{ replyDraft.trim().length }}/500
                        </div>
                        <button
                          :disabled="(!replyDraft.trim() && !replyMediaPreview) || saving"
                          @click="submitReply"
                          class="rounded-xl bg-emerald-600 px-5 py-3 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {{ saving ? '发送中...' : '发送回复' }}
                        </button>
                      </div>
                      <div v-if="showReplyEmojiPicker" ref="replyEmojiPickerPanelRef" @click.stop class="mt-3 rounded-2xl border border-yellow-400/20 bg-[var(--panel-bg)] p-2">
                        <emoji-picker @emoji-click="handleReplyEmojiPick" locale="zh-Hans" preview-position="none" skin-tone-emoji="👍"></emoji-picker>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
                <div class="mb-4 flex items-center justify-between gap-3">
                  <div class="text-xs font-semibold uppercase tracking-[0.2em] text-[color:var(--text-muted)]">
                    回复楼层
                  </div>
                  <div class="text-sm text-[color:var(--text-muted)]">
                    {{ threadReplies.length }} 条回复
                  </div>
                </div>

                <div v-if="threadReplies.length === 0" class="rounded-3xl border border-dashed border-[color:var(--border-color)] px-6 py-10 text-center text-[color:var(--text-muted)]">
                  这条帖子还没有回复，第一条评论会显示在这里。
                </div>

                <div v-else class="space-y-4">
                  <article
                    v-for="group in twoLevelReplies"
                    :key="group.parent.id"
                    class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-5"
                  >
                    <div class="flex items-center gap-2 text-sm">
                      <button
                        @click="goToUserProfile(group.parent.authorId)"
                        class="font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                      >
                        {{ group.parent.author }}
                      </button>
                      <span class="text-[color:var(--text-secondary)]">@{{ group.parent.instance }}</span>
                      <span class="rounded-full bg-[var(--chip-bg)] px-3 py-1 text-[color:var(--text-muted)]">第 2 层</span>
                      <span class="text-[color:var(--text-muted)]">{{ group.parent.time }}</span>
                    </div>
                    <div class="mt-3 whitespace-pre-wrap text-base leading-7 text-[color:var(--text-secondary)]">
                      {{ group.parent.content }}
                    </div>
                    <div
                      v-if="group.parent.media?.preview"
                      class="mt-3 overflow-hidden rounded-2xl"
                    >
                      <img
                        :src="group.parent.media.preview"
                        :alt="group.parent.media.name || '回复图片'"
                        class="max-h-[60vh] w-full h-auto object-contain"
                      />
                    </div>
                    <div class="mt-4 flex flex-wrap items-center gap-3 text-sm">
                      <button
                        @click="setReplyTarget(group.parent)"
                        class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> 回复
                      </button>
                    </div>
                    <div v-if="activeReplyTarget?.id === group.parent.id && !isReplyingRoot" class="mt-4 rounded-2xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] p-4">
                      <textarea
                        ref="replyTextareaRef"
                        v-model="replyDraft"
                        @click="syncReplyCursor"
                        @keyup="syncReplyCursor"
                        @select="syncReplyCursor"
                        rows="3"
                        maxlength="500"
                        placeholder="回复这条评论"
                        class="w-full resize-none rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-3 py-3 text-sm text-[color:var(--text-primary)] outline-none"
                      />
                      <div v-if="replyMediaPreview && replyMediaMeta" class="relative mt-3 overflow-hidden rounded-xl border border-[color:var(--border-color)]">
                        <img :src="replyMediaPreview" :alt="replyMediaMeta.name" class="max-h-40 w-full object-contain bg-[var(--panel-contrast)]" />
                        <button type="button" @click="clearReplyMedia" class="absolute right-2 top-2 rounded-full bg-black/60 p-1 text-white"><X class="w-4 h-4" /></button>
                      </div>
                      <div class="mt-3 flex items-center justify-between">
                        <div class="flex items-center gap-2">
                          <label class="cursor-pointer rounded-lg p-1.5 transition-colors hover:bg-emerald-400/10" title="添加图片">
                            <ImageIcon class="w-4 h-4 text-emerald-300" />
                            <input ref="replyFileInputRef" type="file" accept="image/*" class="hidden" @change="handleReplyMediaChange" />
                          </label>
                          <button ref="replyEmojiTriggerRef" @click.stop="toggleReplyEmojiPicker" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" :class="showReplyEmojiPicker ? 'text-emerald-400 bg-emerald-500/10' : ''" title="表情">
                            <Smile class="w-4 h-4" />
                          </button>
                        </div>
                        <button :disabled="(!replyDraft.trim() && !replyMediaPreview) || saving" @click="submitReply" class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">{{ saving ? '发送中...' : '发送' }}</button>
                      </div>
                      <div v-if="showReplyEmojiPicker" ref="replyEmojiPickerPanelRef" @click.stop class="mt-3 rounded-2xl border border-yellow-400/20 bg-[var(--panel-bg)] p-2">
                        <emoji-picker @emoji-click="handleReplyEmojiPick" locale="zh-Hans" preview-position="none" skin-tone-emoji="👍"></emoji-picker>
                      </div>
                    </div>

                    <div v-if="group.children.length" class="mt-4 border-t border-[color:var(--border-color)]/70 pt-4">
                      <div
                        v-for="child in group.children"
                        :key="child.post.id"
                        class="ml-6 mb-3 rounded-2xl border border-[color:var(--border-color)]/70 bg-[var(--frame-bg)] px-4 py-4 last:mb-0"
                      >
                        <div class="flex items-center gap-2 text-sm">
                          <button
                            @click="goToUserProfile(child.post.authorId)"
                            class="font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                          >
                            {{ child.post.author }}
                          </button>
                          <span class="text-[color:var(--text-secondary)]">@{{ child.post.instance }}</span>
                          <span class="rounded-full bg-[var(--chip-bg)] px-3 py-1 text-[color:var(--text-muted)]">第 3 层</span>
                          <span class="text-[color:var(--text-muted)]">{{ child.post.time }}</span>
                        </div>
                        <div class="mt-2 whitespace-pre-wrap text-base leading-7 text-[color:var(--text-secondary)]">
                          <span v-if="child.replyTo" class="mr-1 text-emerald-400">回复 @{{ child.replyTo.author }}：</span>{{ child.post.content }}
                        </div>
                        <div
                          v-if="child.post.media?.preview"
                          class="mt-3 overflow-hidden rounded-2xl"
                        >
                          <img
                            :src="child.post.media.preview"
                            :alt="child.post.media.name || '回复图片'"
                            class="max-h-[60vh] w-full h-auto object-contain"
                          />
                        </div>
                        <div class="mt-3 flex flex-wrap items-center gap-3 text-sm">
                          <button
                            @click="setReplyTarget(child.post)"
                            class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                          >
                            <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> 回复
                          </button>
                        </div>
                        <div v-if="activeReplyTarget?.id === child.post.id" class="mt-4 rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] p-4">
                          <textarea
                            ref="replyTextareaRef"
                            v-model="replyDraft"
                            @click="syncReplyCursor"
                            @keyup="syncReplyCursor"
                            @select="syncReplyCursor"
                            rows="3"
                            maxlength="500"
                            placeholder="回复这条评论"
                            class="w-full resize-none rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-3 py-3 text-sm text-[color:var(--text-primary)] outline-none"
                          />
                          <div v-if="replyMediaPreview && replyMediaMeta" class="relative mt-3 overflow-hidden rounded-xl border border-[color:var(--border-color)]">
                            <img :src="replyMediaPreview" :alt="replyMediaMeta.name" class="max-h-40 w-full object-contain bg-[var(--panel-contrast)]" />
                            <button type="button" @click="clearReplyMedia" class="absolute right-2 top-2 rounded-full bg-black/60 p-1 text-white"><X class="w-4 h-4" /></button>
                          </div>
                          <div class="mt-3 flex items-center justify-between">
                            <div class="flex items-center gap-2">
                              <label class="cursor-pointer rounded-lg p-1.5 transition-colors hover:bg-emerald-400/10" title="添加图片">
                                <ImageIcon class="w-4 h-4 text-emerald-300" />
                                <input ref="replyFileInputRef" type="file" accept="image/*" class="hidden" @change="handleReplyMediaChange" />
                              </label>
                              <button ref="replyEmojiTriggerRef" @click.stop="toggleReplyEmojiPicker" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" :class="showReplyEmojiPicker ? 'text-emerald-400 bg-emerald-500/10' : ''" title="表情">
                                <Smile class="w-4 h-4" />
                              </button>
                            </div>
                            <button :disabled="(!replyDraft.trim() && !replyMediaPreview) || saving" @click="submitReply" class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">{{ saving ? '发送中...' : '发送' }}</button>
                          </div>
                          <div v-if="showReplyEmojiPicker" ref="replyEmojiPickerPanelRef" @click.stop class="mt-3 rounded-2xl border border-yellow-400/20 bg-[var(--panel-bg)] p-2">
                            <emoji-picker @emoji-click="handleReplyEmojiPick" locale="zh-Hans" preview-position="none" skin-tone-emoji="👍"></emoji-picker>
                          </div>
                        </div>
                      </div>
                    </div>
                  </article>
                </div>
              </div>
            </div>

            <div v-else class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              没有找到对应的帖子内容。
            </div>
          </section>

          <section v-else-if="currentSection === 'followers' || currentSection === 'following'" class="divide-y divide-[color:var(--border-color)]">
            <div class="px-6 py-5">
              <div class="text-xl font-semibold text-[color:var(--text-primary)]">
                {{ currentSection === 'followers' ? '我的关注者' : '我的关注中' }}
              </div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">
                {{ relationUsers.length }} 个账号
              </div>
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

            <article v-for="person in relationUsers" :key="person.id" class="px-6 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0 flex items-start gap-3">
                  <button
                    @click="goToUserProfile(person.id)"
                    class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                  >
                    <img v-if="person.avatarUrl" :src="person.avatarUrl" :alt="person.displayName" class="h-full w-full object-cover" />
                    <template v-else>{{ avatarText(person.displayName) }}</template>
                  </button>
                  <div class="min-w-0">
                    <button
                      @click="goToUserProfile(person.id)"
                      class="text-left text-lg font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                    >
                      {{ person.displayName }}
                    </button>
                    <div class="text-sm text-[color:var(--text-muted)]">{{ formatHandleInstance(person.handle, person.instance) }}</div>
                    <div class="mt-2 text-sm text-[color:var(--text-secondary)] line-clamp-2">
                      {{ person.bio || '这个用户还没有填写简介。' }}
                    </div>
                  </div>
                </div>

                <button
                  @click="toggleFollowRelationUser(person.id)"
                  :disabled="followActionLoading[person.id]"
                  class="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-bold text-white shadow-sm transition hover:bg-emerald-500 hover:shadow-emerald-500/25 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {{
                    followActionLoading[person.id]
                      ? '处理中...'
                      : followedUsers[person.id]
                        ? '取消关注'
                        : '关注'
                  }}
                </button>
              </div>
            </article>
          </section>

          <section v-else-if="currentSection === 'explore'">
            <!-- Tab Navigation (Now part of the explore content) -->
            <div class="px-6 pt-2 pb-6 border-b border-[color:var(--border-color)]">
              <div class="flex items-center gap-8 overflow-x-auto no-scrollbar">
                <button 
                  v-for="tab in [
                    { id: 'posts', label: '摩文' },
                    { id: 'latest', label: '最新' },
                    { id: 'topics', label: '话题' },
                    { id: 'users', label: '用户' },
                    { id: 'news', label: '新闻' }
                  ]"
                  :key="tab.id"
                  @click="activeExploreTab = tab.id as ExploreTab"
                  class="relative pb-4 text-lg font-medium transition-colors"
                  :class="activeExploreTab === tab.id ? 'text-emerald-500' : 'text-[color:var(--text-muted)] hover:text-[color:var(--text-primary)]'"
                >
                  {{ tab.label }}
                  <div 
                    v-if="activeExploreTab === tab.id" 
                    class="absolute bottom-[-1px] left-0 h-[3px] w-full rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.4)]"
                  ></div>
                </button>
              </div>
            </div>

            <!-- Tab Content -->
            <div class="divide-y divide-[color:var(--border-color)]">
              <!-- Posts Tab -->
              <template v-if="activeExploreTab === 'posts'">
                <div v-if="explorePostsLoading" class="p-6 text-sm text-[color:var(--text-muted)]">
                  热门加载中...
                </div>
                <div v-else-if="exploreTimeline.length === 0" class="p-12 text-center text-[color:var(--text-muted)]">
                  目前没有用户发布的热门摩文。
                </div>
                <PostFeedCard
                  v-for="post in exploreTimeline"
                  :key="post.id"
                  :post="post"
                  :avatar-url="userAvatarUrl(post.authorId)"
                  :liked="likedPosts[post.id]"
                  :bookmarked="bookmarkedPosts[post.id]"
                  :current-user-id="currentUser?.id"
                  :mention-users="people"
                  :show-more-menu="true"
                  :more-menu-open="activeMoreMenuId === post.id"
                  @open-profile="goToUserProfile"
                  @open-detail="openPostDetail"
                  @forward="openForwardDialog"
                  @toggle-like="toggleLike"
                  @toggle-bookmark="toggleBookmark"
                  @toggle-more="toggleMoreMenu"
                  @menu-action="handleMenuAction"
                  @vote="handleVote"
                />
              </template>

              <template v-else-if="activeExploreTab === 'latest'">
                <div v-if="latestPostsLoading" class="p-6 text-sm text-[color:var(--text-muted)]">
                  最新加载中...
                </div>
                <div v-else-if="timeline.length === 0" class="p-12 text-center text-[color:var(--text-muted)]">
                  目前没有最新普通摩文。
                </div>
                <PostFeedCard
                  v-for="post in timeline"
                  :key="post.id"
                  :post="post"
                  :avatar-url="userAvatarUrl(post.authorId)"
                  :liked="likedPosts[post.id]"
                  :bookmarked="bookmarkedPosts[post.id]"
                  :current-user-id="currentUser?.id"
                  :mention-users="people"
                  :show-more-menu="true"
                  :more-menu-open="activeMoreMenuId === post.id"
                  @open-profile="goToUserProfile"
                  @open-detail="openPostDetail"
                  @forward="openForwardDialog"
                  @toggle-like="toggleLike"
                  @toggle-bookmark="toggleBookmark"
                  @toggle-more="toggleMoreMenu"
                  @menu-action="handleMenuAction"
                  @vote="handleVote"
                />
              </template>

              <!-- Topics Tab -->
              <template v-else-if="activeExploreTab === 'topics'">
                <template v-if="!selectedTopicTag">
                  <div
                    v-for="tag in trendingTags"
                    :key="tag.tag"
                    class="p-6 transition hover:bg-[var(--panel-soft)] flex items-center justify-between cursor-pointer"
                    @click="openTopicTagFeed(tag.tag)"
                  >
                    <div>
                      <div class="font-bold text-lg text-[color:var(--text-primary)]">#{{ tag.tag }}</div>
                      <div class="text-sm text-[color:var(--text-muted)] mt-1">热门话题</div>
                    </div>
                    <div class="rounded-full bg-emerald-500/10 px-4 py-1.5 text-sm font-semibold text-emerald-600">
                      {{ tag.count }} 摩文
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="px-6 py-4 flex items-center justify-between border-b border-[color:var(--border-color)]">
                    <div class="text-lg font-semibold text-[color:var(--text-primary)]">#{{ selectedTopicTag }}</div>
                    <button @click="clearTopicTagFeed" class="rounded-xl border border-[color:var(--border-color)] px-4 py-2 text-sm font-semibold text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)]">返回话题</button>
                  </div>
                  <div v-if="topicTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">该话题下暂无摩文。</div>
                  <PostFeedCard
                    v-for="post in topicTimeline"
                    v-else
                    :key="post.id"
                    :post="post"
                    :avatar-url="userAvatarUrl(post.authorId)"
                    :liked="likedPosts[post.id]"
                    :bookmarked="bookmarkedPosts[post.id]"
                    :current-user-id="currentUser?.id"
                    :mention-users="people"
                    :show-more-menu="true"
                    :more-menu-open="activeMoreMenuId === post.id"
                    @open-profile="goToUserProfile"
                    @open-detail="openPostDetail"
                    @forward="openForwardDialog"
                    @toggle-like="toggleLike"
                    @toggle-bookmark="toggleBookmark"
                    @toggle-more="toggleMoreMenu"
                    @menu-action="handleMenuAction"
                    @vote="handleVote"
                  />
                </template>
              </template>

              <!-- Users Tab -->
              <template v-else-if="activeExploreTab === 'users'">
                <div v-if="exploreUsersLoading" class="p-6 text-sm text-[color:var(--text-muted)]">
                  用户加载中...
                </div>
                <div v-else-if="recommendedPeople.length === 0" class="p-12 text-center text-[color:var(--text-muted)]">
                  暂无可展示用户。
                </div>
                <article
                  v-for="person in recommendedPeople"
                  :key="person.id"
                  class="cursor-pointer p-6 transition hover:bg-[var(--panel-soft)]"
                  @click="goToUserProfile(person.id)"
                >
                  <div class="flex items-start justify-between gap-4">
                    <div class="flex min-w-0 gap-4">
                      <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                        {{ avatarText(person.displayName) }}
                      </div>
                      <div class="min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                          <span class="text-lg font-semibold text-[color:var(--text-primary)]">{{ person.displayName }}</span>
                          <span class="truncate text-base text-[color:var(--text-muted)]">{{ formatHandleInstance(person.handle, person.instance) }}</span>
                        </div>
                        <div class="mt-1 text-sm text-[color:var(--text-muted)]">{{ person.followers }} 关注者</div>
                        <div class="mt-2 line-clamp-2 text-base leading-relaxed text-[color:var(--text-secondary)]">{{ person.bio }}</div>
                      </div>
                    </div>
                    <div class="flex shrink-0 items-center gap-2">
                      <button
                        @click.stop="startConversation(person)"
                        class="inline-flex items-center gap-2 rounded-xl bg-cyan-600 px-5 py-2 text-sm font-bold tracking-wide text-white shadow-sm transition hover:bg-cyan-500 hover:shadow-cyan-500/25"
                      >
                        <MessageCircle class="h-4 w-4" />
                        <span>{{ isCrossInstanceUser(person) ? '跨联邦发消息' : '发消息' }}</span>
                      </button>
                      <button
                        @click.stop="toggleFollow(person.id)"
                        :disabled="followActionLoading[person.id]"
                        class="rounded-xl bg-emerald-600 px-5 py-2 text-sm font-bold tracking-wide text-white shadow-sm transition hover:bg-emerald-500 hover:shadow-emerald-500/25 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {{ followActionLoading[person.id] ? '处理中...' : (followedUsers[person.id] ? '已关注' : '关注') }}
                      </button>
                    </div>
                  </div>
                </article>
              </template>

              <!-- News Tab -->
              <template v-else-if="activeExploreTab === 'news'">
                <div v-if="newsLoading" class="p-6 text-sm text-[color:var(--text-muted)]">
                  新闻加载中...
                </div>
                <div v-else-if="newsTimeline.length === 0" class="p-12 text-center text-[color:var(--text-muted)]">
                  目前没有最新的新闻摩文。
                </div>
                <article v-for="post in newsTimeline" :key="post.id" class="p-6 transition hover:bg-[var(--panel-soft)]">
                   <div class="flex gap-4">
                    <div class="h-12 w-12 flex-none rounded-2xl bg-emerald-600 flex items-center justify-center text-white">
                      <Newspaper class="w-6 h-6" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between mb-1">
                        <span class="font-bold text-emerald-500">{{ parseNewsContent(post.content).source }}</span>
                        <span class="text-sm text-[color:var(--text-muted)]">{{ post.time }}</span>
                      </div>
                      <div class="text-[17px] leading-relaxed text-[color:var(--text-primary)] font-medium whitespace-pre-wrap">{{ parseNewsContent(post.content).title }}</div>
                      <a
                        v-if="parseNewsContent(post.content).link"
                        :href="parseNewsContent(post.content).link"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="mt-2 inline-block break-all text-sm text-cyan-500 hover:text-cyan-400 hover:underline"
                      >
                        {{ parseNewsContent(post.content).link }}
                      </a>

                      <!-- News Poll Display -->
                      <div v-if="post.poll" class="mt-4 space-y-3 rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-3">
                        <div v-for="(opt, idx) in post.poll.options" :key="idx" class="relative">
                          <div v-if="post.poll.voters.includes(currentUser?.id || '') || new Date(post.poll.expiresAt) < new Date()" class="group overflow-hidden rounded-lg bg-[var(--panel-bg)]">
                            <div 
                              class="absolute inset-y-0 left-0 bg-emerald-500/20 transition-all"
                              :style="{ width: `${(opt.votes / Math.max(1, post.poll.options.reduce((a, b) => a + b.votes, 0))) * 100}%` }"
                            ></div>
                            <div class="relative flex items-center justify-between px-3 py-2 text-sm">
                              <span class="font-medium text-[color:var(--text-primary)]">{{ opt.label }}</span>
                              <span class="font-bold text-emerald-400">
                                {{ Math.round((opt.votes / Math.max(1, post.poll.options.reduce((a, b) => a + b.votes, 0))) * 100) }}%
                              </span>
                            </div>
                          </div>
                          <button 
                            v-else 
                            @click="handleVote(post, [idx])"
                            class="w-full rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 text-left text-sm font-medium text-emerald-400 transition-all hover:bg-emerald-500/10"
                          >
                            {{ opt.label }}
                          </button>
                        </div>
                      </div>
                      
                      <!-- Post Media (News Tab) -->
                      <div v-if="post.media" class="mt-4 overflow-hidden rounded-xl">
                        <img :src="post.media.preview" :alt="post.media.name" class="max-h-[70vh] w-full h-auto object-contain" />
                      </div>
                      
                      <!-- Interaction Row -->
                      <div class="mt-4 flex flex-wrap items-center gap-3 text-sm">
                        <button
                          @click="openPostDetail(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium text-[color:var(--text-secondary)] transition-all hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                        >
                          <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.replies || '' }}
                        </button>
                        <button
                          @click="openForwardDialog(post)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium text-[color:var(--text-secondary)] transition-all hover:bg-emerald-500/10 hover:text-emerald-400"
                        >
                          <Repeat class="w-[18px] h-[18px] mr-1.5" /> 转发
                        </button>
                        <button
                          @click="toggleLike(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-rose-500/10 hover:text-rose-400"
                          :class="likedPosts[post.id] ? 'text-rose-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Heart :class="{'fill-current': likedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.likes ?? 0 }}
                        </button>
                        <button
                          @click="toggleBookmark(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-indigo-500/10 hover:text-indigo-400"
                          :class="bookmarkedPosts[post.id] ? 'text-indigo-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Bookmark :class="{'fill-current': bookmarkedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.bookmarks ?? 0 }}
                        </button>
                        
                        <!-- More Menu Wrapper -->
                        <div class="relative ml-auto">
                          <button 
                            @click="toggleMoreMenu(post.id)"
                            class="inline-flex items-center rounded-lg px-2 py-1.5 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                          >
                            <MoreHorizontal class="w-5 h-5" />
                          </button>
                          
                          <!-- Dropdown Menu -->
                          <div 
                            v-if="activeMoreMenuId === post.id" 
                            class="absolute right-0 top-full mt-2 w-56 rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] shadow-[0_10px_40px_rgba(0,0,0,0.5)] z-50 text-sm overflow-hidden"
                          >
                            <div class="py-1">
                              <button @click="handleMenuAction('share', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">分享</button>
                              <button @click="handleMenuAction('mention', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)] font-medium">提及 {{ post.handle }}</button>
                              <button
                                v-if="currentUser?.id && post.authorId === currentUser.id"
                                @click="handleMenuAction('delete', post)"
                                class="w-full text-left px-4 py-2.5 text-rose-500 hover:bg-rose-500/10"
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
            </div>
          </section>

          <section v-else-if="currentSection === 'messages'" class="h-[calc(100vh-140px)] overflow-hidden">
            <div class="grid h-full min-h-0 lg:grid-cols-[240px_minmax(0,1fr)]">
              <aside class="flex min-h-0 flex-col border-b border-[color:var(--border-color)] bg-[var(--panel-soft)] lg:border-b-0 lg:border-r">
                <div class="border-b border-[color:var(--border-color)] px-5 py-5">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <div class="text-xl font-semibold text-[color:var(--text-primary)]">消息</div>
                      <div class="mt-1 text-sm text-[color:var(--text-muted)]">选择一个联系人开始聊天</div>
                    </div>
                    <div
                      v-if="unreadConversationCount > 0"
                      class="flex h-7 min-w-7 items-center justify-center rounded-full bg-emerald-600 px-1.5 text-[11px] font-bold leading-none text-white shadow-sm"
                    >
                      {{ unreadConversationCount > 99 ? '99+' : unreadConversationCount }}
                    </div>
                  </div>
                </div>

                <div v-if="conversations.length === 0" class="px-5 py-10 text-sm text-[color:var(--text-muted)]">
                  还没有私信消息，去“当前热门 → 用户”里点击“发消息”开始第一段聊天。
                </div>

                <div v-else class="min-h-0 flex-1 overflow-y-auto divide-y divide-[color:var(--border-color)] no-scrollbar">
                  <button
                    v-for="conversation in conversations"
                    :key="conversation.id"
                    @click="openConversation(conversation.id)"
                    class="flex w-full items-start gap-3 px-5 py-4 text-left transition hover:bg-[var(--chip-hover)]"
                    :class="selectedConversationId === conversation.id ? 'bg-emerald-500/10' : ''"
                  >
                    <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                      <img v-if="conversation.avatarUrl" :src="conversation.avatarUrl" class="h-full w-full object-cover" />
                      <template v-else>{{ conversation.avatarLabel }}</template>
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between gap-3">
                        <div class="truncate text-[15px] font-semibold text-[color:var(--text-primary)]">{{ conversation.name }}</div>
                        <div class="flex shrink-0 items-center gap-2">
                          <span
                            v-if="latestPeerMessageTimestamp(conversation) > (readConversationAt[conversation.id] || 0)"
                            class="inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500"
                          />
                          <span class="text-xs text-[color:var(--text-muted)]">{{ conversation.messages[conversation.messages.length - 1]?.time || '' }}</span>
                        </div>
                      </div>
                      <div class="mt-1 truncate text-sm text-[color:var(--text-secondary)]">{{ conversation.handle }}</div>
                      <div v-if="conversation.crossInstance" class="mt-1 truncate text-xs font-medium text-emerald-400">
                        {{ conversation.federationRoute }}
                      </div>
                      <div class="mt-2 truncate text-sm text-[color:var(--text-muted)]">
                        {{ conversation.messages[conversation.messages.length - 1]?.text || '还没有消息，开始打个招呼吧。' }}
                      </div>
                    </div>
                  </button>
                </div>
              </aside>

              <div class="flex min-h-0 flex-col bg-[var(--frame-bg)]">
                <template v-if="activeConversation">
                  <div
                    class="relative flex shrink-0 items-center gap-4 overflow-hidden border-b border-[color:var(--border-color)] px-6 py-5"
                    :style="activeConversation.backgroundUrl ? { backgroundImage: `linear-gradient(rgba(0,0,0,0.30), rgba(0,0,0,0.30)), url(${activeConversation.backgroundUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : {}"
                  >
                    <button
                      v-if="!activeConversation.backgroundUrl && activeConversation.participantId"
                      type="button"
                      class="absolute inset-0 flex items-center justify-center bg-black/20 opacity-0 transition hover:opacity-100"
                      title="查看对方主页"
                      @click.stop="goToUserProfile(activeConversation.participantId)"
                    />
                    <button
                      type="button"
                      class="relative z-10 flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                      title="查看对方主页"
                      @click.stop="activeConversation.participantId && goToUserProfile(activeConversation.participantId)"
                    >
                      <img v-if="activeConversation.avatarUrl" :src="activeConversation.avatarUrl" class="h-full w-full object-cover" />
                      <template v-else>{{ activeConversation.avatarLabel }}</template>
                    </button>
                    <div class="min-w-0">
                      <div class="truncate text-lg font-semibold text-[color:var(--text-primary)]">{{ activeConversation.name }}</div>
                      <div class="mt-1 truncate text-sm text-[color:var(--text-secondary)]">{{ activeConversation.handle }}</div>
                      <div v-if="activeConversation.crossInstance" class="mt-1 inline-flex max-w-full items-center gap-2 rounded-full border border-emerald-500/25 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300">
                        <Globe class="h-3.5 w-3.5 shrink-0" />
                        <span class="truncate">{{ activeConversation.federationRoute }}</span>
                      </div>
                    </div>
                  </div>

                  <div ref="messageListRef" class="min-h-0 flex-1 overflow-y-auto px-6 py-6 no-scrollbar">
                    <div class="space-y-4">
                      <div
                        v-for="message in activeConversation.messages"
                        :key="message.id"
                        class="flex items-end gap-3"
                        :class="message.from === 'me' ? 'justify-end' : 'justify-start'"
                      >
                        <template v-if="message.from === 'peer'">
                          <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                            <img v-if="activeConversation.avatarUrl" :src="activeConversation.avatarUrl" class="h-full w-full object-cover" />
                            <template v-else>{{ activeConversation.avatarLabel }}</template>
                          </div>
                          <div class="max-w-[75%] rounded-[22px] rounded-bl-md border border-[color:var(--border-color)] px-4 py-3 text-sm leading-6 shadow-sm"
                            :class="message.forwardedPost ? 'bg-white text-slate-800' : 'bg-[var(--panel-soft)] text-[color:var(--text-primary)]'">
                            <template v-if="message.forwardedPost">
                              <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-500">转发帖子</div>
                              <button
                                @click="openForwardedPostDetail(message)"
                                class="block w-full overflow-hidden rounded-xl border border-slate-200 bg-white text-left transition hover:border-slate-300 hover:bg-slate-50"
                              >
                                <div class="px-3 pt-3">
                                  <div class="text-xs text-slate-500">{{ forwardedPostAuthorName(message.forwardedPost) }} · {{ formatHandleInstance(message.forwardedPost.handle, message.forwardedPost.instance) }}</div>
                                  <div class="mt-1 line-clamp-2 break-words text-[13px] leading-5 text-slate-800">{{ message.forwardedPost.content }}</div>
                                </div>
                                <div
                                  v-if="message.forwardedPost.media?.preview"
                                  class="mt-2 overflow-hidden border-t border-slate-200 bg-slate-100"
                                >
                                  <img
                                    :src="message.forwardedPost.media.preview"
                                    :alt="message.forwardedPost.media.name || '转发帖子图片'"
                                    class="max-h-[60vh] w-full h-auto object-contain bg-[var(--panel-contrast)]"
                                  />
                                </div>
                              </button>
                            </template>
                            <div v-else>
                              <div v-if="message.text">{{ message.text }}</div>
                              <div v-if="message.imageEmoji?.preview" class="mt-2 overflow-hidden rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]">
                                <img :src="message.imageEmoji.preview" :alt="message.imageEmoji.name || '图片表情'" class="max-h-56 w-full h-auto object-contain" />
                              </div>
                            </div>
                            <div class="mt-2 text-[11px] text-[color:var(--text-muted)]">{{ message.time }}</div>
                          </div>
                        </template>

                        <template v-else>
                          <div class="max-w-[75%] rounded-[22px] rounded-br-md px-4 py-3 text-sm leading-6 shadow-sm"
                            :class="message.forwardedPost ? 'border border-slate-200 bg-white text-slate-800' : 'bg-emerald-600 text-white shadow-[0_10px_30px_rgba(16,185,129,0.22)]'">
                            <template v-if="message.forwardedPost">
                              <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-500">转发帖子</div>
                              <button
                                @click="openForwardedPostDetail(message)"
                                class="block w-full overflow-hidden rounded-xl border border-slate-200 bg-white text-left transition hover:border-slate-300 hover:bg-slate-50"
                              >
                                <div class="px-3 pt-3">
                                  <div class="text-xs text-slate-500">{{ forwardedPostAuthorName(message.forwardedPost) }} · {{ formatHandleInstance(message.forwardedPost.handle, message.forwardedPost.instance) }}</div>
                                  <div class="mt-1 line-clamp-2 break-words text-[13px] leading-5 text-slate-800">{{ message.forwardedPost.content }}</div>
                                </div>
                                <div
                                  v-if="message.forwardedPost.media?.preview"
                                  class="mt-2 overflow-hidden border-t border-slate-200 bg-slate-100"
                                >
                                  <img
                                    :src="message.forwardedPost.media.preview"
                                    :alt="message.forwardedPost.media.name || '转发帖子图片'"
                                    class="max-h-[60vh] w-full h-auto object-contain bg-[var(--panel-contrast)]"
                                  />
                                </div>
                              </button>
                            </template>
                            <div v-else>
                              <div v-if="message.text">{{ message.text }}</div>
                              <div v-if="message.imageEmoji?.preview" class="mt-2 overflow-hidden rounded-xl border border-emerald-300/30 bg-white/10">
                                <img :src="message.imageEmoji.preview" :alt="message.imageEmoji.name || '图片表情'" class="max-h-56 w-full h-auto object-contain" />
                              </div>
                            </div>
                            <div class="mt-2 text-[11px]" :class="message.forwardedPost ? 'text-slate-400' : 'text-emerald-100/80'">{{ message.time }}</div>
                          </div>
                          <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                            <img v-if="currentUser?.avatarUrl" :src="currentUser.avatarUrl" class="h-full w-full object-cover" />
                            <template v-else>{{ avatarText(currentUser?.displayName || 'U') }}</template>
                          </div>
                        </template>
                      </div>

                      <div v-if="activeConversation.messages.length === 0" class="flex min-h-[240px] items-center justify-center">
                        <div class="rounded-3xl border border-dashed border-[color:var(--border-color)] px-8 py-10 text-center text-sm text-[color:var(--text-muted)]">
                          还没有消息，先发一句“你好”吧。
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="shrink-0 border-t border-[color:var(--border-color)] bg-[var(--panel-soft)] px-6 py-5">
                    <div class="flex items-end gap-4">
                      <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900">
                        <img v-if="currentUser?.avatarUrl" :src="currentUser.avatarUrl" class="h-full w-full object-cover" />
                        <template v-else>{{ avatarText(currentUser?.displayName || 'U') }}</template>
                      </div>
                      <div class="min-w-0 flex-1 rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-4 py-3">
                        <div v-if="messageMediaPreview && messageMediaMeta" class="relative mb-3 overflow-hidden rounded-2xl border border-[color:var(--border-color)]">
                          <img :src="messageMediaPreview" :alt="messageMediaMeta.name" class="max-h-44 w-full object-contain bg-[var(--panel-contrast)]" />
                          <button
                            @click="clearMessageMedia"
                            class="absolute right-2 top-2 rounded-full bg-black/55 px-2 py-1 text-[11px] text-white hover:bg-black/70"
                          >
                            移除
                          </button>
                        </div>
                        <textarea
                          ref="messageInputRef"
                          v-model="messageDraft"
                          @keydown.enter.prevent="sendMessage"
                          rows="3"
                          maxlength="1000"
                          placeholder="输入消息..."
                          class="w-full resize-none bg-transparent text-sm leading-6 text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
                        />
                        <div class="mt-3 flex items-center justify-between gap-3">
                          <div class="flex items-center gap-2">
                            <input ref="messageImageInputRef" type="file" accept="image/*" class="hidden" @change="handleMessageMediaChange" />
                            <button @click="triggerMessageImagePicker" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" title="发送图片表情">
                              <ImageIcon class="h-4 w-4" />
                            </button>
                            <button ref="messageStickerTriggerRef" @click.stop="toggleMessageStickerPanel" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" :class="showMessageStickerPanel ? 'text-emerald-500 bg-emerald-500/10' : ''" title="我发表过的表情">
                              <Bookmark class="h-4 w-4" />
                            </button>
                            <button ref="messageEmojiTriggerRef" @click.stop="toggleMessageEmojiPicker" class="rounded-lg p-1.5 transition-colors hover:bg-emerald-500/10" :class="showMessageEmojiPicker ? 'text-emerald-400 bg-emerald-500/10' : ''" title="表情">
                              <Pencil class="h-4 w-4" />
                            </button>
                            <div class="text-xs text-[color:var(--text-muted)]">{{ messageDraft.trim().length }}/1000</div>
                          </div>
                          <button
                            :disabled="(!messageDraft.trim() && !messageMediaPreview) || saving"
                            @click="sendMessage"
                            class="rounded-xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            {{ saving ? '发送中...' : '发送消息' }}
                          </button>
                        </div>

                        <div
                          v-if="showMessageStickerPanel"
                          ref="messageStickerPanelRef"
                          class="mt-3 max-h-52 overflow-y-auto rounded-2xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] p-3"
                        >
                          <div v-if="recentMessageStickers.length === 0" class="text-xs text-[color:var(--text-muted)]">发送过的图片会自动出现在这里。</div>
                          <div v-else class="grid grid-cols-5 gap-2">
                            <button
                              v-for="(sticker, idx) in recentMessageStickers"
                              :key="`${sticker.preview}-${idx}`"
                              @click="useStickerAsMessage(sticker)"
                              class="overflow-hidden rounded-lg border border-[color:var(--border-color)] bg-[var(--panel-contrast)] transition hover:border-emerald-400/60"
                            >
                              <img :src="sticker.preview" :alt="sticker.name || '图片表情'" class="h-14 w-full object-cover" />
                            </button>
                          </div>
                        </div>

                        <div
                          v-if="showMessageEmojiPicker"
                          ref="messageEmojiPanelRef"
                          class="mt-3 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--frame-bg)]"
                        >
                          <emoji-picker @emoji-click="handleMessageEmojiPick" locale="zh-Hans" preview-position="none" skin-tone-emoji="👍"></emoji-picker>
                        </div>
                      </div>
                    </div>
                  </div>
                </template>

                <div v-else class="flex h-full items-center justify-center px-6 py-16 text-center text-[color:var(--text-muted)]">
                  <div>
                    <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-3xl bg-emerald-500/10 text-emerald-400">
                      <Mail class="h-8 w-8" />
                    </div>
                    <div class="mt-5 text-lg font-semibold text-[color:var(--text-primary)]">请选择一条消息</div>
                    <div class="mt-2 text-sm">点击左侧联系人后，这里会切换成聊天室。</div>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section v-else-if="currentSection === 'notifications'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in orderedNotificationItems" :key="item.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="flex items-start gap-4">
                <div class="mt-1 flex h-11 w-11 items-center justify-center rounded-2xl bg-emerald-600/12 text-emerald-600">◌</div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-3">
                    <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                    <div class="text-sm text-[color:var(--text-muted)]">{{ item.time }}</div>
                  </div>
                  <div class="mt-2 line-clamp-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.body }}</div>
                </div>
              </div>
            </article>
          </section>

          <section v-else-if="currentSection === 'lists'" class="divide-y divide-[color:var(--border-color)]">
            <div class="px-6 py-4">
              <div class="text-lg font-semibold text-[color:var(--text-primary)]">联邦实例切换</div>
              <div class="mt-1 text-sm text-[color:var(--text-muted)]">选择实例后将返回主页并按该实例筛选摩文。</div>
            </div>
            <article class="px-6 py-4 transition hover:bg-[var(--panel-soft)]">
              <button @click="selectInstance('all')" class="w-full text-left">
                <div class="text-base font-semibold" :class="selectedInstanceName === 'all' ? 'text-emerald-500' : 'text-[color:var(--text-primary)]'">全部摩尔实例</div>
                <div class="mt-1 text-sm text-[color:var(--text-muted)]">{{ visibleInstances.length }} 个实例</div>
              </button>
            </article>
            <article v-for="instance in visibleInstances" :key="instance.name" class="px-6 py-4 transition hover:bg-[var(--panel-soft)]">
              <button @click="selectInstance(instance.name)" class="w-full text-left">
                <div class="text-base font-semibold" :class="selectedInstanceName === instance.name ? 'text-emerald-500' : 'text-[color:var(--text-primary)]'">{{ instance.name }}</div>
                <div class="mt-1 text-sm text-[color:var(--text-muted)]">{{ instance.focus }} · {{ instance.members }} · {{ instance.latency }}</div>
              </button>
            </article>
          </section>

          <section v-else-if="currentSection === 'topics'" class="divide-y divide-[color:var(--border-color)]">
            <template v-if="!selectedTopicTag">
              <div class="grid grid-cols-1 gap-4 px-6 py-6 sm:grid-cols-2">
                <button
                  v-for="item in followedTopicCards"
                  :key="item.tag"
                  @click="openTopicTagFeed(item.tag)"
                  class="group relative overflow-hidden rounded-2xl border border-emerald-400/25 bg-[linear-gradient(160deg,rgba(7,18,14,0.94)_0%,rgba(10,28,20,0.92)_100%)] px-5 py-5 text-left shadow-[0_16px_36px_rgba(0,0,0,0.35)] transition-all hover:-translate-y-0.5 hover:border-emerald-300/45 hover:shadow-[0_20px_42px_rgba(16,185,129,0.18)]"
                >
                  <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_100%_0%,rgba(52,211,153,0.18)_0%,transparent_48%)]"></div>
                  <div class="pointer-events-none absolute inset-0 bg-[linear-gradient(120deg,rgba(16,185,129,0.08),transparent_45%,rgba(110,231,183,0.06))] opacity-70"></div>
                  <div class="relative text-xl font-semibold tracking-[0.02em] text-emerald-100 transition group-hover:text-white">#{{ item.label }}</div>
                </button>
              </div>
            </template>
            <template v-else>
              <div class="px-6 py-4 flex items-center justify-between border-b border-[color:var(--border-color)]">
                <div class="text-lg font-semibold text-[color:var(--text-primary)]">#{{ selectedTopicTag }}</div>
                <button @click="clearTopicTagFeed" class="rounded-xl border border-[color:var(--border-color)] px-4 py-2 text-sm font-semibold text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)]">返回话题</button>
              </div>
              <div v-if="topicTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">该话题下暂无摩文。</div>
              <PostFeedCard
                v-for="post in topicTimeline"
                v-else
                :key="post.id"
                :post="post"
                :avatar-url="userAvatarUrl(post.authorId)"
                :liked="likedPosts[post.id]"
                :bookmarked="bookmarkedPosts[post.id]"
                :current-user-id="currentUser?.id"
                :mention-users="people"
                :show-more-menu="true"
                :more-menu-open="activeMoreMenuId === post.id"
                @open-profile="goToUserProfile"
                @open-detail="openPostDetail"
                @forward="openForwardDialog"
                @toggle-like="toggleLike"
                @toggle-bookmark="toggleBookmark"
                @toggle-more="toggleMoreMenu"
                @menu-action="handleMenuAction"
                @vote="handleVote"
              />
            </template>
          </section>

          <section v-else-if="currentSection === 'likes'" class="divide-y divide-[color:var(--border-color)]">
            <article v-if="likedTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              你点赞的内容会显示在这里。
            </article>
            <PostFeedCard
              v-for="post in likedTimeline"
              v-else
              :key="post.id"
              :post="post"
              :avatar-url="userAvatarUrl(post.authorId)"
              :liked="likedPosts[post.id]"
              :bookmarked="bookmarkedPosts[post.id]"
              :current-user-id="currentUser?.id"
              :mention-users="people"
              :show-more-menu="true"
              :more-menu-open="activeMoreMenuId === post.id"
              @open-profile="goToUserProfile"
              @open-detail="openPostDetail"
              @forward="openForwardDialog"
              @toggle-like="toggleLike"
              @toggle-bookmark="toggleBookmark"
              @toggle-more="toggleMoreMenu"
              @menu-action="handleMenuAction"
              @vote="handleVote"
            />
          </section>

          <section v-else-if="currentSection === 'bookmarks'" class="divide-y divide-[color:var(--border-color)]">
            <article v-if="bookmarkedTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              收藏的动态会整理在这里，方便稍后继续阅读。
            </article>
            <PostFeedCard
              v-for="post in bookmarkedTimeline"
              v-else
              :key="post.id"
              :post="post"
              :avatar-url="userAvatarUrl(post.authorId)"
              :liked="likedPosts[post.id]"
              :bookmarked="bookmarkedPosts[post.id]"
              :current-user-id="currentUser?.id"
              :mention-users="people"
              :show-more-menu="true"
              :more-menu-open="activeMoreMenuId === post.id"
              @open-profile="goToUserProfile"
              @open-detail="openPostDetail"
              @forward="openForwardDialog"
              @toggle-like="toggleLike"
              @toggle-bookmark="toggleBookmark"
              @toggle-more="toggleMoreMenu"
              @menu-action="handleMenuAction"
              @vote="handleVote"
            />
          </section>

          <section v-else-if="currentSection === 'mentions'" class="divide-y divide-[color:var(--border-color)]">
            <div class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[color:var(--text-muted)]">收到的提及</div>
            <article v-if="mentionItems.length === 0" class="px-6 py-8 text-center text-sm text-[color:var(--text-muted)]">
              暂无别人提及你的内容。
            </article>
            <article v-for="item in mentionItems" v-else :key="item.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="flex items-start gap-4">
                <div class="mt-1 flex h-11 w-11 items-center justify-center rounded-2xl bg-emerald-600/12 text-emerald-600">@</div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-3">
                    <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                    <div class="text-sm text-[color:var(--text-muted)]">{{ item.time }}</div>
                  </div>
                  <div class="mt-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.body }}</div>
                </div>
              </div>
            </article>

            <div class="px-5 py-3 text-xs font-semibold uppercase tracking-wider text-[color:var(--text-muted)]">我发出的提及</div>
            <article v-if="outgoingMentionItems.length === 0" class="px-6 py-8 text-center text-sm text-[color:var(--text-muted)]">
              你还没有发出提及。
            </article>
            <article
              v-for="item in outgoingMentionItems"
              v-else
              :key="item.id"
              class="cursor-pointer px-5 py-5 transition hover:bg-[var(--panel-soft)]"
              @click="openPostDetail(item.postId, false)"
            >
              <div class="flex items-start gap-4">
                <div class="mt-1 flex h-11 w-11 items-center justify-center rounded-2xl bg-cyan-600/12 text-cyan-500">@</div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-3">
                    <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                    <div class="text-sm text-[color:var(--text-muted)]">{{ item.time }}</div>
                  </div>
                  <div class="mt-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.body }}</div>
                </div>
              </div>
            </article>
          </section>

          <!-- preferences section removed - moved to SettingsPage.vue -->

          <section v-else class="px-6 py-10 text-center text-[color:var(--text-muted)]">
            暂无内容。
          </section>
        </main>

        <aside class="border-t border-[color:var(--border-color)] bg-[var(--panel-bg)] lg:h-[calc(100vh-24px)] lg:overflow-y-auto no-scrollbar lg:border-l lg:border-t-0">
          <div class="p-4">
            <div class="mb-5 flex items-center gap-2">
              <img src="/logo.png" alt="MoleSociety logo" class="h-8 w-8 rounded-xl object-cover shadow-sm shadow-emerald-500/20" />
              <div class="text-[17px] font-bold tracking-tight text-[color:var(--text-primary)]">MoleSociety</div>
            </div>

            <div class="space-y-0.5">
              <button
                v-for="item in primaryNavItems"
                :key="item.key"
                @click="item.key === 'preferences' ? router.push('/settings') : setSection(item.key)"
                class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-[15px] font-medium transition-all hover:translate-x-1"
                :class="currentSection === item.key ? 'bg-emerald-600/10 text-emerald-500 font-semibold' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-hover)]'"
              >
                <component :is="item.icon" class="w-[18px] h-[18px] stroke-[1.8]" />
                <span>{{ item.label }}</span>
                <span
                  v-if="item.key === 'notifications' && unreadNotificationCount > 0"
                  class="ml-auto inline-flex min-w-[20px] items-center justify-center rounded-full bg-rose-500 px-1.5 py-0.5 text-[11px] font-semibold leading-none text-white"
                >
                  {{ unreadNotificationCount > 99 ? '99+' : unreadNotificationCount }}
                </span>
              </button>
            </div>

            <div class="mt-5 border-t border-[color:var(--border-color)] pt-5">
              <div class="space-y-0.5">
                <button
                  v-for="item in secondaryNavItems"
                  :key="item.key"
                  @click="setSection(item.key)"
                  class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-[14px] font-medium transition-all hover:translate-x-0.5 hover:bg-[var(--chip-hover)]"
                  :class="currentSection === item.key ? 'bg-emerald-600/10 text-emerald-500' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'"
                >
                  <component :is="item.icon" class="w-[17px] h-[17px] stroke-[1.5]" />
                  <span>{{ item.label }}</span>
                  <span
                    v-if="item.key === 'mentions' && unreadMentionCount > 0"
                    class="ml-auto inline-flex min-w-[20px] items-center justify-center rounded-full bg-rose-500 px-1.5 py-0.5 text-[11px] font-semibold leading-none text-white"
                  >
                    {{ unreadMentionCount > 99 ? '99+' : unreadMentionCount }}
                  </span>
                </button>
              </div>
            </div>

            <div class="mt-6 border-t border-[color:var(--border-color)] pt-6">
              <div class="space-y-1">
                <button
                  v-for="item in utilityNavItems"
                  :key="item.key"
                  @click="item.key === 'preferences' ? router.push('/settings') : setSection(item.key)"
                  class="flex w-full items-center gap-3 rounded-[1.2rem] px-3 py-2.5 text-base font-medium transition-all hover:translate-x-1 hover:bg-[var(--chip-hover)]"
                  :class="currentSection === item.key ? 'bg-emerald-600/12 text-emerald-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'"
                >
                  <component :is="item.icon" class="w-5 h-5 stroke-[1.5]" />
                  <span>{{ item.label }}</span>
                </button>

                <button
                  v-if="isLoggedIn"
                  @click="goToLogout"
                  class="flex w-full items-center gap-3 rounded-[1.2rem] px-3 py-2.5 text-base font-medium text-[color:var(--text-secondary)] transition hover:translate-x-1 hover:bg-[var(--chip-hover)]"
                >
                  <LogOut class="w-5 h-5 stroke-[1.5]" />
                  <span>退出登录</span>
                </button>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <Transition name="modal">
        <div
          v-if="showDeletePostDialog"
          class="fixed inset-0 z-[240] flex items-center justify-center bg-black/45 px-4"
          @click="cancelDeletePost"
        >
          <div
            class="relative w-full max-w-md rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] p-6 shadow-[0_30px_80px_rgba(0,0,0,0.45)]"
            @click.stop
          >
            <div class="text-lg font-semibold text-[color:var(--text-primary)]">删除这条摩文？</div>
            <div class="mt-2 text-sm leading-6 text-[color:var(--text-secondary)]">
              删除后无法恢复，相关内容会从时间线中移除。
            </div>
            <div v-if="deletingPost" class="mt-4 rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-sm text-[color:var(--text-primary)]">
              {{ deletingPost.content || '（无正文内容）' }}
            </div>
            <div class="mt-6 flex items-center justify-end gap-3">
              <button
                @click="cancelDeletePost"
                :disabled="deletingPostLoading"
                class="rounded-xl border border-[color:var(--border-color)] px-4 py-2 text-sm font-semibold text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] disabled:opacity-50"
              >
                取消
              </button>
              <button
                @click="confirmDeletePost"
                :disabled="deletingPostLoading"
                class="rounded-xl bg-rose-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-rose-500 disabled:opacity-50"
              >
                {{ deletingPostLoading ? '删除中...' : '确认删除' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>

      <Transition name="modal">
        <div
          v-if="showForwardDialog"
          class="fixed inset-0 z-[230] flex items-center justify-center bg-black/45 px-4"
          @click="closeForwardDialog"
        >
          <div
            class="relative w-full max-w-lg rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] p-6 shadow-[0_30px_80px_rgba(0,0,0,0.45)]"
            @click.stop
          >
            <div class="mb-4 flex items-center justify-between">
              <div>
                <div class="text-lg font-semibold text-[color:var(--text-primary)]">转发到会话</div>
                <div class="mt-1 text-sm text-[color:var(--text-muted)]">选择一个联系人会话并发送</div>
              </div>
              <button @click="closeForwardDialog" class="rounded-lg p-1.5 text-[color:var(--text-muted)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]">
                <X class="h-4 w-4" />
              </button>
            </div>
            <div class="mb-4 max-h-64 overflow-y-auto rounded-2xl border border-[color:var(--border-color)]">
              <button
                v-for="conversation in conversations"
                :key="conversation.id"
                @click="forwardingConversationId = conversation.id"
                class="flex w-full items-center justify-between border-b border-[color:var(--border-color)] px-4 py-3 text-left last:border-b-0"
                :class="forwardingConversationId === conversation.id ? 'bg-emerald-500/10' : 'hover:bg-[var(--panel-soft)]'"
              >
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold text-[color:var(--text-primary)]">{{ conversation.name }}</div>
                  <div class="truncate text-xs text-[color:var(--text-muted)]">{{ conversation.handle }}</div>
                </div>
                <div v-if="forwardingConversationId === conversation.id" class="text-xs font-semibold text-emerald-300">已选择</div>
              </button>
              <div v-if="conversations.length === 0" class="px-4 py-8 text-center text-sm text-[color:var(--text-muted)]">
                暂无会话，请先在“用户”页发起聊天。
              </div>
            </div>
            <div class="flex items-center justify-end gap-3">
              <button @click="closeForwardDialog" class="rounded-xl border border-[color:var(--border-color)] px-4 py-2 text-sm font-medium text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)]">
                取消
              </button>
              <button
                :disabled="!forwardingConversationId || !forwardingPost || forwarding"
                @click="forwardPostToConversation"
                class="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {{ forwarding ? '转发中...' : '确认转发' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>

      <Transition name="modal">
        <div
          v-if="showVisibilityModal"
          class="fixed inset-0 z-[220] flex items-center justify-center bg-black/45 px-4"
          @click="closeVisibilityModal"
        >
          <div
            class="relative w-full max-w-xl rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] p-6 shadow-[0_30px_80px_rgba(0,0,0,0.45)]"
            @click.stop
          >
            <div class="mb-5 flex items-start justify-between gap-4">
              <div>
                <h3 class="text-lg font-semibold text-[color:var(--text-primary)]">发布范围与引用权限</h3>
                <p class="mt-1 text-sm text-[color:var(--text-muted)]">选择这条内容谁可以看到，以及谁可以引用。</p>
              </div>
              <button
                @click="closeVisibilityModal"
                class="rounded-lg p-1.5 text-[color:var(--text-muted)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                title="关闭"
              >
                <X class="h-4 w-4" />
              </button>
            </div>

            <div class="space-y-6">
              <div>
                <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">可见性</div>
                <div class="space-y-2">
                  <button
                    v-for="option in visibilityOptions"
                    :key="option.id"
                    @click="tempVisibility = option.id"
                    class="flex w-full items-center gap-3 rounded-xl border px-4 py-3 text-left transition"
                    :class="tempVisibility === option.id ? 'border-emerald-500/60 bg-emerald-500/10' : 'border-[color:var(--border-color)] hover:border-emerald-500/30 hover:bg-[var(--panel-soft)]'"
                  >
                    <component :is="option.icon" class="h-4 w-4 text-emerald-400" />
                    <div class="min-w-0 flex-1">
                      <div class="text-sm font-semibold text-[color:var(--text-primary)]">{{ option.label }}</div>
                      <div class="text-xs text-[color:var(--text-muted)]">{{ option.description }}</div>
                    </div>
                    <div class="h-3 w-3 rounded-full border border-emerald-400/70" :class="tempVisibility === option.id ? 'bg-emerald-400' : 'bg-transparent'" />
                  </button>
                </div>
              </div>

              <div>
                <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">引用权限</div>
                <div class="grid gap-2 sm:grid-cols-3">
                  <button
                    v-for="option in interactionOptions"
                    :key="option.id"
                    @click="tempInteraction = option.id"
                    class="rounded-xl border px-3 py-2.5 text-sm font-medium transition"
                    :class="tempInteraction === option.id ? 'border-emerald-500/60 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-500/40'"
                  >
                    {{ option.label }}
                  </button>
                </div>
              </div>
            </div>

            <div class="mt-6 flex items-center justify-end gap-3">
              <button
                @click="closeVisibilityModal"
                class="rounded-xl border border-[color:var(--border-color)] px-4 py-2 text-sm font-medium text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)]"
              >
                取消
              </button>
              <button
                @click="saveVisibilitySettings"
                class="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-emerald-500"
              >
                保存
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
  .modal-enter-active,
  .modal-leave-active {
    transition: opacity 0.3s ease;
  }

  .modal-enter-from,
  .modal-leave-to {
    opacity: 0;
  }

  .modal-enter-active .relative,
  .modal-leave-active .relative {
    transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .modal-enter-from .relative,
  .modal-leave-to .relative {
    transform: scale(0.9) translateY(20px);
  }
.modal-enter-from .relative,
.modal-leave-to .relative {
  transform: scale(0.9) translateY(20px);
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease-out;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

.no-scrollbar::-webkit-scrollbar {
  display: none;
}

.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

emoji-picker {
  width: 100%;
  max-height: min(22rem, 55vh);
  --border-radius: 12px;
}
</style>

