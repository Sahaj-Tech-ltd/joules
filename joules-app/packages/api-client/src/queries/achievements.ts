import { api } from '../api';
import type { Achievement } from '../types';

export function fetchAchievements(): Promise<Achievement[]> {
  return api.get<Achievement[]>('/achievements/');
}

export function checkAchievements(): Promise<Achievement[]> {
  return api.post<Achievement[]>('/achievements/check');
}
