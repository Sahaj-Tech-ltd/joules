import { api } from '../api';
import type { AuthResponse, UserProfile, UserGoals, UserPreferences, User } from '../types';

export function login(email: string, password: string): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/login', { email, password });
}

export function signup(email: string, password: string, name: string): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/signup', { email, password, name });
}

interface RefreshResponse {
  access_token: string;
}

export function refreshToken(): Promise<RefreshResponse> {
  return api.post<RefreshResponse>('/auth/refresh');
}

export function fetchCurrentUser(): Promise<User> {
  return api.get<User>('/auth/me');
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return api.put('/auth/password', {
    current_password: currentPassword,
    new_password: newPassword,
  });
}

export function fetchProfile(): Promise<UserProfile> {
  return api.get<UserProfile>('/user/profile');
}

export function updateProfile(profile: Partial<UserProfile>): Promise<UserProfile> {
  return api.put<UserProfile>('/user/profile', profile);
}

export function fetchGoals(): Promise<UserGoals> {
  return api.get<UserGoals>('/user/goals');
}

export function updateGoals(goals: Partial<UserGoals>): Promise<UserGoals> {
  return api.put<UserGoals>('/user/goals', goals);
}

export function fetchPreferences(): Promise<UserPreferences> {
  return api.get<UserPreferences>('/user/preferences');
}

export function updatePreferences(prefs: Partial<UserPreferences>): Promise<UserPreferences> {
  return api.put<UserPreferences>('/user/preferences', prefs);
}

export function completeOnboarding(data: Record<string, unknown>): Promise<void> {
  return api.post('/user/onboarding', data);
}
