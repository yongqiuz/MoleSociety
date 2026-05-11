/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue';

  const component: DefineComponent<Record<string, never>, Record<string, never>, any>;
  export default component;
}

interface Window {
  ethereum?: {
    isMetaMask?: boolean;
    providers?: Array<{
      isMetaMask?: boolean;
      request(args: { method: string; params?: unknown[] | object }): Promise<unknown>;
    }>;
    request(args: { method: string; params?: unknown[] | object }): Promise<unknown>;
  };
}
