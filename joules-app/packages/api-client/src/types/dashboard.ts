export interface DashboardSummary {
  calories_consumed: number;
  calorie_target: number;
  protein_consumed: number;
  protein_target: number;
  carbs_consumed: number;
  carbs_target: number;
  fat_consumed: number;
  fat_target: number;
  water_ml: number;
  exercises: ExerciseEntry[];
  step_count: number;
  meals: MealSummary[];
  is_cheat_day: boolean;
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
