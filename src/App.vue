<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import {
  createConversationMessage,
  createMediaAsset,
  createPost,
  fetchPostReplies,
  fetchPostThread,
  fetchSocialBootstrap,
  type BootstrapPayload,
  type FederationInstance,
  type MediaAsset,
  type SocialConversation,
  type SocialPost,
  type SocialUser,
} from './api/socialApi';
import { useAppearance, type AppearanceSettings, type ColorScheme } from './composables/useAppearance';

type Section =
  | 'home'
  | 'postDetail'
  | 'explore'
  | 'notifications'
  | 'lists'
  | 'topics'
  | 'likes'
  | 'bookmarks'
  | 'mentions'
  | 'preferences'
  | 'more';
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
  };
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
};

type ConversationCard = {
  id: string;
  name: string;
  handle: string;
  status: string;
  messages: MessageCard[];
};

const primaryNavItems: { label: string; key: Section; icon: string }[] = [
  { label: '主页', key: 'home', icon: '⌂' },
  { label: '当前热门', key: 'explore', icon: '↗' },
  { label: '通知', key: 'notifications', icon: '◌' },
];

const secondaryNavItems: { label: string; key: Section; icon: string }[] = [
  { label: '列表', key: 'lists', icon: '▦' },
  { label: '关注的话题', key: 'topics', icon: '#' },
  { label: '喜欢', key: 'likes', icon: '☆' },
  { label: '书签', key: 'bookmarks', icon: '▯' },
  { label: '私下提及', key: 'mentions', icon: '@' },
];

const utilityNavItems: { label: string; key: Section; icon: string }[] = [
  { label: '偏好设置', key: 'preferences', icon: '⚙' },
  { label: '更多', key: 'more', icon: '⋯' },
];

const settingsMenu: { id: SettingsTab; label: string; icon: string }[] = [
  { id: 'profile', label: '个人资料', icon: '◌' },
  { id: 'privacy', label: '隐私与可达性', icon: '◍' },
  { id: 'preferences', label: '偏好设置', icon: '⚙' },
  { id: 'appearance', label: '外观', icon: '☼' },
  { id: 'posting', label: '发布默认值', icon: '✎' },
  { id: 'notifications', label: '邮件通知', icon: '✉' },
  { id: 'other', label: '其他', icon: '☰' },
  { id: 'follows', label: '关注管理', icon: '☍' },
  { id: 'filters', label: '过滤规则', icon: '⏃' },
  { id: 'cleanup', label: '自动删除嘟文', icon: '↺' },
];

const fallbackUser: SocialUser = {
  id: 'user_local',
  handle: '@vico',
  displayName: 'Vico',
  bio: '在离线模式下继续浏览内容与草稿。',
  instance: 'vault.social',
  wallet: '0xlocal',
  avatarUrl: '',
  followers: 238,
  following: 121,
  createdAt: new Date().toISOString(),
};

const fallbackPeople: SocialUser[] = [
  fallbackUser,
  {
    id: 'user_archive',
    handle: '@archive',
    displayName: 'Whale Archive',
    bio: '为创作者提供永久内容归档与链上身份锚定。',
    instance: 'vault.social',
    wallet: '0xa18f',
    avatarUrl: '',
    followers: 1284,
    following: 312,
    createdAt: new Date().toISOString(),
  },
  {
    id: 'user_librarian',
    handle: '@librarian',
    displayName: 'Node Librarian',
    bio: '把书籍确权、媒体存储和社交关系连接起来。',
    instance: 'readers.polkadot',
    wallet: '0x78fe',
    avatarUrl: '',
    followers: 932,
    following: 221,
    createdAt: new Date().toISOString(),
  },
];

const fallbackInstances: FederationInstance[] = [
  { name: 'vault.social', focus: '创作者主权与链上身份', members: '12.4k', latency: '43 ms', status: 'healthy' },
  { name: 'readers.polkadot', focus: '阅读社群与数字馆藏', members: '8.9k', latency: '51 ms', status: 'healthy' },
  { name: 'relay.zone', focus: '跨实例消息转发', members: '3.1k', latency: '37 ms', status: 'healthy' },
];

const fallbackPosts: FeedCard[] = [
  {
    id: 'local-post-1',
    author: 'Whale Archive',
    handle: '@archive',
    instance: 'vault.social',
    kind: 'post',
    time: '刚刚',
    content: '欢迎来到 Whale Vault Social。即使后端暂时不可用，这里也会保留一个完整的社交界面，而不是只显示技术说明。',
    bio: '离线回退内容',
    tags: ['离线模式', '社交界面'],
    chainProof: 'local://fallback/post-1',
    stats: { replies: 3, boosts: 8, likes: 23 },
  },
];

const fallbackAssets: AssetCard[] = [
  {
    id: 'local-asset-1',
    title: 'fallback-manifesto.png',
    network: 'Arweave 镜像队列',
    cid: 'local-preview',
    size: '0.38 MB',
    retention: '本地预览',
    url: '',
  },
];

const fallbackConversations: ConversationCard[] = [
  {
    id: 'local-conversation',
    name: 'Archive Curator',
    handle: '@curator@vault.social',
    status: '回退模式',
    messages: [
      {
        id: 'local-message',
        from: 'peer',
        text: '后端恢复后，这里会自动切回真实会话数据。',
        time: '刚刚',
      },
    ],
  },
];

