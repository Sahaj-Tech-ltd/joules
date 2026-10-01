import { create } from 'zustand';

export interface StorageAdapter {
  getItem: (key: string) => Promise<string | null> | string | null;
  setItem: (key: string, value: string) => Promise<void> | void;
  removeItem: (key: string) => Promise<void> | void;
}

let storageAdapter: StorageAdapter | null = null;

export function normalizeBaseUrl(url: string): string {
  let clean = url.trim().replace(/\/+$/, '');
  if (!clean.endsWith('/api')) {
    clean += '/api';
  }
  return clean;
}

interface AuthState {
  token: string | null;
  baseUrl: string;
  isHydrated: boolean;
  setToken: (token: string | null) => void;
  setBaseUrl: (url: string) => void;
  clear: () => void;
  setHydrated: (hydrated: boolean) => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: null,
  baseUrl: 'http://localhost:3000/api',
  isHydrated: false,
  setToken: (token) => {
    set({ token });
    if (storageAdapter) {
      if (token) {
        storageAdapter.setItem('joule_auth_token', token);
      } else {
        storageAdapter.removeItem('joule_auth_token');
      }
    }
  },
  setBaseUrl: (url) => {
    const normalized = normalizeBaseUrl(url);
    set({ baseUrl: normalized });
    if (storageAdapter) {
      storageAdapter.setItem('joule_base_url', normalized);
    }
  },
  clear: () => {
    set({ token: null });
    if (storageAdapter) {
      storageAdapter.removeItem('joule_auth_token');
    }
  },
  setHydrated: (isHydrated) => set({ isHydrated }),
}));

export async function initAuth(adapter: StorageAdapter): Promise<void> {
  storageAdapter = adapter;
  try {
    const [savedToken, savedUrl] = await Promise.all([
      adapter.getItem('joule_auth_token'),
      adapter.getItem('joule_base_url'),
    ]);
    if (savedToken) {
      useAuthStore.getState().setToken(savedToken);
    }
    if (savedUrl) {
      useAuthStore.getState().setBaseUrl(savedUrl);
    }
  } catch (e) {
    console.warn('Failed to hydrate auth state from storage', e);
  } finally {
    useAuthStore.getState().setHydrated(true);
  }
}

export function getToken(): string | null {
  return useAuthStore.getState().token;
}

export function setToken(token: string | null): void {
  useAuthStore.getState().setToken(token);
}

export function clearToken(): void {
  useAuthStore.getState().clear();
}

export function getBaseUrl(): string {
  return useAuthStore.getState().baseUrl;
}
