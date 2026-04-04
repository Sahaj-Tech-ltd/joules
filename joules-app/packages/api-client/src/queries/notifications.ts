import { api } from '../api';
import type { NotificationPreferences } from '../types';

export function fetchNotificationPreferences(): Promise<NotificationPreferences> {
  return api.get<NotificationPreferences>('/notifications/preferences');
}

export function updateNotificationPreferences(
  prefs: Partial<NotificationPreferences>
): Promise<NotificationPreferences> {
  return api.put<NotificationPreferences>('/notifications/preferences', prefs);
}

export function registerExpoPushToken(token: string): Promise<void> {
  return api.post('/notifications/subscribe', { endpoint: token });
}
