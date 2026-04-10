import { BrowserProvider } from 'ethers';

export type AuthSession = {
  id: string;
  handle: string;
  displayName: string;
  instance: string;
  bio: string;
  avatarUrl: string;
  wallet: string;
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

const API_BASE = (import.meta.env.VITE_SOCIAL_API_URL || 'http://localhost:8080').replace(/\/$/, '');

async function request<T>(path: string, init?: RequestInit): Promise<T> {
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
    throw new ApiError(payload.error || `Request failed: ${response.status}`, response.status);
  }

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
