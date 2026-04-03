import { writable } from 'svelte/store';

function createAuthToken() {
  const initial = localStorage.getItem('auth_token');
  const store = writable<string | null>(initial);

  return {
    subscribe: store.subscribe,
    set: (token: string | null) => {
      if (token) {
        localStorage.setItem('auth_token', token);
      } else {
        localStorage.removeItem('auth_token');
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
  eating_window_start: string | null;
  fasting_streak: number;
}

export interface FastingStatus {
  is_fasting: boolean;
  fast_start_time: string | null;
  eating_window_start: string;
  eating_window_hours: number;
  fasting_hours: number;
  seconds_elapsed: number;
  seconds_remaining: number;
  fasting_streak: number;
  fasting_window: string;
}

function createPersistedStore<T>(key: string, fallback: T) {
  let initial: T = fallback;
  const raw = localStorage.getItem(key);
  if (raw) {
    try {
      initial = JSON.parse(raw) as T;
    } catch {
      initial = fallback;
    }
  }
  const store = writable<T>(initial);

  return {
    subscribe: store.subscribe,
    set: (value: T) => {
      localStorage.setItem(key, JSON.stringify(value));
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
  daily_fat_g: 0,
  eating_window_start: null,
  fasting_streak: 0,
});

export type Theme = 'dark' | 'light';

function createThemeStore() {
  let initial: Theme = 'dark';
  const stored = localStorage.getItem('theme') as Theme | null;
  if (stored === 'light' || stored === 'dark') {
    initial = stored;
  } else if (window.matchMedia('(prefers-color-scheme: light)').matches) {
    initial = 'light';
  }
  const store = writable<Theme>(initial);

  document.documentElement.classList.remove('dark', 'light');
  document.documentElement.classList.add(initial);

  return {
    subscribe: store.subscribe,
    set: (value: Theme) => {
      localStorage.setItem('theme', value);
      document.documentElement.classList.remove('dark', 'light');
      document.documentElement.classList.add(value);
      const meta = document.querySelector('meta[name="theme-color"]');
      if (meta) {
        meta.setAttribute('content', value === 'dark' ? '#0f172a' : '#f8fafc');
      }
      store.set(value);
    }
  };
}

export const theme = createThemeStore();
