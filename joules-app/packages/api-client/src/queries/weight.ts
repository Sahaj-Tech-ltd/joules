import { api } from '../api';
import type { WeightLog } from '../types';

export function fetchWeightLogs(days?: number): Promise<WeightLog[]> {
  const query = days ? `?days=${days}` : '';
  return api.get<WeightLog[]>(`/weight${query}`);
}

export function logWeight(weightKg: number, date?: string): Promise<WeightLog> {
  return api.post<WeightLog>('/weight/', {
    weight_kg: weightKg,
    ...(date ? { date } : {}),
  });
}
