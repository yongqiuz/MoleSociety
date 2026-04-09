export type SocialUser = {
  id: string;
  handle: string;
  displayName: string;
  bio: string;
  instance: string;
  wallet: string;
  avatarUrl: string;
  followers: number;
  following: number;
  createdAt: string;
};

export type PostMedia = {
  id: string;
  name: string;
  url: string;
  kind: string;
  storageUri: string;
  cid: string;
};

export type SocialPost = {
  id: string;
  authorId: string;
  authorHandle: string;
  authorName: string;
  instance: string;
  kind: string;
  content: string;
  visibility: string;
  storageUri: string;
  attestationUri: string;
  tags: string[];
  media: PostMedia[] | null;
  parentPostId?: string;
  rootPostId?: string;
  replyDepth?: number;
  replies: number;
  boosts: number;
  likes: number;
  createdAt: string;
};

export type PostThread = {
  post: SocialPost;
  ancestors: SocialPost[];
  replies: SocialPost[];
};

export type MediaAsset = {
  id: string;
  ownerId: string;
  name: string;
  kind: string;
  url: string;
  storageUri: string;
  cid: string;
  sizeBytes: number;
  status: string;
  createdAt: string;
};

export type SocialMessage = {
  id: string;
  conversationId: string;
  senderId: string;
  senderHandle: string;
  body: string;
  createdAt: string;
};

export type SocialConversation = {
  id: string;
  title: string;
  participantIds: string[];
  encrypted: boolean;
  messages: SocialMessage[];
  updatedAt: string;
};

export type FederationInstance = {
  name: string;
  focus: string;
  members: string;
  latency: string;
  status: string;
};

export type SocialStats = {
  users: number;
  posts: number;
  mediaAssets: number;
  conversations: number;
};

export type BootstrapPayload = {
  currentUser?: SocialUser;
  stats: SocialStats;
  feed: SocialPost[];
  users: SocialUser[];
  media: MediaAsset[];
  conversations: SocialConversation[];
  instances: FederationInstance[];
};

type ApiEnvelope<T> = {
  ok: boolean;
  data: T;
  error?: string;
};

type CreateMediaRequest = {
  ownerId: string;
  name: string;
  kind: string;
  url: string;
  storageUri: string;
  cid: string;
  sizeBytes: number;
  status: string;
};

type CreatePostRequest = {
  authorId: string;
  kind?: string;
  content: string;
  visibility: string;
  storageUri: string;
  attestationUri: string;
  tags: string[];
  mediaIds: string[];
  parentPostId?: string;
  rootPostId?: string;
};

type CreateMessageRequest = {
  senderId: string;
  body: string;
};

const API_BASE = (import.meta.env.VITE_SOCIAL_API_URL || 'http://localhost:8080').replace(/\/$/, '');

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
    ...init,
  });

  const payload = (await response.json()) as ApiEnvelope<T> | { ok: boolean; error?: string };
  if (!response.ok || !payload.ok) {
    throw new Error(payload.error || `Request failed: ${response.status}`);
  }

  return (payload as ApiEnvelope<T>).data;
}

export async function fetchSocialBootstrap(limit = 20) {
  return request<BootstrapPayload>(`/api/v1/social/bootstrap?limit=${limit}`);
}

export async function createMediaAsset(payload: CreateMediaRequest) {
  return request<MediaAsset>('/api/v1/social/media', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function createPost(payload: CreatePostRequest) {
  return request<SocialPost>('/api/v1/social/posts', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchPostThread(postId: string, limit = 20) {
  return request<PostThread>(`/api/v1/social/posts/${postId}/thread?limit=${limit}`);
}

export async function fetchPostReplies(postId: string, limit = 20) {
  return request<SocialPost[]>(`/api/v1/social/posts/${postId}/replies?limit=${limit}`);
}

export async function createConversationMessage(conversationId: string, payload: CreateMessageRequest) {
  return request<SocialConversation>(`/api/v1/social/conversations/${conversationId}/messages`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
