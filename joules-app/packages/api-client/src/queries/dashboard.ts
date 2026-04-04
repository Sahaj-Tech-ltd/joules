import { api } from '../api';
import type { DashboardSummary } from '../types';

export function fetchDashboardSummary(date?: string): Promise<DashboardSummary> {
  const query = date ? `?date=${encodeURIComponent(date)}` : '';
  return api.get<DashboardSummary>(`/dashboard/summary${query}`);
}

export function markCheatDay(date?: string): Promise<void> {
  return api.post('/dashboard/cheat-day', date ? { date } : undefined);
}

export function unmarkCheatDay(date?: string): Promise<void> {
  return api.del(`/dashboard/cheat-day${date ? `?date=${encodeURIComponent(date)}` : ''}`);
}
