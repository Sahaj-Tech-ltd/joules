import { create } from 'zustand';

interface AuthState {
  token: string | null;
  baseUrl: string;
  setToken: (token: string | null) => void;
  setBaseUrl: (url: string) => void;
  clear: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  baseUrl: 'http://localhost:3000/api',
  setToken: (token) => set({ token }),
  setBaseUrl: (url) => set({ baseUrl: url }),
  clear: () => set({ token: null }),
}));

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
