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

export function registerExpoPushToken(
  token: string,
  platform: 'ios' | 'android' | 'web' = 'ios'
): Promise<void> {
  return api.post('/notifications/register-expo', { token, platform });
}
