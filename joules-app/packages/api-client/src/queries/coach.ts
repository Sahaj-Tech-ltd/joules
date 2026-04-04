import { api } from '../api';
import type { CoachMessage, CoachMemory, CoachReminder } from '../types';

export function fetchCoachMessages(limit?: number): Promise<CoachMessage[]> {
  const query = limit ? `?limit=${limit}` : '';
  return api.get<CoachMessage[]>(`/coach/chat${query}`);
}

export function sendCoachMessage(message: string): Promise<CoachMessage> {
  return api.post<CoachMessage>('/coach/chat', { message });
}

export function fetchCoachMemories(): Promise<CoachMemory[]> {
  return api.get<CoachMemory[]>('/user/coach-memories');
}

export function deleteCoachMemory(id: string): Promise<void> {
  return api.del(`/user/coach-memories/${id}`);
}

export function fetchCoachReminders(): Promise<CoachReminder[]> {
  return api.get<CoachReminder[]>('/coach/reminders');
}

export function updateCoachReminder(id: string, enabled: boolean): Promise<CoachReminder> {
  return api.put<CoachReminder>(`/coach/reminders/${id}`, { enabled });
}

export function deleteCoachReminder(id: string): Promise<void> {
  return api.del(`/coach/reminders/${id}`);
}

export function fetchCoachTips(): Promise<string[]> {
  return api.get<string[]>('/coach/tips');
}