const currentSection = ref<Section>('home');
const currentUser = ref<SocialUser | null>(null);
const people = ref<SocialUser[]>([]);
const posts = ref<FeedCard[]>([]);
const assets = ref<AssetCard[]>([]);
const conversations = ref<ConversationCard[]>([]);
const instances = ref<FederationInstance[]>([]);
const postDraft = ref('');
const messageDraft = ref('');
const searchQuery = ref('');
const selectedConversationId = ref('');
const mediaPreview = ref<string | null>(null);
const mediaMeta = ref<{ name: string; sizeLabel: string; type: string; sizeBytes: number } | null>(null);
const replyDraft = ref('');
const loading = ref(true);
const saving = ref(false);
const apiOnline = ref(false);
const errorMessage = ref('');
const followedUsers = ref<Record<string, boolean>>({});
const likedPosts = ref<Record<string, boolean>>({});
const boostedPosts = ref<Record<string, boolean>>({});
const bookmarkedPosts = ref<Record<string, boolean>>({});
const currentSettingsTab = ref<SettingsTab>('appearance');
const settingsNotice = ref('');
const selectedPostId = ref('');
const threadLoading = ref(false);
const threadError = ref('');
const threadFocusPost = ref<FeedCard | null>(null);
const threadAncestors = ref<FeedCard[]>([]);
const threadReplies = ref<FeedCard[]>([]);
const activeReplyTarget = ref<FeedCard | null>(null);
const replyComposerRef = ref<HTMLDivElement | null>(null);
const replyTextareaRef = ref<HTMLTextAreaElement | null>(null);

const { appearanceSettings, resolvedTheme, saveAppearanceSettings } = useAppearance();
const appearanceDraft = ref<AppearanceSettings>({ ...appearanceSettings.value });

const activeConversation = computed(() =>
  conversations.value.find((conversation) => conversation.id === selectedConversationId.value),
);

const mediaCount = computed(() => posts.value.filter((post) => post.media).length + assets.value.length);

const timeline = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return posts.value;
  return posts.value.filter((post) =>
    [post.author, post.handle, post.content, ...post.tags].join(' ').toLowerCase().includes(query),
  );
});

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

const serviceNotice = computed(() =>
  errorMessage.value ? '当前展示的是本地预览内容，社区恢复后会自动同步。' : '跨实例动态正在持续刷新。',
);

const currentSectionTitle = computed(() => {
  if (currentSection.value === 'postDetail') {
    return '帖子详情';
  }
  switch (currentSection.value) {
    case 'home':
      return '主页';
    case 'explore':
      return '当前热门';
    case 'notifications':
      return '通知';
    case 'lists':
      return '列表';
    case 'topics':
      return '关注的话题';
    case 'likes':
      return '喜欢';
    case 'bookmarks':
      return '书签';
    case 'mentions':
      return '私下提及';
    case 'preferences':
      return '偏好设置';
    default:
      return '更多';
  }
});

const likedTimeline = computed(() => posts.value.filter((post) => likedPosts.value[post.id]));

const bookmarkedTimeline = computed(() => posts.value.filter((post) => bookmarkedPosts.value[post.id]));

const notificationItems = computed(() => {
  const suggestedUsers = recommendedPeople.value.slice(0, 2).map((person) => ({
    id: `follow-${person.id}`,
    title: `${person.displayName} 开始在社区活跃`,
    body: `${person.handle}@${person.instance} 发布了新的动态，适合加入你的关注流。`,
    time: '刚刚',
  }));

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
    count: `${recommendedPeople.value.length || 3} 位成员`,
  },
  {
    id: 'list-readers',
    title: '阅读与知识节点',
    summary: '聚合阅读社群、图书馆节点和内容策展者。',
    count: `${instances.value.length || 3} 个实例`,
  },
  {
    id: 'list-archives',
    title: '永久存储观察',
    summary: '跟踪媒体上链、归档状态和内容留存趋势。',
    count: `${assets.value.length || 1} 个资源`,
  },
]);

const followedTopicCards = computed(() =>
  trendingTags.value.map((item) => ({
    ...item,
    summary: timeline.value.find((post) => post.tags.includes(item.tag))?.content ?? '正在汇聚新的讨论内容。',
  })),
);

