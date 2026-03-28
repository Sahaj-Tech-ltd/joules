import { writable } from 'svelte/store';

function createAuthToken() {
  const initial = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
  const store = writable<string | null>(initial);

  return {
    subscribe: store.subscribe,
    set: (token: string | null) => {
      if (typeof window !== 'undefined') {
        if (token) {
          localStorage.setItem('auth_token', token);
        } else {
          localStorage.removeItem('auth_token');
        }
      }
      store.set(token);
    }
  };
}

export const authToken = createAuthToken();

export interface UserProfile {
  name: string;
  age: number | null;
  sex: string | null;
  height_cm: number | null;
  weight_kg: number | null;
  target_weight_kg: number | null;
  activity_level: string | null;
  onboarding_complete: boolean;
  is_admin: boolean;
  avatar_url: string | null;
}

export interface UserGoals {
  objective: string;
  diet_plan: string;
  fasting_window: string | null;
  daily_calorie_target: number;
  daily_protein_g: number;
  daily_carbs_g: number;
  daily_fat_g: number;
}

function createPersistedStore<T>(key: string, fallback: T) {
  let initial: T = fallback;
  if (typeof window !== 'undefined') {
    const raw = localStorage.getItem(key);
    if (raw) {
      try {
        initial = JSON.parse(raw) as T;
      } catch {
        initial = fallback;
      }
    }
  }
  const store = writable<T>(initial);

  return {
    subscribe: store.subscribe,
    set: (value: T) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem(key, JSON.stringify(value));
      }
      store.set(value);
    }
  };
}

export const userProfile = createPersistedStore<UserProfile>('user_profile', {
  name: '',
  age: null,
  sex: null,
  height_cm: null,
  weight_kg: null,
  target_weight_kg: null,
  activity_level: null,
  onboarding_complete: false,
  is_admin: false,
  avatar_url: null,
});

export const userGoals = createPersistedStore<UserGoals>('user_goals', {
  objective: '',
  diet_plan: '',
  fasting_window: null,
  daily_calorie_target: 0,
  daily_protein_g: 0,
  daily_carbs_g: 0,
  daily_fat_g: 0
});

export type Theme = 'dark' | 'light';

function createThemeStore() {
  let initial: Theme = 'dark';
  if (typeof window !== 'undefined') {
    const stored = localStorage.getItem('theme') as Theme | null;
    if (stored === 'light' || stored === 'dark') {
      initial = stored;
    } else if (window.matchMedia('(prefers-color-scheme: light)').matches) {
      initial = 'light';
    }
  }
  const store = writable<Theme>(initial);

  return {
    subscribe: store.subscribe,
    set: (value: Theme) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem('theme', value);
        document.documentElement.classList.remove('dark', 'light');
        document.documentElement.classList.add(value);
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) {
          meta.setAttribute('content', value === 'dark' ? '#0f172a' : '#f8fafc');
        }
      }
      store.set(value);
    }
  };
}

export const theme = createThemeStore();
