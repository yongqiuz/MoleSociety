import { BrowserProvider } from 'ethers';
import type { UserField } from './socialApi';

export type AuthSession = {
  id: string;
  handle: string;
  displayName: string;
  instance: string;
  bio: string;
  avatarUrl: string;
  wallet: string;
  fields: UserField[];
  featuredTags: string[];
  isBot: boolean;
};

type ApiEnvelope<T> = {
  ok: boolean;
  data: T;
  error?: string;
};

type ChallengeResponse = {
  nonce: string;
  message: string;
  chainId: number;
  issuedAt: string;
  expiresAt: string;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

const fallbackHost = typeof window !== 'undefined' ? window.location.hostname : 'localhost';
const defaultApiUrl = `http://${fallbackHost}:8080`;
const API_BASE = (import.meta.env.VITE_SOCIAL_API_URL || defaultApiUrl).replace(/\/$/, '');

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = init?.method || 'GET';
  console.log(`[Auth] ${method} ${path}`);
  const response = await fetch(`${API_BASE}${path}`, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
    ...init,
  });

  const payload = (await response.json()) as ApiEnvelope<T> | { ok: boolean; error?: string };
  if (!response.ok || !payload.ok) {
    console.warn(`[Auth] ${method} ${path} → ${response.status}`, payload.error);
    throw new ApiError(payload.error || `Request failed: ${response.status}`, response.status);
  }

  console.log(`[Auth] ${method} ${path} → ${response.status} OK`);
  return (payload as ApiEnvelope<T>).data;
}

export async function fetchCurrentSession() {
  return request<AuthSession>('/api/v1/auth/me');
}

export async function logoutSession() {
  return request<{ loggedOut: boolean }>('/api/v1/auth/logout', {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export async function connectWalletAndLogin() {
  if (typeof window === 'undefined' || !window.ethereum) {
    throw new Error('No injected wallet found. Please install MetaMask or another EVM wallet.');
  }

  const provider = new BrowserProvider(window.ethereum);
  await provider.send('eth_requestAccounts', []);
  const signer = await provider.getSigner();
  const address = await signer.getAddress();
  const network = await provider.getNetwork();

  const challenge = await request<ChallengeResponse>('/api/v1/auth/challenge', {
    method: 'POST',
    body: JSON.stringify({
      address,
      chainId: Number(network.chainId),
    }),
  });

  const signature = await signer.signMessage(challenge.message);
  return request<AuthSession>('/api/v1/auth/verify', {
    method: 'POST',
    body: JSON.stringify({
      address,
      nonce: challenge.nonce,
      signature,
    }),
  });
}

export async function passwordLogin(identifier: string, password: string) {
  return request<AuthSession>('/api/v1/auth/password-login', {
    method: 'POST',
    body: JSON.stringify({ identifier, password }),
  });
}

export async function registerAccount(payload: {
  username: string;
  email?: string;
  password: string;
  walletAddress?: string;
  chainId?: number;
  signature?: string;
  nonce?: string;
}) {
  return request<AuthSession>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchBindChallenge(walletAddress: string, chainId: number) {
  return request<ChallengeResponse>('/api/v1/auth/bind-challenge', {
    method: 'POST',
    body: JSON.stringify({ walletAddress, chainId }),
  });
}
