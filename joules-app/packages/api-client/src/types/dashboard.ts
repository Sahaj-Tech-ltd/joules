export interface DashboardSummary {
  total_calories?: number;
  calories_consumed?: number;
  calorie_target?: number;
  total_protein?: number;
  protein_consumed?: number;
  protein_target?: number;
  total_carbs?: number;
  carbs_consumed?: number;
  carbs_target?: number;
  total_fat?: number;
  fat_consumed?: number;
  fat_target?: number;
  total_water_ml?: number;
  water_ml?: number;
  exercises?: ExerciseEntry[];
  total_steps?: number;
  step_count?: number;
  meals?: MealSummary[];
  is_cheat_day?: boolean;
}

export interface MealSummary {
  id: string;
  timestamp: string;
  meal_type: string;
  total_calories: number;
  food_count: number;
}

export interface ExerciseEntry {
  id: string;
  name: string;
  duration_min: number;
  calories_burned: number;
  timestamp: string;
}

export interface WeightLog {
  id: string;
  date: string;
  weight_kg: number;
}

export interface WaterLog {
  id: string;
  date: string;
  amount_ml: number;
}
