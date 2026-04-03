import { writable } from 'svelte/store';

type Features = Record<string, boolean>;

const defaults: Features = {
  coach: true,
  ai_food_id: true,
  barcode: true,
  groups: true,
  gamification: true,
  fasting: true,
  achievements: true,
  steps: true,
  export: true,
  recipes: true,
  tips: true,
  notifications: true,
};

function createFeaturesStore() {
  const { subscribe, set } = writable<Features>(defaults);
  let loaded = false;

  return {
    subscribe,
    async load() {
      if (loaded) return;
      try {
        const res = await fetch('/api/features');
        if (res.ok) {
          const data = await res.json();
          if (data.data) {
            set({ ...defaults, ...data.data });
          }
        }
        loaded = true;
      } catch {}
    },
  };
}

export const features = createFeaturesStore();
