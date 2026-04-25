<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import 'emoji-picker-element';
import {
  createConversation,
  createConversationMessage,
  createMediaAsset,
  createPost,
  fetchPostReplies,
  fetchPostThread,
  fetchSocialBootstrap,
  fetchSocialBootstrapMine,
  getConversation,
  listConversations,
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
  Home, Compass, Bell, List, Hash, Star, Bookmark, AtSign, Settings,
  MoreHorizontal, User, Shield, PenTool, Mail, AlignJustify, Users,
  Filter, Trash2, Image as ImageIcon, CheckSquare, Smile, Search,
  ArrowLeft, ChevronLeft, LogOut, MessageCircle, Repeat, Heart, Pencil, TrendingUp, Newspaper,
  Globe, Moon, Lock, ChevronDown, ChevronUp, X, BarChart3, RefreshCw
} from 'lucide-vue-next';
import { useAppearance } from '../composables/useAppearance';

type Section =
  | 'home'
  | 'myFeed'
  | 'postDetail'
  | 'explore'
  | 'messages'
  | 'notifications'
  | 'lists'
  | 'topics'
  | 'likes'
  | 'bookmarks'
  | 'mentions'
  | 'preferences'
  | 'more';

type ExploreTab = 'posts' | 'topics' | 'users' | 'news';

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
  };
  interaction: string;
  poll?: Poll;
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
  time: string;
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
  { label: '我的内容', key: 'myFeed', icon: User },
  { label: '通知', key: 'notifications', icon: Bell },
];

const secondaryNavItems: { label: string; key: Section; icon: Component }[] = [
  { label: '列表', key: 'lists', icon: List },
  { label: '关注的话题', key: 'topics', icon: Hash },
  { label: '喜欢', key: 'likes', icon: Star },
  { label: '书签', key: 'bookmarks', icon: Bookmark },
  { label: '提及', key: 'mentions', icon: AtSign },
];

const utilityNavItems: { label: string; key: Section; icon: Component }[] = [
  { label: '偏好设置', key: 'preferences', icon: Settings },
  { label: '更多', key: 'more', icon: MoreHorizontal },
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
const defaultTagOptions = ['创作者动态', '联邦社交', '产品更新', '技术分享', '问答', '公告'];
const recentPostTags = ref<string[]>([]);
const pollOptions = ref(['', '']);
const pollExpiresIn = ref(1440); // 1 day
const pollMultiple = ref(false);
const messageDraft = ref('');
const searchQuery = ref('');
const selectedConversationId = ref('');
const mediaPreview = ref<string | null>(null);
const mediaMeta = ref<{ name: string; sizeLabel: string; type: string; sizeBytes: number } | null>(null);
const replyDraft = ref('');
const messageListRef = ref<HTMLElement | null>(null);
const loading = ref(true);
const saving = ref(false);
const apiOnline = ref(false);
const errorMessage = ref('');
const followedUsers = ref<Record<string, boolean>>({});
const likedPosts = ref<Record<string, boolean>>({});
const boostedPosts = ref<Record<string, boolean>>({});
const bookmarkedPosts = ref<Record<string, boolean>>({});
const selectedInstanceName = ref('all');
const showInstanceDropdown = ref(false);
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
const postComposerRef = ref<HTMLTextAreaElement | null>(null);
const emojiPickerPanelRef = ref<HTMLElement | null>(null);
const emojiTriggerRef = ref<HTMLElement | null>(null);
const postSelectionStart = ref(0);
const postSelectionEnd = ref(0);
const router = useRouter();
const { session: authSession } = useAuth();

const MAX_POST_LENGTH = 500;
const MAX_POST_TAGS = 5;
const MAX_TAG_LENGTH = 24;
const RECENT_TAGS_STORAGE_KEY = 'mole-compose-recent-tags';
const PULL_MAX_DISTANCE = 120;
const PULL_REFRESH_THRESHOLD = 72;

const isLoggedIn = computed(() => !!authSession.value);

const { themeStyles, appearanceSettings } = useAppearance();

const activeExploreTab = ref<ExploreTab>('posts');

const newsPosts = computed(() => posts.value.filter(p => p.type === 'news'));
const activeMoreMenuId = ref<string | null>(null);

function toggleMoreMenu(postId: string) {
  activeMoreMenuId.value = activeMoreMenuId.value === postId ? null : postId;
}

function handleMenuAction(action: string, post: FeedCard) {
  activeMoreMenuId.value = null; // Close menu after action
  // Placeholder actions
  console.log(`Action [${action}] on post:`, post.id);
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

const selectedInstance = computed(() =>
  instances.value.find((instance) => instance.name === selectedInstanceName.value) ?? null,
);

const activeProfileInstanceName = computed(() =>
  selectedInstanceName.value === 'all'
    ? currentUser.value?.instance || '摩尔1号'
    : selectedInstanceName.value,
);

const homeTimeline = computed(() => {
  if (selectedInstanceName.value === 'all') return timeline.value;
  return timeline.value.filter((post) => post.instance === selectedInstanceName.value);
});

async function selectInstance(name: string) {
  selectedInstanceName.value = name;
  showInstanceDropdown.value = false;
  if (name === 'all') return;
  if (!currentUser.value) return;
  if (!apiOnline.value) {
    goToNotFound();
    return;
  }

  try {
    const updatedUser = await updateUserProfile(currentUser.value.id, { instance: name });
    currentUser.value = updatedUser;
    people.value = people.value.map((person) => (person.id === updatedUser.id ? updatedUser : person));
    const payload = await fetchSocialBootstrap();
    applyBootstrap(payload);
    apiOnline.value = true;
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.code === 'AUTH_SESSION_REQUIRED')) {
      void router.push({ path: '/login', query: { redirect: '/app' } });
      return;
    }
    goToNotFound();
  }
}

function toggleInstanceDropdown() {
  showInstanceDropdown.value = !showInstanceDropdown.value;
}

const recommendedPeople = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  return people.value
    .filter((person) => person.id !== currentUser.value?.id)
    .filter((person) =>
      !query ||
      [person.displayName, person.handle, person.instance, person.bio]
        .join(' ')
        .toLowerCase()
        .includes(query),
    );
});

const trendingTags = computed(() => {
  const bucket = new Map<string, number>();
  posts.value.forEach((post) => {
    post.tags.forEach((tag) => {
      bucket.set(tag, (bucket.get(tag) ?? 0) + 1);
    });
  });
  return [...bucket.entries()]
    .sort((left, right) => right[1] - left[1])
    .slice(0, 6)
    .map(([tag, count]) => ({ tag, count }));
});

const availablePostTags = computed(() => {
  const pool = [...defaultTagOptions, ...recentPostTags.value, ...trendingTags.value.map((item) => item.tag)];
  return [...new Set(pool.map((tag) => tag.trim()).filter(Boolean))];
});

const serviceNotice = computed(() =>
  errorMessage.value || '跨实例动态正在持续刷新。',
);