const mentionItems = computed(() => {
  const handle = currentUser.value?.handle ?? '@vico';
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

const themeStyles = computed<Record<string, string>>(() => {
  if (resolvedTheme.value === 'dark') {
    return {
      '--app-bg': '#0f1117',
      '--frame-bg': '#171a22',
      '--panel-bg': '#131720',
      '--panel-soft': '#1b2130',
      '--panel-muted': '#202738',
      '--panel-contrast': '#0f1320',
      '--border-color': 'rgba(148, 163, 184, 0.18)',
      '--text-primary': '#f8fafc',
      '--text-secondary': '#cbd5e1',
      '--text-muted': '#94a3b8',
      '--text-soft': '#e2e8f0',
      '--chip-bg': 'rgba(255,255,255,0.06)',
      '--chip-hover': 'rgba(255,255,255,0.1)',
    };
  }

  return {
    '--app-bg': '#f4f7fb',
    '--frame-bg': '#ffffff',
    '--panel-bg': '#f8fafc',
    '--panel-soft': '#ffffff',
    '--panel-muted': '#eef2f7',
    '--panel-contrast': '#edf2f9',
    '--border-color': 'rgba(15, 23, 42, 0.08)',
    '--text-primary': '#0f172a',
    '--text-secondary': '#334155',
    '--text-muted': '#64748b',
    '--text-soft': '#1e293b',
    '--chip-bg': 'rgba(148, 163, 184, 0.12)',
    '--chip-hover': 'rgba(148, 163, 184, 0.2)',
  };
});

const hasAppearanceChanges = computed(
  () => JSON.stringify(appearanceDraft.value) !== JSON.stringify(appearanceSettings.value),
);

watch(
  appearanceSettings,
  (value) => {
    appearanceDraft.value = { ...value };
  },
  { deep: true },
);

watch(
  appearanceDraft,
  () => {
    settingsNotice.value = '';
  },
  { deep: true },
);

function setSection(section: Section) {
  currentSection.value = section;
  if (section === 'preferences') {
    currentSettingsTab.value = 'appearance';
  }
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

function shortProof(proof: string) {
  if (!proof) return '链上凭证待同步';
  return proof.length > 36 ? `${proof.slice(0, 16)}...${proof.slice(-10)}` : proof;
}

function toneClass(option: ColorScheme) {
  const current = appearanceDraft.value.colorScheme;
  if (current === 'light' && option !== 'light') {
    return 'text-white';
  }
  return 'text-[color:var(--text-primary)]';
}

function openSettings(tab: SettingsTab = 'appearance') {
  currentSection.value = 'preferences';
  currentSettingsTab.value = tab;
}

function saveAppearancePreferences() {
  saveAppearanceSettings({ ...appearanceDraft.value });
  settingsNotice.value = '偏好设置已保存';
}

function formatTimestamp(input: string) {
  if (!input) return '刚刚';
  const date = new Date(input);
  if (Number.isNaN(date.getTime())) return input;
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
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
  return `${user.handle}@${user.instance}`;
}

function toFeedCard(post: SocialPost): FeedCard {
  const firstMedia = Array.isArray(post.media) ? post.media[0] : undefined;
  const person = people.value.find((item) => item.id === post.authorId);
  return {
    id: post.id,
    author: post.authorName,
    handle: post.authorHandle,
    instance: post.instance,
    kind: post.kind || (post.parentPostId ? 'reply' : 'post'),
    parentPostId: post.parentPostId,
    rootPostId: post.rootPostId,
    replyDepth: post.replyDepth ?? 0,
    time: formatTimestamp(post.createdAt),
    content: post.content,
    bio: person?.bio,
    tags: post.tags,
    chainProof: post.attestationUri || post.storageUri || 'unverified://pending',
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
  return {
    id: conversation.id,
    name: conversation.title,
    handle: conversation.participantIds.join(', '),
    status: conversation.encrypted ? '端到端加密会话' : '标准会话',
    messages: conversation.messages.map((message) => ({
      id: message.id,
      from: message.senderId === userId ? 'me' : 'peer',
      text: message.body,
      time: formatTimestamp(message.createdAt),
    })),
  };
}

function applyBootstrap(payload: BootstrapPayload) {
  currentUser.value = payload.currentUser ?? payload.users[0] ?? fallbackUser;
  people.value = payload.users.length ? payload.users : fallbackPeople;
  posts.value = payload.feed.map(toFeedCard);
  assets.value = payload.media.map(toAssetCard);
  conversations.value = payload.conversations.map((conversation) =>
    toConversationCard(conversation, currentUser.value?.id ?? null),
  );
  instances.value = payload.instances.length ? payload.instances : fallbackInstances;
  selectedConversationId.value = conversations.value[0]?.id ?? '';
}

function applyFallback(message: string) {
  apiOnline.value = false;
  errorMessage.value = message;
  currentUser.value = fallbackUser;
  people.value = fallbackPeople;
  posts.value = fallbackPosts;
  assets.value = fallbackAssets;
  conversations.value = fallbackConversations;
  instances.value = fallbackInstances;
  selectedConversationId.value = fallbackConversations[0].id;
}

function buildLocalThread(postId: string) {
  const focusPost = posts.value.find((post) => post.id === postId) ?? null;
  threadFocusPost.value = focusPost;
  threadAncestors.value = [];
  threadReplies.value = posts.value.filter((post) => post.parentPostId === postId);
  activeReplyTarget.value = focusPost;
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
  replyComposerRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  replyTextareaRef.value?.focus({ preventScroll: true });
}

async function loadBootstrap() {
  loading.value = true;
  try {
    const payload = await fetchSocialBootstrap();
    applyBootstrap(payload);
    apiOnline.value = true;
    errorMessage.value = '';
  } catch (error) {
    const message = error instanceof Error ? error.message : '暂时无法连接社区服务';
    applyFallback(message);
  } finally {
    loading.value = false;
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
    buildLocalThread(postId);
    threadLoading.value = false;
    if (focusComposer) {
      await focusReplyComposer();
    }
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
    const message = error instanceof Error ? error.message : '暂时无法加载讨论串';
    threadError.value = message;
    buildLocalThread(postId);
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
    if (apiOnline.value) {
      await createPost({
        authorId: currentUser.value.id,
        kind: 'reply',
        content: replyDraft.value.trim(),
        visibility: 'public',
        storageUri: `draft://reply/${Date.now()}`,
        attestationUri: `attestation://reply/${Date.now()}`,
        tags: targetPost.tags.slice(0, 3),
        mediaIds: [],
        parentPostId: targetPost.id,
        rootPostId,
      });
      bumpReplyCount(targetPost.id);
      await openPostDetail(selectedPostId.value || rootPostId);
    } else {
      const localId = `local-reply-${Date.now()}`;
      const replyCard: FeedCard = {
        id: localId,
        author: currentUser.value.displayName,
        handle: currentUser.value.handle,
        instance: currentUser.value.instance,
        kind: 'reply',
        parentPostId: targetPost.id,
        rootPostId,
        replyDepth: (targetPost.replyDepth ?? 0) + 1,
        time: '刚刚',
        content: replyDraft.value.trim(),
        tags: targetPost.tags.slice(0, 2),
        chainProof: `local://${localId}`,
        stats: { replies: 0, boosts: 0, likes: 0 },
      };
      threadReplies.value = [...threadReplies.value, replyCard];
      bumpReplyCount(targetPost.id);
    }

    replyDraft.value = '';
    activeReplyTarget.value = threadFocusPost.value;
    errorMessage.value = '';
    threadError.value = '';
    await focusReplyComposer();
  } catch (error) {
    threadError.value = error instanceof Error ? error.message : '暂时无法发送回复';
  } finally {
    saving.value = false;
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

function toggleFollow(userId: string) {
  followedUsers.value = {
    ...followedUsers.value,
    [userId]: !followedUsers.value[userId],
  };
}

async function publishPost() {
  if ((!postDraft.value.trim() && !mediaPreview.value) || !currentUser.value || saving.value) return;

  saving.value = true;
  try {
    let createdAsset: MediaAsset | null = null;

    if (apiOnline.value && mediaPreview.value && mediaMeta.value) {
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

    if (apiOnline.value) {
      const createdPost = await createPost({
        authorId: currentUser.value.id,
        content: postDraft.value.trim() || '分享了一条新的媒体动态。',
        visibility: 'public',
        storageUri: createdAsset?.storageUri || `draft://post/${Date.now()}`,
        attestationUri: `attestation://frontend/${Date.now()}`,
        tags: ['创作者动态', '联邦社交'],
        mediaIds: createdAsset ? [createdAsset.id] : [],
      });
      posts.value = [toFeedCard(createdPost), ...posts.value];
      errorMessage.value = '';
    } else {
      const localId = `local-post-${Date.now()}`;
      posts.value = [
        {
          id: localId,
          author: currentUser.value.displayName,
          handle: currentUser.value.handle,
          instance: currentUser.value.instance,
          kind: 'post',
          time: '刚刚',
          content: postDraft.value.trim() || '分享了一条新的媒体动态。',
          tags: ['离线草稿'],
          chainProof: `local://${localId}`,
          media:
            mediaPreview.value && mediaMeta.value
              ? {
                  name: mediaMeta.value.name,
                  preview: mediaPreview.value,
                  type: mediaMeta.value.type,
                  sizeLabel: mediaMeta.value.sizeLabel,
                }
              : undefined,
          stats: { replies: 0, boosts: 0, likes: 0 },
        },
        ...posts.value,
      ];
    }

    postDraft.value = '';
    mediaPreview.value = null;
    mediaMeta.value = null;
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '发布内容失败';
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

    if (apiOnline.value) {
      const updatedConversation = await createConversationMessage(targetConversation.id, {
        senderId: currentUser.value.id,
        body: messageDraft.value.trim(),
      });

      conversations.value = conversations.value.map((conversation) =>
        conversation.id === updatedConversation.id
          ? toConversationCard(updatedConversation, currentUser.value?.id ?? null)
          : conversation,
      );
    } else {
      conversations.value = conversations.value.map((conversation) =>
        conversation.id === targetConversation.id
          ? {
              ...conversation,
              messages: [
                ...conversation.messages,
                {
                  id: `local-message-${Date.now()}`,
                  from: 'me',
                  text: messageDraft.value.trim(),
                  time: '刚刚',
                },
              ],
            }
          : conversation,
      );
    }

    messageDraft.value = '';
    errorMessage.value = '';
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '发送消息失败';
  } finally {
    saving.value = false;
  }
}

onMounted(loadBootstrap);
</script>

<template>
  <div class="min-h-screen bg-[var(--app-bg)] text-[color:var(--text-primary)] transition-colors duration-300 lg:h-screen lg:overflow-hidden" :style="themeStyles">
    <div class="mx-auto max-w-[1540px] px-4 py-4 lg:h-screen lg:max-w-none lg:px-6 lg:overflow-hidden">
      <div v-if="errorMessage" class="mb-4 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
        {{ serviceNotice }}
      </div>

      <div v-if="loading" class="rounded-[24px] border border-[color:var(--border-color)] bg-[var(--frame-bg)] px-6 py-12 text-center text-[color:var(--text-secondary)]">
        正在载入社区内容...
      </div>

      <div v-else class="grid gap-0 overflow-hidden rounded-[28px] border border-[color:var(--border-color)] bg-[var(--frame-bg)] shadow-[0_20px_60px_rgba(15,23,42,0.08)] lg:h-[calc(100vh-32px)] lg:grid-cols-[360px_minmax(0,1fr)_320px]">
        <aside class="border-b border-[color:var(--border-color)] bg-[var(--panel-bg)] lg:h-[calc(100vh-32px)] lg:overflow-hidden lg:border-b-0 lg:border-r">
          <div class="space-y-4 p-5">
            <div class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-4">
              <input
                v-model="searchQuery"
                placeholder="搜索或输入网址"
                class="w-full bg-transparent text-sm text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
              />
            </div>

            <div class="flex items-center gap-4">
              <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-lime-200 to-cyan-200 text-xl font-bold text-slate-900">
                {{ avatarText(currentUser?.displayName || 'W') }}
              </div>
              <div class="min-w-0">
                <div class="truncate text-xl font-semibold text-[color:var(--text-primary)]">{{ currentUser?.displayName }}</div>
                <div class="truncate text-base text-[color:var(--text-secondary)]">{{ profileLabel(currentUser) }}</div>
              </div>
            </div>

            <div class="rounded-[22px] border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-4">
              <div class="mb-4 flex flex-wrap gap-2">
                <span class="rounded-xl border border-violet-400/30 bg-violet-500/10 px-3 py-2 text-sm text-violet-200">公开，允许引用</span>
                <span class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-muted)] px-3 py-2 text-sm text-[color:var(--text-secondary)]">简体中文</span>
              </div>

              <textarea
                v-model="postDraft"
                placeholder="想写什么？"
                class="min-h-[110px] w-full resize-none bg-transparent text-base text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
              />

              <div v-if="mediaPreview && mediaMeta" class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)]">
                <img :src="mediaPreview" :alt="mediaMeta.name" class="max-h-40 w-full object-cover" />
              </div>

              <div class="mt-5 flex items-center justify-between gap-3">
                <div class="flex items-center gap-4 text-xl text-[color:var(--text-secondary)]">
                  <label class="cursor-pointer transition hover:text-violet-300" title="上传图片或视频">
                    ⊞
                    <input type="file" accept="image/*,video/*" class="hidden" @change="handleMediaChange" />
                  </label>
                  <span title="投票">▤</span>
                  <span title="预警标签">△</span>
                  <span title="表情">☺</span>
                </div>

                <div class="flex items-center gap-4">
                  <span class="text-2xl font-light text-[color:var(--text-secondary)]">500</span>
                  <button
                    :disabled="saving"
                    @click="publishPost"
                    class="rounded-xl bg-violet-600 px-7 py-3 text-base font-semibold text-white transition hover:bg-violet-500 disabled:opacity-60"
                  >
                    {{ saving ? '发布中' : '发布' }}
                  </button>
                </div>
              </div>
            </div>

            <div class="space-y-2 pt-4 text-sm text-[color:var(--text-muted)]">
              <div>whale-vault.social · 关于本站 · 社区公约 · 隐私政策</div>
              <div>Whale Vault Social · 创作者支持 · 媒体规范 · 开放生态</div>
            </div>
          </div>
        </aside>

        <main class="bg-[var(--frame-bg)] lg:h-[calc(100vh-32px)] lg:overflow-y-auto no-scrollbar">
          <div class="border-b border-[color:var(--border-color)] px-6 py-5">
            <div class="flex items-center gap-3 text-xl font-semibold text-[color:var(--text-primary)]">
              <span class="text-lg text-[color:var(--text-secondary)]">⌕</span>
              <span>{{ currentSectionTitle }}</span>
            </div>
          </div>

          <section v-if="currentSection === 'home'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="post in timeline" :key="post.id" class="px-6 py-6">
              <div class="flex gap-4">
                <div class="flex h-14 w-14 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-violet-300 to-cyan-200 text-lg font-bold text-slate-900">
                  {{ avatarText(post.author) }}
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <span class="text-[22px] font-semibold text-[color:var(--text-primary)]">{{ post.author }}</span>
                    <span class="text-lg text-[color:var(--text-secondary)]">{{ post.handle }}@{{ post.instance }}</span>
                    <span class="text-sm text-[color:var(--text-muted)]">{{ post.time }}</span>
                  </div>
                  <div v-if="post.bio" class="mt-1 text-sm text-[color:var(--text-muted)]">{{ post.bio }}</div>
                  <div class="mt-4 whitespace-pre-wrap text-[17px] leading-8 text-[color:var(--text-soft)]">{{ post.content }}</div>

                  <div v-if="post.media" class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]">
                    <img :src="post.media.preview" :alt="post.media.name" class="max-h-[420px] w-full object-cover" />
                    <div class="flex items-center justify-between px-4 py-3 text-sm text-[color:var(--text-secondary)]">
                      <span>{{ post.media.name }}</span>
                      <span>{{ post.media.sizeLabel }}</span>
                    </div>
                  </div>

                  <div v-if="post.tags.length" class="mt-4 flex flex-wrap gap-2">
                    <span v-for="tag in post.tags" :key="tag" class="rounded-full bg-violet-500/10 px-3 py-1 text-sm text-violet-200">
                      #{{ tag }}
                    </span>
                  </div>

                  <div class="mt-5 flex flex-wrap items-center gap-3 text-sm">
                    <button
                      @click="openPostDetail(post.id)"
                      class="rounded-full border border-[color:var(--border-color)] px-3 py-2 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                    >
                      回复 {{ post.stats.replies }}
                    </button>
                    <button
                      @click="toggleBoost(post.id)"
                      class="rounded-full border px-3 py-2 transition"
                      :class="boostedPosts[post.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                    >
                      转发 {{ post.stats.boosts + (boostedPosts[post.id] ? 1 : 0) }}
                    </button>
                    <button
                      @click="toggleLike(post.id)"
                      class="rounded-full border px-3 py-2 transition"
                      :class="likedPosts[post.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                    >
                      喜欢 {{ post.stats.likes + (likedPosts[post.id] ? 1 : 0) }}
                    </button>
                    <button
                      @click="toggleBookmark(post.id)"
                      class="rounded-full border px-3 py-2 transition"
                      :class="bookmarkedPosts[post.id] ? 'border-violet-400/40 bg-violet-500/10 text-violet-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-violet-300/30 hover:text-violet-200'"
                    >
                      {{ bookmarkedPosts[post.id] ? '已收藏' : '收藏' }}
                    </button>
                    <span class="ml-auto truncate rounded-full bg-[var(--chip-bg)] px-3 py-2 text-[color:var(--text-muted)]">
                      {{ shortProof(post.chainProof) }}
                    </span>
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

              <div v-if="threadAncestors.length" class="px-6 py-6">
                <div class="mb-4 text-xs font-semibold uppercase tracking-[0.2em] text-[color:var(--text-muted)]">
                  上下文
                </div>
                <div class="space-y-4">
                  <article
                    v-for="ancestor in threadAncestors"
                    :key="ancestor.id"
                    class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-5 py-4"
                  >
                    <div class="flex items-center gap-2 text-sm">
                      <span class="font-semibold text-[color:var(--text-primary)]">{{ ancestor.author }}</span>
                      <span class="text-[color:var(--text-secondary)]">{{ ancestor.handle }}@{{ ancestor.instance }}</span>
                      <span class="text-[color:var(--text-muted)]">{{ ancestor.time }}</span>
                    </div>
                    <div class="mt-3 whitespace-pre-wrap text-base leading-7 text-[color:var(--text-secondary)]">
                      {{ ancestor.content }}
                    </div>
                  </article>
                </div>
              </div>

              <article class="px-6 py-6">
                <div class="flex gap-4">
                  <div class="flex h-14 w-14 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-violet-300 to-cyan-200 text-lg font-bold text-slate-900">
                    {{ avatarText(threadFocusPost.author) }}
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <span class="text-[24px] font-semibold text-[color:var(--text-primary)]">{{ threadFocusPost.author }}</span>
                      <span class="text-lg text-[color:var(--text-secondary)]">{{ threadFocusPost.handle }}@{{ threadFocusPost.instance }}</span>
                      <span class="rounded-full bg-violet-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-violet-500">
                        {{ threadFocusPost.kind === 'reply' ? '回复' : '帖子' }}
                      </span>
                      <span class="text-sm text-[color:var(--text-muted)]">{{ threadFocusPost.time }}</span>
                    </div>
                    <div v-if="threadFocusPost.bio" class="mt-1 text-sm text-[color:var(--text-muted)]">{{ threadFocusPost.bio }}</div>
                    <div class="mt-4 whitespace-pre-wrap text-[18px] leading-8 text-[color:var(--text-soft)]">{{ threadFocusPost.content }}</div>

                    <div
                      v-if="threadFocusPost.media"
                      class="mt-4 overflow-hidden rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-contrast)]"
                    >
                      <img :src="threadFocusPost.media.preview" :alt="threadFocusPost.media.name" class="max-h-[420px] w-full object-cover" />
                      <div class="flex items-center justify-between px-4 py-3 text-sm text-[color:var(--text-secondary)]">
                        <span>{{ threadFocusPost.media.name }}</span>
                        <span>{{ threadFocusPost.media.sizeLabel }}</span>
                      </div>
                    </div>

                    <div v-if="threadFocusPost.tags.length" class="mt-4 flex flex-wrap gap-2">
                      <span
                        v-for="tag in threadFocusPost.tags"
                        :key="tag"
                        class="rounded-full bg-violet-500/10 px-3 py-1 text-sm text-violet-200"
                      >
                        #{{ tag }}
                      </span>
                    </div>

                    <div class="mt-5 flex flex-wrap items-center gap-3 text-sm">
                      <button
                        class="rounded-full border border-[color:var(--border-color)] px-3 py-2 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        回复 {{ threadFocusPost.stats.replies }}
                      </button>
                      <button
                        @click="toggleBoost(threadFocusPost.id)"
                        class="rounded-full border px-3 py-2 transition"
                        :class="boostedPosts[threadFocusPost.id] ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-emerald-300/30 hover:text-emerald-200'"
                      >
                        转发 {{ threadFocusPost.stats.boosts + (boostedPosts[threadFocusPost.id] ? 1 : 0) }}
                      </button>
                      <button
                        @click="toggleLike(threadFocusPost.id)"
                        class="rounded-full border px-3 py-2 transition"
                        :class="likedPosts[threadFocusPost.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                      >
                        喜欢 {{ threadFocusPost.stats.likes + (likedPosts[threadFocusPost.id] ? 1 : 0) }}
                      </button>
                      <button
                        @click="toggleBookmark(threadFocusPost.id)"
                        class="rounded-full border px-3 py-2 transition"
                        :class="bookmarkedPosts[threadFocusPost.id] ? 'border-violet-400/40 bg-violet-500/10 text-violet-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-violet-300/30 hover:text-violet-200'"
                      >
                        {{ bookmarkedPosts[threadFocusPost.id] ? '已收藏' : '收藏' }}
                      </button>
                      <span class="ml-auto truncate rounded-full bg-[var(--chip-bg)] px-3 py-2 text-[color:var(--text-muted)]">
                        {{ shortProof(threadFocusPost.chainProof) }}
                      </span>
                    </div>
                  </div>
                </div>
              </article>

              <div ref="replyComposerRef" class="px-6 py-6">
                <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-5">
                  <div class="flex items-start gap-4">
                    <div class="flex h-12 w-12 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-violet-300 to-cyan-200 text-base font-bold text-slate-900">
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
                          class="rounded-xl bg-violet-600 px-5 py-3 text-sm font-semibold text-white transition hover:bg-violet-500 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {{ saving ? '发送中...' : '发送回复' }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="px-6 py-6">
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
                      <span class="font-semibold text-[color:var(--text-primary)]">{{ reply.author }}</span>
                      <span class="text-[color:var(--text-secondary)]">{{ reply.handle }}@{{ reply.instance }}</span>
                      <span class="rounded-full bg-[var(--chip-bg)] px-3 py-1 text-[color:var(--text-muted)]">
                        第 {{ (reply.replyDepth ?? 0) + 1 }} 层
                      </span>
                      <span class="text-[color:var(--text-muted)]">{{ reply.time }}</span>
                    </div>
                    <div class="mt-3 whitespace-pre-wrap text-base leading-7 text-[color:var(--text-secondary)]">
                      {{ reply.content }}
                    </div>
                    <div v-if="reply.tags.length" class="mt-4 flex flex-wrap gap-2">
                      <span v-for="tag in reply.tags" :key="tag" class="rounded-full bg-violet-500/10 px-3 py-1 text-xs text-violet-200">
                        #{{ tag }}
                      </span>
                    </div>
                    <div class="mt-4 flex flex-wrap items-center gap-3 text-sm">
                      <button
                        @click="setReplyTarget(reply)"
                        class="rounded-full border border-[color:var(--border-color)] px-3 py-2 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        回复此层
                      </button>
                      <button
                        @click="openPostDetail(reply.id)"
                        class="rounded-full border border-[color:var(--border-color)] px-3 py-2 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]"
                      >
                        查看子讨论
                      </button>
                      <button
                        @click="toggleLike(reply.id)"
                        class="rounded-full border px-3 py-2 transition"
                        :class="likedPosts[reply.id] ? 'border-rose-400/40 bg-rose-500/10 text-rose-300' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-rose-300/30 hover:text-rose-200'"
                      >
                        喜欢 {{ reply.stats.likes + (likedPosts[reply.id] ? 1 : 0) }}
                      </button>
                      <button
                        @click="toggleBookmark(reply.id)"
                        class="rounded-full border px-3 py-2 transition"
                        :class="bookmarkedPosts[reply.id] ? 'border-violet-400/40 bg-violet-500/10 text-violet-200' : 'border-[color:var(--border-color)] text-[color:var(--text-secondary)] hover:border-violet-300/30 hover:text-violet-200'"
                      >
                        {{ bookmarkedPosts[reply.id] ? '已收藏' : '收藏' }}
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

          <section v-else-if="currentSection === 'explore'">
            <div class="border-b border-[color:var(--border-color)] px-6 py-5">
              <div class="rounded-2xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-4">
                <input
                  v-model="searchQuery"
                  placeholder="搜索"
                  class="w-full bg-transparent text-base text-[color:var(--text-primary)] outline-none placeholder:text-[color:var(--text-muted)]"
                />
              </div>
            </div>

            <div class="divide-y divide-[color:var(--border-color)]">
              <article v-for="person in recommendedPeople" :key="person.id" class="px-6 py-5">
                <div class="flex items-start justify-between gap-4">
                  <div class="flex min-w-0 gap-4">
                    <div class="flex h-14 w-14 flex-none items-center justify-center rounded-2xl bg-gradient-to-br from-amber-200 to-violet-200 text-lg font-bold text-slate-900">
                      {{ avatarText(person.displayName) }}
                    </div>
                    <div class="min-w-0">
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="text-[18px] font-semibold text-[color:var(--text-primary)]">{{ person.displayName }}</span>
                        <span class="truncate text-[18px] text-[color:var(--text-secondary)]">{{ person.handle }}@{{ person.instance }}</span>
                      </div>
                      <div class="mt-1 text-lg text-[color:var(--text-secondary)]">{{ person.followers }} 关注者</div>
                      <div class="mt-2 line-clamp-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ person.bio }}</div>
                    </div>
                  </div>

                  <button
                    @click="toggleFollow(person.id)"
                    class="rounded-xl bg-violet-600 px-6 py-3 text-lg font-semibold text-white transition hover:bg-violet-500"
                  >
                    {{ followedUsers[person.id] ? '已关注' : '关注' }}
                  </button>
                </div>
              </article>
            </div>
          </section>

          <section v-else-if="currentSection === 'notifications'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in notificationItems" :key="item.id" class="px-6 py-6">
              <div class="flex items-start gap-4">
                <div class="mt-1 flex h-11 w-11 items-center justify-center rounded-2xl bg-violet-600/12 text-violet-600">◌</div>
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
            <article v-for="item in curatedLists" :key="item.id" class="px-6 py-6">
              <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-6">
                <div class="flex items-center justify-between gap-3">
                  <div class="text-xl font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                  <div class="text-sm text-violet-600">{{ item.count }}</div>
                </div>
                <div class="mt-3 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.summary }}</div>
              </div>
            </article>
          </section>

          <section v-else-if="currentSection === 'topics'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in followedTopicCards" :key="item.tag" class="px-6 py-6">
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
            <article v-for="post in likedTimeline" v-else :key="post.id" class="px-6 py-6">
              <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ post.author }}</div>
              <div class="mt-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ post.content }}</div>
            </article>
          </section>

          <section v-else-if="currentSection === 'bookmarks'" class="divide-y divide-[color:var(--border-color)]">
            <article v-if="bookmarkedTimeline.length === 0" class="px-6 py-12 text-center text-[color:var(--text-muted)]">
              收藏的动态会整理在这里，方便稍后继续阅读。
            </article>
            <article v-for="post in bookmarkedTimeline" v-else :key="post.id" class="px-6 py-6">
              <div class="text-lg font-semibold text-[color:var(--text-primary)]">{{ post.author }}</div>
              <div class="mt-2 text-base leading-7 text-[color:var(--text-secondary)]">{{ post.content }}</div>
            </article>
          </section>

          <section v-else-if="currentSection === 'mentions'" class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in mentionItems" :key="item.id" class="px-6 py-6">
              <div class="flex items-start gap-4">
                <div class="mt-1 flex h-11 w-11 items-center justify-center rounded-2xl bg-violet-600/12 text-violet-600">@</div>
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

          <section v-else-if="currentSection === 'preferences'" class="min-h-[calc(100vh-140px)] px-0 py-0">
            <div class="grid min-h-[calc(100vh-140px)] lg:grid-cols-[260px_minmax(0,1fr)]">
              <aside class="border-r border-[color:var(--border-color)] bg-[var(--panel-bg)] px-6 py-8">
                <button
                  @click="setSection('home')"
                  class="mb-8 flex items-center gap-3 text-base text-[color:var(--text-secondary)] transition hover:text-violet-500"
                >
                  <span>‹</span>
                  <span>返回社区</span>
                </button>

                <div class="space-y-2">
                  <button
                    v-for="item in settingsMenu"
                    :key="item.id"
                    @click="currentSettingsTab = item.id"
                    class="flex w-full items-center gap-3 rounded-2xl px-4 py-3 text-left text-base transition"
                    :class="currentSettingsTab === item.id ? 'bg-violet-600/12 text-violet-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'"
                  >
                    <span class="w-5 text-center">{{ item.icon }}</span>
                    <span>{{ item.label }}</span>
                  </button>
                </div>
              </aside>

              <div class="px-8 py-8 lg:px-10">
                <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <div class="text-[34px] font-semibold tracking-tight text-[color:var(--text-primary)]">外观</div>
                    <div class="mt-2 text-sm text-[color:var(--text-muted)]">调整界面语言、配色方案和可访问性偏好。</div>
                  </div>

                  <div class="flex items-center gap-3">
                    <span v-if="settingsNotice" class="text-sm text-emerald-500">{{ settingsNotice }}</span>
                    <button
                      :disabled="!hasAppearanceChanges"
                      @click="saveAppearancePreferences"
                      class="rounded-xl bg-violet-600 px-6 py-3 text-base font-semibold text-white transition hover:bg-violet-500 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      保存更改
                    </button>
                  </div>
                </div>

                <div class="grid gap-6 lg:grid-cols-2">
                  <label class="block">
                    <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">界面语言</div>
                    <select
                      v-model="appearanceDraft.language"
                      class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
                    >
                      <option>简体中文</option>
                      <option>English</option>
                    </select>
                  </label>

                  <label class="block">
                    <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">时区</div>
                    <select
                      v-model="appearanceDraft.timezone"
                      class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
                    >
                      <option>(GMT+08:00) Asia/Shanghai</option>
                      <option>(GMT+00:00) UTC</option>
                      <option>(GMT-07:00) America/Los_Angeles</option>
                    </select>
                  </label>
                </div>

                <div class="mt-10 space-y-10">
                  <div>
                    <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">配色方案</div>
                  <div class="flex flex-wrap gap-8 text-base">
                    <label
                      class="flex items-center gap-2"
                      :class="toneClass('auto')"
                    >
                      <input v-model="appearanceDraft.colorScheme" type="radio" value="auto" class="accent-violet-600" />
                      <span>自动</span>
                    </label>
                    <label
                      class="flex items-center gap-2"
                      :class="toneClass('light')"
                    >
                      <input v-model="appearanceDraft.colorScheme" type="radio" value="light" class="accent-violet-600" />
                      <span>浅色</span>
                    </label>
                    <label
                      class="flex items-center gap-2"
                      :class="toneClass('dark')"
                    >
                      <input v-model="appearanceDraft.colorScheme" type="radio" value="dark" class="accent-violet-600" />
                      <span>深色</span>
                    </label>
                  </div>
                  </div>

                  <div>
                    <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">对比度</div>
                    <div class="flex flex-wrap gap-8 text-base text-[color:var(--text-primary)]">
                      <label class="flex items-center gap-2">
                        <input v-model="appearanceDraft.contrast" type="radio" value="auto" class="accent-violet-600" />
                        <span>自动</span>
                      </label>
                      <label class="flex items-center gap-2">
                        <input v-model="appearanceDraft.contrast" type="radio" value="high" class="accent-violet-600" />
                        <span>高</span>
                      </label>
                    </div>
                  </div>

                  <label class="block">
                    <div class="mb-3 text-sm font-semibold text-[color:var(--text-primary)]">表情符号样式</div>
                    <select
                      v-model="appearanceDraft.emojiStyle"
                      class="w-full rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-4 py-3 text-base text-[color:var(--text-primary)] outline-none"
                    >
                      <option>自动</option>
                      <option>原生 Emoji</option>
                      <option>Twemoji</option>
                    </select>
                    <div class="mt-3 rounded-2xl border border-violet-500/30 bg-violet-500/5 px-4 py-4 text-sm leading-7 text-[color:var(--text-secondary)]">
                      你可以在这里选择社区默认外观。配色方案默认显示为浅色，保存后会在整个前端界面生效。
                    </div>
                  </label>

                  <div class="border-t border-[color:var(--border-color)] pt-8">
                    <div class="mb-4 text-sm font-semibold text-[color:var(--text-primary)]">动画与可访问性</div>
                    <div class="space-y-5 text-[color:var(--text-primary)]">
                      <label class="flex items-start gap-3">
                        <input v-model="appearanceDraft.slowMode" type="checkbox" class="mt-1 h-4 w-4 rounded accent-violet-600" />
                        <div>
                          <div class="font-medium">慢速模式</div>
                          <div class="mt-1 text-sm text-[color:var(--text-muted)]">点击查看时间线更新，而非自动滚动更新动态。</div>
                        </div>
                      </label>

                      <label class="flex items-start gap-3">
                        <input v-model="appearanceDraft.autoplayGif" type="checkbox" class="mt-1 h-4 w-4 rounded accent-violet-600" />
                        <div>
                          <div class="font-medium">自动播放 GIF 动画</div>
                          <div class="mt-1 text-sm text-[color:var(--text-muted)]">推荐开启，在动态流中直接预览轻量动画内容。</div>
                        </div>
                      </label>

                      <label class="flex items-start gap-3">
                        <input v-model="appearanceDraft.reduceMotion" type="checkbox" class="mt-1 h-4 w-4 rounded accent-violet-600" />
                        <div>
                          <div class="font-medium">降低动态效果</div>
                          <div class="mt-1 text-sm text-[color:var(--text-muted)]">减少转场、悬浮和自动播放带来的视觉刺激。</div>
                        </div>
                      </label>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section v-else class="divide-y divide-[color:var(--border-color)]">
            <article v-for="item in moreCards" :key="item.id" class="px-6 py-6">
              <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-6">
                <div class="text-xl font-semibold text-[color:var(--text-primary)]">{{ item.title }}</div>
                <div class="mt-3 text-base leading-7 text-[color:var(--text-secondary)]">{{ item.description }}</div>
              </div>
            </article>
          </section>
        </main>

        <aside class="border-t border-[color:var(--border-color)] bg-[var(--panel-bg)] lg:h-[calc(100vh-32px)] lg:overflow-hidden lg:border-l lg:border-t-0">
          <div class="p-5">
            <div class="mb-10 flex items-center gap-3">
              <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-violet-600 text-xl font-bold text-white">w</div>
              <div class="text-[24px] font-semibold tracking-tight text-[color:var(--text-primary)]">whaleodon</div>
            </div>

            <div class="space-y-1">
              <button
                v-for="item in primaryNavItems"
                :key="item.key"
                @click="item.key === 'preferences' ? openSettings('appearance') : setSection(item.key)"
                class="flex w-full items-center gap-4 rounded-2xl px-4 py-4 text-left text-[18px] transition"
                :class="currentSection === item.key ? 'bg-violet-600/12 text-violet-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'"
              >
                <span class="w-6 text-center text-lg">{{ item.icon }}</span>
                <span>{{ item.label }}</span>
              </button>
            </div>

            <div class="mt-8 border-t border-[color:var(--border-color)] pt-8">
              <div class="space-y-1">
                <button
                  v-for="item in secondaryNavItems"
                  :key="item.key"
                  @click="setSection(item.key)"
                  class="flex w-full items-center gap-4 rounded-2xl px-4 py-4 text-left text-[18px] transition"
                  :class="currentSection === item.key ? 'bg-violet-600/12 text-violet-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'"
                >
                  <span class="w-6 text-center text-lg">{{ item.icon }}</span>
                  <span>{{ item.label }}</span>
                </button>
              </div>
            </div>

            <div class="mt-8 border-t border-[color:var(--border-color)] pt-8">
              <div class="space-y-1">
                <button
                  v-for="item in utilityNavItems"
                  :key="item.key"
                  @click="item.key === 'preferences' ? openSettings('appearance') : setSection(item.key)"
                  class="flex w-full items-center gap-4 rounded-2xl px-4 py-4 text-left text-[18px] transition"
                  :class="currentSection === item.key ? 'bg-violet-600/12 text-violet-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'"
                >
                  <span class="w-6 text-center text-lg">{{ item.icon }}</span>
                  <span>{{ item.label }}</span>
                </button>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </div>
  </div>
</template>

<style scoped>
.no-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.no-scrollbar::-webkit-scrollbar {
  width: 0;
  height: 0;
}
</style>
