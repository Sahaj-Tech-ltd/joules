import { api } from '../api';

export interface StepEntry {
  date: string;
  steps: number;
  source: string;
}

export function fetchSteps(date?: string): Promise<StepEntry> {
  const query = date ? `?date=${encodeURIComponent(date)}` : '';
  return api.get<StepEntry>(`/steps${query}`);
}

export function logSteps(steps: number, date?: string): Promise<StepEntry> {
  return api.post<StepEntry>('/steps/', {
    steps,
    ...(date ? { date } : {}),
  });
}

export function fetchStepHistory(days?: number): Promise<StepEntry[]> {
  const query = days ? `?days=${days}` : '';
  return api.get<StepEntry[]>(`/steps/history${query}`);
}
