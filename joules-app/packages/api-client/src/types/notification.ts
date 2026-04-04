export interface NotificationPreferences {
  water_reminders: boolean;
  water_interval_hours: number;
  meal_reminders: boolean;
  if_window_reminders: boolean;
  streak_reminders: boolean;
  quiet_start: number;
  quiet_end: number;
}

export interface HabitPhase {
  phase: 'scaffolding' | 'identity_building' | 'intrinsic' | 'maintenance';
  days_in_phase: number;
  total_days: number;
  next_phase_date: string | null;
  consistency_percentage: number;
  grace_days_used_this_week: number;
  grace_days_max_per_week: number;
}

export interface ImplementationIntention {
  id: string;
  meal_type: string;
  trigger_text: string;
  action_text: string;
  notification_time: string | null;
  enabled: boolean;
}
