import { api } from '../api';
import type { HabitPhase, ImplementationIntention } from '../types';

export interface HabitSummary {
  phase: HabitPhase;
  streak: number;
  longest_streak: number;
  consistency_percentage: number;
  total_points: number;
  level: number;
  level_name: string;
  streak_days: number;
  pet_mood: string;
  today_points: number;
  today_checked_in: boolean;
}

export function fetchHabitSummary(): Promise<HabitSummary> {
  return api.get<HabitSummary>('/habits/summary');
}

export function fetchHabitPhase(): Promise<HabitPhase> {
  return api.get<HabitPhase>('/habits/phase');
}

export function habitCheckin(date?: string): Promise<void> {
  return api.post('/habits/checkin', date ? { date } : undefined);
}

export function fetchIntentions(): Promise<ImplementationIntention[]> {
  return api.get<ImplementationIntention[]>('/habits/intentions');
}

export function createIntention(
  intention: Omit<ImplementationIntention, 'id'>
): Promise<ImplementationIntention> {
  return api.post<ImplementationIntention>('/habits/intentions', intention);
}

export function updateIntention(
  id: string,
  updates: Partial<ImplementationIntention>
): Promise<ImplementationIntention> {
  return api.put<ImplementationIntention>(`/habits/intentions/${id}`, updates);
}

export function deleteIntention(id: string): Promise<void> {
  return api.del(`/habits/intentions/${id}`);
}
