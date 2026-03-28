import { writable } from 'svelte/store';

export interface NewAchievement {
  id: string;
  title: string;
  description: string;
}

export const newAchievements = writable<NewAchievement[]>([]);

export function showAchievement(a: NewAchievement) {
  newAchievements.update(list => [...list, a]);
  setTimeout(() => {
    newAchievements.update(list => list.filter(x => x.id !== a.id));
  }, 5000);
}