const currentSectionInfo = computed(() => {
  const allNavItems = [...primaryNavItems, ...secondaryNavItems, ...utilityNavItems];
  const navItem = allNavItems.find(item => item.key === currentSection.value);
  
  if (navItem) return navItem;
  
  if (currentSection.value === 'postDetail') {
    return { label: '摩文详情', icon: MessageCircle };
  }

  if (currentSection.value === 'myFeed') {
    return { label: '我的内容', icon: User };
  }
  
  return { label: '更多', icon: MoreHorizontal };
});

const likedTimeline = computed(() => posts.value.filter((post) => likedPosts.value[post.id]));

const bookmarkedTimeline = computed(() => posts.value.filter((post) => bookmarkedPosts.value[post.id]));

const notificationItems = computed(() => {
  const suggestedUsers = recommendedPeople.value.slice(0, 2).map((person) => {
    const hasPublished = posts.value.some((post) => post.authorId === person.id);
    return {
      id: `follow-${person.id}`,
      title: `${person.displayName} 开始在社区活跃`,
      body: hasPublished
        ? `${person.handle}@${person.instance} 已发布动态，适合加入你的关注流。`
        : `${person.handle}@${person.instance} 刚加入社区，欢迎关注。`,
      time: '刚刚',
    };
  });

  const postEvents = timeline.value.slice(0, 2).map((post) => ({
    id: `post-${post.id}`,
    title: `${post.author} 发布了新内容`,
    body: post.content,
    time: post.time,
  }));

  return [...suggestedUsers, ...postEvents];
});

const curatedLists = computed(() => [
  {
    id: 'list-creators',
    title: '链上创作者',
    summary: '关注独立创作者、长期写作者和内容档案馆。',
    count: `${recommendedPeople.value.length} 位成员`,
  },
  {
    id: 'list-readers',
    title: '阅读与知识节点',
    summary: '聚合阅读社群、图书馆节点和内容策展者。',
    count: `${instances.value.length} 个实例`,
  },
  {
    id: 'list-archives',
    title: '永久存储观察',
    summary: '跟踪媒体上链、归档状态和内容留存趋势。',
    count: `${assets.value.length} 个资源`,
  },
]);

const followedTopicCards = computed(() =>
  trendingTags.value.map((item) => ({
    ...item,
    summary: timeline.value.find((post) => post.tags.includes(item.tag))?.content ?? '正在汇聚新的讨论内容。',
  })),
);

const mentionItems = computed(() => {
  const handle = currentUser.value?.handle ?? '';
  return conversations.value.map((conversation) => ({
    id: conversation.id,
    title: conversation.name,
    body: `${conversation.handle} 在私密对话中提到了 ${handle}`,
    time: conversation.messages[conversation.messages.length - 1]?.time ?? '刚刚',
  }));
});

const moreCards = computed(() => [
  {
    id: 'chat',
    title: '会话聊天',
    description: `${conversations.value.length} 个会话正在使用中，后续会继续并入私密沟通入口。`,
  },
  {
    id: 'media',
    title: '媒体资源',
    description: `${assets.value.length} 个资源已进入存储面板，可继续扩展到对象存储与永久归档。`,
  },
  {
    id: 'federation',
    title: '联邦实例',
    description: `${instances.value.length} 个实例已接入展示，用于跨社区发现与联邦观察。`,
  },
]);

// themeStyles moved to composable

// appearance computed removed

// appearance watches removed

function setSection(section: Section) {
  currentSection.value = section;
  // settings transition removed
  if (section !== 'postDetail') {
    threadError.value = '';
    threadLoading.value = false;
    replyDraft.value = '';
    activeReplyTarget.value = null;
  }
}

function toggleLike(postId: string) {
  likedPosts.value = { ...likedPosts.value, [postId]: !likedPosts.value[postId] };
}

function toggleBoost(postId: string) {
  boostedPosts.value = { ...boostedPosts.value, [postId]: !boostedPosts.value[postId] };
}

