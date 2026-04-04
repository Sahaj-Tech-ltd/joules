export interface User {
  id: string;
  email: string;
  is_admin: boolean;
  verified: boolean;
  plan: 'free' | 'premium';
  plan_expires_at: string | null;
  trial_started_at: string | null;
}

export interface UserProfile {
  name: string;
  age: number | null;
  sex: string | null;
  height_cm: number | null;
  weight_kg: number | null;
  target_weight_kg: number | null;
  activity_level: string | null;
  onboarding_complete: boolean;
  avatar_url: string | null;
  identity_aspiration: string | null;
}

export interface UserGoals {
  objective: string;
  diet_plan: string;
  fasting_window: string | null;
  daily_calorie_target: number;
  daily_protein_g: number;
  daily_carbs_g: number;
  daily_fat_g: number;
  eating_window_start: string | null;
  fasting_streak: number;
}

export interface UserPreferences {
  diet_type: string;
  allergies: string[];
  food_notes: string;
  eating_context: string;
  height_unit: string;
  weight_unit: string;
  energy_unit: string;
  dietary_restrictions: string[];
}

export interface AuthResponse {
  access_token: string;
  user: User;
}

export interface FastingStatus {
  is_fasting: boolean;
  fast_start_time: string | null;
  eating_window_start: string;
  eating_window_hours: number;
  fasting_hours: number;
  seconds_elapsed: number;
  seconds_remaining: number;
  fasting_streak: number;
  fasting_window: string;
}
