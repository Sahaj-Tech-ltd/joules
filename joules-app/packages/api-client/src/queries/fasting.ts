import { api } from '../api';
import type { FastingStatus } from '../types';

export function fetchFastingStatus(): Promise<FastingStatus> {
  return api.get<FastingStatus>('/fasting/status');
}

export function startFast(): Promise<FastingStatus> {
  return api.post<FastingStatus>('/fasting/start');
}

export function breakFast(): Promise<FastingStatus> {
  return api.post<FastingStatus>('/fasting/break');
}
