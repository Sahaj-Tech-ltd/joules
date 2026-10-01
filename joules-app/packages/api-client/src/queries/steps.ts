import { api } from '../api';

export interface StepEntry {
  date: string;
  steps: number;
  step_count?: number;
  source: string;
}

export async function fetchSteps(date?: string): Promise<StepEntry> {
  const query = date ? `?date=${encodeURIComponent(date)}` : '';
  const data = await api.get<any>(`/steps${query}`);
  if (!data) {
    return {
      date: date || new Date().toISOString().slice(0, 10),
      steps: 0,
      source: 'Manual',
    };
  }
  return {
    date: data.date,
    steps: data.step_count ?? data.steps ?? 0,
    source: data.source || 'Manual',
  };
}

export async function logSteps(steps: number, date?: string): Promise<StepEntry> {
  const res = await api.post<any>('/steps', {
    step_count: steps,
    steps,
    ...(date ? { date } : {}),
  });
  return {
    date: res.date,
    steps: res.step_count ?? res.steps ?? steps,
    source: res.source || 'Manual',
  };
}

export function fetchStepHistory(days?: number): Promise<StepEntry[]> {
  const query = days ? `?days=${days}` : '';
  return api.get<StepEntry[]>(`/steps/history${query}`);
}
