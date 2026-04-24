import { BrowserProvider } from 'ethers';
import type { UserField } from './socialApi';
import { ApiError, type ApiEnvelope } from './apiError';

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

export { ApiError };

type ChallengeResponse = {
  nonce: string;
  message: string;
  chainId: number;
  issuedAt: string;
  expiresAt: string;
};

const API_BASE = (import.meta.env.VITE_SOCIAL_API_URL || '').replace(/\/$/, '');
console.log('[API] auth base =', API_BASE);

async function request<T>(path: string, init?: RequestInit): Promise<T> {
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
    console.error('[AUTH ERROR]', {
      status: response.status,
      error: payload.error,
      code: payload.code,
      type: payload.type,
      raw: payload,
    });
    throw new ApiError(
      payload.error || `Request failed: ${response.status}`,
      response.status,
      payload.code,
      payload.type,
    );
  }

  return payload.data;
}

function createNoWalletError() {
  return new ApiError('No injected wallet found. Please install MetaMask or another EVM wallet.', 400, 'AUTH_WALLET_APP_MISSING', 'wallet');
}

async function connectWallet() {
  if (typeof window === 'undefined' || !window.ethereum) {
    throw createNoWalletError();
  }

  const provider = new BrowserProvider(window.ethereum);
  await provider.send('eth_requestAccounts', []);
  const signer = await provider.getSigner();
  const address = await signer.getAddress();
  const network = await provider.getNetwork();

  return {
    signer,
    address,
    chainId: Number(network.chainId),
  };
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
  const { signer, address, chainId } = await connectWallet();

  const challenge = await request<ChallengeResponse>('/api/v1/auth/challenge', {
    method: 'POST',
    body: JSON.stringify({ address, chainId }),
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
  console.log('[AUTH API] passwordLogin called');
  return request<AuthSession>('/api/v1/auth/password-login', {
    method: 'POST',
    body: JSON.stringify({ identifier, password }),
  });
}

export async function registerAccount(payload: {
  username: string;
  email?: string;
  password: string;
  autoWallet?: boolean;
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

export async function connectWalletForRegistration() {
  const { signer, address, chainId } = await connectWallet();
  const challenge = await fetchBindChallenge(address, chainId);
  const signature = await signer.signMessage(challenge.message);

  return {
    walletAddress: address,
    chainId,
    nonce: challenge.nonce,
    signature,
  };
}