function toggleBookmark(postId: string) {
  bookmarkedPosts.value = { ...bookmarkedPosts.value, [postId]: !bookmarkedPosts.value[postId] };
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
  const person = people.value.find((item) => item.id === post.authorId);
  return {
    id: post.id,
    authorId: post.authorId,
    author: post.authorName,
    handle: post.authorHandle,
    instance: post.instance,
    kind: post.kind || (post.parentPostId ? 'reply' : 'post'),
    parentPostId: post.parentPostId,
    rootPostId: post.rootPostId,
    replyDepth: post.replyDepth ?? 0,
    time: formatTimestamp(post.createdAt),
    content: post.content,
    type: post.type || 'post',
    interaction: post.interaction || 'anyone',
    bio: person?.bio,
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
      likes: post.likes,
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

  const resolvedTitle =
    conversation.title.trim() ||
    displayParticipants.map((person) => person.displayName).join('、') ||
    otherParticipantIds.join(', ') ||
    participantIds.join(', ') ||
    '新会话';

  const resolvedHandle =
    displayParticipants.map((person) => `${person.handle}@${person.instance}`).join(', ') ||
    (fallbackPeer ? `${fallbackPeer.handle}@${fallbackPeer.instance}` : '') ||
    otherParticipantIds.join(', ') ||
    participantIds.join(', ');

  return {
    id: conversation.id,
    name: resolvedTitle,
    handle: resolvedHandle,
    status: conversation.crossInstance ? '跨联邦会话' : conversation.encrypted ? '端到端加密会话' : '同实例会话',
    crossInstance: Boolean(conversation.crossInstance),
    federationRoute: conversation.federationRoute || '',
    assetUri: conversation.assetUri,
    chainId: conversation.chainId,
    txHash: conversation.txHash,
    contractAddress: conversation.contractAddress,
    explorerUrl: conversation.explorerUrl,
    avatarLabel: avatarText(resolvedTitle),
    participantId: fallbackPeer?.id,
    messages: conversation.messages.map((message) => ({
      id: message.id,
      from: message.senderId === userId ? 'me' : 'peer',
      text: message.body,
      time: formatTimestamp(message.createdAt),
      assetUri: message.assetUri,
      chainId: message.chainId,
      txHash: message.txHash,
      contractAddress: message.contractAddress,
      explorerUrl: message.explorerUrl,
    })),
  };
}

function upsertConversation(conversation: ConversationCard) {
  const remaining = conversations.value.filter((item) => item.id !== conversation.id);
  conversations.value = [conversation, ...remaining];
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
    const mapped = items.map((conversation) => toConversationCard(conversation, currentUser.value?.id ?? null));
    conversations.value = mapped;
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
  await loadConversationMessages(conversationId);
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
      title: isCrossInstanceUser(targetUser) ? `跨联邦：${targetUser.displayName}` : targetUser.displayName,
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
  people.value = payload.users;
  posts.value = payload.feed.map(toFeedCard);
  assets.value = payload.media.map(toAssetCard);
  const mappedConversations = payload.conversations.map((conversation) =>
    toConversationCard(conversation, currentUser.value?.id ?? null),
  );
  conversations.value = mappedConversations;
  instances.value = payload.instances;
  selectedConversationId.value = conversations.value[0]?.id ?? '';
}

async function loadMyFeed() {
  if (!authSession.value) {
    myPosts.value = [];
    return;
  }

  const payload = await fetchSocialBootstrapMine();
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
  showVisibilityModal.value = false;
}

async function loadBootstrap() {
  if (!authSession.value) {
    await router.replace({ path: '/login', query: { redirect: '/app' } });
    return;
  }

  loading.value = true;
  try {
    const payload = await fetchSocialBootstrap();
    applyBootstrap(payload);
    await loadMyFeed();
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
      fetchSocialBootstrap(),
      authSession.value ? fetchSocialBootstrapMine() : Promise.resolve(null),
    ]);
    applyBootstrap(payload);
    myPosts.value = minePayload ? minePayload.feed.map(toFeedCard) : [];
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

function replyOffset(post: FeedCard) {
  const depth = Math.max(0, post.replyDepth ?? 0);
  return `${Math.min(depth, 4) * 28}px`;
}

async function submitReply() {
  if (!replyDraft.value.trim() || !currentUser.value || !threadFocusPost.value || !activeReplyTarget.value || saving.value) return;

  const targetPost = activeReplyTarget.value;
  const rootPostId = threadFocusPost.value.rootPostId || threadFocusPost.value.id;
  saving.value = true;
  try {
    if (!apiOnline.value) {
      goToNotFound();
      return;
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
      mediaIds: [],
      parentPostId: targetPost.id,
      rootPostId,
      type: 'post',
    });
    bumpReplyCount(targetPost.id);
    await openPostDetail(selectedPostId.value || rootPostId);

    replyDraft.value = '';
    activeReplyTarget.value = threadFocusPost.value;
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

function toggleFollow(userId: string) {
  followedUsers.value = {
    ...followedUsers.value,
    [userId]: !followedUsers.value[userId],
  };
}

function goToUserProfile(userId: string) {
  if (!userId) return;
  void router.push(`/profile/${userId}`);
}

function togglePollEditor() {
  showPollEditor.value = !showPollEditor.value;
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

function loadRecentPostTags() {
  if (typeof window === 'undefined') return;
  try {
    const raw = window.localStorage.getItem(RECENT_TAGS_STORAGE_KEY);
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
  window.localStorage.setItem(RECENT_TAGS_STORAGE_KEY, JSON.stringify(recentPostTags.value));
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

function toggleEmojiPicker() {
  showEmojiPicker.value = !showEmojiPicker.value;
  if (showEmojiPicker.value) {
    syncPostCursor();
  }
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

function handleDocumentClick(event: MouseEvent) {
  if (!showEmojiPicker.value) return;
  const target = event.target as Node | null;
  if (!target) return;
  if (emojiPickerPanelRef.value?.contains(target)) return;
  if (emojiTriggerRef.value?.contains(target)) return;
  showEmojiPicker.value = false;
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
      pollMultiple: showPollEditor.value ? pollMultiple.value : false,
    });
    posts.value = [toFeedCard(createdPost), ...posts.value];
    errorMessage.value = '';

    postDraft.value = '';
    mediaPreview.value = null;
    mediaMeta.value = null;
    updateRecentTags(selectedPostTags.value);
    selectedPostTags.value = [];
    customTagInput.value = '';
    showTagPicker.value = false;
    showEmojiPicker.value = false;
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
  if (!messageDraft.value.trim() || !currentUser.value || !activeConversation.value || saving.value) return;

  saving.value = true;
  try {
    const targetConversation = activeConversation.value;
    if (!targetConversation) return;

    if (!apiOnline.value) {
      goToNotFound();
      return;
    }

    const updatedConversation = await createConversationMessage(targetConversation.id, {
      senderId: currentUser.value.id,
      body: messageDraft.value.trim(),
    });

    const mapped = toConversationCard(updatedConversation, currentUser.value?.id ?? null);
    upsertConversation(mapped);
    selectedConversationId.value = mapped.id;

    messageDraft.value = '';
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

onMounted(() => {
  void loadBootstrap();
  loadRecentPostTags();
  document.addEventListener('click', handleDocumentClick);
});

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick);
});

watch(
  () => currentSection.value,
  (section) => {
    if (section !== 'messages') return;
    void refreshConversations(true);
    if (selectedConversationId.value) {
      void loadConversationMessages(selectedConversationId.value);
    }
  },
);
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
        <aside class="relative z-[80] min-h-0 max-h-[calc(100vh-24px)] overflow-hidden border-b border-[color:var(--border-color)] bg-[var(--panel-bg)] lg:h-[calc(100vh-32px)] lg:max-h-none lg:border-b-0 lg:border-r">
          <div class="max-h-[calc(100vh-24px)] min-h-0 space-y-3 overflow-y-auto overscroll-contain p-4 no-scrollbar lg:h-full lg:max-h-none">
            <div class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-4">
              <input
                v-model="searchQuery"
                placeholder="搜索或输入网址"
                class="w-full bg-transparent text-sm text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
              />
            </div>

            <div class="flex items-center gap-3">
              <button
                @click="currentUser?.id && goToUserProfile(currentUser.id)"
                class="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-lime-200 to-cyan-200 text-lg font-bold text-slate-900"
                title="查看我的主页"
              >
                {{ avatarText(currentUser?.displayName || 'W') }}
              </button>
              <div class="min-w-0">
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
                <span><strong class="text-[color:var(--text-primary)]">{{ currentUser?.followers ?? 0 }}</strong> 关注者</span>
                <span><strong class="text-[color:var(--text-primary)]">{{ currentUser?.following ?? 0 }}</strong> 关注中</span>
              </div>
              <div class="flex items-center gap-2">
                <button
                  @click="router.push('/profile/edit')"
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
                      <button @click="pollMultiple = !pollMultiple" class="block w-full text-left text-sm font-bold text-emerald-400">
                        {{ pollMultiple ? '多选' : '单选' }}
                      </button>
                    </div>
                  </div>
                </div>
              </Transition>

              <div class="mt-4 flex flex-col gap-3">
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
                      :class="showTagPicker ? 'text-cyan-400 bg-cyan-500/10' : 'hover:bg-cyan-500/10'"
                      title="选择标签"
                    >
                      <Hash class="w-5 h-5 cursor-pointer transition-transform hover:scale-110" :class="showTagPicker ? 'text-cyan-400' : 'hover:text-cyan-400'" />
                    </button>
                    <button ref="emojiTriggerRef" @click.stop="toggleEmojiPicker" class="rounded-lg p-1.5 transition-colors hover:bg-yellow-400/10" :class="showEmojiPicker ? 'text-yellow-400 bg-yellow-400/10' : ''" title="表情">
                      <Smile class="w-5 h-5 hover:text-yellow-400 cursor-pointer transition-transform hover:scale-110" />
                    </button>
                  </div>
                  <span
                    class="text-sm font-medium pr-1 transition-colors"
                    :class="remainingPostChars <= 0 ? 'text-rose-400 font-bold' : remainingPostChars <= 50 ? 'text-amber-400' : 'text-[color:var(--text-muted)]'"
                  >{{ remainingPostChars }}</span>
                </div>

                <div v-if="showEmojiPicker" ref="emojiPickerPanelRef" @click.stop class="rounded-2xl border border-yellow-400/20 bg-[var(--panel-bg)] p-2">
                  <emoji-picker @emoji-click="handleEmojiPick" locale="zh-Hans" preview-position="none" skin-tone-emoji="👍"></emoji-picker>
                  <div class="mt-2 px-2 text-[11px] text-[color:var(--text-muted)]">点击表情即可插入</div>
                </div>

                <Transition name="expand">
                  <div v-if="showTagPicker" class="rounded-2xl border border-cyan-500/25 bg-cyan-500/5 p-4">
                    <div class="mb-3 text-xs font-semibold uppercase tracking-wider text-[color:var(--text-muted)]">选择标签（点击 #XXX，最多 5 个）</div>
                    <div class="flex flex-wrap gap-2">
                      <button
                        v-for="tag in availablePostTags"
                        :key="tag"
                        @click="togglePostTag(tag)"
                        class="rounded-full border px-3 py-1 text-xs font-semibold transition"
                        :class="selectedPostTags.includes(tag) ? 'border-cyan-400/60 bg-cyan-500/20 text-cyan-300' : 'border-[color:var(--border-color)] bg-[var(--panel-bg)] text-[color:var(--text-secondary)] hover:border-cyan-400/50 hover:text-cyan-300'"
                      >
                        #{{ tag }}
                      </button>
                    </div>
                    <div class="mt-3 flex gap-2">
                      <input
                        v-model="customTagInput"
                        @keydown.enter.prevent="addCustomTag"
                        placeholder="输入自定义标签，例如 #开发日志"
                        class="flex-1 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-3 py-2 text-sm text-[color:var(--text-primary)] outline-none focus:border-cyan-400"
                      />
                      <button
                        @click="addCustomTag"
                        class="rounded-xl bg-cyan-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-cyan-500"
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
                    class="inline-flex items-center gap-1 rounded-full border border-cyan-500/40 bg-cyan-500/10 px-3 py-1 text-xs font-semibold text-cyan-300 transition hover:bg-cyan-500/20"
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

            <div v-if="currentSection === 'home'" class="relative z-[120]">
              <button
                @click="toggleInstanceDropdown"
                class="flex w-full items-center justify-between rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-2.5 text-left transition hover:border-emerald-500/50"
              >
                <span class="min-w-0">
                  <span class="block truncate text-sm font-semibold text-[color:var(--text-primary)]">
                    {{ selectedInstanceName === 'all' ? '全部摩尔实例' : selectedInstanceName }}
                  </span>
                  <span class="block truncate text-xs text-[color:var(--text-muted)]">
                    {{ selectedInstanceName === 'all' ? `${instances.length} 个实例` : selectedInstance?.focus }}
                  </span>
                </span>
                <ChevronDown class="ml-3 h-4 w-4 shrink-0 text-[color:var(--text-muted)]" />
              </button>
              <div
                v-if="showInstanceDropdown"
                class="mt-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--frame-bg,#ffffff)] text-[color:var(--text-primary,#0f172a)] shadow-[0_18px_48px_rgba(0,0,0,0.35)]"
              >
                <button
                  @click="selectInstance('all')"
                  class="w-full px-4 py-3 text-left transition first:rounded-t-xl hover:bg-[var(--panel-soft)]"
                  :class="selectedInstanceName === 'all' ? 'text-emerald-400' : 'text-[color:var(--text-primary)]'"
                >
                  <span class="block text-sm font-semibold">全部摩尔实例</span>
                  <span class="block text-xs text-[color:var(--text-muted)]">显示所有首页动态</span>
                </button>
                <button
                  v-for="instance in instances"
                  :key="instance.name"
                  @click="selectInstance(instance.name)"
                  class="w-full px-4 py-3 text-left transition last:rounded-b-xl hover:bg-[var(--panel-soft)]"
                  :class="selectedInstanceName === instance.name ? 'text-emerald-400' : 'text-[color:var(--text-primary)]'"
                >
                  <span class="block text-sm font-semibold">{{ instance.name }}</span>
                  <span class="block truncate text-xs text-[color:var(--text-muted)]">{{ instance.focus }} · {{ instance.members }} · {{ instance.latency }}</span>
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
          <div class="border-b border-[color:var(--border-color)] px-6 py-6 transition-all duration-300">
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
              当前实例暂无动态。
            </div>
            <article v-for="post in homeTimeline" :key="post.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="flex gap-3">
                <button
                  @click="goToUserProfile(post.authorId)"
                  class="flex h-12 w-12 flex-none items-center justify-center rounded-xl bg-gradient-to-br from-emerald-300 to-cyan-200 text-base font-bold text-slate-900"
                  title="查看用户主页"
                >
                  {{ avatarText(post.author) }}
                </button>
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                    <button
                      @click="goToUserProfile(post.authorId)"
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
                        @click="handleVote(post, [idx])"
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
                      <button @click="refreshPost(post.id)" class="text-emerald-500/70 hover:text-emerald-400 transition-colors">刷新</button>
                    </div>
                  </div>

                  <div v-if="post.media" class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]">
                    <img :src="post.media.preview" :alt="post.media.name" class="max-h-[60vh] w-full object-contain bg-[var(--panel-contrast)]" />
                  </div>

                  <div v-if="post.tags.length" class="mt-4 flex flex-wrap gap-2">
                    <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-emerald-500/10 px-3 py-1 text-sm text-emerald-200">
                      #{{ tag }}
                    </span>
                  </div>

                  <div class="mt-5 flex flex-wrap items-center gap-3 text-sm">
                    <button
                      @click="openPostDetail(post.id)"
                      class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                    >
                      <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.replies || '' }}
                    </button>
                    <button
                      @click="toggleBoost(post.id)"
                      class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                      :class="boostedPosts[post.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                    >
                      <Repeat class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.boosts + (boostedPosts[post.id] ? 1 : 0) || '' }}
                    </button>
                    <button
                      @click="toggleLike(post.id)"
                      class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                      :class="likedPosts[post.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                    >
                      <Heart :class="{'fill-current': likedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.likes + (likedPosts[post.id] ? 1 : 0) || '' }}
                    </button>
                    <button
                      @click="toggleBookmark(post.id)"
                      class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                      :class="bookmarkedPosts[post.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                    >
                      <Bookmark :class="{'fill-current': bookmarkedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" />
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
                          <button @click="handleMenuAction('openOriginal', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">打开原始页面</button>
                          <button @click="handleMenuAction('copyLink', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">复制摩文链接</button>
                          <button @click="handleMenuAction('share', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">分享</button>
                          <button @click="handleMenuAction('embed', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">获取嵌入代码</button>
                        </div>
                        <div class="border-t border-[color:var(--border-color)] py-1">
                          <button @click="handleMenuAction('mention', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)] font-medium">提及 {{ post.handle }}</button>
                        </div>
                        <div class="border-t border-[color:var(--border-color)] py-1 flex flex-col items-start text-rose-500 font-medium">
                          <button @click="handleMenuAction('hide', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">隐藏 {{ post.handle }}</button>
                          <button @click="handleMenuAction('block', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">屏蔽 {{ post.handle }}</button>
                          <button @click="handleMenuAction('report', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">举报 {{ post.handle }}</button>
                        </div>
                        <div class="border-t border-[color:var(--border-color)] py-1">
                          <button @click="handleMenuAction('blockInstance', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 text-rose-500 font-medium">屏蔽 {{ post.instance }} 实例</button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </article>
          </section>

          <section v-else-if="currentSection === 'postDetail'" class="min-h-[calc(100vh-140px)]">
            <div class="border-b border-[color:var(--border-color)] px-6 py-4">
              <button
                @click="setSection('home')"
                class="inline-flex items-center gap-2 rounded-full border border-[color:var(--border-color)] px-4 py-2 text-sm text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
              >
                <span>←</span>
                <span>返回主页</span>
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
                    class="flex h-12 w-12 flex-none items-center justify-center rounded-xl bg-gradient-to-br from-emerald-300 to-cyan-200 text-lg font-bold text-slate-900"
                    title="查看用户主页"
                  >
                    {{ avatarText(threadFocusPost.author) }}
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

                    <div v-if="threadFocusPost.poll" class="mt-4 space-y-2 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-4">
                      <div v-for="(opt, idx) in threadFocusPost.poll.options" :key="idx" class="relative">
                        <div v-if="threadFocusPost.poll.voters.includes(currentUser?.id || '') || new Date(threadFocusPost.poll.expiresAt) < new Date()" class="group overflow-hidden rounded-lg bg-[var(--frame-bg)]">
                          <div 
                            class="absolute inset-y-0 left-0 bg-emerald-500/20 transition-all duration-1000"
                            :style="{ width: `${(opt.votes / Math.max(1, threadFocusPost.poll.options.reduce((a, b) => a + b.votes, 0))) * 100}%` }"
                          ></div>
                          <div class="relative flex items-center justify-between px-4 py-2.5 text-sm">
                            <span class="font-medium text-[color:var(--text-primary)]">{{ opt.label }}</span>
                            <span class="font-bold text-emerald-400">
                              {{ Math.round((opt.votes / Math.max(1, threadFocusPost.poll.options.reduce((a, b) => a + b.votes, 0))) * 100) }}%
                            </span>
                          </div>
                        </div>
                        <button 
                          v-else 
                          @click="handleVote(threadFocusPost, [idx])"
                          class="w-full rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-4 py-2.5 text-left text-sm font-medium text-emerald-400 transition-all hover:bg-emerald-500/10 hover:border-emerald-500/50"
                        >
                          {{ opt.label }}
                        </button>
                      </div>
                      
                      <div class="mt-3 flex items-center justify-between text-[11px] font-bold uppercase tracking-wider text-[color:var(--text-muted)]">
                        <div class="flex items-center gap-3">
                          <span>{{ threadFocusPost.poll.options.reduce((a, b) => a + b.votes, 0) }} 票</span>
                          <span class="opacity-30">·</span>
                          <span>{{ new Date(threadFocusPost.poll.expiresAt) < new Date() ? '已结束' : '进行中' }}</span>
                        </div>
                      </div>
                    </div>

                    <div
                      v-if="threadFocusPost.media"
                      class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]"
                    >
                      <img :src="threadFocusPost.media.preview" :alt="threadFocusPost.media.name" class="max-h-[70vh] w-full object-contain bg-[var(--panel-contrast)]" />
                    </div>

                    <div v-if="threadFocusPost.tags.length" class="mt-4 flex flex-wrap gap-2">
                      <span
                        v-for="tag in threadFocusPost.tags"
                        :key="tag"
                        class="rounded-full bg-emerald-500/10 px-3 py-1 text-sm text-emerald-200"
                      >
                        #{{ tag }}
                      </span>
                    </div>

                    <div class="mt-5 flex flex-wrap items-center gap-3 text-sm">
                      <button
                        class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        <MessageCircle class="w-[18px] h-[18px] mr-1.5" /> {{ threadFocusPost.stats.replies || '' }}
                      </button>
                      <button
                        @click="toggleBoost(threadFocusPost.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="boostedPosts[threadFocusPost.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                      >
                        <Repeat class="w-[18px] h-[18px] mr-1.5" /> {{ threadFocusPost.stats.boosts + (boostedPosts[threadFocusPost.id] ? 1 : 0) || '' }}
                      </button>
                      <button
                        @click="toggleLike(threadFocusPost.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="likedPosts[threadFocusPost.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                      >
                        <Heart :class="{'fill-current': likedPosts[threadFocusPost.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ threadFocusPost.stats.likes + (likedPosts[threadFocusPost.id] ? 1 : 0) || '' }}
                      </button>
                      <button
                        @click="toggleBookmark(threadFocusPost.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="bookmarkedPosts[threadFocusPost.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                      >
                        <Bookmark :class="{'fill-current': bookmarkedPosts[threadFocusPost.id]}" class="w-[18px] h-[18px] mr-1.5" />
                      </button>
                    </div>
                  </div>
                </div>
              </article>

              <div ref="replyComposerRef" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
                <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-5">
                  <div class="flex items-start gap-4">
                    <div class="flex h-12 w-12 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-emerald-300 to-cyan-200 text-base font-bold text-slate-900">
                      {{ avatarText(currentUser?.displayName || 'U') }}
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
                        rows="4"
                        maxlength="500"
                        placeholder="写下你的看法，让讨论继续发生"
                        class="w-full resize-none rounded-2xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-4 py-4 text-base leading-7 text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
                      />
                      <div class="mt-4 flex items-center justify-between gap-4">
                        <div class="text-sm text-[color:var(--text-muted)]">
                          {{ replyDraft.trim().length }}/500
                        </div>
                        <button
                          :disabled="!replyDraft.trim() || saving"
                          @click="submitReply"
                          class="rounded-xl bg-emerald-600 px-5 py-3 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {{ saving ? '发送中...' : '发送回复' }}
                        </button>
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
                    v-for="reply in threadReplies"
                    :key="reply.id"
                    class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-5"
                    :style="{ marginLeft: replyOffset(reply) }"
                  >
                    <div class="flex items-center gap-2 text-sm">
                      <button
                        @click="goToUserProfile(reply.authorId)"
                        class="font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                      >
                        {{ reply.author }}
                      </button>
                      <span class="text-[color:var(--text-secondary)]">@{{ reply.instance }}</span>
                      <span class="rounded-full bg-[var(--chip-bg)] px-3 py-1 text-[color:var(--text-muted)]">
                        第 {{ (reply.replyDepth ?? 0) + 1 }} 层
                      </span>
                      <span class="text-[color:var(--text-muted)]">{{ reply.time }}</span>
                    </div>
                    <div class="mt-3 whitespace-pre-wrap text-base leading-7 text-[color:var(--text-secondary)]">
                      {{ reply.content }}
                    </div>
                    <div v-if="reply.tags.length" class="mt-4 flex flex-wrap gap-2">
                      <span v-for="tag in reply.tags" :key="tag" class="rounded-full bg-emerald-500/10 px-3 py-1 text-xs text-emerald-200">
                        #{{ tag }}
                      </span>
                    </div>
                    <div class="mt-4 flex flex-wrap items-center gap-3 text-sm">
                      <button
                        @click="setReplyTarget(reply)"
                        class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        <MessageCircle class="w-[18px] h-[18px] mr-1.5" />
                      </button>
                      <button
                        @click="openPostDetail(reply.id)"
                        class="inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        <List class="w-[18px] h-[18px] mr-1.5" />
                      </button>
                      <button
                        @click="toggleLike(reply.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="likedPosts[reply.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                      >
                        <Heart :class="{'fill-current': likedPosts[reply.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ reply.stats.likes + (likedPosts[reply.id] ? 1 : 0) || '' }}
                      </button>
                      <button
                        @click="toggleBookmark(reply.id)"
                        class="inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm"
                        :class="bookmarkedPosts[reply.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                      >
                        <Bookmark :class="{'fill-current': bookmarkedPosts[reply.id]}" class="w-[18px] h-[18px] mr-1.5" />
                      </button>
                    </div>
                  </article>
                </div>
              </div>
            </div>

            <div v-else class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              没有找到对应的帖子内容。
            </div>
          </section>

          <section v-else-if="currentSection === 'myFeed'" class="divide-y divide-[color:var(--border-color)]">
            <article v-if="myTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              你发布的摩文会显示在这里。
            </article>
            <article v-for="post in myTimeline" v-else :key="post.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="flex gap-3">
                <button
                  @click="goToUserProfile(post.authorId)"
                  class="flex h-12 w-12 flex-none items-center justify-center rounded-xl bg-gradient-to-br from-emerald-300 to-cyan-200 text-base font-bold text-slate-900"
                  title="查看用户主页"
                >
                  {{ avatarText(post.author) }}
                </button>
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                    <button
                      @click="goToUserProfile(post.authorId)"
                      class="text-lg font-semibold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                    >
                      {{ post.author }}
                    </button>
                    <span class="text-sm text-[color:var(--text-secondary)]">@{{ post.instance }}</span>
                    <span class="text-xs text-[color:var(--text-muted)]">{{ post.time }}</span>
                  </div>
                  <div v-if="post.bio" class="mt-0.5 text-xs text-[color:var(--text-muted)]">{{ post.bio }}</div>
                  <div class="mt-3 whitespace-pre-wrap text-[15px] leading-7 text-[color:var(--text-soft)]">{{ post.content }}</div>
                  <div v-if="post.media" class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]">
                    <img :src="post.media.preview" :alt="post.media.name" class="max-h-[60vh] w-full object-contain bg-[var(--panel-contrast)]" />
                  </div>
                  <div v-if="post.tags.length" class="mt-4 flex flex-wrap gap-2">
                    <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-emerald-500/10 px-3 py-1 text-sm text-emerald-200">
                      #{{ tag }}
                    </span>
                  </div>
                </div>
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
                <article v-for="post in timeline" :key="post.id" class="p-6 transition hover:bg-[var(--panel-soft)]">
                  <!-- Reuse Post View Here - Similar to Home Feed -->
                  <div class="flex gap-4">
                    <button
                      @click="goToUserProfile(post.authorId)"
                      class="h-12 w-12 flex-none rounded-2xl bg-gradient-to-br from-indigo-200 to-emerald-200 flex items-center justify-center font-bold text-slate-800"
                      title="查看用户主页"
                    >
                      {{ avatarText(post.author) }}
                    </button>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between gap-2">
                        <div class="flex items-center gap-2 truncate">
                          <button
                            @click="goToUserProfile(post.authorId)"
                            class="font-bold text-[color:var(--text-primary)] transition hover:text-emerald-500"
                          >
                            {{ post.author }}
                          </button>
                          <span class="text-sm text-[color:var(--text-muted)] truncate">@{{ post.instance }}</span>
                        </div>
                        <span class="text-sm text-[color:var(--text-muted)]">{{ post.time }}</span>
                      </div>
                      <div class="mt-2 text-[17px] leading-relaxed text-[color:var(--text-primary)] whitespace-pre-wrap">{{ post.content }}</div>

                      <!-- Explore Poll Display -->
                      <div v-if="post.poll" class="mt-4 space-y-3 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-3">
                        <div v-for="(opt, idx) in post.poll.options" :key="idx" class="relative">
                          <div v-if="post.poll.voters.includes(currentUser?.id || '') || new Date(post.poll.expiresAt) < new Date()" class="group overflow-hidden rounded-lg bg-[var(--frame-bg)]">
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
                      
                      <!-- Post Media (Explore Tab) -->
                      <div v-if="post.media" class="mt-4 overflow-hidden rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]">
                        <img :src="post.media.preview" :alt="post.media.name" class="max-h-[70vh] w-full object-contain bg-[var(--panel-contrast)]" />
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
                          @click="toggleBoost(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-emerald-500/10 hover:text-emerald-400"
                          :class="boostedPosts[post.id] ? 'text-emerald-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Repeat class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.boosts + (boostedPosts[post.id] ? 1 : 0) || '' }}
                        </button>
                        <button
                          @click="toggleLike(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-rose-500/10 hover:text-rose-400"
                          :class="likedPosts[post.id] ? 'text-rose-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Heart :class="{'fill-current': likedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.likes + (likedPosts[post.id] ? 1 : 0) || '' }}
                        </button>
                        <button
                          @click="toggleBookmark(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-indigo-500/10 hover:text-indigo-400"
                          :class="bookmarkedPosts[post.id] ? 'text-indigo-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Bookmark :class="{'fill-current': bookmarkedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" />
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
                              <button @click="handleMenuAction('openOriginal', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">打开原始页面</button>
                              <button @click="handleMenuAction('copyLink', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">复制摩文链接</button>
                              <button @click="handleMenuAction('share', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">分享</button>
                              <button @click="handleMenuAction('embed', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">获取嵌入代码</button>
                            </div>
                            <div class="border-t border-[color:var(--border-color)] py-1">
                              <button @click="handleMenuAction('mention', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)] font-medium">提及 {{ post.handle }}</button>
                            </div>
                            <div class="border-t border-[color:var(--border-color)] py-1 flex flex-col items-start text-rose-500 font-medium">
                              <button @click="handleMenuAction('hide', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">隐藏 {{ post.handle }}</button>
                              <button @click="handleMenuAction('block', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">屏蔽 {{ post.handle }}</button>
                              <button @click="handleMenuAction('report', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">举报 {{ post.handle }}</button>
                            </div>
                            <div class="border-t border-[color:var(--border-color)] py-1">
                              <button @click="handleMenuAction('blockInstance', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 text-rose-500 font-medium">屏蔽 {{ post.instance }} 实例</button>
                            </div>
                          </div>
                        </div>

                      </div>
                    </div>
                  </div>
                </article>
              </template>

              <!-- Topics Tab -->
              <template v-else-if="activeExploreTab === 'topics'">
                <div v-for="tag in trendingTags" :key="tag.tag" class="p-6 transition hover:bg-[var(--panel-soft)] flex items-center justify-between cursor-pointer">
                  <div>
                    <div class="font-bold text-lg text-[color:var(--text-primary)]">#{{ tag.tag }}</div>
                    <div class="text-sm text-[color:var(--text-muted)] mt-1">热门话题</div>
                  </div>
                  <div class="rounded-full bg-emerald-500/10 px-4 py-1.5 text-sm font-semibold text-emerald-600">
                    {{ tag.count }} 摩文
                  </div>
                </div>
              </template>

              <!-- Users Tab -->
              <template v-else-if="activeExploreTab === 'users'">
                <article v-for="person in recommendedPeople" :key="person.id" class="p-6 transition hover:bg-[var(--panel-soft)]">
                  <div class="flex items-start justify-between gap-4">
                    <div class="flex min-w-0 gap-4">
                      <div class="h-14 w-14 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-amber-200 to-emerald-200 text-lg font-bold text-slate-900">
                        {{ avatarText(person.displayName) }}
                      </div>
                      <div class="min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                          <span class="text-lg font-semibold text-[color:var(--text-primary)]">{{ person.displayName }}</span>
                          <span class="truncate text-base text-[color:var(--text-muted)]">{{ person.handle }}@{{ person.instance }}</span>
                        </div>
                        <div class="mt-1 text-sm text-[color:var(--text-muted)]">{{ person.followers }} 关注者</div>
                        <div class="mt-2 line-clamp-2 text-base leading-relaxed text-[color:var(--text-secondary)]">{{ person.bio }}</div>
                      </div>
                    </div>
                    <div class="flex shrink-0 items-center gap-2">
                      <button
                        @click="goToUserProfile(person.id)"
                        class="rounded-xl border border-[color:var(--border-color)] px-4 py-2 text-sm font-semibold text-[color:var(--text-secondary)] transition hover:border-cyan-500/40 hover:bg-cyan-500/10 hover:text-cyan-300"
                      >
                        查看主页
                      </button>
                      <button
                        @click="startConversation(person)"
                        class="inline-flex items-center gap-2 rounded-xl border border-emerald-500/30 bg-emerald-500/8 px-4 py-2 text-sm font-semibold text-emerald-300 transition hover:border-emerald-400/50 hover:bg-emerald-500/12 hover:text-emerald-200"
                      >
                        <MessageCircle class="h-4 w-4" />
                        <span>{{ isCrossInstanceUser(person) ? '跨联邦发消息' : '发消息' }}</span>
                      </button>
                      <button
                        @click="toggleFollow(person.id)"
                        class="rounded-xl bg-emerald-600 px-5 py-2 text-sm font-bold text-white transition hover:bg-emerald-500"
                      >
                        {{ followedUsers[person.id] ? '已关注' : '关注' }}
                      </button>
                    </div>
                  </div>
                </article>
              </template>

              <!-- News Tab -->
              <template v-else-if="activeExploreTab === 'news'">
                <div v-if="newsPosts.length === 0" class="p-12 text-center text-[color:var(--text-muted)]">
                  目前没有最新的新闻摩文。
                </div>
                <article v-for="post in newsPosts" :key="post.id" class="p-6 transition hover:bg-[var(--panel-soft)]">
                   <div class="flex gap-4">
                    <div class="h-12 w-12 flex-none rounded-2xl bg-emerald-600 flex items-center justify-center text-white">
                      <Newspaper class="w-6 h-6" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between mb-1">
                        <span class="font-bold text-emerald-500">官方新闻</span>
                        <span class="text-sm text-[color:var(--text-muted)]">{{ post.time }}</span>
                      </div>
                      <div class="text-[17px] leading-relaxed text-[color:var(--text-primary)] font-medium whitespace-pre-wrap">{{ post.content }}</div>

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
                      <div v-if="post.media" class="mt-4 overflow-hidden rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]">
                        <img :src="post.media.preview" :alt="post.media.name" class="max-h-[70vh] w-full object-contain bg-[var(--panel-contrast)]" />
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
                          @click="toggleBoost(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-emerald-500/10 hover:text-emerald-400"
                          :class="boostedPosts[post.id] ? 'text-emerald-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Repeat class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.boosts + (boostedPosts[post.id] ? 1 : 0) || '' }}
                        </button>
                        <button
                          @click="toggleLike(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-rose-500/10 hover:text-rose-400"
                          :class="likedPosts[post.id] ? 'text-rose-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Heart :class="{'fill-current': likedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" /> {{ post.stats.likes + (likedPosts[post.id] ? 1 : 0) || '' }}
                        </button>
                        <button
                          @click="toggleBookmark(post.id)"
                          class="inline-flex items-center rounded-lg border border-transparent px-2 py-1.5 font-medium transition-all hover:bg-indigo-500/10 hover:text-indigo-400"
                          :class="bookmarkedPosts[post.id] ? 'text-indigo-400' : 'text-[color:var(--text-secondary)]'"
                        >
                          <Bookmark :class="{'fill-current': bookmarkedPosts[post.id]}" class="w-[18px] h-[18px] mr-1.5" />
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
                              <button @click="handleMenuAction('openOriginal', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">打开原始页面</button>
                              <button @click="handleMenuAction('copyLink', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">复制摩文链接</button>
                              <button @click="handleMenuAction('share', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">分享</button>
                              <button @click="handleMenuAction('embed', post)" class="w-full text-left px-4 py-2.5 hover:bg-[var(--panel-soft)] text-[color:var(--text-primary)]">获取嵌入代码</button>
                            </div>
                            <div class="border-t border-[color:var(--border-color)] py-1 flex flex-col items-start text-rose-500 font-medium">
                              <button @click="handleMenuAction('hide', post)" class="w-full text-left px-4 py-2.5 hover:bg-rose-500/10 hover:text-rose-400">隐藏此新闻</button>
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
                    <div class="rounded-full border border-emerald-500/20 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300">
                      {{ conversations.length }} 条消息
                    </div>
                  </div>
                </div>

                <div v-if="conversations.length === 0" class="px-5 py-10 text-sm text-[color:var(--text-muted)]">
                  还没有私信消息，去“当前热门 → 用户”里点击“发消息”开始第一段聊天。
                </div>

                <div v-else class="min-h-0 flex-1 overflow-y-auto divide-y divide-[color:var(--border-color)]">
                  <button
                    v-for="conversation in conversations"
                    :key="conversation.id"
                    @click="openConversation(conversation.id)"
                    class="flex w-full items-start gap-3 px-5 py-4 text-left transition hover:bg-[var(--chip-hover)]"
                    :class="selectedConversationId === conversation.id ? 'bg-emerald-500/10' : ''"
                  >
                    <div class="flex h-12 w-12 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-amber-200 to-emerald-200 text-base font-bold text-slate-900">
                      {{ conversation.avatarLabel }}
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between gap-3">
                        <div class="truncate text-[15px] font-semibold text-[color:var(--text-primary)]">{{ conversation.name }}</div>
                        <div class="shrink-0 text-xs text-[color:var(--text-muted)]">{{ conversation.messages[conversation.messages.length - 1]?.time || '' }}</div>
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
                  <div class="flex shrink-0 items-center gap-4 border-b border-[color:var(--border-color)] px-6 py-5">
                    <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-amber-200 to-emerald-200 text-base font-bold text-slate-900">
                      {{ activeConversation.avatarLabel }}
                    </div>
                    <div class="min-w-0">
                      <div class="truncate text-lg font-semibold text-[color:var(--text-primary)]">{{ activeConversation.name }}</div>
                      <div class="mt-1 truncate text-sm text-[color:var(--text-secondary)]">{{ activeConversation.handle }}</div>
                      <div v-if="activeConversation.crossInstance" class="mt-1 inline-flex max-w-full items-center gap-2 rounded-full border border-emerald-500/25 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300">
                        <Globe class="h-3.5 w-3.5 shrink-0" />
                        <span class="truncate">{{ activeConversation.federationRoute }}</span>
                      </div>
                    </div>
                  </div>

                  <div ref="messageListRef" class="min-h-0 flex-1 overflow-y-auto px-6 py-6">
                    <div class="space-y-4">
                      <div
                        v-for="message in activeConversation.messages"
                        :key="message.id"
                        class="flex items-end gap-3"
                        :class="message.from === 'me' ? 'justify-end' : 'justify-start'"
                      >
                        <template v-if="message.from === 'peer'">
                          <div class="flex h-10 w-10 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-amber-200 to-emerald-200 text-sm font-bold text-slate-900">
                            {{ activeConversation.avatarLabel }}
                          </div>
                          <div class="max-w-[75%] rounded-[22px] rounded-bl-md border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-sm leading-6 text-[color:var(--text-primary)] shadow-sm">
                            <div>{{ message.text }}</div>
                            <div class="mt-2 text-[11px] text-[color:var(--text-muted)]">{{ message.time }}</div>
                          </div>
                        </template>

                        <template v-else>
                          <div class="max-w-[75%] rounded-[22px] rounded-br-md bg-emerald-600 px-4 py-3 text-sm leading-6 text-white shadow-[0_10px_30px_rgba(16,185,129,0.22)]">
                            <div>{{ message.text }}</div>
                            <div class="mt-2 text-[11px] text-emerald-100/80">{{ message.time }}</div>
                          </div>
                          <div class="flex h-10 w-10 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-lime-200 to-cyan-200 text-sm font-bold text-slate-900">
                            {{ avatarText(currentUser?.displayName || 'U') }}
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
                      <div class="flex h-11 w-11 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-lime-200 to-cyan-200 text-sm font-bold text-slate-900">
                        {{ avatarText(currentUser?.displayName || 'U') }}
                      </div>
                      <div class="min-w-0 flex-1 rounded-3xl border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-4 py-3">
                        <textarea
                          v-model="messageDraft"
                          @keydown.enter.prevent="sendMessage"
                          rows="3"
                          maxlength="1000"
                          placeholder="输入消息..."
                          class="w-full resize-none bg-transparent text-sm leading-6 text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
                        />
                        <div class="mt-3 flex items-center justify-between gap-3">
                          <div class="text-xs text-[color:var(--text-muted)]">{{ messageDraft.trim().length }}/1000</div>
                          <button
                            :disabled="!messageDraft.trim() || saving"
                            @click="sendMessage"
                            class="rounded-xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            {{ saving ? '发送中...' : '发送消息' }}
                          </button>
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
            <article v-for="item in notificationItems" :key="item.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
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
            <article v-for="item in curatedLists" :key="item.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-6">
                <div class="flex items-center justify-between gap-3">
                  <div class="text-xl font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                  <div class="text-sm text-emerald-600">{{ item.count }}</div>
                </div>
                <div class="mt-3 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.summary }}</div>
              </div>
            </article>
          </section>

          <section v-else-if="currentSection === 'topics'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in followedTopicCards" :key="item.tag" class="px-8 py-8 transition hover:bg-[var(--panel-soft)]">
              <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-6">
                <div class="flex items-center justify-between gap-3">
                  <div class="text-xl font-semibold text-[color:var(--text-primary)]">#{{ item.tag }}</div>
                  <div class="text-sm text-[color:var(--text-muted)]">{{ item.count }} 条动态</div>
                </div>
                <div class="mt-3 line-clamp-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.summary }}</div>
              </div>
            </article>
          </section>

          <section v-else-if="currentSection === 'likes'" class="divide-y divide-[color:var(--border-color)]">
            <article v-if="likedTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              你点赞的内容会显示在这里。
            </article>
            <article v-for="post in likedTimeline" v-else :key="post.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ post.author }}</div>
              <div class="mt-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ post.content }}</div>
            </article>
          </section>

          <section v-else-if="currentSection === 'bookmarks'" class="divide-y divide-[color:var(--border-color)]">
            <article v-if="bookmarkedTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              收藏的动态会整理在这里，方便稍后继续阅读。
            </article>
            <article v-for="post in bookmarkedTimeline" v-else :key="post.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
              <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ post.author }}</div>
              <div class="mt-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ post.content }}</div>
            </article>
          </section>

          <section v-else-if="currentSection === 'mentions'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in mentionItems" :key="item.id" class="px-5 py-5 transition hover:bg-[var(--panel-soft)]">
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
          </section>

          <!-- preferences section removed - moved to SettingsPage.vue -->

          <section v-else class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in moreCards" :key="item.id" class="px-8 py-8 transition hover:bg-[var(--panel-soft)]">
              <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-6">
                <div class="text-xl font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                <div class="mt-3 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.description }}</div>
              </div>
            </article>
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
                    :class="tempInteraction === option.id ? 'border-cyan-500/60 bg-cyan-500/10 text-cyan-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-cyan-500/40'"
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
</style>
