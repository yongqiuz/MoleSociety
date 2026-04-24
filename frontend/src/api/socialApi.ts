import { ApiError, type ApiEnvelope } from './apiError';

export type UserField = {
  name: string;
  value: string;
};

export type SocialUser = {
  id: string;
  handle: string;
  displayName: string;
  bio: string;
  instance: string;
  wallet: string;
  avatarUrl: string;
  fields: UserField[];
  featuredTags: string[];
  isBot: boolean;
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
  chainId?: string;
  txHash?: string;
  contractAddress?: string;
  explorerUrl?: string;
  tags: string[];
  media: PostMedia[] | null;
  type: string;
  interaction: string;
  parentPostId?: string;
  rootPostId?: string;
  replyDepth?: number;
  replies: number;
  boosts: number;
  likes: number;
  poll?: Poll;
  createdAt: string;
};

export type PollOption = {
  label: string;
  votes: number;
};

export type Poll = {
  options: PollOption[];
  expiresAt: string;
  multiple: boolean;
  voters: string[];
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
  assetUri?: string;
  chainId?: string;
  txHash?: string;
  contractAddress?: string;
  explorerUrl?: string;
  createdAt: string;
};

export type SocialConversation = {
  id: string;
  title: string;
  participantIds: string[];
  encrypted: boolean;
  assetUri?: string;
  chainId?: string;
  txHash?: string;
  contractAddress?: string;
  explorerUrl?: string;
  crossInstance?: boolean;
  federationRoute?: string;
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
  instance?: string;
  kind?: string;
  content: string;
  visibility: string;
  interaction: string;
  storageUri: string;
  attestationUri: string;
  tags: string[];
  mediaIds: string[];
  type: string;
  pollOptions?: string[];
  pollExpiresIn?: number;
  pollMultiple?: boolean;
  parentPostId?: string;
  rootPostId?: string;
};

type CreateMessageRequest = {
  senderId: string;
  body: string;
};

type CreateConversationRequest = {
  title: string;
  participantIds: string[];
  encrypted: boolean;
};

export type UpdateUserRequest = {
  displayName?: string;
  bio?: string;
  instance?: string;
  avatarUrl?: string;
  fields?: UserField[];
  featuredTags?: string[];
  isBot?: boolean;
};

const fallbackHost = typeof window !== 'undefined' ? window.location.hostname : '127.0.0.1';
const defaultApiUrl = `http://${fallbackHost}:8080`;
const API_BASE = (import.meta.env.VITE_SOCIAL_API_URL || defaultApiUrl).replace(/\/$/, '');
console.log('[API] social base =', API_BASE);

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = init?.method || 'GET';
  console.log(`[API] ${method} ${path}`);
  const response = await fetch(`${API_BASE}${path}`, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
    ...init,
  });

  const payload = (await response.json()) as ApiEnvelope<T>;
  if (!response.ok || !payload.ok) {
    console.warn(`[API] ${method} ${path} → ${response.status}`, payload.error);
    throw new ApiError(payload.error || `Request failed: ${response.status}`, response.status, payload.code, payload.type);
  }

  console.log(`[API] ${method} ${path} → ${response.status} OK`);
  return (payload as ApiEnvelope<T>).data;
}

export async function fetchSocialBootstrap(limit = 20) {
  return request<BootstrapPayload>(`/api/v1/social/bootstrap?limit=${limit}`);
}

export async function fetchSocialBootstrapMine(limit = 20) {
  return request<BootstrapPayload>(`/api/v1/social/bootstrap?limit=${limit}&mine=1`);
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

export async function listConversations(limit = 20) {
  return request<SocialConversation[]>(`/api/v1/social/conversations?limit=${limit}`);
}

export async function getConversation(conversationId: string) {
  return request<SocialConversation>(`/api/v1/social/conversations/${conversationId}`);
}

export async function createConversation(payload: CreateConversationRequest) {
  return request<SocialConversation>('/api/v1/social/conversations', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function updateUserProfile(userId: string, payload: UpdateUserRequest) {
  return request<SocialUser>(`/api/v1/social/users/${userId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export async function voteOnPoll(postId: string, optionIndices: number[]) {
  return request<SocialPost>(`/api/v1/social/posts/${postId}/poll/vote`, {
    method: 'POST',
    body: JSON.stringify({ optionIndices }),
  });
}
